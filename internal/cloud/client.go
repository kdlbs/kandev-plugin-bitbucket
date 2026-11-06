// Package cloud implements Bitbucket Cloud v2 API access.
package cloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"kandev-plugin-bitbucket/internal/domain"
)

const (
	defaultMaxResponseBytes      int64 = 1 << 20
	maxPageLength                      = 100
	maxPullRequestPageLength           = 50
	cloudRepositoryCursorVersion       = 1
)

type cloudRepositoryCursor struct {
	Version   int    `json:"version"`
	Scope     string `json:"scope"`
	Workspace string `json:"workspace"`
	Query     string `json:"query"`
	Limit     int    `json:"limit"`
	NextURL   string `json:"next_url"`
}

// TokenSource provides an OAuth access token or API token without exposing persistence.
type TokenSource interface {
	AccessToken(context.Context) (string, error)
}

type expiryAwareTokenSource interface {
	TokenSource
	AccessTokenCredential(context.Context) (string, time.Time, error)
}

// AuthenticationMode selects one Bitbucket Cloud authentication contract.
type AuthenticationMode string

const (
	AuthenticationOAuth       AuthenticationMode = "oauth"
	AuthenticationAccessToken AuthenticationMode = "access_token"
	AuthenticationAPIToken    AuthenticationMode = "api_token"
)

// Authentication contains non-secret Cloud authentication data.
// Email is required only for API-token Basic authentication.
type Authentication struct {
	Mode      AuthenticationMode
	Email     string
	Workspace string
}

// ClientOptions configures a Cloud client. APIBase exists for hermetic tests.
type ClientOptions struct {
	APIBase          *url.URL
	HTTPClient       *http.Client
	TokenSource      TokenSource
	Authentication   Authentication
	MaxResponseBytes int64
	RetryDelay       func(int, time.Duration) time.Duration
}

// Client calls the Bitbucket Cloud v2 API.
type Client struct {
	apiBase          *url.URL
	httpClient       *http.Client
	tokenSource      TokenSource
	authentication   Authentication
	maxResponseBytes int64
	retryDelay       func(int, time.Duration) time.Duration
}

// NewClient creates a Cloud API client with bounded responses and redirects.
func NewClient(options ClientOptions) *Client {
	apiBase := options.APIBase
	if apiBase == nil {
		apiBase, _ = url.Parse("https://api.bitbucket.org/2.0")
	}
	client := hardenedHTTPClient(options.HTTPClient, apiBase)
	maxResponseBytes := options.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = defaultMaxResponseBytes
	}
	retryDelay := options.RetryDelay
	if retryDelay == nil {
		retryDelay = defaultRetryDelay
	}
	authentication := options.Authentication
	if authentication.Mode == "" {
		authentication.Mode = AuthenticationOAuth
	}
	return &Client{
		apiBase:          copyURL(apiBase),
		httpClient:       client,
		tokenSource:      options.TokenSource,
		authentication:   authentication,
		maxResponseBytes: maxResponseBytes,
		retryDelay:       retryDelay,
	}
}

func (c *Client) authorize(request *http.Request, token string) error {
	switch c.authentication.Mode {
	case AuthenticationOAuth, AuthenticationAccessToken:
		request.Header.Set("Authorization", "Bearer "+token)
	case AuthenticationAPIToken:
		email := strings.TrimSpace(c.authentication.Email)
		if email == "" {
			return fmt.Errorf("Cloud API-token email is required")
		}
		request.SetBasicAuth(email, token)
	default:
		return fmt.Errorf("unsupported Cloud authentication mode")
	}
	return nil
}

func (c *Client) gitUsername() (string, error) {
	switch c.authentication.Mode {
	case AuthenticationOAuth, AuthenticationAccessToken:
		return "x-token-auth", nil
	case AuthenticationAPIToken:
		if strings.TrimSpace(c.authentication.Email) == "" {
			return "", fmt.Errorf("Cloud API-token email is required")
		}
		return "x-bitbucket-api-token-auth", nil
	default:
		return "", fmt.Errorf("unsupported Cloud authentication mode")
	}
}

func (c *Client) accessToken(ctx context.Context) (string, time.Time, error) {
	if source, ok := c.tokenSource.(expiryAwareTokenSource); ok {
		return source.AccessTokenCredential(ctx)
	}
	token, err := c.tokenSource.AccessToken(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(5 * time.Minute), nil
}

// ListRepositories follows v2 cursor pagination until limit repositories exist.
func (c *Client) ListRepositories(ctx context.Context, workspace string, limit int) ([]domain.Repository, error) {
	return c.listRepositories(ctx, workspace, "", limit)
}

// ListRepositoriesPage performs one server-filtered page request. Cursor is
// opaque to callers and bound to the configured API scope, workspace, query,
// and page size before its provider URL is followed.
func (c *Client) ListRepositoriesPage(ctx context.Context, workspace string, query domain.RepositoryQuery) (domain.RepositoryPage, error) {
	if !isPathSegment(workspace) || query.Limit <= 0 {
		return domain.RepositoryPage{}, fmt.Errorf("workspace and positive repository limit are required")
	}
	search := strings.TrimSpace(query.Text)
	endpoint := *c.apiBase
	endpoint.Path = path.Join(endpoint.Path, "repositories", workspace)
	endpoint.RawPath = ""
	if query.Cursor == "" {
		values := endpoint.Query()
		values.Set("pagelen", fmt.Sprintf("%d", min(query.Limit, maxPageLength)))
		if search != "" {
			values.Set("q", `name ~ "`+escapeQueryLiteral(search)+`"`)
		}
		endpoint.RawQuery = values.Encode()
	} else {
		nextURL, err := c.parseRepositoryCursor(query.Cursor, workspace, search, min(query.Limit, maxPageLength))
		if err != nil {
			return domain.RepositoryPage{}, err
		}
		parsed, err := c.nextURL(&endpoint, nextURL)
		if err != nil || parsed == nil || parsed.Path != endpoint.Path {
			return domain.RepositoryPage{}, fmt.Errorf("invalid Bitbucket Cloud repository cursor")
		}
		endpoint = *parsed
	}
	payload, err := c.repositoryPage(ctx, &endpoint)
	if err != nil {
		return domain.RepositoryPage{}, err
	}
	repositories := make([]domain.Repository, 0, min(len(payload.Values), query.Limit))
	for _, value := range payload.Values {
		mapped, err := c.mapRepository(workspace, value)
		if err != nil {
			return domain.RepositoryPage{}, err
		}
		repositories = append(repositories, mapped)
		if len(repositories) == query.Limit {
			break
		}
	}
	next, err := c.nextURL(&endpoint, payload.Next)
	if err != nil {
		return domain.RepositoryPage{}, err
	}
	nextCursor := ""
	if next != nil {
		nextCursor = c.encodeRepositoryCursor(workspace, search, min(query.Limit, maxPageLength), next.String())
	}
	return domain.RepositoryPage{Repositories: repositories, NextCursor: nextCursor}, nil
}

func (c *Client) encodeRepositoryCursor(workspace, query string, limit int, nextURL string) string {
	payload, _ := json.Marshal(cloudRepositoryCursor{
		Version: cloudRepositoryCursorVersion, Scope: c.apiBase.String(),
		Workspace: workspace, Query: query, Limit: limit, NextURL: nextURL,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func (c *Client) parseRepositoryCursor(raw, workspace, query string, limit int) (string, error) {
	if len(raw) > 8192 {
		return "", fmt.Errorf("invalid Bitbucket Cloud repository cursor")
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("invalid Bitbucket Cloud repository cursor")
	}
	var cursor cloudRepositoryCursor
	if err := json.Unmarshal(payload, &cursor); err != nil ||
		cursor.Version != cloudRepositoryCursorVersion ||
		cursor.Scope != c.apiBase.String() || cursor.Workspace != workspace ||
		cursor.Query != query || cursor.Limit != limit || cursor.NextURL == "" {
		return "", fmt.Errorf("invalid Bitbucket Cloud repository cursor")
	}
	return cursor.NextURL, nil
}

// SearchRepositories delegates filtering to Bitbucket before following its
// opaque cursor. This keeps matches beyond the first UI page discoverable.
func (c *Client) SearchRepositories(ctx context.Context, workspace, query string, limit int) ([]domain.Repository, error) {
	return c.listRepositories(ctx, workspace, strings.TrimSpace(query), limit)
}

func (c *Client) listRepositories(ctx context.Context, workspace, search string, limit int) ([]domain.Repository, error) {
	if !isPathSegment(workspace) {
		return nil, fmt.Errorf("workspace must be a URL path segment")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("repository limit must be positive")
	}

	var repositories []domain.Repository
	cursor := ""
	for len(repositories) < limit {
		page, err := c.ListRepositoriesPage(ctx, workspace, domain.RepositoryQuery{
			Text: search, Limit: min(limit, maxPageLength), Cursor: cursor,
		})
		if err != nil {
			return nil, err
		}
		for _, repository := range page.Repositories {
			repositories = append(repositories, repository)
			if len(repositories) == limit {
				break
			}
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	return repositories, nil
}

func escapeQueryLiteral(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

type repositoryPage struct {
	Values []repositoryPayload `json:"values"`
	Next   string              `json:"next"`
}

type repositoryPayload struct {
	UUID       string `json:"uuid"`
	Slug       string `json:"slug"`
	MainBranch struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Links struct {
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func (c *Client) repositoryPage(ctx context.Context, endpoint *url.URL) (repositoryPage, error) {
	if c.tokenSource == nil {
		return repositoryPage{}, fmt.Errorf("Cloud token source is not configured")
	}
	token, _, err := c.accessToken(ctx)
	if err != nil {
		return repositoryPage{}, fmt.Errorf("resolve Cloud access token: %w", err)
	}
	for attempt := 0; attempt < 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return repositoryPage{}, err
		}
		request.Header.Set("Accept", "application/json")
		if err := c.authorize(request, token); err != nil {
			return repositoryPage{}, err
		}
		response, err := c.httpClient.Do(request)
		if err != nil {
			return repositoryPage{}, fmt.Errorf("request Bitbucket Cloud: %w", err)
		}
		if (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError) && attempt < 2 {
			delay := c.retryDelay(attempt, retryAfter(response.Header.Get("Retry-After")))
			response.Body.Close()
			if err := wait(ctx, delay); err != nil {
				return repositoryPage{}, err
			}
			continue
		}
		if response.StatusCode != http.StatusOK {
			responseErr := providerResponseError(response)
			response.Body.Close()
			return repositoryPage{}, responseErr
		}
		body, err := readBounded(response.Body, c.maxResponseBytes)
		response.Body.Close()
		if err != nil {
			return repositoryPage{}, err
		}
		var page repositoryPage
		if err := json.Unmarshal(body, &page); err != nil {
			return repositoryPage{}, fmt.Errorf("decode Bitbucket Cloud repositories: %w", err)
		}
		return page, nil
	}
	return repositoryPage{}, fmt.Errorf("Bitbucket Cloud repository retry limit exceeded")
}

func (c *Client) nextURL(current *url.URL, rawNext string) (*url.URL, error) {
	if rawNext == "" {
		return nil, nil
	}
	parsed, err := url.Parse(rawNext)
	if err != nil {
		return nil, fmt.Errorf("parse Bitbucket Cloud next page: %w", err)
	}
	next := current.ResolveReference(parsed)
	if !validCloudAPIURL(next, c.apiBase) {
		return nil, fmt.Errorf("Bitbucket Cloud next page is outside the API origin or context")
	}
	return next, nil
}

func (c *Client) mapRepository(workspace string, payload repositoryPayload) (domain.Repository, error) {
	if !isPathSegment(workspace) || !isPathSegment(payload.Slug) {
		return domain.Repository{}, fmt.Errorf("Cloud repository workspace and slug must be URL path segments")
	}
	for _, clone := range payload.Links.Clone {
		if clone.Name != "https" {
			continue
		}
		cloneURL, err := url.Parse(clone.Href)
		expectedPath := "/" + path.Join(workspace, payload.Slug+".git")
		if err != nil || cloneURL.Scheme != "https" || cloneURL.Host != "bitbucket.org" || cloneURL.RawQuery != "" || cloneURL.Fragment != "" || cloneURL.Path != expectedPath {
			return domain.Repository{}, fmt.Errorf("Cloud repository has invalid HTTPS clone URL")
		}
		// Bitbucket Cloud currently includes the authenticated account name as
		// URL userinfo in otherwise canonical HTTPS clone links. Never propagate
		// it into host repository data; reconstruct the validated credential-free
		// URL instead.
		cloneURL = &url.URL{Scheme: "https", Host: "bitbucket.org", Path: expectedPath}
		if strings.TrimSpace(payload.UUID) == "" {
			return domain.Repository{}, fmt.Errorf("Cloud repository has no immutable UUID")
		}
		return domain.Repository{ID: payload.UUID, ProviderScope: "https://bitbucket.org", Namespace: workspace, Slug: payload.Slug, CloneURL: cloneURL, DefaultBranch: payload.MainBranch.Name}, nil
	}
	return domain.Repository{}, fmt.Errorf("Cloud repository has no HTTPS clone URL")
}

func hardenedHTTPClient(base *http.Client, origin *url.URL) *http.Client {
	if base == nil {
		base = &http.Client{}
	}
	client := *base
	if client.Timeout == 0 {
		client.Timeout = 15 * time.Second
	}
	client.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
		if !validCloudAPIURL(request.URL, origin) {
			return fmt.Errorf("redirect changed Bitbucket API origin")
		}
		return nil
	}
	return &client
}

func validCloudAPIURL(value, apiBase *url.URL) bool {
	if value.User != nil || value.Fragment != "" || !sameOrigin(value, apiBase) || value.Path != path.Clean(value.Path) {
		return false
	}
	basePath := path.Clean(apiBase.Path)
	return value.Path == basePath || strings.HasPrefix(value.Path, basePath+"/")
}

func readBounded(body io.Reader, maxBytes int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("Bitbucket response exceeds %d-byte limit", maxBytes)
	}
	return data, nil
}

func sameOrigin(left, right *url.URL) bool {
	return strings.EqualFold(left.Scheme, right.Scheme) && strings.EqualFold(left.Host, right.Host)
}

func copyURL(value *url.URL) *url.URL {
	copy := *value
	return &copy
}

func isPathSegment(value string) bool {
	return value != "" && !strings.ContainsAny(value, "/\\?#")
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func defaultRetryDelay(attempt int, retryAfterDelay time.Duration) time.Duration {
	if retryAfterDelay > 0 {
		return retryAfterDelay
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func retryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

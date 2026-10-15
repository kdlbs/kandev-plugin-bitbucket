package cloud

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"kandev-plugin-bitbucket/internal/domain"
	"kandev-plugin-bitbucket/internal/redact"
)

const sourcePageLimit = 20
const sourceLogLimit = 64 << 10

var sourceSegment = regexp.MustCompile(`^[A-Za-z0-9_{}.-]+$`)
var sourceHash = regexp.MustCompile(`^(?:[a-fA-F0-9]{40}|[a-fA-F0-9]{64})$`)

// SourceInput contains no URL, identity, credential, branch, or target selector.
type SourceInput struct {
	Operation   string `json:"operation"`
	Number      int    `json:"number,omitempty"`
	PipelineID  string `json:"pipeline_id,omitempty"`
	StepID      string `json:"step_id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// SourceOperation executes one bounded operation against a host-derived checkout.
func (c *Client) SourceOperation(ctx context.Context, repository domain.Repository, source, destination string, input SourceInput) (map[string]any, error) {
	if !validSourceSegment(repository.Namespace) || !validSourceSegment(repository.Slug) || repository.ID == "" || repository.ProviderScope != "https://bitbucket.org" || !validSourceBranch(source) || !validSourceBranch(destination) {
		return nil, fmt.Errorf("bitbucket_source_scope_denied")
	}
	if err := c.validateSourceAuthentication(); err != nil {
		return nil, err
	}
	if err := validateSourceInput(input); err != nil {
		return nil, err
	}
	switch input.Operation {
	case "pr_lookup", "pr_get", "pr_create":
		return c.sourcePullRequest(ctx, repository, source, destination, input)
	case "pipelines", "pipeline_get", "pipeline_steps", "pipeline_log", "statuses":
		return c.sourcePipeline(ctx, repository, source, input)
	default:
		return nil, fmt.Errorf("bitbucket_source_operation_unsupported")
	}
}

func validSourceSegment(value string) bool {
	return len(value) <= 128 && sourceSegment.MatchString(value) && value != "." && value != ".."
}
func validSourceBranch(value string) bool {
	return value != "" && len(value) <= 255 && !strings.ContainsAny(value, "\x00\r\n\\?#") && !strings.Contains(value, "..")
}
func (c *Client) validateSourceAuthentication() error {
	switch c.authentication.Mode {
	case AuthenticationOAuth:
		return nil
	case AuthenticationAPIToken:
		email := strings.TrimSpace(c.authentication.Email)
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email || strings.ContainsAny(email, "\r\n:") {
			return fmt.Errorf("bitbucket_registered_email_required")
		}
		return nil
	default:
		return fmt.Errorf("bitbucket_api_auth_kind_unsupported")
	}
}

func (c *Client) sourceEndpoint(repository domain.Repository, elements ...string) *url.URL {
	endpoint := *c.apiBase
	segments := append([]string{"repositories", repository.Namespace, repository.Slug}, elements...)
	escaped := strings.TrimSuffix(endpoint.EscapedPath(), "/")
	for _, segment := range segments {
		escaped += "/" + url.PathEscape(segment)
	}
	endpoint.Path, _ = url.PathUnescape(escaped)
	endpoint.RawPath = escaped
	endpoint.RawQuery = ""
	return &endpoint
}

// sourceRequest never follows redirects, returns provider error bodies, or retries writes.
func (c *Client) sourceRequest(ctx context.Context, method string, endpoint *url.URL, input any, log bool) ([]byte, bool, error) {
	if !validCloudAPIURL(endpoint, c.apiBase) || c.tokenSource == nil {
		return nil, false, fmt.Errorf("bitbucket_source_scope_denied")
	}
	token, expiry, err := c.accessToken(ctx)
	if err != nil || token == "" || (!expiry.IsZero() && !expiry.After(time.Now())) {
		return nil, false, fmt.Errorf("bitbucket_api_credential_unavailable")
	}
	var payload io.Reader
	if input != nil {
		encoded, encodeErr := json.Marshal(input)
		if encodeErr != nil {
			return nil, false, fmt.Errorf("bitbucket_source_input_invalid")
		}
		payload = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), payload)
	if err != nil {
		return nil, false, fmt.Errorf("bitbucket_source_input_invalid")
	}
	request.Header.Set("Accept", "application/json")
	if log {
		request.Header.Set("Accept", "text/plain")
	}
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if err = c.authorize(request, token); err != nil {
		return nil, false, fmt.Errorf("bitbucket_registered_email_required")
	}
	client := *c.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		return nil, false, fmt.Errorf("bitbucket_rest_transport_unavailable")
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case 401:
		return nil, false, fmt.Errorf("bitbucket_api_identity_unauthorized")
	case 403:
		return nil, false, fmt.Errorf("bitbucket_api_permission_denied")
	case 404:
		return nil, false, fmt.Errorf("bitbucket_rest_resource_unavailable")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, false, fmt.Errorf("bitbucket_rest_status_%d", response.StatusCode)
	}
	limit := min(int(c.maxResponseBytes), 1<<20)
	if log {
		limit = sourceLogLimit
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil {
		return nil, false, fmt.Errorf("bitbucket_rest_read_failed")
	}
	truncated := len(body) > limit
	if truncated && !log {
		return nil, false, fmt.Errorf("bitbucket_source_response_limit")
	}
	if truncated {
		body = body[:limit]
	}
	// Include JSON-escaped and encoded variants before any data leaves the subprocess.
	secretJSON, _ := json.Marshal(token)
	redactor := redact.New(token, string(secretJSON[1:len(secretJSON)-1]), url.QueryEscape(token), base64.StdEncoding.EncodeToString([]byte(token)), strings.TrimPrefix(request.Header.Get("Authorization"), "Basic "), request.Header.Get("Authorization"))
	return []byte(redactor.String(string(body))), truncated, nil
}

func (c *Client) sourceJSON(ctx context.Context, method string, endpoint *url.URL, input, target any) error {
	body, _, err := c.sourceRequest(ctx, method, endpoint, input, false)
	if err != nil {
		return err
	}
	if json.Unmarshal(body, target) != nil {
		return fmt.Errorf("bitbucket_rest_response_invalid")
	}
	return nil
}

func (c *Client) sourcePullRequest(ctx context.Context, repository domain.Repository, source, destination string, input SourceInput) (map[string]any, error) {
	if input.Operation == "pr_get" {
		if input.Number <= 0 {
			return nil, fmt.Errorf("bitbucket_pr_number_required")
		}
		var pr pullRequestPayload
		err := c.sourceJSON(ctx, http.MethodGet, c.sourceEndpoint(repository, "pullrequests", fmt.Sprint(input.Number)), nil, &pr)
		if err != nil {
			return nil, err
		}
		if pr.ID != input.Number || !sourcePRMatches(pr, repository, source, destination) {
			return nil, fmt.Errorf("bitbucket_pr_source_mismatch")
		}
		return sourcePRView(pr, false), nil
	}
	endpoint := c.sourceEndpoint(repository, "pullrequests")
	query := endpoint.Query()
	query.Set("pagelen", fmt.Sprint(sourcePageLimit))
	query.Set("q", `state = "OPEN" AND source.branch.name = "`+escapeQueryLiteral(source)+`" AND destination.branch.name = "`+escapeQueryLiteral(destination)+`"`)
	endpoint.RawQuery = query.Encode()
	var page pullRequestPage
	if err := c.sourceJSON(ctx, http.MethodGet, endpoint, nil, &page); err != nil {
		return nil, err
	}
	if page.Values == nil || page.Next != "" || len(page.Values) > sourcePageLimit {
		return nil, fmt.Errorf("bitbucket_pr_lookup_incomplete")
	}
	var matches []pullRequestPayload
	for _, pr := range page.Values {
		if pr.ID <= 0 || pr.State == "" || pr.Source.Repository == nil || pr.Source.Repository.UUID == "" || pr.Source.Branch.Name == "" || pr.Destination.Branch.Name == "" {
			return nil, fmt.Errorf("bitbucket_pr_lookup_incomplete")
		}
		if sourcePRMatches(pr, repository, source, destination) && pr.State == "OPEN" {
			matches = append(matches, pr)
		}
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("bitbucket_pr_source_ambiguous")
	}
	if len(matches) == 1 {
		return sourcePRView(matches[0], true), nil
	}
	if input.Operation == "pr_lookup" {
		return map[string]any{"found": false}, nil
	}
	if strings.TrimSpace(input.Title) == "" || len(input.Title) > 512 || len(input.Description) > 32768 {
		return nil, fmt.Errorf("bitbucket_pr_presentation_invalid")
	}
	body := map[string]any{"title": input.Title, "description": input.Description, "source": map[string]any{"branch": map[string]string{"name": source}}, "destination": map[string]any{"branch": map[string]string{"name": destination}}, "close_source_branch": false}
	var pr pullRequestPayload
	if err := c.sourceJSON(ctx, http.MethodPost, c.sourceEndpoint(repository, "pullrequests"), body, &pr); err != nil {
		return nil, err
	}
	if !sourcePRMatches(pr, repository, source, destination) {
		return nil, fmt.Errorf("bitbucket_pr_source_mismatch")
	}
	return sourcePRView(pr, false), nil
}
func sourcePRMatches(pr pullRequestPayload, repository domain.Repository, source, destination string) bool {
	return pr.ID > 0 && pr.Source.Branch.Name == source && pr.Destination.Branch.Name == destination && pr.Source.Repository != nil && pr.Source.Repository.UUID == repository.ID
}
func sourcePRView(pr pullRequestPayload, reused bool) map[string]any {
	return map[string]any{"found": true, "number": pr.ID, "title": pr.Title, "description": pr.Description, "state": pr.State, "source_branch": pr.Source.Branch.Name, "target_branch": pr.Destination.Branch.Name, "source_head": pr.Source.Commit.Hash, "reused": reused}
}

func validateSourceInput(input SourceInput) error {
	switch input.Operation {
	case "pr_lookup", "pr_create":
	case "pr_get":
		if input.Number <= 0 {
			return fmt.Errorf("bitbucket_pr_number_required")
		}
	case "pipelines", "statuses":
	case "pipeline_get", "pipeline_steps", "pipeline_log":
		if !validSourceSegment(input.PipelineID) {
			return fmt.Errorf("bitbucket_pipeline_id_required")
		}
		if input.Operation == "pipeline_log" && !validSourceSegment(input.StepID) {
			return fmt.Errorf("bitbucket_step_id_required")
		}
	default:
		return fmt.Errorf("bitbucket_source_operation_unsupported")
	}
	if len(input.Title) > 512 || len(input.Description) > 32768 {
		return fmt.Errorf("bitbucket_pr_presentation_invalid")
	}
	return nil
}

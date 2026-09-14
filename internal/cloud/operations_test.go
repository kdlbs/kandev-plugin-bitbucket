package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"kandev-plugin-bitbucket/internal/domain"
)

func TestCloudListsBranchesAndPullRequestsWithV2QueryPagination(t *testing.T) {
	branches, err := os.ReadFile("testdata/branches.json")
	require.NoError(t, err)
	pullRequests, err := os.ReadFile("testdata/pullrequests.json")
	require.NoError(t, err)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer cloud-token", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/2.0/repositories/acme/widgets/refs/branches":
			require.Equal(t, "2", r.URL.Query().Get("pagelen"))
			require.Equal(t, `name~"feat"`, r.URL.Query().Get("q"))
			_, _ = w.Write(branches)
		case "/2.0/repositories/acme/widgets/pullrequests":
			require.Equal(t, "OPEN", r.URL.Query().Get("state"))
			require.Equal(t, `title~"race"`, r.URL.Query().Get("q"))
			_, _ = w.Write(pullRequests)
		default:
			t.Fatalf("unexpected Cloud path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, TokenSource: staticTokenSource("cloud-token")})
	repository := domain.Repository{Namespace: "acme", Slug: "widgets"}

	gotBranches, err := client.listBranches(context.Background(), repository, "feat", 2)
	require.NoError(t, err)
	require.Equal(t, []domain.Branch{{Name: "feature/race", Commit: "abc123"}}, gotBranches)

	gotPullRequests, err := client.listPullRequests(context.Background(), repository, "race", 2)
	require.NoError(t, err)
	require.Len(t, gotPullRequests, 1)
	require.Equal(t, "acme/widgets#42", gotPullRequests[0].Key())
	require.Equal(t, "feature/race", gotPullRequests[0].Source.Name)
	require.Equal(t, domain.Repository{ID: "{repo-forked-widgets}", ProviderScope: "https://bitbucket.org", Namespace: "forker", Slug: "forked-widgets", CloneURL: mustURL(t, "https://bitbucket.org/forker/forked-widgets.git")}, gotPullRequests[0].SourceRepository)
	require.Equal(t, "main", gotPullRequests[0].Repository.DefaultBranch)
	require.Equal(t, "https://bitbucket.org/acme/widgets.git", gotPullRequests[0].Repository.CloneURL.String())
}

func TestCloudListBranchesPagesThroughAllBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer cloud-token", r.Header.Get("Authorization"))
		require.Equal(t, "/2.0/repositories/acme/widgets/refs/branches", r.URL.Path)
		require.Equal(t, "100", r.URL.Query().Get("pagelen"))
		switch r.URL.Query().Get("page") {
		case "":
			_, _ = w.Write([]byte(`{"values":[{"name":"one","target":{"hash":"hash-1"}},{"name":"two","target":{"hash":"hash-2"}}],"next":"/2.0/repositories/acme/widgets/refs/branches?page=2&pagelen=100"}`))
		case "2":
			_, _ = w.Write([]byte(`{"values":[{"name":"three","target":{"hash":"hash-3"}}],"next":"/2.0/repositories/acme/widgets/refs/branches?page=3&pagelen=100"}`))
		case "3":
			_, _ = w.Write([]byte(`{"values":[{"name":"four","target":{"hash":"hash-4"}}]}`))
		default:
			t.Fatalf("unexpected branch page %q", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, TokenSource: staticTokenSource("cloud-token")})

	branches, err := client.ListBranches(context.Background(), domain.Repository{Namespace: "acme", Slug: "widgets"})
	require.NoError(t, err)
	require.Equal(t, []domain.Branch{
		{Name: "one", Commit: "hash-1"},
		{Name: "two", Commit: "hash-2"},
		{Name: "three", Commit: "hash-3"},
		{Name: "four", Commit: "hash-4"},
	}, branches)
}

func TestCloudListBranchesRejectsEndlessPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/2.0/repositories/acme/widgets/refs/branches", r.URL.Path)
		_, _ = w.Write([]byte(`{"values":[{"name":"one","target":{"hash":"hash-1"}}],"next":"/2.0/repositories/acme/widgets/refs/branches?page=2"}`))
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, TokenSource: staticTokenSource("cloud-token")})

	_, err = client.ListBranches(context.Background(), domain.Repository{Namespace: "acme", Slug: "widgets"})
	require.ErrorContains(t, err, "Cloud branch pagination limit exceeded")
}

func TestCloudSearchPullRequestsPagePreservesProviderCursorAndAuthor(t *testing.T) {
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Query().Get("page") == "2" {
			_, _ = w.Write([]byte(`{"values":[{"id":2,"title":"Two","state":"OPEN","author":{"account_id":"account-2"},"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/2"}},"source":{"branch":{"name":"two"}},"destination":{"branch":{"name":"main"},"repository":{"mainbranch":{"name":"main"}}}}]}`))
			return
		}
		require.Equal(t, "OPEN", r.URL.Query().Get("state"))
		require.Equal(t, "50", r.URL.Query().Get("pagelen"))
		next := fmt.Sprintf("%s/2.0/repositories/acme/widgets/pullrequests?page=2&pagelen=50&state=OPEN", server.URL)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"values":[{"id":1,"title":"One","state":"OPEN","author":{"account_id":"account-1"},"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/1"}},"source":{"branch":{"name":"one"}},"destination":{"branch":{"name":"main"},"repository":{"mainbranch":{"name":"main"}}}}],"next":%q}`, next)))
	}))
	defer server.Close()
	base, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: base, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})
	query := domain.PullRequestQuery{Repository: domain.Repository{ID: "repo-uuid", ProviderScope: "https://bitbucket.org", Namespace: "acme", Slug: "widgets"}, State: "OPEN", Limit: 100}

	first, err := client.SearchPullRequestsPage(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "account-1", first.PullRequests[0].Author)
	require.NotEmpty(t, first.NextCursor)
	query.Cursor = first.NextCursor
	second, err := client.SearchPullRequestsPage(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "account-2", second.PullRequests[0].Author)
	require.Empty(t, second.NextCursor)
	require.Equal(t, 2, requests)
}

func TestCloudSearchPullRequestsPageCursorIsBoundToImmutableRepositoryAndQuery(t *testing.T) {
	requests := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		next := fmt.Sprintf("%s/2.0/repositories/acme/widgets/pullrequests?page=2&pagelen=50&q=title%%7E%%22race%%22&state=OPEN", server.URL)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"values":[],"next":%q}`, next)))
	}))
	defer server.Close()
	base, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: base, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})
	query := domain.PullRequestQuery{
		Repository: domain.Repository{ID: "repo-original", ProviderScope: "https://bitbucket.org", Namespace: "acme", Slug: "widgets"},
		Text:       "race", State: "OPEN", Limit: 100,
	}
	page, err := client.SearchPullRequestsPage(context.Background(), query)
	require.NoError(t, err)
	require.NotEmpty(t, page.NextCursor)

	recreated := query
	recreated.Repository.ID = "repo-recreated"
	recreated.Cursor = page.NextCursor
	_, err = client.SearchPullRequestsPage(context.Background(), recreated)
	require.ErrorContains(t, err, "cursor")
	changedQuery := query
	changedQuery.Text = "other"
	changedQuery.Cursor = page.NextCursor
	_, err = client.SearchPullRequestsPage(context.Background(), changedQuery)
	require.ErrorContains(t, err, "cursor")
	require.Equal(t, 1, requests, "invalid cursors must fail before any provider request")
}

func TestCloudMapsPullRequestDisplayAuthorAndCreatedTime(t *testing.T) {
	var payload pullRequestPayload
	require.NoError(t, json.Unmarshal([]byte(`{
		"id":42,"title":"Fix race","state":"OPEN",
		"author":{"account_id":"account-1","display_name":"Ada Lovelace"},
		"created_on":"2026-07-31T12:00:00Z",
		"links":{"html":{"href":"https://bitbucket.org/acme/widgets/pull-requests/42"}},
		"source":{"branch":{"name":"feature/race"}},
		"destination":{"branch":{"name":"main"},"repository":{"mainbranch":{"name":"main"}}}
	}`), &payload))

	pullRequest, err := mapPullRequest(domain.Repository{Namespace: "acme", Slug: "widgets"}, payload)
	require.NoError(t, err)
	require.Equal(t, "account-1", pullRequest.Author)
	require.Equal(t, "Ada Lovelace", pullRequest.AuthorDisplayName)
	require.Equal(t, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC), pullRequest.CreatedAt)
}

func TestCloudSearchPullRequestsPageAllStatesUsesOpaqueStateContinuation(t *testing.T) {
	states := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		states = append(states, r.URL.Query().Get("state"))
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	defer server.Close()
	base, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: base, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})
	query := domain.PullRequestQuery{Repository: domain.Repository{ID: "repo-uuid", ProviderScope: "https://bitbucket.org", Namespace: "acme", Slug: "widgets"}, State: "ALL", Limit: 100}

	for index := 0; index < 4; index++ {
		page, err := client.SearchPullRequestsPage(context.Background(), query)
		require.NoError(t, err)
		query.Cursor = page.NextCursor
	}
	require.Empty(t, query.Cursor)
	require.Equal(t, []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"}, states)
}

func TestCloudSearchPullRequestsUsesRequestedState(t *testing.T) {
	pullRequests, err := os.ReadFile("testdata/pullrequests.json")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "MERGED", r.URL.Query().Get("state"))
		_, _ = w.Write(pullRequests)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})

	_, err = client.SearchPullRequests(context.Background(), domain.PullRequestQuery{Repository: domain.Repository{Namespace: "acme", Slug: "widgets"}, State: "MERGED", Limit: 1})
	require.NoError(t, err)
}

func TestCloudSearchPullRequestsExpandsAllStates(t *testing.T) {
	pullRequests, err := os.ReadFile("testdata/pullrequests.json")
	require.NoError(t, err)
	states := make([]string, 0, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		states = append(states, r.URL.Query().Get("state"))
		_, _ = w.Write(pullRequests)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})

	_, err = client.SearchPullRequests(context.Background(), domain.PullRequestQuery{Repository: domain.Repository{Namespace: "acme", Slug: "widgets"}, State: "ALL", Limit: 1})
	require.NoError(t, err)
	require.Equal(t, []string{"OPEN", "MERGED", "DECLINED", "SUPERSEDED"}, states)
}

func TestCloudGetReviewMapsGoldenReviewData(t *testing.T) {
	responses := map[string]string{
		"/2.0/user": "review-current-user.json",
		"/2.0/repositories/acme/widgets/pullrequests/42":             "review-pullrequest.json",
		"/2.0/repositories/acme/widgets/pullrequests/42/commits":     "review-commits.json",
		"/2.0/repositories/acme/widgets/pullrequests/42/comments":    "review-comments.json",
		"/2.0/repositories/acme/widgets/commit/source-hash/statuses": "review-statuses.json",
		"/2.0/repositories/acme/widgets/pullrequests/42/diff":        "review.diff",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer cloud-token", r.Header.Get("Authorization"))
		fixture, ok := responses[r.URL.Path]
		if r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/42/commits" || r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/42/comments" || r.URL.Path == "/2.0/repositories/acme/widgets/commit/source-hash/statuses" {
			if r.URL.Query().Get("page") == "2" {
				fixture += "-second.json"
			} else {
				require.Equal(t, "100", r.URL.Query().Get("pagelen"))
			}
		}
		if r.URL.Path == "/2.0/repositories/acme/widgets/pullrequests/42/diffstat" {
			if r.URL.Query().Get("page") == "2" {
				fixture = "review-diffstat-second.json"
			} else {
				require.Equal(t, "100", r.URL.Query().Get("pagelen"))
				fixture = "review-diffstat-first.json"
			}
			ok = true
		}
		if !ok {
			t.Fatalf("unexpected Cloud path %s", r.URL.Path)
		}
		body, err := os.ReadFile("testdata/" + fixture)
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})

	review, err := client.GetReview(context.Background(), domain.Repository{Namespace: "acme", Slug: "widgets"}, 42)
	require.NoError(t, err)
	require.Equal(t, "diff --git a/race.go b/race.go\nindex 1111111..2222222 100644\n--- a/race.go\n+++ b/race.go\n@@ -1 +1 @@\n-unsafe()\n+safe()\ndiff --git a/README.md b/README.md\nnew file mode 100644\n--- /dev/null\n+++ b/README.md\n@@ -0,0 +1 @@\n+safe usage\n", review.Diff)
	require.Equal(t, []domain.Commit{{Hash: "source-hash", Message: "Fix race", Author: "Ada <ada@example.test>"}, {Hash: "source-hash-2", Message: "Add coverage", Author: "Bob <bob@example.test>"}}, review.Commits)
	require.Equal(t, []domain.Participant{{ID: "user-1", Name: "Ada", Role: "REVIEWER", Approved: true, Verdict: domain.ReviewVerdictApproved}}, review.Participants)
	require.Equal(t, "user-1", review.ViewerID)
	require.Equal(t, []domain.Thread{{ID: "10", Comments: []domain.Comment{
		{ID: "10", Author: "Ada", Body: "Please add a test.", When: time.Date(2026, time.July, 31, 12, 1, 0, 0, time.UTC)},
		{ID: "11", ParentID: "10", Author: "Bob", Body: "Done.", When: time.Date(2026, time.July, 31, 12, 2, 0, 0, time.UTC)},
	}}}, review.Threads)
	require.Equal(t, []domain.BuildStatus{{Key: "build-7", Name: "CI", State: "SUCCESSFUL", URL: "https://ci.example.test/build/7", Target: "source-hash"}, {Key: "build-8", Name: "Lint", State: "FAILED", URL: "https://ci.example.test/build/8", Target: "source-hash"}}, review.Statuses)
	require.Equal(t, []domain.ReviewFile{
		{Path: "race.go", Status: "modified", Additions: 1, Deletions: 1, Patch: "diff --git a/race.go b/race.go\nindex 1111111..2222222 100644\n--- a/race.go\n+++ b/race.go\n@@ -1 +1 @@\n-unsafe()\n+safe()\n"},
		{Path: "README.md", Status: "added", Additions: 1, Patch: "diff --git a/README.md b/README.md\nnew file mode 100644\n--- /dev/null\n+++ b/README.md\n@@ -0,0 +1 @@\n+safe usage\n"},
	}, review.Files)
	require.Equal(t, "main", review.PullRequest.Repository.DefaultBranch)
	require.Equal(t, domain.Repository{ID: "{repo-forked-widgets}", ProviderScope: "https://bitbucket.org", Namespace: "forker", Slug: "forked-widgets", CloneURL: mustURL(t, "https://bitbucket.org/forker/forked-widgets.git")}, review.PullRequest.SourceRepository)
}

func TestCloudProjectedReviewSkipsDiffFilesCommitsCommentsAndViewer(t *testing.T) {
	pullRequest, err := os.ReadFile("testdata/review-pullrequest.json")
	require.NoError(t, err)
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/2.0/repositories/acme/widgets/pullrequests/42":
			_, _ = w.Write(pullRequest)
		case "/2.0/repositories/acme/widgets/commit/source-hash/statuses":
			_, _ = w.Write([]byte(`{"values":[{"key":"ci","name":"CI","state":"SUCCESSFUL"}]}`))
		default:
			t.Fatalf("unexpected heavy Cloud review request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	baseURL, err := url.Parse(server.URL + "/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: baseURL, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})

	review, err := client.GetReviewProjected(context.Background(), domain.Repository{Namespace: "acme", Slug: "widgets"}, 42, domain.ReviewProjection{
		Participants: true, Statuses: true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, review.Participants)
	require.NotEmpty(t, review.Statuses)
	require.Empty(t, review.Diff)
	require.Empty(t, review.Files)
	require.Empty(t, review.Commits)
	require.Empty(t, review.Threads)
	require.Empty(t, review.ViewerID)
	require.Equal(t, []string{
		"/2.0/repositories/acme/widgets/pullrequests/42",
		"/2.0/repositories/acme/widgets/commit/source-hash/statuses",
	}, paths)
}

func TestMapParticipantsPreservesChangesRequestedVerdict(t *testing.T) {
	changesRequested := cloudParticipantPayload{Role: "REVIEWER", State: "changes_requested"}
	changesRequested.User.AccountID, changesRequested.User.DisplayName = "user-1", "Ada"
	pending := cloudParticipantPayload{Role: "REVIEWER"}
	pending.User.AccountID, pending.User.DisplayName = "user-2", "Grace"
	participants := mapParticipants([]cloudParticipantPayload{changesRequested, pending})

	require.Equal(t, []domain.Participant{
		{ID: "user-1", Name: "Ada", Role: "REVIEWER", Verdict: domain.ReviewVerdictChangesRequested},
		{ID: "user-2", Name: "Grace", Role: "REVIEWER", Verdict: domain.ReviewVerdictPending},
	}, participants)
}

func TestCloudPullRequestRejectsUnsafeHTMLURL(t *testing.T) {
	data, err := os.ReadFile("testdata/review-pullrequest.json")
	require.NoError(t, err)
	var payload pullRequestPayload
	require.NoError(t, json.Unmarshal(data, &payload))
	for _, value := range []string{
		"https://token@bitbucket.org/acme/widgets/pull-requests/42",
		"https://attacker.example.test/acme/widgets/pull-requests/42",
	} {
		payload.Links.HTML.Href = value
		_, err := mapPullRequest(domain.Repository{Namespace: "acme", Slug: "widgets"}, payload)
		require.ErrorContains(t, err, "invalid HTML URL")
	}
}

func TestCloudReviewPaginationRejectsRepeatedPage(t *testing.T) {
	apiBase, err := url.Parse("https://api.bitbucket.org/2.0")
	require.NoError(t, err)
	client := NewClient(ClientOptions{APIBase: apiBase})
	endpoint := client.repositoryEndpoint(domain.Repository{Namespace: "acme", Slug: "widgets"}, "pullrequests", "42", "commits")
	err = client.forReviewPages(context.Background(), endpoint, func(next *url.URL) (string, error) {
		return next.String(), nil
	})
	require.ErrorContains(t, err, "pagination repeated a page")
}

func TestCloudReviewActionRejectsUnavailableCapability(t *testing.T) {
	client := NewClient(ClientOptions{TokenSource: staticTokenSource("cloud-token")})
	_, err := client.ApplyReviewAction(context.Background(), domain.PullRequest{Repository: domain.Repository{Namespace: "acme", Slug: "widgets"}, Number: 42}, domain.ReviewAction{Kind: domain.MutationTriggerBuild})
	require.ErrorContains(t, err, "does not support build_actions")
}

func TestCloudReviewActionsUseV2EndpointsAndBodies(t *testing.T) {
	pullRequest, err := os.ReadFile("testdata/review-pullrequest.json")
	require.NoError(t, err)
	type expectation struct {
		action domain.ReviewAction
		method string
		path   string
		body   string
	}
	expectations := []expectation{
		{action: domain.ReviewAction{Kind: domain.MutationApprove}, method: http.MethodPost, path: "/approve"},
		{action: domain.ReviewAction{Kind: domain.MutationUnapprove}, method: http.MethodDelete, path: "/approve"},
		{action: domain.ReviewAction{Kind: domain.MutationMerge}, method: http.MethodPost, path: "/merge"},
		{action: domain.ReviewAction{Kind: domain.MutationDecline}, method: http.MethodPost, path: "/decline"},
		{action: domain.ReviewAction{Kind: domain.MutationAddComment, Comment: "Looks good."}, method: http.MethodPost, path: "/comments", body: `{"content":{"raw":"Looks good."}}`},
		{action: domain.ReviewAction{Kind: domain.MutationReply, Comment: "Done.", ParentCommentID: "10"}, method: http.MethodPost, path: "/comments", body: `{"content":{"raw":"Done."},"parent":{"id":10}}`},
	}
	for _, expected := range expectations {
		t.Run(string(expected.action.Kind), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				base := "/2.0/repositories/acme/widgets/pullrequests/42"
				if r.Method == http.MethodGet && r.URL.Path == base {
					_, _ = w.Write(pullRequest)
					return
				}
				require.Equal(t, expected.method, r.Method)
				require.Equal(t, base+expected.path, r.URL.Path)
				if expected.body != "" {
					var gotBody, wantBody any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
					require.NoError(t, json.NewDecoder(bytes.NewBufferString(expected.body)).Decode(&wantBody))
					require.Equal(t, wantBody, gotBody)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			baseURL, err := url.Parse(server.URL + "/2.0")
			require.NoError(t, err)
			client := NewClient(ClientOptions{APIBase: baseURL, HTTPClient: server.Client(), TokenSource: staticTokenSource("cloud-token")})
			_, err = client.ApplyReviewAction(context.Background(), domain.PullRequest{Repository: domain.Repository{Namespace: "acme", Slug: "widgets"}, Number: 42}, expected.action)
			require.NoError(t, err)
		})
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	require.NoError(t, err)
	return value
}

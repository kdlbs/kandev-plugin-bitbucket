package datacenter

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

func TestDataCenterListsBranchesAndPullRequestsWithStartLimitPagination(t *testing.T) {
	branches, err := os.ReadFile("testdata/branches.json")
	require.NoError(t, err)
	pullRequests, err := os.ReadFile("testdata/pullrequests.json")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer dc-token", r.Header.Get("Authorization"))
		require.Equal(t, "2", r.URL.Query().Get("limit"))
		switch r.URL.Path {
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/branches":
			require.Equal(t, "feat", r.URL.Query().Get("filterText"))
			_, _ = w.Write(branches)
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests":
			require.Equal(t, "OPEN", r.URL.Query().Get("state"))
			require.Equal(t, "race", r.URL.Query().Get("filterText"))
			_, _ = w.Write(pullRequests)
		default:
			t.Fatalf("unexpected Data Center path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
	})
	require.NoError(t, err)
	repository := domain.Repository{Namespace: "ENG", Slug: "widgets"}

	gotBranches, err := client.listBranches(context.Background(), repository, "feat", 2)
	require.NoError(t, err)
	require.Equal(t, []domain.Branch{{Name: "feature/race", Commit: "abc123", IsDefault: true}}, gotBranches)

	gotPullRequests, err := client.listPullRequests(context.Background(), repository, "race", 2)
	require.NoError(t, err)
	require.Len(t, gotPullRequests, 1)
	require.Equal(t, "ENG/widgets#42", gotPullRequests[0].Key())
	require.Equal(t, "feature/race", gotPullRequests[0].Source.Name)
	require.Equal(t, domain.Repository{ID: "84", ProviderScope: server.URL + "/bitbucket", Namespace: "FORK", Slug: "fork-widgets", CloneURL: mustURL(t, server.URL+"/bitbucket/scm/FORK/fork-widgets.git")}, gotPullRequests[0].SourceRepository)
	require.Equal(t, "main", gotPullRequests[0].Repository.DefaultBranch)
	require.Equal(t, server.URL+"/bitbucket/scm/ENG/widgets.git", gotPullRequests[0].Repository.CloneURL.String())
	require.Equal(t, server.URL+"/bitbucket/projects/ENG/repos/widgets/pull-requests/42", gotPullRequests[0].URL)
}

func TestDataCenterListBranchesPagesThroughAllBranches(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer dc-token", r.Header.Get("Authorization"))
		require.Equal(t, "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/branches", r.URL.Path)
		require.Equal(t, "100", r.URL.Query().Get("limit"))
		switch r.URL.Query().Get("start") {
		case "0":
			_, _ = w.Write([]byte(`{"isLastPage":false,"nextPageStart":2,"values":[{"displayId":"one","latestCommit":"commit-1"},{"displayId":"two","latestCommit":"commit-2"}]}`))
		case "2":
			_, _ = w.Write([]byte(`{"isLastPage":true,"nextPageStart":2,"values":[{"displayId":"three","latestCommit":"commit-3"},{"displayId":"four","latestCommit":"commit-4"}]}`))
		default:
			t.Fatalf("unexpected branch start %q", r.URL.Query().Get("start"))
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
	})
	require.NoError(t, err)

	branches, err := client.ListBranches(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"})
	require.NoError(t, err)
	require.Equal(t, []domain.Branch{
		{Name: "one", Commit: "commit-1"},
		{Name: "two", Commit: "commit-2"},
		{Name: "three", Commit: "commit-3"},
		{Name: "four", Commit: "commit-4"},
	}, branches)
}

func TestDataCenterListBranchesRejectsEndlessPagination(t *testing.T) {
	nextStart := 1
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/branches", r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"isLastPage":false,"nextPageStart":%d,"values":[]}`, nextStart)
		nextStart++
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
	})
	require.NoError(t, err)

	_, err = client.ListBranches(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"})
	require.ErrorContains(t, err, "Data Center branch pagination limit exceeded")
}

func TestDataCenterSearchPullRequestsUsesRequestedState(t *testing.T) {
	pullRequests, err := os.ReadFile("testdata/pullrequests.json")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "DECLINED", r.URL.Query().Get("state"))
		_, _ = w.Write(pullRequests)
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(), TokenSource: staticTokenSource("dc-token"),
	})
	require.NoError(t, err)

	_, err = client.SearchPullRequests(context.Background(), domain.PullRequestQuery{Repository: domain.Repository{Namespace: "ENG", Slug: "widgets"}, State: "DECLINED", Limit: 1})
	require.NoError(t, err)
}

func TestDataCenterSearchPullRequestsUsesAllState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "ALL", r.URL.Query().Get("state"))
		_, _ = w.Write([]byte(`{"isLastPage":true,"values":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(), TokenSource: staticTokenSource("dc-token"),
	})
	require.NoError(t, err)

	_, err = client.SearchPullRequests(context.Background(), domain.PullRequestQuery{Repository: domain.Repository{Namespace: "ENG", Slug: "widgets"}, State: "ALL", Limit: 1})
	require.NoError(t, err)
}

func TestDataCenterSearchPullRequestsPageResumesStartCursorAndMapsAuthor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := r.URL.Query().Get("start")
		if start == "1" {
			_, _ = w.Write([]byte(`{"isLastPage":true,"values":[{"id":2,"title":"Two","state":"OPEN","author":{"slug":"user-2"},"fromRef":{"displayId":"two"},"toRef":{"displayId":"main","repository":{"defaultBranch":"main"}}}]}`))
			return
		}
		require.Equal(t, "0", start)
		_, _ = w.Write([]byte(`{"isLastPage":false,"nextPageStart":1,"values":[{"id":1,"title":"One","state":"OPEN","author":{"slug":"user-1"},"fromRef":{"displayId":"one"},"toRef":{"displayId":"main","repository":{"defaultBranch":"main"}}}]}`))
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(), TokenSource: staticTokenSource("dc-token"),
	})
	require.NoError(t, err)
	query := domain.PullRequestQuery{Repository: domain.Repository{Namespace: "ENG", Slug: "widgets"}, State: "OPEN", Limit: 100}

	first, err := client.SearchPullRequestsPage(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "user-1", first.PullRequests[0].Author)
	require.Equal(t, "1", first.NextCursor)
	query.Cursor = first.NextCursor
	second, err := client.SearchPullRequestsPage(context.Background(), query)
	require.NoError(t, err)
	require.Equal(t, "user-2", second.PullRequests[0].Author)
	require.Empty(t, second.NextCursor)
}

func TestDataCenterMapsPullRequestDisplayAuthorAndCreatedTime(t *testing.T) {
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: "https://dc.example.test/bitbucket"},
		TokenSource:       staticTokenSource("dc-token"),
	})
	require.NoError(t, err)
	var payload pullRequestPayload
	require.NoError(t, json.Unmarshal([]byte(`{
		"id":42,"title":"Fix race","state":"OPEN","createdDate":1785499200000,
		"author":{"slug":"ada","displayName":"Ada Lovelace"},
		"fromRef":{"displayId":"feature/race"},
		"toRef":{"displayId":"main","repository":{"defaultBranch":"main"}}
	}`), &payload))

	pullRequest, err := client.mapPullRequest(domain.Repository{Namespace: "ENG", Slug: "widgets"}, payload)
	require.NoError(t, err)
	require.Equal(t, "ada", pullRequest.Author)
	require.Equal(t, "Ada Lovelace", pullRequest.AuthorDisplayName)
	require.Equal(t, time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC), pullRequest.CreatedAt)
}

func TestDataCenterGetReviewMapsGoldenReviewData(t *testing.T) {
	responses := map[string]string{
		"/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42":         "review-pullrequest.json",
		"/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/commits": "review-commits.json",
		"/bitbucket/rest/build-status/latest/commits/source-hash":                        "review-statuses.json",
		"/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42.diff":    "review.diff",
		"/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/changes": "review-changes.json",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Basic ZGV2OmRjLXRva2Vu", r.Header.Get("Authorization"))
		if r.URL.Path == "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/changes" {
			require.Equal(t, "100", r.URL.Query().Get("limit"))
			if r.URL.Query().Get("start") == "1" {
				body, err := os.ReadFile("testdata/review-changes-second.json")
				require.NoError(t, err)
				_, _ = w.Write(body)
				return
			}
		}
		if r.URL.Path == "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/commits" || r.URL.Path == "/bitbucket/rest/build-status/latest/commits/source-hash" {
			require.Equal(t, "100", r.URL.Query().Get("limit"))
			if r.URL.Query().Get("start") == "1" {
				fixture, ok := responses[r.URL.Path]
				require.True(t, ok)
				body, err := os.ReadFile("testdata/" + fixture + "-second.json")
				require.NoError(t, err)
				_, _ = w.Write(body)
				return
			}
		}
		if r.URL.Path == "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/activities" {
			require.Equal(t, "true", r.URL.Query().Get("withComments"))
			require.Equal(t, "100", r.URL.Query().Get("limit"))
			if r.URL.Query().Get("start") == "1" {
				body, err := os.ReadFile("testdata/review-activities-second.json")
				require.NoError(t, err)
				_, _ = w.Write(body)
				return
			}
			body, err := os.ReadFile("testdata/review-activities-first.json")
			require.NoError(t, err)
			_, _ = w.Write(body)
			return
		}
		fixture, ok := responses[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected Data Center path %s", r.URL.Path)
		}
		body, err := os.ReadFile("testdata/" + fixture)
		require.NoError(t, err)
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
		Authentication:    Authentication{Mode: AuthenticationPAT, Username: "dev"},
	})
	require.NoError(t, err)

	review, err := client.GetReview(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"}, 42)
	require.NoError(t, err)
	require.Equal(t, "diff --git a/race.go b/race.go\nindex 1111111..2222222 100644\n--- a/race.go\n+++ b/race.go\n@@ -1 +1 @@\n-unsafe()\n+safe()\n", review.Diff)
	require.Equal(t, []domain.Commit{{Hash: "source-hash", Message: "Fix race", Author: "Ada"}, {Hash: "source-hash-2", Message: "Add coverage", Author: "Bob"}}, review.Commits)
	require.Equal(t, []domain.Participant{{ID: "ada", Name: "Ada", Role: "REVIEWER", Approved: true, Verdict: domain.ReviewVerdictApproved}}, review.Participants)
	require.Equal(t, "dev", review.ViewerID)
	require.Equal(t, []domain.Thread{{ID: "10", Comments: []domain.Comment{
		{ID: "10", Author: "Ada", Body: "Please add a test.", When: time.Date(2026, time.July, 31, 12, 1, 0, 0, time.UTC)},
		{ID: "11", ParentID: "10", Author: "Bob", Body: "Done.", When: time.Date(2026, time.July, 31, 12, 2, 0, 0, time.UTC)},
	}}}, review.Threads)
	require.Equal(t, []domain.BuildStatus{{Key: "build-7", Name: "CI", State: "SUCCESSFUL", URL: "https://ci.example.test/build/7", Target: "source-hash"}, {Key: "build-8", Name: "Lint", State: "FAILED", URL: "https://ci.example.test/build/8", Target: "source-hash"}}, review.Statuses)
	require.Equal(t, []domain.ReviewFile{{
		Path:      "race.go",
		Status:    "modified",
		Additions: 1,
		Deletions: 1,
		Patch:     "--- a/race.go\n+++ b/race.go\n@@ -1,1 +1,1 @@\n-unsafe()\n+safe()\n",
	}, {
		Path:      "new.go",
		Status:    "added",
		Additions: 1,
		Deletions: 0,
		Patch:     "--- /dev/null\n+++ b/new.go\n@@ -0,0 +1,1 @@\n+new()\n",
	}}, review.Files)
	require.Equal(t, "main", review.PullRequest.Repository.DefaultBranch)
	require.Equal(t, domain.Repository{ID: "84", ProviderScope: server.URL + "/bitbucket", Namespace: "FORK", Slug: "fork-widgets", CloneURL: mustURL(t, server.URL+"/bitbucket/scm/FORK/fork-widgets.git")}, review.PullRequest.SourceRepository)
}

func TestDataCenterProjectedReviewSkipsDiffFilesCommitsComments(t *testing.T) {
	pullRequest, err := os.ReadFile("testdata/review-pullrequest.json")
	require.NoError(t, err)
	paths := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42":
			_, _ = w.Write(pullRequest)
		case "/bitbucket/rest/build-status/latest/commits/source-hash":
			_, _ = w.Write([]byte(`{"isLastPage":true,"values":[{"key":"ci","name":"CI","state":"SUCCESSFUL"}]}`))
		default:
			t.Fatalf("unexpected heavy Data Center review request %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(), TokenSource: staticTokenSource("dc-token"),
		Authentication: Authentication{Mode: AuthenticationPAT, Username: "dev"},
	})
	require.NoError(t, err)

	review, err := client.GetReviewProjected(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"}, 42, domain.ReviewProjection{
		Participants: true, Statuses: true,
	})

	require.NoError(t, err)
	require.NotEmpty(t, review.Participants)
	require.NotEmpty(t, review.Statuses)
	require.Empty(t, review.Diff)
	require.Empty(t, review.Files)
	require.Empty(t, review.Commits)
	require.Empty(t, review.Threads)
	require.Equal(t, []string{
		"/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42",
		"/bitbucket/rest/build-status/latest/commits/source-hash",
	}, paths)
}

func TestMapParticipantsPreservesNeedsWorkVerdict(t *testing.T) {
	needsWork := dataCenterParticipantPayload{Role: "REVIEWER", Status: "NEEDS_WORK"}
	needsWork.User.Slug, needsWork.User.DisplayName = "ada", "Ada"
	pending := dataCenterParticipantPayload{Role: "REVIEWER", Status: "UNAPPROVED"}
	pending.User.Slug, pending.User.DisplayName = "grace", "Grace"
	participants := mapParticipants([]dataCenterParticipantPayload{needsWork, pending})

	require.Equal(t, []domain.Participant{
		{ID: "ada", Name: "Ada", Role: "REVIEWER", Verdict: domain.ReviewVerdictChangesRequested},
		{ID: "grace", Name: "Grace", Role: "REVIEWER", Verdict: domain.ReviewVerdictPending},
	}, participants)
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	require.NoError(t, err)
	return value
}

func TestDataCenterReviewChangesPaginationRejectsRepeatedStart(t *testing.T) {
	pullRequest, err := os.ReadFile("testdata/review-pullrequest.json")
	require.NoError(t, err)
	diff, err := os.ReadFile("testdata/review.diff")
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42":
			_, _ = w.Write(pullRequest)
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42.diff":
			_, _ = w.Write(diff)
		case "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42/changes":
			require.Equal(t, "100", r.URL.Query().Get("limit"))
			require.Equal(t, "0", r.URL.Query().Get("start"))
			_, _ = w.Write([]byte(`{"isLastPage":false,"nextPageStart":0,"values":[]}`))
		default:
			t.Fatalf("unexpected Data Center path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
		Authentication:    Authentication{Mode: AuthenticationPAT, Username: "dev"},
	})
	require.NoError(t, err)

	_, err = client.GetReview(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"}, 42)
	require.ErrorContains(t, err, "review change pagination did not advance")
}

func TestDataCenterReviewCommitPaginationRejectsRepeatedStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "0", r.URL.Query().Get("start"))
		require.Equal(t, "100", r.URL.Query().Get("limit"))
		_, _ = w.Write([]byte(`{"isLastPage":false,"nextPageStart":0,"values":[]}`))
	}))
	defer server.Close()
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
		HTTPClient:        server.Client(),
		TokenSource:       staticTokenSource("dc-token"),
		Authentication:    Authentication{Mode: AuthenticationPAT, Username: "dev"},
	})
	require.NoError(t, err)

	_, err = client.reviewCommits(context.Background(), domain.Repository{Namespace: "ENG", Slug: "widgets"}, 42)
	require.ErrorContains(t, err, "pagination did not advance")
}

func TestDataCenterReviewActionRejectsUnavailableCapability(t *testing.T) {
	client, err := NewClient(ClientOptions{ConnectionOptions: ConnectionOptions{BaseURL: "https://dc.example.test"}, TokenSource: staticTokenSource("dc-token")})
	require.NoError(t, err)
	_, err = client.ApplyReviewAction(context.Background(), domain.PullRequest{Repository: domain.Repository{Namespace: "ENG", Slug: "widgets"}, Number: 42}, domain.ReviewAction{Kind: domain.MutationTriggerBuild})
	require.ErrorContains(t, err, "does not support build_actions")
}

func TestDataCenterOAuthKeepsSupportedReviewOperations(t *testing.T) {
	client, err := NewClient(ClientOptions{
		ConnectionOptions: ConnectionOptions{BaseURL: "https://dc.example.test"},
		TokenSource:       staticTokenSource("dc-token"),
		Authentication:    Authentication{Mode: AuthenticationOAuth, Username: "oauth-user"},
	})
	require.NoError(t, err)
	require.True(t, client.Capabilities().Supports(domain.CapabilityIncomingOAuth))
	require.True(t, client.Capabilities().Supports(domain.CapabilityReview))
	require.True(t, client.Capabilities().Supports(domain.CapabilityPullRequests))
	require.True(t, client.Capabilities().Supports(domain.CapabilityComments))
	require.True(t, client.Capabilities().Supports(domain.CapabilityThreadReplies))
	require.True(t, client.Capabilities().Supports(domain.CapabilityApprove))
	require.True(t, client.Capabilities().Supports(domain.CapabilityMerge))
	require.True(t, client.Capabilities().Supports(domain.CapabilityDecline))
	require.True(t, client.Capabilities().Supports(domain.CapabilityBuildStatuses))
}

func TestDataCenterReviewActionsUseLatestEndpointsAndBodies(t *testing.T) {
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
		{action: domain.ReviewAction{Kind: domain.MutationMerge}, method: http.MethodPost, path: "/merge?version=3"},
		{action: domain.ReviewAction{Kind: domain.MutationDecline}, method: http.MethodPost, path: "/decline?version=3"},
		{action: domain.ReviewAction{Kind: domain.MutationAddComment, Comment: "Looks good."}, method: http.MethodPost, path: "/comments", body: `{"text":"Looks good."}`},
		{action: domain.ReviewAction{Kind: domain.MutationReply, Comment: "Done.", ParentCommentID: "10"}, method: http.MethodPost, path: "/comments", body: `{"parent":{"id":10},"text":"Done."}`},
	}
	for _, expected := range expectations {
		t.Run(string(expected.action.Kind), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				base := "/bitbucket/rest/api/latest/projects/ENG/repos/widgets/pull-requests/42"
				if r.Method == http.MethodGet && r.URL.Path == base {
					_, _ = w.Write(pullRequest)
					return
				}
				require.Equal(t, expected.method, r.Method)
				require.Equal(t, base+expected.path, r.URL.RequestURI())
				if expected.body != "" {
					var gotBody, wantBody any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
					require.NoError(t, json.NewDecoder(bytes.NewBufferString(expected.body)).Decode(&wantBody))
					require.Equal(t, wantBody, gotBody)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			client, err := NewClient(ClientOptions{
				ConnectionOptions: ConnectionOptions{BaseURL: server.URL + "/bitbucket", AllowInsecureHTTP: true},
				HTTPClient:        server.Client(),
				TokenSource:       staticTokenSource("dc-token"),
				Authentication:    Authentication{Mode: AuthenticationPAT, Username: "dev"},
			})
			require.NoError(t, err)
			_, err = client.ApplyReviewAction(context.Background(), domain.PullRequest{Repository: domain.Repository{Namespace: "ENG", Slug: "widgets"}, Number: 42, Version: 1}, expected.action)
			require.NoError(t, err)
		})
	}
}

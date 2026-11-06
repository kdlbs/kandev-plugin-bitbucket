import {
  changeRequestDetailActions,
  changeRequestDetailModel,
  connectionActionBody,
  connectionIdentity,
  connectionOAuthRegistration,
  connectionSaveBody,
  currentWatchFilter,
  deriveOAuthCallbackURL,
  displayPullRequestAuthor,
  disconnectConnectionInput,
  integrationSettingsHref,
  linkPullRequestBody,
  normalizeRepositoryInspection,
  normalizeRepositories,
  normalizePullRequests,
  normalizePullRequestAssociations,
  normalizeSavedQueries,
  newSavedQuery,
  canSaveDashboardQuery,
  normalizeReviewDetail,
  normalizeWatches,
  taskDialogInitialValues,
  taskFromLaunchResult,
  taskLaunchBody,
  taskLaunchPresets,
  usePluginTaskCreation,
  oauthStartInput,
  pullRequestCreateBody,
  pluginRepositoryInput,
  pullRequestListRequest,
  statusTone,
  taskSupportsBitbucketRepository,
  taskRepositoryDescriptor,
  validateCloudWorkspace,
  validateConnectionIdentity,
  validateOAuthRegistration,
  workspaceReviewAction,
} from "../src/view-models";
import { describe, expect, it } from "vitest";

describe("Bitbucket view models", () => {
  it("does not request an account email for Cloud scoped access tokens", () => {
    expect(connectionIdentity("cloud", "access_token")).toBeNull();
    expect(connectionIdentity("cloud", "api_token")?.inputType).toBe("email");
  });
  it("builds workspace-scoped integration settings links with a global fallback", () => {
    expect(integrationSettingsHref("workspace one")).toBe(
      "/settings/workspaces/workspace%20one/integrations/bitbucket",
    );
    expect(integrationSettingsHref()).toBe("/settings/integrations/bitbucket");
  });

  it("shows readable authors but hides provider-opaque Cloud identities", () => {
    expect(displayPullRequestAuthor("Ari Almeida")).toBe("Ari Almeida");
    expect(displayPullRequestAuthor("dev-user")).toBe("dev-user");
    expect(
      displayPullRequestAuthor("712020:49e8a296-5268-4c02-a6b3-20a4b8e60149"),
    ).toBeUndefined();
    expect(displayPullRequestAuthor("  ")).toBeUndefined();
  });

  it("normalizes queue data without leaking provider payload shape into components", () => {
    expect(
      normalizePullRequests({
        values: [
          {
            id: 42,
            title: "Improve mobile review",
            repository: { id: "repo-1", name: "mobile" },
            state: "OPEN",
            source_branch: "feature/mobile-review",
            destination_branch: "main",
            author: "712020:49e8a296-5268-4c02-a6b3-20a4b8e60149",
            author_display_name: "Ari",
            created_at: "2026-08-06T08:00:00Z",
            capabilities: "merge, decline",
            associations: [
              { id: "link-1", task_id: "task-1", title: "Review mobile" },
            ],
          },
        ],
      }),
    ).toEqual([
      expect.objectContaining({
        key: "repo-1:42",
        number: 42,
        repositoryName: "mobile",
        author: "Ari",
        createdAt: "2026-08-06T08:00:00Z",
        sourceBranch: "feature/mobile-review",
        destinationBranch: "main",
        tasks: [
          { id: "link-1", taskId: "task-1", fallbackTitle: "Review mobile" },
        ],
        capabilities: ["merge", "decline"],
      }),
    ]);

    expect(
      normalizePullRequests({
        pull_requests: [
          {
            id: "42",
            number: 42,
            title: "Scoped repository",
            state: "OPEN",
            repository: {
              provider_repository_id: "repo-uuid",
              provider_scope: "https://dc.example.test/context-a",
              name: "mobile",
            },
          },
        ],
      })[0],
    ).toEqual(
      expect.objectContaining({
        repositoryId: "repo-uuid",
        providerScope: "https://dc.example.test/context-a",
      }),
    );
  });

  it("extracts provider links without coercing Bitbucket link objects", () => {
    const cloud = normalizePullRequests({
      values: [
        {
          id: 42,
          title: "Cloud link",
          repository: { id: "acme/cloud", name: "cloud" },
          state: "OPEN",
          links: {
            html: { href: "https://bitbucket.org/acme/cloud/pull-requests/42" },
          },
        },
      ],
    });
    const dataCenter = normalizePullRequests({
      values: [
        {
          id: 7,
          title: "Data Center link",
          repository: { id: "ENG/widgets", name: "widgets" },
          state: "OPEN",
          links: {
            self: [
              {
                href: "https://bitbucket.example.test/projects/ENG/repos/widgets/pull-requests/7",
              },
            ],
          },
        },
      ],
    });

    expect(cloud[0]?.url).toBe(
      "https://bitbucket.org/acme/cloud/pull-requests/42",
    );
    expect(dataCenter[0]?.url).toBe(
      "https://bitbucket.example.test/projects/ENG/repos/widgets/pull-requests/7",
    );
    expect(cloud[0]?.url).not.toBe("[object Object]");
  });

  it("groups task associations by canonical review key", () => {
    expect(
      normalizePullRequestAssociations({
        associations: [
          {
            review_key: "acme/widgets#42",
            task_id: "task-1",
            task_title: "Review mobile",
          },
        ],
      }),
    ).toEqual({
      "acme/widgets#42": [
        {
          id: "acme/widgets#42:task-1",
          taskId: "task-1",
          fallbackTitle: "Review mobile",
        },
      ],
    });
  });

  it("also indexes task associations by immutable repository identity", () => {
    const associations = normalizePullRequestAssociations({
      associations: [
        {
          review_key: "old-name/widgets#42",
          provider_scope: "https://bitbucket.org",
          repository_id: "repo-uuid",
          number: 42,
          task_id: "task-1",
          task_title: "Review rename",
        },
      ],
    });

    expect(
      associations[
        "connection:https://bitbucket.org\u0000repository:repo-uuid\u0000pull-request:42"
      ],
    ).toEqual([
      expect.objectContaining({
        taskId: "task-1",
        connectionScope: "https://bitbucket.org",
        repositoryId: "repo-uuid",
        changeRequestNumber: 42,
      }),
    ]);
  });

  it("validates, bounds, and creates workspace saved queries from committed filters", () => {
    expect(
      normalizeSavedQueries([
        {
          id: "saved-1",
          label: "Needs review",
          query: "state:open reviewer:me",
          repositoryId: "repo-1",
          state: "open",
          createdAt: "2026-08-10T10:00:00Z",
        },
        { id: "invalid", label: "", query: 42 },
      ]),
    ).toEqual([
      {
        id: "saved-1",
        label: "Needs review",
        query: "state:open reviewer:me",
        repositoryId: "repo-1",
        state: "open",
        createdAt: "2026-08-10T10:00:00Z",
      },
    ]);
    expect(canSaveDashboardQuery("state:open reviewer:me", "")).toBe(true);
    expect(canSaveDashboardQuery("", "repo-1")).toBe(true);
    expect(canSaveDashboardQuery("", "")).toBe(false);
    expect(
      newSavedQuery(
        {
          label: " Needs review ",
          query: "state:open",
          repositoryId: "repo-1",
          state: "open",
        },
        "saved-2",
        "2026-08-10T11:00:00Z",
      ),
    ).toEqual({
      id: "saved-2",
      label: "Needs review",
      query: "state:open",
      repositoryId: "repo-1",
      state: "open",
      createdAt: "2026-08-10T11:00:00Z",
    });
  });

  it("maps Bitbucket repository descriptors between plugin snake case and host camel case", () => {
    const serverRepository = {
      id: "acme/widgets",
      name: "widgets",
      owner_or_project: "acme",
      provider_id: "bitbucket",
      provider_host: "bitbucket.org",
      provider_repository_id: "acme/widgets",
      clone_url: "https://bitbucket.org/acme/widgets.git",
      default_branch: "main",
    };
    const expectedRepository = {
      ownerOrProject: "acme",
      providerId: "bitbucket",
      providerHost: "bitbucket.org",
      repositoryId: "acme/widgets",
      repositoryName: "widgets",
      cloneUrl: "https://bitbucket.org/acme/widgets.git",
      defaultBranch: "main",
    };
    const repository = normalizeRepositoryInspection(serverRepository);

    expect(repository).toEqual(expectedRepository);
    expect(normalizeRepositories({ repositories: [serverRepository] })).toEqual(
      [expectedRepository],
    );
    expect(pluginRepositoryInput(repository)).toEqual({
      provider_id: "bitbucket",
      provider_host: "bitbucket.org",
      owner_or_project: "acme",
      provider_repository_id: "acme/widgets",
      name: "widgets",
      clone_url: "https://bitbucket.org/acme/widgets.git",
      default_branch: "main",
    });
  });

  it("uses the repository-scoped search action instead of filtering the 25-item workspace queue", () => {
    const repository = {
      providerId: "bitbucket",
      providerHost: "bitbucket.org",
      ownerOrProject: "acme",
      repositoryId: "repository-26",
      repositoryName: "selected",
      cloneUrl: "https://bitbucket.org/acme/selected.git",
    };

    expect(
      pullRequestListRequest(repository, "fix", "open", "repo-page-2", 25),
    ).toEqual({
      actionKey: "pullrequests.search",
      body: {
        repository: pluginRepositoryInput(repository),
        query: "fix",
        state: "open",
        cursor: "repo-page-2",
        limit: 25,
      },
    });
    expect(
      pullRequestListRequest(null, "fix", "open", "queue-page-2", 25),
    ).toEqual({
      actionKey: "pullrequests.queue",
      body: {
        view: "queue",
        query: "fix",
        state: "open",
        cursor: "queue-page-2",
        limit: 25,
      },
    });
    expect(
      pullRequestListRequest(null, "fix login state:merged", "open"),
    ).toEqual({
      actionKey: "pullrequests.queue",
      body: {
        view: "queue",
        query: "fix login",
        state: "merged",
        cursor: "",
        limit: 25,
      },
    });
  });

  it("preserves pull-request inspection metadata required by native task import", () => {
    expect(
      normalizeRepositoryInspection({
        repository: {
          provider_id: "bitbucket",
          provider_host: "bitbucket.example.test",
          owner_or_project: "ENG",
          provider_repository_id: "ENG/widgets",
          name: "widgets",
          clone_url: "https://bitbucket.example.test/scm/eng/widgets.git",
        },
        pull_request: { number: 42, title: "Ship native Bitbucket import" },
        base_branch: "main",
        head_branch: "feature/bitbucket",
      }),
    ).toEqual(
      expect.objectContaining({
        baseBranch: "main",
        headBranch: "feature/bitbucket",
        pullRequest: { number: 42, title: "Ship native Bitbucket import" },
      }),
    );
  });

  it("normalizes build statuses for the native review panel", () => {
    expect(
      normalizeReviewDetail({
        id: "42",
        number: 42,
        title: "Verify Pipelines",
        repository_id: "ENG/widgets",
        repository_name: "widgets",
        state: "OPEN",
        statuses: [
          {
            key: "build-42",
            name: "Bitbucket Pipelines",
            state: "SUCCESSFUL",
            url: "https://bitbucket.example.test/builds/42",
            target: "feature/bitbucket",
          },
        ],
      })?.statuses,
    ).toEqual([
      {
        key: "build-42",
        name: "Bitbucket Pipelines",
        state: "SUCCESSFUL",
        url: "https://bitbucket.example.test/builds/42",
        target: "feature/bitbucket",
      },
    ]);
  });

  it("preserves every comment in a review thread so replies remain visible", () => {
    expect(
      normalizeReviewDetail({
        id: "42",
        number: 42,
        title: "Show threaded discussion",
        repository_id: "ENG/widgets",
        repository_name: "widgets",
        state: "OPEN",
        threads: [
          {
            id: "10",
            comments: [
              { ID: "10", Author: "Ada", Body: "Please add a test." },
              { ID: "11", ParentID: "10", Author: "Bob", Body: "Done." },
            ],
          },
        ],
      })?.threads,
    ).toEqual([
      {
        id: "10",
        author: "Ada",
        body: "Please add a test.",
        resolved: false,
        comments: [
          { id: "10", author: "Ada", body: "Please add a test." },
          { id: "11", parentId: "10", author: "Bob", body: "Done." },
        ],
      },
    ]);
  });

  it("preserves real comment timestamps and omits Go zero times", () => {
    const detail = normalizeReviewDetail({
      id: "42",
      number: 42,
      title: "Show honest comment ages",
      repository_id: "ENG/widgets",
      repository_name: "widgets",
      state: "OPEN",
      threads: [
        {
          id: "10",
          comments: [
            {
              ID: "10",
              Author: "Ada",
              Body: "Timed",
              When: "2026-07-31T12:01:00Z",
            },
            {
              ID: "11",
              ParentID: "10",
              Author: "Bob",
              Body: "Untimed",
              When: "0001-01-01T00:00:00Z",
            },
          ],
        },
      ],
    });

    expect(detail?.threads[0]?.comments).toEqual([
      {
        id: "10",
        author: "Ada",
        body: "Timed",
        createdAt: "2026-07-31T12:01:00Z",
      },
      { id: "11", parentId: "10", author: "Bob", body: "Untimed" },
    ]);
  });

  it("maps Bitbucket review data into the host-owned change-request detail model", () => {
    const review = normalizeReviewDetail({
      id: "42",
      review_key: "acme/widgets#42",
      number: 42,
      title: "Use shared review UI",
      url: "https://bitbucket.org/acme/widgets/pull-requests/42",
      repository_id: "acme/widgets",
      repository_name: "widgets",
      state: "OPEN",
      author_display_name: "Ada",
      created_at: "2026-08-06T08:00:00Z",
      source_branch: "feature/shared-review",
      destination_branch: "main",
      description: "One host-owned review surface.",
      files: [
        { path: "ui.tsx", status: "modified", additions: 12, deletions: 3 },
        { path: "ui.test.tsx", status: "added", additions: 8, deletions: 0 },
      ],
      participants: [
        { name: "Grace", role: "REVIEWER", approved: true },
        {
          name: "Linus",
          role: "REVIEWER",
          approved: false,
          verdict: "changes_requested",
        },
        {
          name: "Margaret",
          role: "REVIEWER",
          approved: false,
          verdict: "pending",
        },
      ],
      statuses: [{ key: "pipeline", name: "Pipelines", state: "SUCCESSFUL" }],
      threads: [
        {
          id: "10",
          file: "ui.tsx",
          comments: [
            { id: "10", author: "Grace", body: "Please reuse the host." },
            { id: "11", parent_id: "10", author: "Ada", body: "Done." },
          ],
        },
      ],
    });

    expect(review && changeRequestDetailModel(review)).toEqual({
      providerId: "bitbucket",
      reviewKey: "acme/widgets#42",
      number: 42,
      title: "Use shared review UI",
      url: "https://bitbucket.org/acme/widgets/pull-requests/42",
      state: "open",
      author: { name: "Ada" },
      createdAt: "2026-08-06T08:00:00Z",
      sourceBranch: "feature/shared-review",
      targetBranch: "main",
      additions: 20,
      deletions: 3,
      description: "One host-owned review surface.",
      reviewState: "changes_requested",
      pendingReviewCount: 1,
      reviews: [
        { id: "Grace", author: { name: "Grace" }, state: "APPROVED" },
        { id: "Linus", author: { name: "Linus" }, state: "CHANGES_REQUESTED" },
      ],
      requestedReviewers: [{ name: "Margaret" }],
      checks: [{ id: "pipeline", name: "Pipelines", state: "SUCCESSFUL" }],
      comments: [
        {
          id: "10",
          author: { name: "Grace" },
          body: "Please reuse the host.",
          path: "ui.tsx",
          resolved: false,
        },
        {
          id: "11",
          parentId: "10",
          author: { name: "Ada" },
          body: "Done.",
          path: "ui.tsx",
          resolved: false,
        },
      ],
    });
  });

  it("normalizes Cloud identity, head checks, and viewer approval for shared host UI", () => {
    const review = normalizeReviewDetail({
      id: 42,
      title: "Cloud review",
      state: "OPEN",
      links: {
        html: { href: "https://bitbucket.org/acme/widgets/pull-requests/42" },
      },
      repository: { full_name: "acme/widgets", name: "widgets" },
      author: {
        display_name: "Ada Cloud",
        links: {
          html: { href: "https://bitbucket.org/ada" },
          avatar: { href: "https://avatar.example.test/ada.png" },
        },
      },
      created_on: "2026-08-06T08:00:00Z",
      updated_on: "2026-08-06T09:00:00Z",
      source: {
        branch: { name: "feature/cloud" },
        commit: { hash: "cloud-head" },
      },
      destination: { branch: { name: "main" } },
      participants: [
        {
          user: { display_name: "Ada Cloud" },
          role: "REVIEWER",
          approved: true,
          is_current_user: true,
        },
      ],
      statuses: {
        values: [
          {
            key: "pipeline",
            name: "Pipelines",
            state: "SUCCESSFUL",
            url: "https://bitbucket.org/acme/widgets/pipelines/results/7",
            description: "Cloud pipeline",
            created_on: "2026-08-06T08:10:00Z",
            updated_on: "2026-08-06T08:12:00Z",
          },
        ],
      },
    });

    expect(review).toMatchObject({
      key: "acme/widgets:42",
      repositoryId: "acme/widgets",
      author: "Ada Cloud",
      authorUrl: "https://bitbucket.org/ada",
      authorAvatarUrl: "https://avatar.example.test/ada.png",
      createdAt: "2026-08-06T08:00:00Z",
      updatedAt: "2026-08-06T09:00:00Z",
      sourceBranch: "feature/cloud",
      destinationBranch: "main",
      headCommit: "cloud-head",
      viewerApproved: true,
      statuses: [
        {
          key: "pipeline",
          name: "Pipelines",
          state: "SUCCESSFUL",
          url: "https://bitbucket.org/acme/widgets/pipelines/results/7",
          output: "Cloud pipeline",
          startedAt: "2026-08-06T08:10:00Z",
          completedAt: "2026-08-06T08:12:00Z",
        },
      ],
    });
    expect(review && changeRequestDetailModel(review)).toMatchObject({
      author: {
        name: "Ada Cloud",
        url: "https://bitbucket.org/ada",
        avatarUrl: "https://avatar.example.test/ada.png",
      },
      checks: [
        {
          id: "pipeline",
          name: "Pipelines",
          state: "SUCCESSFUL",
          output: "Cloud pipeline",
          startedAt: "2026-08-06T08:10:00Z",
          completedAt: "2026-08-06T08:12:00Z",
        },
      ],
    });
  });

  it("normalizes Data Center aliases into the same shared detail model", () => {
    const review = normalizeReviewDetail({
      id: 7,
      title: "Data Center review",
      state: "DECLINED",
      links: {
        self: [
          {
            href: "https://bitbucket.example.test/projects/ENG/repos/widgets/pull-requests/7",
          },
        ],
      },
      repository: { slug: "widgets", project: { key: "ENG" } },
      author: { displayName: "Dana DC", slug: "dana" },
      createdDate: 1786003200000,
      updatedDate: 1786006800000,
      fromRef: { displayId: "feature/dc", latestCommit: "dc-head" },
      toRef: { displayId: "main" },
      reviewers: [
        {
          user: { displayName: "Dana DC", slug: "dana" },
          role: "REVIEWER",
          status: "UNAPPROVED",
          currentUser: true,
        },
      ],
      builds: {
        values: [
          {
            key: "build-7",
            name: "CI",
            state: "FAILED",
            url: "https://ci.example.test/7",
          },
        ],
      },
    });

    expect(review).toMatchObject({
      repositoryId: "ENG/widgets",
      state: "DECLINED",
      author: "Dana DC",
      sourceBranch: "feature/dc",
      destinationBranch: "main",
      headCommit: "dc-head",
      viewerApproved: false,
      statuses: [
        {
          key: "build-7",
          name: "CI",
          state: "FAILED",
          url: "https://ci.example.test/7",
        },
      ],
    });
    expect(review && changeRequestDetailModel(review)).toMatchObject({
      state: "closed",
    });
  });

  it("exposes only terminal-safe, capability- and viewer-aware shared detail actions", () => {
    const base = normalizeReviewDetail({
      id: 42,
      review_key: "acme/widgets#42",
      number: 42,
      title: "Review actions",
      repository_id: "acme/widgets",
      repository_name: "widgets",
      state: "OPEN",
      capabilities: [
        "approve",
        "merge",
        "decline",
        "comments",
        "thread_replies",
      ],
    })!;

    expect(
      changeRequestDetailActions({ ...base, viewerApproved: false }).map(
        ({ id }) => id,
      ),
    ).toEqual(["approve", "merge", "decline", "comment", "reply"]);
    expect(
      changeRequestDetailActions({ ...base, viewerApproved: true }).map(
        ({ id }) => id,
      ),
    ).toEqual(["unapprove", "merge", "decline", "comment", "reply"]);
    expect(changeRequestDetailActions({ ...base, state: "MERGED" })).toEqual(
      [],
    );
    expect(
      changeRequestDetailActions({ ...base, capabilities: ["comments"] }).map(
        ({ id }) => id,
      ),
    ).toEqual(["comment"]);
  });

  it("turns current queue filters into an exact watch filter", () => {
    expect(currentWatchFilter("  mobile review  ", "open")).toEqual({
      query: "mobile review",
      states: ["open"],
    });
    expect(currentWatchFilter("", "all")).toEqual({ query: "", states: [] });
  });

  it("derives a pull-request repository and branches from task context", () => {
    expect(
      taskRepositoryDescriptor(
        [
          {
            repository_id: "host-repo-1",
            base_branch: "release",
            checkout_branch: "feature/task-42",
          },
        ],
        [
          {
            id: "host-repo-1",
            provider: "bitbucket",
            provider_host: "bitbucket.org",
            provider_owner: "acme",
            provider_repo_id: "acme/widgets",
            provider_name: "widgets",
            remote_url: "https://bitbucket.org/acme/widgets.git",
            default_branch: "main",
          },
        ],
      ),
    ).toEqual({
      providerId: "bitbucket",
      providerHost: "bitbucket.org",
      ownerOrProject: "acme",
      repositoryId: "acme/widgets",
      repositoryName: "widgets",
      cloneUrl: "https://bitbucket.org/acme/widgets.git",
      defaultBranch: "main",
      baseBranch: "release",
      headBranch: "feature/task-42",
    });
  });

  it("sends only optional presentation overrides when creating from a task", () => {
    expect(
      pullRequestCreateBody({
        title: "Review task work",
        description: "Please review this task checkout.",
        destination: "release",
        closeSourceOnMerge: true,
      }),
    ).toEqual({
      title: "Review task work",
      description: "Please review this task checkout.",
      destination: "release",
      close_source_on_merge: true,
    });
    expect(
      pullRequestCreateBody({
        title: " ",
        description: "",
        destination: "",
        closeSourceOnMerge: false,
      }),
    ).toEqual({});
  });

  it("accepts Bitbucket pull-request keys or URLs for linking, but only existing keys for unlinking", () => {
    expect(linkPullRequestBody("acme/widgets#42")).toEqual({
      review_key: "acme/widgets#42",
    });
    expect(
      linkPullRequestBody(
        "https://bitbucket.org/acme/widgets/pull-requests/42",
      ),
    ).toEqual({
      review_key: "acme/widgets#42",
    });
    expect(
      linkPullRequestBody(
        "https://bitbucket.example.test/projects/ENG/repos/widgets/pull-requests/42/overview",
      ),
    ).toEqual({ review_key: "ENG/widgets#42" });
    expect(
      linkPullRequestBody(
        "https://bitbucket.example.test/bitbucket/projects/ENG/repos/widgets/pull-requests/42/overview",
      ),
    ).toEqual({ review_key: "ENG/widgets#42" });
    expect(
      linkPullRequestBody("https://bitbucket.org/acme/widgets/issues/42"),
    ).toBeNull();
  });

  it("shows create only when task repository context is Bitbucket or not yet known", () => {
    expect(taskSupportsBitbucketRepository([])).toBe(true);
    expect(
      taskSupportsBitbucketRepository([{ repository_id: "host-repository" }]),
    ).toBe(true);
    expect(taskSupportsBitbucketRepository([{ provider: "github" }])).toBe(
      false,
    );
    expect(
      taskSupportsBitbucketRepository([{ provider_id: "bitbucket" }]),
    ).toBe(true);
  });

  it("builds bounded approve, comment, and reply review mutations", () => {
    const identity = {
      reviewKey: "acme/widgets#42",
      connectionScope: "https://bitbucket.org",
      repositoryId: "repository-uuid",
      changeRequestNumber: 42,
    };
    expect(
      workspaceReviewAction("workspace-1", "task-1", identity, "42", "approve"),
    ).toEqual({
      workspaceId: "workspace-1",
      taskId: "task-1",
      body: {
        review_key: "acme/widgets#42",
        provider_scope: "https://bitbucket.org",
        repository_id: "repository-uuid",
        number: 42,
        pull_request_id: "42",
        kind: "approve",
      },
    });
    expect(
      workspaceReviewAction("workspace-1", "task-1", identity, "42", "reply", {
        comment: "  Looks good  ",
        parentCommentId: "thread-1",
      }),
    ).toEqual({
      workspaceId: "workspace-1",
      taskId: "task-1",
      body: {
        review_key: "acme/widgets#42",
        provider_scope: "https://bitbucket.org",
        repository_id: "repository-uuid",
        number: 42,
        pull_request_id: "42",
        kind: "reply",
        comment: "Looks good",
        parent_comment_id: "thread-1",
      },
    });
  });

  it("uses verified workspace scope for disconnect without a request body", () => {
    expect(disconnectConnectionInput("workspace-1")).toEqual({
      workspaceId: "workspace-1",
    });
  });

  it("offers the native GitHub/GitLab task presets", () => {
    expect(
      taskLaunchPresets().map(({ id, label, hint, iconName }) => ({
        id,
        label,
        hint,
        iconName,
      })),
    ).toEqual([
      {
        id: "review",
        label: "Review",
        hint: "Read the diff, flag issues",
        iconName: "eye",
      },
      {
        id: "address-feedback",
        label: "Address feedback",
        hint: "Apply review comments",
        iconName: "message",
      },
      {
        id: "fix-ci",
        label: "Fix CI",
        hint: "Diagnose failing checks",
        iconName: "tool",
      },
    ]);
  });

  it("prefills native task creation from the Bitbucket pull request", () => {
    const pullRequest = normalizePullRequests({
      pull_requests: [
        {
          id: 42,
          review_key: "acme/widgets#42",
          title: "Improve mobile review",
          url: "https://bitbucket.org/acme/widgets/pull-requests/42",
          repository_id: "acme/widgets",
          repository_name: "widgets",
          source_branch: "feature/mobile-review",
          destination_branch: "main",
        },
      ],
    })[0]!;
    const review = taskLaunchPresets()[0]!;

    expect(taskDialogInitialValues(pullRequest, review)).toEqual({
      title: "Review: Improve mobile review",
      description: expect.stringContaining(
        "https://bitbucket.org/acme/widgets/pull-requests/42",
      ),
      remoteUrl: "https://bitbucket.org/acme/widgets/pull-requests/42",
      branch: "feature/mobile-review",
      checkoutBranch: "feature/mobile-review",
    });
  });

  it("passes an authorized repository descriptor to the native task dialog", () => {
    const pullRequest = normalizePullRequests({
      pull_requests: [
        {
          id: 42,
          title: "Improve mobile review",
          url: "https://bitbucket.org/acme/widgets/pull-requests/42",
          repository_id: "widgets",
          repository_name: "widgets",
        },
      ],
    })[0]!;
    const review = taskLaunchPresets()[0]!;
    const repository = {
      providerId: "bitbucket",
      providerHost: "https://bitbucket.org",
      ownerOrProject: "acme",
      repositoryId: "widgets",
      repositoryName: "widgets",
      cloneUrl: "https://bitbucket.org/acme/widgets.git",
      defaultBranch: "main",
    };

    expect(
      taskDialogInitialValues(pullRequest, review, undefined, repository),
    ).toMatchObject({
      remoteUrl: "https://bitbucket.org/acme/widgets/pull-requests/42",
      remoteRepository: repository,
    });
  });

  it("maps the native task dialog payload to the narrow authenticated launch action", () => {
    const pullRequest = normalizePullRequests({
      pull_requests: [
        {
          id: 42,
          review_key: "acme/widgets#42",
          title: "Improve mobile review",
          provider_scope: "https://bitbucket.org",
          repository_id: "repository-uuid",
          repository_name: "widgets",
        },
      ],
    })[0]!;

    expect(
      taskLaunchBody(
        pullRequest,
        {
          workspace_id: "workspace-1",
          workflow_id: "workflow-1",
          workflow_step_id: "step-2",
          title: "Review: Improve mobile review",
          description: "Inspect the diff",
          repositories: [
            {
              remote_url: "https://bitbucket.org/acme/widgets/pull-requests/42",
            },
          ],
          state: "IN_PROGRESS",
          start_agent: true,
          agent_profile_id: "agent-1",
          executor_id: "executor-runtime-1",
          executor_profile_id: "executor-profile-1",
          plan_mode: true,
          attachments: [
            { type: "image", data: "not-forwarded", mime_type: "image/png" },
          ],
        },
        "launch-123",
      ),
    ).toEqual({
      review_key: "acme/widgets#42",
      provider_scope: "https://bitbucket.org",
      repository_id: "repository-uuid",
      number: 42,
      pull_request_id: "42",
      launch_id: "launch-123",
      task: {
        title: "Review: Improve mobile review",
        description: "Inspect the diff",
        workflow_id: "workflow-1",
        workflow_step_id: "step-2",
        agent_profile_id: "agent-1",
        executor_profile_id: "executor-profile-1",
        start_agent: true,
        plan_mode: true,
      },
    });
  });

  it("fails closed when task launch lacks immutable pull-request identity", () => {
    const pullRequest = normalizePullRequests({
      pull_requests: [
        {
          id: 42,
          review_key: "acme/widgets#42",
          title: "Stale pull request",
          repository_id: "repository-uuid",
          repository_name: "widgets",
        },
      ],
    })[0]!;

    expect(() => taskLaunchBody(pullRequest, {}, "launch-123")).toThrow(
      "identity is unavailable",
    );
  });

  it("keeps persisted host repositories on the native REST task transport", () => {
    expect(usePluginTaskCreation("host-repository-1")).toBe(false);
    expect(usePluginTaskCreation()).toBe(true);
  });

  it("returns a task-shaped launch result only after the pull request link persists", () => {
    expect(taskFromLaunchResult({ task_id: "task-1", linked: true })).toEqual({
      id: "task-1",
      bitbucketLinked: true,
    });
    expect(
      taskFromLaunchResult({
        task_id: "task-1",
        linked: false,
        association_error: "task association could not be saved",
      }),
    ).toEqual({ id: "task-1", bitbucketLinked: false });
    expect(() => taskFromLaunchResult({ linked: true })).toThrow("no task id");
  });

  it("normalizes watch controls without exposing persisted internals", () => {
    expect(
      normalizeWatches({
        watches: [
          {
            id: "team-open",
            status: "running",
            last_polled: "2026-07-31T12:00:00Z",
          },
          { id: "paused", status: "paused" },
        ],
      }),
    ).toEqual([
      {
        id: "team-open",
        status: "running",
        lastPolled: "2026-07-31T12:00:00Z",
      },
      { id: "paused", status: "paused" },
    ]);
  });

  it("maps status states to a presentation tone", () => {
    expect(statusTone("build failed")).toBe("danger");
    expect(statusTone("pending review")).toBe("warning");
  });

  it("uses auth_identity only for Cloud API tokens and Data Center user credentials", () => {
    expect(connectionIdentity("cloud", "api_token")).toMatchObject({
      field: "auth_identity",
      label: "Atlassian account email",
      inputType: "email",
    });
    expect(connectionIdentity("data_center", "user_pat")).toMatchObject({
      field: "auth_identity",
      label: "Bitbucket username",
      inputType: "text",
    });
    expect(connectionIdentity("data_center", "oauth")).not.toBeNull();
    expect(connectionIdentity("data_center", "project_token")).toBeNull();
    expect(connectionIdentity("data_center", "repository_token")).toBeNull();
    expect(connectionIdentity("cloud", "oauth")).toBeNull();
  });

  it("submits required non-secret identity and omits it for Cloud OAuth", () => {
    const cloudToken = {
      product: "cloud",
      baseUrl: "",
      authMethod: "api_token",
      token: "secret-token",
      identity: "ari@example.test",
      cloudWorkspace: "acme-platform",
      oauthRegistrationConfigured: false,
      oauthClientId: "",
      oauthClientSecret: "",
      oauthCallbackUrl:
        "https://kandev.example.test/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    } as const;
    expect(validateCloudWorkspace(cloudToken)).toBeNull();
    expect(validateConnectionIdentity(cloudToken)).toBeNull();
    expect(connectionActionBody(cloudToken)).toMatchObject({
      product: "cloud",
      auth_method: "api_token",
      token: "secret-token",
      auth_identity: "ari@example.test",
      cloud_workspace: "acme-platform",
    });
    expect(
      validateCloudWorkspace({ ...cloudToken, cloudWorkspace: "" }),
    ).toContain("workspace");
    expect(
      validateConnectionIdentity({ ...cloudToken, identity: "" }),
    ).toContain("email");
    const cloudOAuth = connectionActionBody({
      ...cloudToken,
      authMethod: "oauth",
    });
    expect(cloudOAuth).toHaveProperty("cloud_workspace", "acme-platform");
    expect(cloudOAuth).not.toHaveProperty("auth_identity");
    expect(cloudOAuth).not.toHaveProperty("token");

    const dataCenterProjectToken = connectionActionBody({
      ...cloudToken,
      product: "data_center",
      baseUrl: "https://bitbucket.example.test/bitbucket",
      cloudWorkspace: "",
      authMethod: "project_token",
      identity: "",
    });
    expect(dataCenterProjectToken).toMatchObject({
      product: "data_center",
      auth_method: "project_token",
      token: "secret-token",
    });
    expect(dataCenterProjectToken).not.toHaveProperty("auth_identity");
  });

  it("uses a derived callback and saved OAuth registration state without reflecting secrets", () => {
    const oauth = {
      product: "data_center",
      baseUrl: "https://bitbucket.example.test/bitbucket",
      cloudWorkspace: "",
      authMethod: "oauth",
      token: "",
      identity: "dev",
      oauthRegistrationConfigured: false,
      oauthClientId: "client-id",
      oauthClientSecret: "client-secret",
      oauthCallbackUrl:
        "https://kandev.example.test/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    } as const;

    expect(validateOAuthRegistration(oauth)).toBeNull();
    expect(connectionSaveBody(oauth)).toMatchObject({
      oauth_client_id: "client-id",
      oauth_client_secret: "client-secret",
      oauth_redirect_url:
        "https://kandev.example.test/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    });
    expect(connectionSaveBody(oauth)).not.toHaveProperty(
      "oauth_authorization_url",
    );
    expect(connectionSaveBody(oauth)).not.toHaveProperty("oauth_token_url");
    expect(connectionActionBody(oauth)).not.toHaveProperty(
      "oauth_client_secret",
    );
    expect(oauthStartInput("workspace-1")).toEqual({
      workspaceId: "workspace-1",
    });
    expect(
      validateOAuthRegistration({ ...oauth, oauthClientSecret: "" }),
    ).toContain("secret");
    const savedRegistration = {
      ...oauth,
      oauthRegistrationConfigured: true,
      oauthClientId: "",
      oauthClientSecret: "",
    };
    expect(validateOAuthRegistration(savedRegistration)).toBeNull();
    expect(connectionSaveBody(savedRegistration)).not.toHaveProperty(
      "oauth_client_id",
    );
    expect(connectionSaveBody(savedRegistration)).not.toHaveProperty(
      "oauth_client_secret",
    );
    expect(connectionSaveBody(savedRegistration)).not.toHaveProperty(
      "oauth_redirect_url",
    );
    expect(
      connectionOAuthRegistration({
        oauth_client_secret: "must-not-reflect",
        oauth_registration_configured: true,
      }),
    ).toEqual({
      configured: true,
    });
    expect(
      deriveOAuthCallbackURL(
        "https://api.kandev.example.test",
        "https://app.kandev.example.test/workspaces/acme",
      ),
    ).toBe(
      "https://api.kandev.example.test/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    );
    expect(
      deriveOAuthCallbackURL("", "https://kandev.example.test/workspaces/acme"),
    ).toBe(
      "https://kandev.example.test/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    );
  });
});

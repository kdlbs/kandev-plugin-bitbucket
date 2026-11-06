import { expect, test, type APIRequestContext, type APIResponse } from "@playwright/test";

const pluginID = "kandev-plugin-bitbucket";

type JsonRecord = Record<string, unknown>;

type LiveTarget = {
  product: "cloud" | "data_center";
  baseUrl?: string;
  cloudWorkspace?: string;
  authMethod: "api_token" | "access_token" | "user_pat" | "project_token" | "repository_token";
  authIdentity?: string;
  token: string;
  kandevWorkspaceId?: string;
  repositoryNamespace: string;
  repositorySlug: string;
  pullRequestNumber: number;
  reviewWrites: boolean;
  approvalPullRequestNumber?: number;
  declinePullRequestNumber?: number;
};

function requiredEnvironment(name: string): string {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`${name} is required for configured Bitbucket acceptance`);
  return value;
}

function optionalEnvironment(name: string): string | undefined {
  return process.env[name]?.trim() || undefined;
}

function positiveEnvironmentInteger(name: string, required: boolean): number | undefined {
  const value = optionalEnvironment(name);
  if (!value && !required) return undefined;
  if (!value) throw new Error(`${name} is required for configured Bitbucket acceptance`);
  const number = Number(value);
  if (!Number.isSafeInteger(number) || number <= 0) {
    throw new Error(`${name} must be a positive integer`);
  }
  return number;
}

function liveTarget(): LiveTarget {
  const product = requiredEnvironment("KANDEV_BITBUCKET_LIVE_PRODUCT");
  if (product !== "cloud" && product !== "data_center") {
    throw new Error("KANDEV_BITBUCKET_LIVE_PRODUCT must be cloud or data_center");
  }
  const authMethod = requiredEnvironment("KANDEV_BITBUCKET_LIVE_AUTH_METHOD");
  const allowedAuthMethods =
    product === "cloud" ? ["api_token", "access_token"] : ["user_pat", "project_token", "repository_token"];
  if (!allowedAuthMethods.includes(authMethod)) {
    throw new Error(
      `KANDEV_BITBUCKET_LIVE_AUTH_METHOD must be one of ${allowedAuthMethods.join(", ")} for ${product}`,
    );
  }

  const reviewWrites = optionalEnvironment("KANDEV_BITBUCKET_LIVE_REVIEW_WRITES") === "1";
  const target: LiveTarget = {
    product,
    authMethod: authMethod as LiveTarget["authMethod"],
    token: requiredEnvironment("KANDEV_BITBUCKET_LIVE_TOKEN"),
    repositoryNamespace: requiredEnvironment("KANDEV_BITBUCKET_LIVE_REPOSITORY_NAMESPACE"),
    repositorySlug: requiredEnvironment("KANDEV_BITBUCKET_LIVE_REPOSITORY_SLUG"),
    pullRequestNumber: positiveEnvironmentInteger("KANDEV_BITBUCKET_LIVE_PR_NUMBER", true)!,
    reviewWrites,
  };

  target.kandevWorkspaceId = optionalEnvironment("KANDEV_BITBUCKET_LIVE_KANDEV_WORKSPACE_ID");
  target.authIdentity = optionalEnvironment("KANDEV_BITBUCKET_LIVE_AUTH_IDENTITY");
  if (product === "cloud") {
    target.cloudWorkspace = requiredEnvironment("KANDEV_BITBUCKET_LIVE_CLOUD_WORKSPACE");
    if (authMethod === "api_token") {
      target.authIdentity = requiredEnvironment("KANDEV_BITBUCKET_LIVE_AUTH_IDENTITY");
    }
  } else {
    target.baseUrl = requiredEnvironment("KANDEV_BITBUCKET_LIVE_BASE_URL");
    if (authMethod === "user_pat") {
      target.authIdentity = requiredEnvironment("KANDEV_BITBUCKET_LIVE_AUTH_IDENTITY");
    }
  }

  if (reviewWrites) {
    target.approvalPullRequestNumber = positiveEnvironmentInteger(
      "KANDEV_BITBUCKET_LIVE_APPROVAL_PR_NUMBER",
      true,
    );
    target.declinePullRequestNumber = positiveEnvironmentInteger(
      "KANDEV_BITBUCKET_LIVE_DECLINE_PR_NUMBER",
      true,
    );
  }
  return target;
}

async function responseJSON(response: APIResponse): Promise<JsonRecord> {
  const text = await response.text();
  if (!response.ok()) {
    throw new Error(`Bitbucket live action failed with ${response.status()}: ${text}`);
  }
  return JSON.parse(text) as JsonRecord;
}

async function invokeAction(
  request: APIRequestContext,
  workspaceId: string,
  key: string,
  body: JsonRecord = {},
): Promise<JsonRecord> {
  const response = await request.post(`/api/plugins/${pluginID}/actions/${key}`, {
    data: { workspaceId, body },
  });
  return responseJSON(response);
}

async function resolveKandevWorkspace(
  request: APIRequestContext,
  configured?: string,
): Promise<string> {
  const response = await request.get("/api/v1/workspaces");
  const text = await response.text();
  if (!response.ok()) throw new Error(`Could not list disposable Kandev workspaces: ${text}`);
  const payload = JSON.parse(text) as { workspaces?: Array<{ id?: string }> };
  const ids = (payload.workspaces ?? []).flatMap((workspace) =>
    workspace.id ? [workspace.id] : [],
  );
  if (configured) {
    if (!ids.includes(configured)) {
      throw new Error(
        "KANDEV_BITBUCKET_LIVE_KANDEV_WORKSPACE_ID is not present on the disposable host",
      );
    }
    return configured;
  }
  if (ids.length !== 1) {
    throw new Error(
      "Set KANDEV_BITBUCKET_LIVE_KANDEV_WORKSPACE_ID when the disposable host does not have exactly one workspace",
    );
  }
  return ids[0];
}

function connectionBody(target: LiveTarget): JsonRecord {
  const body: JsonRecord = {
    product: target.product,
    auth_method: target.authMethod,
    token: target.token,
    probe: true,
  };
  if (target.baseUrl) body.base_url = target.baseUrl;
  if (target.cloudWorkspace) body.cloud_workspace = target.cloudWorkspace;
  if (target.authIdentity) body.auth_identity = target.authIdentity;
  return body;
}

function records(value: unknown): JsonRecord[] {
  return Array.isArray(value)
    ? value.filter((item): item is JsonRecord => Boolean(item) && typeof item === "object")
    : [];
}

function reviewKey(target: LiveTarget, number: number): string {
  return `${target.repositoryNamespace}/${target.repositorySlug}#${number}`;
}

test("exercises a configured disposable Bitbucket target through the packaged plugin", async ({
  request,
}) => {
  const target = liveTarget();
  const workspaceId = await resolveKandevWorkspace(request, target.kandevWorkspaceId);

  try {
    const connection = await invokeAction(
      request,
      workspaceId,
      "connection.save",
      connectionBody(target),
    );
    expect(connection).toMatchObject({
      state: "connected",
      healthy: true,
      product: target.product,
    });

    const repositoryResult = await invokeAction(request, workspaceId, "repositories.list", {
      query: target.repositorySlug,
      limit: 100,
    });
    const repository = records(repositoryResult.repositories).find(
      (candidate) =>
        candidate.owner_or_project === target.repositoryNamespace &&
        candidate.name === target.repositorySlug,
    );
    expect(repository, "configured repository must be discoverable").toBeDefined();
    expect(repository?.clone_url).toEqual(expect.stringMatching(/^https:\/\/[^@]+$/));

    const branches = await invokeAction(request, workspaceId, "repositories.branches", {
      repository,
    });
    expect(records(branches.branches).length).toBeGreaterThan(0);

    const key = reviewKey(target, target.pullRequestNumber);
    const pullRequest = await invokeAction(request, workspaceId, "pullrequests.inspect", {
      review_key: key,
    });
    expect(pullRequest).toMatchObject({
      review_key: key,
      number: target.pullRequestNumber,
    });
    expect(pullRequest.url).toEqual(expect.stringMatching(/^https:\/\/[^@]+$/));

    const review = await invokeAction(request, workspaceId, "reviews.get", {
      review_key: key,
    });
    expect(review).toMatchObject({
      review_key: key,
      number: target.pullRequestNumber,
    });
    expect(Array.isArray(review.files)).toBe(true);
    expect(Array.isArray(review.commits)).toBe(true);
    expect(Array.isArray(review.threads)).toBe(true);
    expect(Array.isArray(review.statuses)).toBe(true);

    if (target.reviewWrites) {
      const marker = `Kandev live acceptance ${Date.now()}`;
      await invokeAction(request, workspaceId, "reviews.action", {
        review_key: key,
        kind: "add_comment",
        comment: `${marker}. This comment was created in a disposable test repository.`,
      });

      const approvalKey = reviewKey(target, target.approvalPullRequestNumber!);
      let approved = false;
      try {
        await invokeAction(request, workspaceId, "reviews.action", {
          review_key: approvalKey,
          kind: "approve",
        });
        approved = true;
      } finally {
        if (approved) {
          await invokeAction(request, workspaceId, "reviews.action", {
            review_key: approvalKey,
            kind: "unapprove",
          });
        }
      }

      const declineKey = reviewKey(target, target.declinePullRequestNumber!);
      await invokeAction(request, workspaceId, "reviews.action", {
        review_key: declineKey,
        kind: "decline",
      });
    }
  } finally {
    await invokeAction(request, workspaceId, "connection.disconnect").catch(() => undefined);
  }
});

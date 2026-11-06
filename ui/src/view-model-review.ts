import type {
  HostChangeRequestDetail,
  HostChangeRequestDetailAction,
  JsonRecord,
  PullRequest,
  ReviewComment,
  ReviewDetail,
} from "./view-model-base";
import { translateEnglish, type Translate } from "./i18n";
import {
  boolean,
  itemList,
  normalizeReviewComment,
  normalizeTaskLinks,
  number,
  personLink,
  pullRequestURL,
  record,
  statusTone,
  string,
  timestamp,
  toCapabilities,
} from "./view-model-base";
export function normalizePullRequests(
  value: unknown,
  t: Translate = translateEnglish,
): PullRequest[] {
  return itemList(value, ["pull_requests", "pullRequests", "items", "values"])
    .map((item) => {
      const source = record(item);
      const id =
        string(source.id) ??
        string(source.key) ??
        string(source.uuid) ??
        String(number(source.number) ?? number(source.id) ?? "");
      const repository = record(source.repository);
      const repositorySlug = string(repository.slug) ?? string(repository.name);
      const repositoryNamespace =
        string(record(repository.project).key) ??
        string(record(repository.owner).username) ??
        string(record(repository.workspace).slug);
      const repositoryId =
        string(source.repository_id) ??
        string(repository.provider_repository_id) ??
        string(source.repositoryId) ??
        string(repository.full_name) ??
        (repositoryNamespace && repositorySlug
          ? `${repositoryNamespace}/${repositorySlug}`
          : undefined) ??
        string(repository.id) ??
        "";
      const numberValue = number(source.number) ?? number(source.id) ?? 0;
      const title =
        string(source.title) ??
        t("pullRequestNumber", { values: { number: numberValue || id } });
      if (!id || !repositoryId || !numberValue) return null;
      const status = record(source.status);
      const state = string(source.state) ?? string(source.status) ?? "UNKNOWN";
      const providerScope =
        string(source.provider_scope) ?? string(repository.provider_scope);
      const reviewKey =
        string(source.review_key) ??
        string(source.reviewKey) ??
        `${repositoryId}:${id}`;
      const author = record(source.author);
      const sourceRef = record(source.source);
      const destinationRef = record(source.destination);
      const fromRef = record(source.fromRef);
      const toRef = record(source.toRef);
      return {
        key: reviewKey,
        id,
        number: numberValue,
        title,
        url: string(source.url) ?? pullRequestURL(source.links) ?? "",
        repositoryId,
        ...(providerScope ? { providerScope } : {}),
        repositoryName:
          string(source.repository_name) ??
          string(source.repositoryName) ??
          string(repository.name) ??
          repositoryId,
        state,
        author:
          string(source.author_display_name) ??
          string(source.authorDisplayName) ??
          string(author.display_name) ??
          string(author.displayName) ??
          string(author.name) ??
          string(source.author),
        authorUrl:
          string(source.author_url) ??
          string(source.authorUrl) ??
          personLink(author, "html"),
        authorAvatarUrl:
          string(source.author_avatar_url) ??
          string(source.authorAvatarUrl) ??
          personLink(author, "avatar"),
        createdAt:
          timestamp(source.created_at) ??
          timestamp(source.createdAt) ??
          timestamp(source.created_on) ??
          timestamp(source.createdDate),
        updatedAt:
          timestamp(source.updated_at) ??
          timestamp(source.updatedAt) ??
          timestamp(source.updated_on) ??
          timestamp(source.updatedDate),
        mergedAt: timestamp(source.merged_at) ?? timestamp(source.mergedAt),
        closedAt: timestamp(source.closed_at) ?? timestamp(source.closedAt),
        sourceBranch:
          string(source.source_branch) ??
          string(source.sourceBranch) ??
          string(record(sourceRef.branch).name) ??
          string(sourceRef.branch) ??
          string(fromRef.displayId) ??
          string(fromRef.id)?.replace(/^refs\/heads\//, ""),
        destinationBranch:
          string(source.destination_branch) ??
          string(source.destinationBranch) ??
          string(record(destinationRef.branch).name) ??
          string(destinationRef.branch) ??
          string(toRef.displayId) ??
          string(toRef.id)?.replace(/^refs\/heads\//, ""),
        headCommit:
          string(source.head_commit) ??
          string(source.headCommit) ??
          string(record(sourceRef.commit).hash) ??
          string(fromRef.latestCommit),
        statusLabel:
          string(status.label) ??
          string(source.status_label) ??
          string(source.statusLabel),
        statusTone: statusTone(string(status.state) ?? state),
        tasks: normalizeTaskLinks(
          source.associations ?? source.tasks,
          reviewKey,
          t,
        ),
        capabilities: toCapabilities(source.capabilities),
      };
    })
    .flatMap((item): PullRequest[] => (item ? [item] : []));
}

function isCurrentUserParticipant(participant: JsonRecord): boolean {
  const user = record(participant.user);
  return (
    participant.is_current_user === true ||
    participant.isCurrentUser === true ||
    participant.currentUser === true ||
    user.is_current_user === true ||
    user.isCurrentUser === true
  );
}

function normalizeParticipantVerdict(
  participant: JsonRecord,
  approved: boolean | undefined,
): ReviewDetail["participants"][number]["verdict"] {
  const raw = (string(participant.verdict) ?? string(participant.status))
    ?.trim()
    .toUpperCase();
  if (
    ["NEEDS_WORK", "CHANGES_REQUESTED", "REQUEST_CHANGES"].includes(raw ?? "")
  ) {
    return "changes_requested";
  }
  if (raw === "APPROVED" || approved) return "approved";
  return "pending";
}

function normalizeViewerApproval(
  source: JsonRecord,
  participants: unknown[],
): boolean | undefined {
  const direct =
    boolean(source.viewer_approved) ??
    boolean(source.viewerApproved) ??
    boolean(source.current_user_approved) ??
    boolean(source.currentUserApproved);
  if (direct !== undefined) return direct;
  const viewer = participants.map(record).find(isCurrentUserParticipant);
  if (!viewer) return undefined;
  return (
    boolean(viewer.approved) ??
    string(viewer.status)?.toUpperCase() === "APPROVED"
  );
}

export function normalizeReviewDetail(
  value: unknown,
  t: Translate = translateEnglish,
): ReviewDetail | null {
  const source = record(value);
  const pr = normalizePullRequests({ pull_requests: [source] }, t)[0];
  if (!pr) return null;
  const participantItems = itemList(source.participants ?? source.reviewers, [
    "items",
    "values",
  ]);
  return {
    ...pr,
    description:
      string(source.description) ?? string(record(source.description).raw),
    sourceBranch: pr.sourceBranch,
    destinationBranch: pr.destinationBranch,
    files: itemList(source.files, ["items", "values"]).flatMap((entry) => {
      const file = record(entry);
      const path = string(file.path) ?? string(file.name);
      return path
        ? [
            {
              path,
              status: string(file.status) ?? "modified",
              additions: number(file.additions),
              deletions: number(file.deletions),
              patch: string(file.patch),
            },
          ]
        : [];
    }),
    commits: itemList(source.commits, ["items", "values"]).flatMap((entry) => {
      const commit = record(entry);
      const id = string(commit.id) ?? string(commit.hash);
      return id
        ? [
            {
              id,
              message: string(commit.message) ?? id,
              author:
                string(commit.author) ?? string(record(commit.author).name),
            },
          ]
        : [];
    }),
    participants: participantItems.flatMap((entry) => {
      const participant = record(entry);
      const user = record(participant.user);
      const name =
        string(participant.name) ??
        string(participant.display_name) ??
        string(participant.displayName) ??
        string(user.display_name) ??
        string(user.displayName) ??
        string(user.name);
      const approved =
        boolean(participant.approved) ??
        string(participant.status)?.toUpperCase() === "APPROVED";
      const verdict = normalizeParticipantVerdict(participant, approved);
      const isCurrentUser =
        boolean(participant.is_current_user) ??
        boolean(participant.isCurrentUser) ??
        boolean(participant.currentUser) ??
        boolean(user.is_current_user) ??
        boolean(user.isCurrentUser);
      return name
        ? [
            {
              id:
                string(participant.id) ??
                string(user.account_id) ??
                string(user.slug),
              name,
              role: string(participant.role),
              approved,
              verdict,
              isCurrentUser,
              url: personLink(user, "html"),
              avatarUrl: personLink(user, "avatar"),
            },
          ]
        : [];
    }),
    threads: itemList(source.threads, ["items", "values"]).flatMap((entry) => {
      const thread = record(entry);
      const id = string(thread.id) ?? string(thread.comment_id);
      if (!id) return [];
      const comments = itemList(thread.comments, ["items", "values"]).flatMap(
        (comment): ReviewComment[] => {
          const normalized = normalizeReviewComment(comment, t);
          return normalized ? [normalized] : [];
        },
      );
      const rootComment = comments[0];
      return [
        {
          id,
          author:
            rootComment?.author ??
            string(thread.author) ??
            string(record(thread.author).display_name) ??
            t("unknown"),
          body:
            rootComment?.body ??
            string(thread.body) ??
            string(thread.content) ??
            "",
          createdAt:
            rootComment?.createdAt ??
            timestamp(thread.created_at) ??
            timestamp(thread.createdAt),
          resolved: thread.resolved === true,
          file: string(thread.file) ?? string(thread.path),
          comments:
            comments.length > 0
              ? comments
              : [
                  {
                    id,
                    author:
                      string(thread.author) ??
                      string(record(thread.author).display_name) ??
                      t("unknown"),
                    body: string(thread.body) ?? string(thread.content) ?? "",
                  },
                ],
        },
      ];
    }),
    statuses: itemList(source.statuses ?? source.builds, [
      "items",
      "values",
    ]).flatMap((entry) => {
      const status = record(entry);
      const key = string(status.key) ?? string(status.id);
      const name = string(status.name) ?? key;
      const state = string(status.state);
      const url = string(status.url) ?? pullRequestURL(status.links);
      const target = string(status.target) ?? string(status.refname);
      const output = string(status.output) ?? string(status.description);
      const startedAt =
        timestamp(status.started_at) ??
        timestamp(status.startedAt) ??
        timestamp(status.created_on) ??
        timestamp(status.createdDate);
      const completedAt =
        timestamp(status.completed_at) ??
        timestamp(status.completedAt) ??
        timestamp(status.updated_on) ??
        timestamp(status.updatedDate);
      return key && name && state
        ? [
            {
              key,
              name,
              state,
              ...(url ? { url } : {}),
              ...(target ? { target } : {}),
              ...(output ? { output } : {}),
              ...(startedAt ? { startedAt } : {}),
              ...(completedAt ? { completedAt } : {}),
            },
          ]
        : [];
    }),
    viewerApproved: normalizeViewerApproval(source, participantItems),
    unresolvedThreadCount: number(source.unresolved_thread_count),
  };
}

function hostChangeRequestState(value: string): string {
  const state = value.trim().toUpperCase();
  if (state === "MERGED") return "merged";
  if (["DECLINED", "CLOSED", "SUPERSEDED"].includes(state)) return "closed";
  if (state === "DRAFT") return "draft";
  return "open";
}

export function changeRequestDetailModel(
  detail: ReviewDetail,
  t: Translate = translateEnglish,
): HostChangeRequestDetail {
  const reviewers = detail.participants.filter((participant) =>
    ["REVIEWER", "APPROVER"].includes(
      participant.role?.toUpperCase() ?? "REVIEWER",
    ),
  );
  const approved = reviewers.filter((participant) => participant.approved);
  const changesRequested = reviewers.filter(
    (participant) => participant.verdict === "changes_requested",
  );
  const requested = reviewers.filter(
    (participant) =>
      participant.verdict !== "approved" &&
      participant.verdict !== "changes_requested",
  );
  const person = (participant: ReviewDetail["participants"][number]) => ({
    name: participant.name,
    ...(participant.url ? { url: participant.url } : {}),
    ...(participant.avatarUrl ? { avatarUrl: participant.avatarUrl } : {}),
  });
  const comments = detail.threads.flatMap((thread) =>
    thread.comments.map((comment) => ({
      id: comment.id,
      ...(comment.parentId ? { parentId: comment.parentId } : {}),
      author: { name: comment.author },
      body: comment.body,
      ...(comment.createdAt ? { createdAt: comment.createdAt } : {}),
      ...(thread.file ? { path: thread.file } : {}),
      ...(comment.line ? { line: comment.line } : {}),
      resolved: thread.resolved,
    })),
  );
  return {
    providerId: "bitbucket",
    reviewKey: detail.key,
    number: detail.number,
    title: detail.title,
    url: detail.url,
    state: hostChangeRequestState(detail.state),
    ...(detail.state.trim().toUpperCase() === "DRAFT" ? { draft: true } : {}),
    author: {
      name: detail.author ?? t("unknown"),
      ...(detail.authorUrl ? { url: detail.authorUrl } : {}),
      ...(detail.authorAvatarUrl ? { avatarUrl: detail.authorAvatarUrl } : {}),
    },
    ...(detail.createdAt ? { createdAt: detail.createdAt } : {}),
    ...(detail.mergedAt ? { mergedAt: detail.mergedAt } : {}),
    ...(detail.closedAt ? { closedAt: detail.closedAt } : {}),
    sourceBranch: detail.sourceBranch ?? "source",
    targetBranch: detail.destinationBranch ?? "destination",
    additions: detail.files.reduce(
      (total, file) => total + (file.additions ?? 0),
      0,
    ),
    deletions: detail.files.reduce(
      (total, file) => total + (file.deletions ?? 0),
      0,
    ),
    ...(detail.description ? { description: detail.description } : {}),
    ...(changesRequested.length > 0
      ? { reviewState: "changes_requested" }
      : approved.length > 0
        ? { reviewState: "approved" }
        : requested.length > 0
          ? { reviewState: "pending" }
          : {}),
    ...(requested.length > 0 ? { pendingReviewCount: requested.length } : {}),
    reviews: [
      ...approved.map((participant) => ({
        id: participant.id ?? participant.name,
        author: person(participant),
        state: "APPROVED",
      })),
      ...changesRequested.map((participant) => ({
        id: participant.id ?? participant.name,
        author: person(participant),
        state: "CHANGES_REQUESTED",
      })),
    ],
    requestedReviewers: requested.map(person),
    checks: detail.statuses.map((status) => ({
      id: status.key,
      name: status.name,
      state: status.state,
      ...(status.url ? { url: status.url } : {}),
      ...(status.output ? { output: status.output } : {}),
      ...(status.startedAt ? { startedAt: status.startedAt } : {}),
      ...(status.completedAt ? { completedAt: status.completedAt } : {}),
    })),
    comments,
    ...(detail.updatedAt ? { lastSyncedAt: detail.updatedAt } : {}),
  };
}

export function changeRequestDetailActions(
  detail: ReviewDetail,
  t: Translate = translateEnglish,
): HostChangeRequestDetailAction[] {
  if (hostChangeRequestState(detail.state) !== "open") return [];
  const can = (capability: string) => detail.capabilities.includes(capability);
  const actions: HostChangeRequestDetailAction[] = [];
  if (can("approve")) {
    const approveAction = {
      id: "approve",
      label: t("approve"),
      pendingLabel: t("approving"),
      placement: "header" as const,
      tone: "success" as const,
    };
    const unapproveAction = {
      id: "unapprove",
      label: t("removeApproval"),
      pendingLabel: t("removingApproval"),
      placement: "header" as const,
      tone: "secondary" as const,
    };
    if (detail.viewerApproved === undefined) {
      actions.push(approveAction, unapproveAction);
    } else {
      actions.push(detail.viewerApproved ? unapproveAction : approveAction);
    }
  }
  if (can("merge")) {
    actions.push({
      id: "merge",
      label: t("merge"),
      pendingLabel: t("merging"),
      placement: "header",
    });
  }
  if (can("decline")) {
    actions.push({
      id: "decline",
      label: t("decline"),
      pendingLabel: t("declining"),
      placement: "header",
      tone: "danger",
    });
  }
  if (can("comments")) {
    actions.push({
      id: "comment",
      label: t("comment"),
      placement: "comment",
      input: "text",
    });
  }
  if (can("thread_replies")) {
    actions.push({
      id: "reply",
      label: t("reply"),
      placement: "thread",
      input: "text",
    });
  }
  return actions;
}

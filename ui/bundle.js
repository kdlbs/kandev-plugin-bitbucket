// ui/src/i18n.ts
var english = {
  bitbucket: "Bitbucket",
  settings: "Settings",
  openSettings: "Open Bitbucket settings",
  pullRequests: "Pull requests",
  pullRequest: "Pull request",
  pullRequestNumber: "Pull request {{number}}",
  connectDescription: "Connect Bitbucket Cloud or Data Center for this workspace.",
  connection: "Connection",
  notConfigured: "Not configured",
  checkingConnection: "Checking connection",
  connected: "Connected",
  authenticationRequired: "Authentication required",
  unavailable: "Unavailable",
  connectionSaved: "Connection saved and health check started.",
  oauthReady: "OAuth authorization is ready. Continue in the connection settings.",
  disconnected: "Bitbucket disconnected. Stored credentials cleared.",
  disconnectBitbucket: "Disconnect Bitbucket",
  disconnectTitle: "Disconnect Bitbucket connection?",
  disconnectDescription: "Stored Bitbucket credentials and connection settings for this workspace will be removed.",
  disconnectWorkspaceDescription: "Remove this workspace Bitbucket connection.",
  disconnecting: "Disconnecting\u2026",
  cancel: "Cancel",
  bitbucketProduct: "Bitbucket product",
  bitbucketCloud: "Bitbucket Cloud",
  bitbucketDataCenter: "Bitbucket Data Center",
  bitbucketWorkspace: "Bitbucket workspace",
  workspaceHelp: "Workspace slug or ID from bitbucket.org/workspace; required to list repositories.",
  dataCenterUrl: "Data Center URL",
  authentication: "Authentication",
  apiToken: "API token",
  cloudAccessToken: "Repository, project or workspace access token",
  oauth: "OAuth 2.0",
  personalAccessToken: "Personal access token",
  projectAccessToken: "Project access token",
  repositoryAccessToken: "Repository access token",
  accessToken: "Access token",
  tokenPlaceholder: "Stored only by Bitbucket secret handling",
  atlassianEmail: "Atlassian account email",
  atlassianEmailHelp: "Used with this API token for Bitbucket Cloud REST. Git uses x-bitbucket-api-token-auth.",
  bitbucketUsername: "Bitbucket username",
  bitbucketUsernameHelp: "Used for HTTPS Git with this Data Center PAT or OAuth credential.",
  enterIdentity: "Enter {{identity}}.",
  invalidAtlassianEmail: "Enter a valid Atlassian account email.",
  enterCloudWorkspace: "Enter Bitbucket Cloud workspace slug or ID.",
  oauthClientRegistration: "OAuth client registration",
  oauthRegistrationConfigured: "OAuth app registration is configured. Enter both values only to replace it.",
  oauthClientId: "OAuth client ID",
  oauthClientIdReplace: "OAuth client ID (optional to replace)",
  oauthClientSecret: "OAuth client secret",
  oauthClientSecretReplace: "OAuth client secret (optional to replace)",
  oauthSecretHelp: "Stored only by Bitbucket secret handling. Existing secrets are never displayed.",
  oauthCallbackUrl: "OAuth callback URL",
  oauthCallbackHelp: "Copy this Kandev callback URL into your OAuth app. It is derived from this Kandev origin.",
  connectWithOauth: "Connect with OAuth",
  checkConnection: "Check connection",
  linkPullRequestMenu: "Bitbucket Pull Request",
  linkPullRequestTitle: "Link Bitbucket pull request",
  linkPullRequestDescription: "Use a Bitbucket pull request URL or canonical key for this task.",
  linkPullRequestEmpty: "Enter a Bitbucket pull request URL or key.",
  linkPullRequestFailure: "Failed to link Bitbucket pull request.",
  linkPullRequestSuccess: "Bitbucket pull request linked",
  pullRequestNoun: "pull request",
  filterRepositoryAria: "Filter Bitbucket pull requests by repository",
  allRepositories: "All repositories",
  filterRepositories: "Filter repositories...",
  noRepositories: "No repositories found.",
  open: "Open",
  all: "All",
  merged: "Merged",
  declined: "Declined",
  openFilters: "Open Bitbucket filters",
  filters: "Filters",
  bitbucketFilters: "Bitbucket filters",
  filtersDescription: "Narrow pull requests by repository and state.",
  repository: "Repository",
  checkingBitbucketConnection: "Checking Bitbucket connection",
  verifyingConnection: "Kandev is verifying the saved connection.",
  connectToLoad: "Connect Bitbucket for this workspace to load pull requests.",
  bitbucketNeedsAttention: "Bitbucket needs attention",
  configureBitbucket: "Configure Bitbucket",
  customQueryPlaceholder: 'Custom query: press Enter, for example "state:open fix login"',
  pullRequestResults: "Pull request results",
  saveQueryDescription: "Save this Bitbucket pull-request search for the current workspace.",
  repositoryPullRequests: "Repository pull requests",
  savedQuery: "Saved query",
  chooseWorkspace: "Choose a workspace",
  chooseWorkspaceDescription: "Open Bitbucket from a workspace to connect and browse pull requests.",
  noMatchingPullRequests: "No pull requests match this filter.",
  byAuthor: "by {{author}}",
  openedAgo: "opened {{value}}",
  watches: "Watches",
  watchesDescription: "Poll saved pull-request criteria and create only plugin-owned tasks.",
  noSavedWatches: "No saved watches.",
  addFilterWatch: "Add current filter watch",
  lastPolled: "Last polled {{value}}",
  runNow: "Run now",
  running: "Running",
  paused: "Paused",
  pause: "Pause",
  resume: "Resume",
  reset: "Reset",
  delete: "Delete",
  confirmReset: "Confirm reset",
  confirmDelete: "Confirm delete",
  watchResetWarning: "Resetting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchResetWarning_one: "Resetting this watch will remove {{count}} plugin-owned task. Adopted and manual tasks stay untouched.",
  watchResetWarning_other: "Resetting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchDeleteWarning: "Deleting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchDeleteWarning_one: "Deleting this watch will remove {{count}} plugin-owned task. Adopted and manual tasks stay untouched.",
  watchDeleteWarning_other: "Deleting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  reviewPreset: "Review",
  reviewPresetHint: "Read the diff, flag issues",
  reviewPresetPrompt: "Review Bitbucket pull request {{reference}}. Inspect the changes, run relevant tests, and report concrete findings.",
  addressFeedbackPreset: "Address feedback",
  addressFeedbackPresetHint: "Apply review comments",
  addressFeedbackPresetPrompt: "Address the review feedback on Bitbucket pull request {{reference}}. Make the requested changes, verify them, and summarize what changed.",
  fixCiPreset: "Fix CI",
  fixCiPresetHint: "Diagnose failing checks",
  fixCiPresetPrompt: "Fix the failing CI checks on Bitbucket pull request {{reference}}. Reproduce the failures, implement the smallest correct fix, and run the relevant checks.",
  removeApproval: "Remove approval",
  removingApproval: "Removing approval\u2026",
  approve: "Approve",
  approving: "Approving\u2026",
  merge: "Merge",
  merging: "Merging\u2026",
  decline: "Decline",
  declining: "Declining\u2026",
  comment: "Comment",
  reply: "Reply",
  unknown: "Unknown",
  bitbucketTask: "Bitbucket task",
  taskLaunchUnavailable: "Bitbucket task launch is unavailable.",
  pullRequestIdentityUnavailable: "Bitbucket pull request identity is unavailable. Refresh and try again.",
  taskLaunchMissingId: "Bitbucket task launch returned no task id.",
  enterOauthClientId: "Enter OAuth client ID.",
  enterOauthClientSecret: "Enter OAuth client secret.",
  oauthCallbackUnavailable: "OAuth callback URL is unavailable.",
  requestFailed: "Bitbucket request failed. Try again."
};
var pseudoGlyphs = {
  a: "\xE0",
  b: "\u0180",
  c: "\xE7",
  d: "\u010F",
  e: "\xE9",
  f: "\u0192",
  g: "\u011F",
  h: "\u0125",
  i: "\xED",
  j: "\u0135",
  k: "\u0137",
  l: "\u013C",
  m: "\u1E3F",
  n: "\xF1",
  o: "\xF6",
  p: "\xFE",
  q: "q",
  r: "\u0159",
  s: "\u0161",
  t: "\u0163",
  u: "\xFC",
  v: "\u1E7D",
  w: "\u0175",
  x: "\u1E8B",
  y: "\xFD",
  z: "\u017E"
};
function pseudoMessage(message) {
  const transformed = message.split(/(\{\{[^}]+\}\})/g).map(
    (part) => part.startsWith("{{") ? part : part.replace(/[A-Za-z]/g, (letter) => {
      const glyph = pseudoGlyphs[letter.toLowerCase()] ?? letter;
      return letter === letter.toUpperCase() ? glyph.toUpperCase() : glyph;
    })
  ).join("");
  return `[${transformed}${"~".repeat(Math.ceil(message.length * 0.3))}]`;
}
var pseudo = Object.fromEntries(
  Object.entries(english).map(([key, message]) => [
    key,
    pseudoMessage(message)
  ])
);
var translationCatalogs = { en: english, pseudo };
function interpolate(message, options) {
  const values = {
    count: options?.count,
    ...options?.values
  };
  return message.replace(
    /\{\{([^}]+)\}\}/g,
    (_match, key) => values[key] === void 0 ? `{{${key}}}` : String(values[key])
  );
}
var translateEnglish = (key, options) => {
  const pluralKey = `${key}_${options?.count === 1 ? "one" : "other"}`;
  const message = options?.count !== void 0 && pluralKey in english ? english[pluralKey] : english[key];
  return interpolate(message, options);
};
function usePluginTranslation(host) {
  const translation = host.i18n.useTranslation();
  return {
    locale: translation.locale,
    t: translation.t
  };
}
function pluginTranslate(host) {
  return (key, options) => host.i18n.t(key, options);
}
function registerTranslations(registry) {
  registry.registerTranslations(translationCatalogs);
}

// ui/src/view-model-base.ts
function pullRequestAssociationIdentity(connectionScope, repositoryId, number2) {
  return connectionScope && repositoryId && number2 && number2 > 0 ? `connection:${connectionScope}\0repository:${repositoryId}\0pull-request:${number2}` : void 0;
}
function integrationSettingsHref(workspaceId) {
  return workspaceId ? `/settings/workspaces/${encodeURIComponent(workspaceId)}/integrations/bitbucket` : "/settings/integrations/bitbucket";
}
function displayPullRequestAuthor(author) {
  const value = author?.trim();
  if (!value || /^\d+:[a-z0-9-]{20,}$/i.test(value)) return void 0;
  return value;
}
function oauthStartInput(workspaceId) {
  return { workspaceId };
}
function disconnectConnectionInput(workspaceId) {
  return { workspaceId };
}
function deriveOAuthCallbackURL(apiBaseUrl, browserOrigin) {
  const backendOrigin = apiBaseUrl.trim() || browserOrigin;
  return new URL(
    "/api/plugins/kandev-plugin-bitbucket/webhooks/oauth-callback",
    backendOrigin
  ).toString();
}
function record(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value) ? value : {};
}
function string(value) {
  return typeof value === "string" && value.trim() ? value : void 0;
}
function number(value) {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() && Number.isFinite(Number(value)))
    return Number(value);
  return void 0;
}
function array(value) {
  return Array.isArray(value) ? value : [];
}
function timestamp(value) {
  if (typeof value === "number" && Number.isFinite(value)) {
    const date2 = new Date(value);
    return Number.isFinite(date2.getTime()) && date2.getUTCFullYear() > 1 ? date2.toISOString() : void 0;
  }
  const text3 = string(value);
  if (!text3) return void 0;
  if (/^\d+$/.test(text3)) {
    const date2 = new Date(Number(text3));
    return Number.isFinite(date2.getTime()) && date2.getUTCFullYear() > 1 ? date2.toISOString() : void 0;
  }
  const date = new Date(text3);
  return Number.isFinite(date.getTime()) && date.getUTCFullYear() > 1 ? text3 : void 0;
}
function boolean(value) {
  return typeof value === "boolean" ? value : void 0;
}
function linkHref(value) {
  const direct = string(value);
  if (direct) return direct;
  if (Array.isArray(value)) {
    for (const candidate of value) {
      const href = linkHref(candidate);
      if (href) return href;
    }
    return void 0;
  }
  const link = record(value);
  return string(link.href) ?? string(link.url);
}
function pullRequestURL(value) {
  const links = record(value);
  return linkHref(links.html) ?? linkHref(links.self) ?? linkHref(value);
}
function personLink(value, kind) {
  const person = record(value);
  return linkHref(record(person.links)[kind]) ?? linkHref(person[kind === "html" ? "url" : "avatarUrl"]);
}
function itemList(value, keys) {
  if (Array.isArray(value)) return value;
  const source = record(value);
  for (const key of keys) {
    if (Array.isArray(source[key])) return source[key];
  }
  return [];
}
function toCapabilities(value) {
  if (typeof value === "string") {
    return value.split(",").map((capability) => capability.trim()).filter(Boolean);
  }
  return array(value).flatMap(
    (entry) => typeof entry === "string" ? [entry] : []
  );
}
function normalizeTaskLinks(value, reviewKey = "", t = translateEnglish) {
  return itemList(value, ["associations", "tasks", "items", "values"]).flatMap(
    (entry) => {
      const source = record(entry);
      const taskId = string(source.task_id) ?? string(source.taskId);
      const entryReviewKey = string(source.review_key) ?? string(source.reviewKey) ?? reviewKey;
      if (!taskId || reviewKey && entryReviewKey !== reviewKey) return [];
      return [
        {
          id: string(source.id) ?? `${entryReviewKey}:${taskId}`,
          taskId,
          fallbackTitle: string(source.task_title) ?? string(source.taskTitle) ?? string(source.title) ?? t("bitbucketTask")
        }
      ];
    }
  );
}
function normalizePullRequestAssociations(value, t = translateEnglish) {
  const result = {};
  for (const entry of itemList(value, ["associations", "items", "values"])) {
    const source = record(entry);
    const reviewKey = string(source.review_key) ?? string(source.reviewKey);
    if (!reviewKey) continue;
    const links = normalizeTaskLinks([source], reviewKey, t);
    const repositoryId = string(source.repository_id) ?? string(source.repositoryId);
    const connectionScope = string(source.provider_scope) ?? string(source.connectionScope);
    const changeRequestNumber = number(source.number) ?? number(source.changeRequestNumber);
    for (const link of links) {
      link.repositoryId = repositoryId;
      link.connectionScope = connectionScope;
      link.changeRequestNumber = changeRequestNumber;
    }
    if (links.length) {
      result[reviewKey] = [...result[reviewKey] ?? [], ...links];
      const identity = pullRequestAssociationIdentity(
        connectionScope,
        repositoryId,
        changeRequestNumber
      );
      if (identity) result[identity] = [...result[identity] ?? [], ...links];
    }
  }
  return result;
}
function normalizeReviewComment(value, t = translateEnglish) {
  const comment = record(value);
  const id = string(comment.id) ?? string(comment.ID) ?? string(comment.comment_id);
  if (!id) return null;
  const normalized = {
    id,
    author: string(comment.author) ?? string(comment.Author) ?? string(record(comment.author).display_name) ?? string(record(comment.author).displayName) ?? t("unknown"),
    body: string(comment.body) ?? string(comment.Body) ?? string(comment.content) ?? string(record(comment.content).raw) ?? string(comment.text) ?? ""
  };
  const parentId = string(comment.parent_id) ?? string(comment.parentId) ?? string(comment.ParentID);
  const createdAt = timestamp(comment.created_at) ?? timestamp(comment.createdAt) ?? timestamp(comment.When);
  const line = number(comment.line) ?? number(record(comment.inline).to) ?? number(record(comment.anchor).line);
  if (parentId) normalized.parentId = parentId;
  if (createdAt) normalized.createdAt = createdAt;
  if (line) normalized.line = line;
  return normalized;
}
function statusTone(state) {
  const normalized = state.toLowerCase();
  if (/(success|passed|approved|merged|open)/.test(normalized))
    return "success";
  if (/(fail|declined|error|blocked)/.test(normalized)) return "danger";
  if (/(pending|build|review|draft)/.test(normalized)) return "warning";
  return "neutral";
}

// ui/src/view-model-repository.ts
function normalizeRepository(value) {
  const source = record(value);
  const repositoryId = string(source.provider_repository_id) ?? string(source.repositoryId) ?? string(source.provider_repo_id) ?? string(source.id) ?? string(source.uuid) ?? string(source.slug);
  const repositoryName = string(source.repository_name) ?? string(source.repositoryName) ?? string(source.provider_name) ?? string(source.name) ?? string(source.slug) ?? repositoryId;
  if (!repositoryId || !repositoryName) return null;
  const repository = {
    providerId: string(source.provider_id) ?? string(source.providerId) ?? string(source.provider) ?? "bitbucket",
    providerHost: string(source.provider_host) ?? string(source.providerHost) ?? string(source.host) ?? "",
    providerScope: string(source.provider_scope) ?? string(source.providerScope),
    ownerOrProject: string(source.owner_or_project) ?? string(source.ownerOrProject) ?? string(source.provider_owner) ?? string(source.project) ?? string(record(source.owner).username) ?? "",
    repositoryId,
    repositoryName,
    cloneUrl: string(source.clone_url) ?? string(source.cloneUrl) ?? string(source.remote_url) ?? string(source.url) ?? ""
  };
  const defaultBranch = string(source.default_branch) ?? string(source.defaultBranch);
  const baseBranch = string(source.base_branch) ?? string(source.baseBranch);
  const headBranch = string(source.head_branch) ?? string(source.headBranch) ?? string(source.checkout_branch);
  if (defaultBranch) repository.defaultBranch = defaultBranch;
  if (baseBranch) repository.baseBranch = baseBranch;
  if (headBranch) repository.headBranch = headBranch;
  return repository;
}
function normalizeRepositories(value) {
  return itemList(value, ["repositories", "items", "values"]).flatMap(
    (item) => {
      const repository = normalizeRepository(item);
      return repository ? [repository] : [];
    }
  );
}
function normalizeRepositoryInspection(value) {
  const source = record(value);
  const repository = normalizeRepository(
    Object.keys(record(source.repository)).length ? source.repository : source
  );
  if (!repository) return null;
  const baseBranch = string(source.base_branch) ?? string(source.baseBranch);
  const headBranch = string(source.head_branch) ?? string(source.headBranch);
  const pullRequest = record(source.pull_request ?? source.pullRequest);
  const pullRequestNumber = number(pullRequest.number);
  const pullRequestTitle = string(pullRequest.title);
  if (baseBranch) repository.baseBranch = baseBranch;
  if (headBranch) repository.headBranch = headBranch;
  if (pullRequestNumber && pullRequestTitle) {
    repository.pullRequest = {
      number: pullRequestNumber,
      title: pullRequestTitle
    };
  }
  return repository;
}
function workspaceReviewAction(workspaceId, taskId, identity, pullRequestID, kind, options = {}) {
  const body = {
    review_key: identity.reviewKey,
    provider_scope: identity.connectionScope,
    repository_id: identity.repositoryId,
    number: identity.changeRequestNumber,
    pull_request_id: pullRequestID,
    kind
  };
  const comment = options.comment?.trim();
  if (comment) body.comment = comment;
  if (options.parentCommentId) body.parent_comment_id = options.parentCommentId;
  if (options.buildKey) body.build_key = options.buildKey;
  return { workspaceId, taskId, body };
}
function linkPullRequestBody(reference) {
  const key = reference.trim();
  if (/^[^/#\s]+\/[^/#\s]+#[1-9]\d*$/.test(key)) return { review_key: key };
  let url;
  try {
    url = new URL(key);
  } catch {
    return null;
  }
  if (url.protocol !== "https:") return null;
  const cloud = url.pathname.match(
    /^\/([^/]+)\/([^/]+)\/pull-requests\/(\d+)(?:\/|$)/i
  );
  if (cloud) return { review_key: `${cloud[1]}/${cloud[2]}#${cloud[3]}` };
  const dataCenter = url.pathname.match(
    /(?:^|\/)projects\/([^/]+)\/repos\/([^/]+)\/pull-requests\/(\d+)(?:\/|$)/i
  );
  return dataCenter ? { review_key: `${dataCenter[1]}/${dataCenter[2]}#${dataCenter[3]}` } : null;
}
function pluginRepositoryInput(value) {
  const repository = normalizeRepository(value);
  if (!repository) return {};
  const body = {
    provider_id: repository.providerId,
    provider_host: repository.providerHost,
    provider_scope: repository.providerScope,
    owner_or_project: repository.ownerOrProject,
    provider_repository_id: repository.repositoryId,
    name: repository.repositoryName,
    clone_url: repository.cloneUrl
  };
  if (repository.defaultBranch) body.default_branch = repository.defaultBranch;
  if (repository.baseBranch) body.base_branch = repository.baseBranch;
  if (repository.headBranch) body.head_branch = repository.headBranch;
  return body;
}
function pullRequestListRequest(repository, query, state, cursor = "", limit = 25) {
  const parsed = parsePullRequestListQuery(query, state);
  if (repository) {
    return {
      actionKey: "pullrequests.search",
      body: {
        repository: pluginRepositoryInput(repository),
        query: parsed.query,
        state: parsed.state,
        cursor,
        limit
      }
    };
  }
  return {
    actionKey: "pullrequests.queue",
    body: {
      view: "queue",
      query: parsed.query,
      state: parsed.state,
      cursor,
      limit
    }
  };
}
var PULL_REQUEST_STATES = /* @__PURE__ */ new Set(["open", "all", "merged", "declined"]);
var PULL_REQUEST_STATE_TOKEN = /(^|\s)state:(open|all|merged|declined)(?=\s|$)/gi;
function normalizedPullRequestState(value) {
  const state = value.trim().toLowerCase();
  return PULL_REQUEST_STATES.has(state) ? state : "open";
}
function pullRequestScopeQuery(state) {
  return `state:${normalizedPullRequestState(state)}`;
}
function parsePullRequestListQuery(query, fallbackState) {
  let state = normalizedPullRequestState(fallbackState);
  const textQuery = query.replace(
    PULL_REQUEST_STATE_TOKEN,
    (_match, prefix, value) => {
      state = normalizedPullRequestState(value);
      return prefix;
    }
  );
  return { query: textQuery.trim().replace(/\s+/g, " "), state };
}
var MAX_SAVED_QUERIES = 50;
function normalizeSavedQueries(value) {
  return array(value).flatMap((entry) => {
    const source = record(entry);
    const id = string(source.id);
    const label = string(source.label);
    const query = typeof source.query === "string" ? source.query.trim() : void 0;
    const repositoryId = typeof source.repositoryId === "string" ? source.repositoryId.trim() : void 0;
    const state = string(source.state);
    const createdAt = string(source.createdAt);
    if (!id || !label || query === void 0 || repositoryId === void 0 || !state || !PULL_REQUEST_STATES.has(state.toLowerCase()) || !createdAt) {
      return [];
    }
    return [
      {
        id,
        label,
        query,
        repositoryId,
        state: state.toLowerCase(),
        createdAt
      }
    ];
  }).slice(-MAX_SAVED_QUERIES);
}
function canSaveDashboardQuery(query, repositoryId) {
  return Boolean(query.trim() || repositoryId.trim());
}
function newSavedQuery(input, id, createdAt) {
  return {
    id,
    label: input.label.trim(),
    query: input.query.trim(),
    repositoryId: input.repositoryId.trim(),
    state: normalizedPullRequestState(input.state),
    createdAt
  };
}

// ui/src/view-model-review.ts
function normalizePullRequests(value, t = translateEnglish) {
  return itemList(value, ["pull_requests", "pullRequests", "items", "values"]).map((item) => {
    const source = record(item);
    const id = string(source.id) ?? string(source.key) ?? string(source.uuid) ?? String(number(source.number) ?? number(source.id) ?? "");
    const repository = record(source.repository);
    const repositorySlug = string(repository.slug) ?? string(repository.name);
    const repositoryNamespace = string(record(repository.project).key) ?? string(record(repository.owner).username) ?? string(record(repository.workspace).slug);
    const repositoryId = string(source.repository_id) ?? string(repository.provider_repository_id) ?? string(source.repositoryId) ?? string(repository.full_name) ?? (repositoryNamespace && repositorySlug ? `${repositoryNamespace}/${repositorySlug}` : void 0) ?? string(repository.id) ?? "";
    const numberValue = number(source.number) ?? number(source.id) ?? 0;
    const title = string(source.title) ?? t("pullRequestNumber", { values: { number: numberValue || id } });
    if (!id || !repositoryId || !numberValue) return null;
    const status = record(source.status);
    const state = string(source.state) ?? string(source.status) ?? "UNKNOWN";
    const providerScope = string(source.provider_scope) ?? string(repository.provider_scope);
    const reviewKey = string(source.review_key) ?? string(source.reviewKey) ?? `${repositoryId}:${id}`;
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
      ...providerScope ? { providerScope } : {},
      repositoryName: string(source.repository_name) ?? string(source.repositoryName) ?? string(repository.name) ?? repositoryId,
      state,
      author: string(source.author_display_name) ?? string(source.authorDisplayName) ?? string(author.display_name) ?? string(author.displayName) ?? string(author.name) ?? string(source.author),
      authorUrl: string(source.author_url) ?? string(source.authorUrl) ?? personLink(author, "html"),
      authorAvatarUrl: string(source.author_avatar_url) ?? string(source.authorAvatarUrl) ?? personLink(author, "avatar"),
      createdAt: timestamp(source.created_at) ?? timestamp(source.createdAt) ?? timestamp(source.created_on) ?? timestamp(source.createdDate),
      updatedAt: timestamp(source.updated_at) ?? timestamp(source.updatedAt) ?? timestamp(source.updated_on) ?? timestamp(source.updatedDate),
      mergedAt: timestamp(source.merged_at) ?? timestamp(source.mergedAt),
      closedAt: timestamp(source.closed_at) ?? timestamp(source.closedAt),
      sourceBranch: string(source.source_branch) ?? string(source.sourceBranch) ?? string(record(sourceRef.branch).name) ?? string(sourceRef.branch) ?? string(fromRef.displayId) ?? string(fromRef.id)?.replace(/^refs\/heads\//, ""),
      destinationBranch: string(source.destination_branch) ?? string(source.destinationBranch) ?? string(record(destinationRef.branch).name) ?? string(destinationRef.branch) ?? string(toRef.displayId) ?? string(toRef.id)?.replace(/^refs\/heads\//, ""),
      headCommit: string(source.head_commit) ?? string(source.headCommit) ?? string(record(sourceRef.commit).hash) ?? string(fromRef.latestCommit),
      statusLabel: string(status.label) ?? string(source.status_label) ?? string(source.statusLabel),
      statusTone: statusTone(string(status.state) ?? state),
      tasks: normalizeTaskLinks(
        source.associations ?? source.tasks,
        reviewKey,
        t
      ),
      capabilities: toCapabilities(source.capabilities)
    };
  }).flatMap((item) => item ? [item] : []);
}
function isCurrentUserParticipant(participant) {
  const user = record(participant.user);
  return participant.is_current_user === true || participant.isCurrentUser === true || participant.currentUser === true || user.is_current_user === true || user.isCurrentUser === true;
}
function normalizeParticipantVerdict(participant, approved) {
  const raw = (string(participant.verdict) ?? string(participant.status))?.trim().toUpperCase();
  if (["NEEDS_WORK", "CHANGES_REQUESTED", "REQUEST_CHANGES"].includes(raw ?? "")) {
    return "changes_requested";
  }
  if (raw === "APPROVED" || approved) return "approved";
  return "pending";
}
function normalizeViewerApproval(source, participants) {
  const direct = boolean(source.viewer_approved) ?? boolean(source.viewerApproved) ?? boolean(source.current_user_approved) ?? boolean(source.currentUserApproved);
  if (direct !== void 0) return direct;
  const viewer = participants.map(record).find(isCurrentUserParticipant);
  if (!viewer) return void 0;
  return boolean(viewer.approved) ?? string(viewer.status)?.toUpperCase() === "APPROVED";
}
function normalizeReviewDetail(value, t = translateEnglish) {
  const source = record(value);
  const pr = normalizePullRequests({ pull_requests: [source] }, t)[0];
  if (!pr) return null;
  const participantItems = itemList(source.participants ?? source.reviewers, [
    "items",
    "values"
  ]);
  return {
    ...pr,
    description: string(source.description) ?? string(record(source.description).raw),
    sourceBranch: pr.sourceBranch,
    destinationBranch: pr.destinationBranch,
    files: itemList(source.files, ["items", "values"]).flatMap((entry) => {
      const file = record(entry);
      const path = string(file.path) ?? string(file.name);
      return path ? [
        {
          path,
          status: string(file.status) ?? "modified",
          additions: number(file.additions),
          deletions: number(file.deletions),
          patch: string(file.patch)
        }
      ] : [];
    }),
    commits: itemList(source.commits, ["items", "values"]).flatMap((entry) => {
      const commit = record(entry);
      const id = string(commit.id) ?? string(commit.hash);
      return id ? [
        {
          id,
          message: string(commit.message) ?? id,
          author: string(commit.author) ?? string(record(commit.author).name)
        }
      ] : [];
    }),
    participants: participantItems.flatMap((entry) => {
      const participant = record(entry);
      const user = record(participant.user);
      const name = string(participant.name) ?? string(participant.display_name) ?? string(participant.displayName) ?? string(user.display_name) ?? string(user.displayName) ?? string(user.name);
      const approved = boolean(participant.approved) ?? string(participant.status)?.toUpperCase() === "APPROVED";
      const verdict = normalizeParticipantVerdict(participant, approved);
      const isCurrentUser = boolean(participant.is_current_user) ?? boolean(participant.isCurrentUser) ?? boolean(participant.currentUser) ?? boolean(user.is_current_user) ?? boolean(user.isCurrentUser);
      return name ? [
        {
          id: string(participant.id) ?? string(user.account_id) ?? string(user.slug),
          name,
          role: string(participant.role),
          approved,
          verdict,
          isCurrentUser,
          url: personLink(user, "html"),
          avatarUrl: personLink(user, "avatar")
        }
      ] : [];
    }),
    threads: itemList(source.threads, ["items", "values"]).flatMap((entry) => {
      const thread = record(entry);
      const id = string(thread.id) ?? string(thread.comment_id);
      if (!id) return [];
      const comments = itemList(thread.comments, ["items", "values"]).flatMap(
        (comment) => {
          const normalized = normalizeReviewComment(comment, t);
          return normalized ? [normalized] : [];
        }
      );
      const rootComment = comments[0];
      return [
        {
          id,
          author: rootComment?.author ?? string(thread.author) ?? string(record(thread.author).display_name) ?? t("unknown"),
          body: rootComment?.body ?? string(thread.body) ?? string(thread.content) ?? "",
          createdAt: rootComment?.createdAt ?? timestamp(thread.created_at) ?? timestamp(thread.createdAt),
          resolved: thread.resolved === true,
          file: string(thread.file) ?? string(thread.path),
          comments: comments.length > 0 ? comments : [
            {
              id,
              author: string(thread.author) ?? string(record(thread.author).display_name) ?? t("unknown"),
              body: string(thread.body) ?? string(thread.content) ?? ""
            }
          ]
        }
      ];
    }),
    statuses: itemList(source.statuses ?? source.builds, [
      "items",
      "values"
    ]).flatMap((entry) => {
      const status = record(entry);
      const key = string(status.key) ?? string(status.id);
      const name = string(status.name) ?? key;
      const state = string(status.state);
      const url = string(status.url) ?? pullRequestURL(status.links);
      const target = string(status.target) ?? string(status.refname);
      const output = string(status.output) ?? string(status.description);
      const startedAt = timestamp(status.started_at) ?? timestamp(status.startedAt) ?? timestamp(status.created_on) ?? timestamp(status.createdDate);
      const completedAt = timestamp(status.completed_at) ?? timestamp(status.completedAt) ?? timestamp(status.updated_on) ?? timestamp(status.updatedDate);
      return key && name && state ? [
        {
          key,
          name,
          state,
          ...url ? { url } : {},
          ...target ? { target } : {},
          ...output ? { output } : {},
          ...startedAt ? { startedAt } : {},
          ...completedAt ? { completedAt } : {}
        }
      ] : [];
    }),
    viewerApproved: normalizeViewerApproval(source, participantItems),
    unresolvedThreadCount: number(source.unresolved_thread_count)
  };
}
function hostChangeRequestState(value) {
  const state = value.trim().toUpperCase();
  if (state === "MERGED") return "merged";
  if (["DECLINED", "CLOSED", "SUPERSEDED"].includes(state)) return "closed";
  if (state === "DRAFT") return "draft";
  return "open";
}
function changeRequestDetailModel(detail, t = translateEnglish) {
  const reviewers = detail.participants.filter(
    (participant) => ["REVIEWER", "APPROVER"].includes(
      participant.role?.toUpperCase() ?? "REVIEWER"
    )
  );
  const approved = reviewers.filter((participant) => participant.approved);
  const changesRequested = reviewers.filter(
    (participant) => participant.verdict === "changes_requested"
  );
  const requested = reviewers.filter(
    (participant) => participant.verdict !== "approved" && participant.verdict !== "changes_requested"
  );
  const person = (participant) => ({
    name: participant.name,
    ...participant.url ? { url: participant.url } : {},
    ...participant.avatarUrl ? { avatarUrl: participant.avatarUrl } : {}
  });
  const comments = detail.threads.flatMap(
    (thread) => thread.comments.map((comment) => ({
      id: comment.id,
      ...comment.parentId ? { parentId: comment.parentId } : {},
      author: { name: comment.author },
      body: comment.body,
      ...comment.createdAt ? { createdAt: comment.createdAt } : {},
      ...thread.file ? { path: thread.file } : {},
      ...comment.line ? { line: comment.line } : {},
      resolved: thread.resolved
    }))
  );
  return {
    providerId: "bitbucket",
    reviewKey: detail.key,
    number: detail.number,
    title: detail.title,
    url: detail.url,
    state: hostChangeRequestState(detail.state),
    ...detail.state.trim().toUpperCase() === "DRAFT" ? { draft: true } : {},
    author: {
      name: detail.author ?? t("unknown"),
      ...detail.authorUrl ? { url: detail.authorUrl } : {},
      ...detail.authorAvatarUrl ? { avatarUrl: detail.authorAvatarUrl } : {}
    },
    ...detail.createdAt ? { createdAt: detail.createdAt } : {},
    ...detail.mergedAt ? { mergedAt: detail.mergedAt } : {},
    ...detail.closedAt ? { closedAt: detail.closedAt } : {},
    sourceBranch: detail.sourceBranch ?? "source",
    targetBranch: detail.destinationBranch ?? "destination",
    additions: detail.files.reduce(
      (total, file) => total + (file.additions ?? 0),
      0
    ),
    deletions: detail.files.reduce(
      (total, file) => total + (file.deletions ?? 0),
      0
    ),
    ...detail.description ? { description: detail.description } : {},
    ...changesRequested.length > 0 ? { reviewState: "changes_requested" } : approved.length > 0 ? { reviewState: "approved" } : requested.length > 0 ? { reviewState: "pending" } : {},
    ...requested.length > 0 ? { pendingReviewCount: requested.length } : {},
    reviews: [
      ...approved.map((participant) => ({
        id: participant.id ?? participant.name,
        author: person(participant),
        state: "APPROVED"
      })),
      ...changesRequested.map((participant) => ({
        id: participant.id ?? participant.name,
        author: person(participant),
        state: "CHANGES_REQUESTED"
      }))
    ],
    requestedReviewers: requested.map(person),
    checks: detail.statuses.map((status) => ({
      id: status.key,
      name: status.name,
      state: status.state,
      ...status.url ? { url: status.url } : {},
      ...status.output ? { output: status.output } : {},
      ...status.startedAt ? { startedAt: status.startedAt } : {},
      ...status.completedAt ? { completedAt: status.completedAt } : {}
    })),
    comments,
    ...detail.updatedAt ? { lastSyncedAt: detail.updatedAt } : {}
  };
}
function changeRequestDetailActions(detail, t = translateEnglish) {
  if (hostChangeRequestState(detail.state) !== "open") return [];
  const can = (capability) => detail.capabilities.includes(capability);
  const actions = [];
  if (can("approve")) {
    const approveAction = {
      id: "approve",
      label: t("approve"),
      pendingLabel: t("approving"),
      placement: "header",
      tone: "success"
    };
    const unapproveAction = {
      id: "unapprove",
      label: t("removeApproval"),
      pendingLabel: t("removingApproval"),
      placement: "header",
      tone: "secondary"
    };
    if (detail.viewerApproved === void 0) {
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
      placement: "header"
    });
  }
  if (can("decline")) {
    actions.push({
      id: "decline",
      label: t("decline"),
      pendingLabel: t("declining"),
      placement: "header",
      tone: "danger"
    });
  }
  if (can("comments")) {
    actions.push({
      id: "comment",
      label: t("comment"),
      placement: "comment",
      input: "text"
    });
  }
  if (can("thread_replies")) {
    actions.push({
      id: "reply",
      label: t("reply"),
      placement: "thread",
      input: "text"
    });
  }
  return actions;
}

// ui/src/view-model-connection.ts
function connectionIdentity(product, authMethod, t = translateEnglish) {
  if (product === "cloud" && authMethod === "api_token") {
    return {
      field: "auth_identity",
      label: t("atlassianEmail"),
      help: t("atlassianEmailHelp"),
      inputType: "email"
    };
  }
  if (product === "data_center" && (authMethod === "user_pat" || authMethod === "oauth")) {
    return {
      field: "auth_identity",
      label: t("bitbucketUsername"),
      help: t("bitbucketUsernameHelp"),
      inputType: "text"
    };
  }
  return null;
}
function validateConnectionIdentity(input, t = translateEnglish) {
  const identity = connectionIdentity(input.product, input.authMethod, t);
  const value = input.identity.trim();
  if (!identity) return null;
  if (!value)
    return t("enterIdentity", {
      values: { identity: identity.label.toLowerCase() }
    });
  if (identity.inputType === "email" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
    return t("invalidAtlassianEmail");
  }
  return null;
}
function validateCloudWorkspace(input, t = translateEnglish) {
  if (input.product !== "cloud" || input.cloudWorkspace.trim()) return null;
  return t("enterCloudWorkspace");
}
function taskLaunchPresets(t = translateEnglish) {
  return [
    {
      id: "review",
      label: t("reviewPreset"),
      hint: t("reviewPresetHint"),
      iconName: "eye",
      prompt: (pullRequest) => t("reviewPresetPrompt", {
        values: { reference: pullRequest.url || pullRequest.key }
      })
    },
    {
      id: "address-feedback",
      label: t("addressFeedbackPreset"),
      hint: t("addressFeedbackPresetHint"),
      iconName: "message",
      prompt: (pullRequest) => t("addressFeedbackPresetPrompt", {
        values: { reference: pullRequest.url || pullRequest.key }
      })
    },
    {
      id: "fix-ci",
      label: t("fixCiPreset"),
      hint: t("fixCiPresetHint"),
      iconName: "tool",
      prompt: (pullRequest) => t("fixCiPresetPrompt", {
        values: { reference: pullRequest.url || pullRequest.key }
      })
    }
  ];
}
function taskDialogInitialValues(pullRequest, preset, hostRepositoryId, remoteRepository) {
  const values = {
    title: `${preset.label}: ${pullRequest.title}`,
    description: preset.prompt(pullRequest)
  };
  if (hostRepositoryId) values.repositoryId = hostRepositoryId;
  else if (pullRequest.url) {
    values.remoteUrl = pullRequest.url;
    if (remoteRepository) values.remoteRepository = remoteRepository;
  }
  if (pullRequest.sourceBranch) {
    values.branch = pullRequest.sourceBranch;
    values.checkoutBranch = pullRequest.sourceBranch;
  }
  return values;
}
function usePluginTaskCreation(hostRepositoryId) {
  return !hostRepositoryId?.trim();
}
function taskLaunchBody(pullRequest, payload, launchId, t = translateEnglish) {
  const task = {
    title: string(payload.title) ?? "",
    description: string(payload.description) ?? "",
    workflow_id: string(payload.workflow_id) ?? "",
    start_agent: payload.start_agent === true,
    plan_mode: payload.plan_mode === true
  };
  const workflowStepID = string(payload.workflow_step_id);
  const agentProfileID = string(payload.agent_profile_id);
  const executorProfileID = string(payload.executor_profile_id);
  if (workflowStepID) task.workflow_step_id = workflowStepID;
  if (agentProfileID) task.agent_profile_id = agentProfileID;
  if (executorProfileID) task.executor_profile_id = executorProfileID;
  return {
    ...pullRequestLookupBody(pullRequest, t),
    launch_id: launchId,
    task
  };
}
function pullRequestLookupBody(pullRequest, t = translateEnglish) {
  const providerScope = pullRequest.providerScope?.trim();
  if (!providerScope || !pullRequest.repositoryId.trim() || pullRequest.number <= 0 || !pullRequest.id.trim()) {
    throw new Error(t("pullRequestIdentityUnavailable"));
  }
  return {
    review_key: pullRequest.key,
    provider_scope: providerScope,
    repository_id: pullRequest.repositoryId,
    number: pullRequest.number,
    pull_request_id: pullRequest.id
  };
}
function taskFromLaunchResult(value, t = translateEnglish) {
  const result = record(value);
  const taskID = string(result.task_id);
  if (!taskID) throw new Error(t("taskLaunchMissingId"));
  return { id: taskID, bitbucketLinked: result.linked === true };
}
function normalizeWatches(value) {
  return itemList(value, ["watches", "items", "values"]).flatMap(
    (entry) => {
      const watch = record(entry);
      const id = string(watch.id);
      const rawStatus = string(watch.status)?.toLowerCase();
      if (!id || rawStatus !== "running" && rawStatus !== "paused") return [];
      const summary = { id, status: rawStatus };
      const lastPolled = string(watch.last_polled) ?? string(watch.lastPolled);
      if (lastPolled) summary.lastPolled = lastPolled;
      return [summary];
    }
  );
}
function validateOAuthRegistration(input, t = translateEnglish) {
  if (input.authMethod !== "oauth") return null;
  const clientID = input.oauthClientId.trim();
  const clientSecret = input.oauthClientSecret.trim();
  if (input.oauthRegistrationConfigured && !clientID && !clientSecret)
    return null;
  if (!clientID) return t("enterOauthClientId");
  if (!clientSecret) return t("enterOauthClientSecret");
  if (!input.oauthCallbackUrl.trim()) return t("oauthCallbackUnavailable");
  return null;
}
function connectionOAuthRegistration(value) {
  const source = record(value);
  return {
    configured: source.oauth_registration_configured === true
  };
}
function connectionActionBody(input) {
  const body = {
    product: input.product,
    auth_method: input.authMethod
  };
  const baseUrl = input.baseUrl.trim();
  const cloudWorkspace = input.cloudWorkspace.trim();
  const token = input.token.trim();
  const identity = input.identity.trim();
  const identityField = connectionIdentity(input.product, input.authMethod);
  if (baseUrl) body.base_url = baseUrl;
  if (input.product === "cloud" && cloudWorkspace)
    body.cloud_workspace = cloudWorkspace;
  if (input.authMethod !== "oauth" && token) body.token = token;
  if (identityField && identity) body[identityField.field] = identity;
  return body;
}
function connectionSaveBody(input) {
  const body = connectionActionBody(input);
  if (input.authMethod !== "oauth") return body;
  const clientID = input.oauthClientId.trim();
  const clientSecret = input.oauthClientSecret.trim();
  if (clientID || clientSecret) {
    body.oauth_client_id = clientID;
    body.oauth_client_secret = clientSecret;
    body.oauth_redirect_url = input.oauthCallbackUrl.trim();
  }
  return body;
}
function connectionState(value) {
  const candidate = string(record(value).state) ?? string(record(value).status) ?? "unconfigured";
  return [
    "unconfigured",
    "checking",
    "connected",
    "auth_required",
    "unavailable"
  ].includes(candidate) ? candidate : "unavailable";
}
function errorMessage(error, t = translateEnglish) {
  if (error instanceof Error && error.message) return error.message;
  const source = record(error);
  return string(source.message) ?? string(source.error) ?? t("requestFailed");
}

// ui/src/ui-runtime.ts
function record2(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value) ? value : {};
}
function text(value, fallback = "") {
  return typeof value === "string" && value.trim() ? value : fallback;
}
function useActiveWorkspaceId(host) {
  const { React } = host;
  const [activeWorkspaceId, setActiveWorkspaceId] = React.useState(
    () => host.context.getActiveWorkspaceId()
  );
  React.useEffect(() => {
    setActiveWorkspaceId(host.context.getActiveWorkspaceId());
    return host.context.subscribeActiveWorkspace(setActiveWorkspaceId);
  }, [host]);
  return activeWorkspaceId;
}
function useTaskCreationContext(host, workspaceId) {
  const { React } = host;
  const read = () => workspaceId ? host.context.getTaskCreationContext(workspaceId) : null;
  const [context, setContext] = React.useState(read);
  React.useEffect(() => {
    setContext(read());
    if (!workspaceId) return;
    return host.context.subscribeTaskCreationContext(workspaceId, setContext);
  }, [host, workspaceId]);
  return context;
}
var SAVED_QUERIES_KEY = "dashboard-saved-queries";
function useSavedQueries(host, workspaceId) {
  const { React } = host;
  const [queries, setQueries] = React.useState([]);
  const load = async () => {
    if (!workspaceId) {
      setQueries([]);
      return;
    }
    const entry = await host.storage.get(
      "workspace",
      workspaceId,
      SAVED_QUERIES_KEY
    );
    setQueries(normalizeSavedQueries(entry?.value));
  };
  React.useEffect(() => {
    if (!workspaceId) {
      setQueries([]);
      return;
    }
    let active = true;
    const sync = async () => {
      const entry = await host.storage.get(
        "workspace",
        workspaceId,
        SAVED_QUERIES_KEY
      );
      if (active) setQueries(normalizeSavedQueries(entry?.value));
    };
    void sync();
    const unsubscribe = host.storage.subscribe(
      { scope: "workspace", scopeId: workspaceId, key: SAVED_QUERIES_KEY },
      () => void sync()
    );
    return () => {
      active = false;
      unsubscribe();
    };
  }, [host, workspaceId]);
  const persist = async (next) => {
    if (!workspaceId) return;
    const normalized = normalizeSavedQueries(next);
    setQueries(normalized);
    try {
      await host.storage.set(
        "workspace",
        workspaceId,
        SAVED_QUERIES_KEY,
        normalized
      );
    } catch (error) {
      await load();
      throw error;
    }
  };
  return {
    queries,
    async save(input) {
      const created = newSavedQuery(
        input,
        `saved-${host.utils.generateUUID()}`,
        (/* @__PURE__ */ new Date()).toISOString()
      );
      await persist([...queries, created]);
      return created;
    },
    remove(id) {
      void persist(queries.filter((query) => query.id !== id));
    }
  };
}
function requestBody(input) {
  if (!input) return void 0;
  return JSON.parse(JSON.stringify(input));
}
function usePluginQuery(host, key, input, enabled = true) {
  const { React } = host;
  const { t } = usePluginTranslation(host);
  const serializedInput = JSON.stringify(input ?? {});
  const [reload, setReload] = React.useState(0);
  const [state, setState] = React.useState({
    data: null,
    loading: enabled,
    error: null,
    lastFetchedAt: null
  });
  React.useEffect(() => {
    let active = true;
    const controller = new AbortController();
    if (!enabled) {
      setState({
        data: null,
        loading: false,
        error: null,
        lastFetchedAt: null
      });
      return () => {
        active = false;
        controller.abort();
      };
    }
    setState((previous) => ({ ...previous, loading: true, error: null }));
    void host.api.invokeAction(
      key,
      requestBody(JSON.parse(serializedInput)),
      {
        signal: controller.signal
      }
    ).then((data) => {
      if (active)
        setState({
          data,
          loading: false,
          error: null,
          lastFetchedAt: /* @__PURE__ */ new Date()
        });
    }).catch((error) => {
      if (active)
        setState((previous) => ({
          ...previous,
          loading: false,
          error: errorMessage(error, t)
        }));
    });
    return () => {
      active = false;
      controller.abort();
    };
  }, [enabled, host.api, key, reload, serializedInput, t]);
  const refresh = React.useCallback(() => setReload((value) => value + 1), []);
  return { ...state, refresh };
}
async function collectPluginActionPages(api, key, input, itemKey, signal) {
  const items = [];
  const seenCursors = /* @__PURE__ */ new Set();
  let cursor = "";
  let lastPage = {};
  for (let page = 0; page < 1e3; page += 1) {
    const body = { ...record2(input?.body), cursor };
    const response = await api.invokeAction(
      key,
      requestBody({ ...input, body }),
      {
        signal
      }
    );
    if (signal.aborted) throw new DOMException("Request aborted", "AbortError");
    lastPage = record2(response);
    const pageItems = lastPage[itemKey];
    if (Array.isArray(pageItems)) items.push(...pageItems);
    const nextCursor = text(lastPage.next_cursor);
    if (!nextCursor) return { ...lastPage, [itemKey]: items, next_cursor: "" };
    if (seenCursors.has(nextCursor))
      throw new Error(`${key} pagination did not advance`);
    seenCursors.add(nextCursor);
    cursor = nextCursor;
  }
  throw new Error(`${key} pagination limit exceeded`);
}
function usePagedPluginQuery(host, key, input, itemKey, enabled = true) {
  const { React } = host;
  const { t } = usePluginTranslation(host);
  const serializedInput = JSON.stringify(input ?? {});
  const [reload, setReload] = React.useState(0);
  const [state, setState] = React.useState({ data: null, loading: enabled, error: null, lastFetchedAt: null });
  React.useEffect(() => {
    let active = true;
    const controller = new AbortController();
    if (!enabled) {
      setState({
        data: null,
        loading: false,
        error: null,
        lastFetchedAt: null
      });
      return () => {
        active = false;
        controller.abort();
      };
    }
    setState((previous) => ({ ...previous, loading: true, error: null }));
    void collectPluginActionPages(
      host.api,
      key,
      JSON.parse(serializedInput),
      itemKey,
      controller.signal
    ).then((data) => {
      if (active)
        setState({
          data,
          loading: false,
          error: null,
          lastFetchedAt: /* @__PURE__ */ new Date()
        });
    }).catch((error) => {
      if (active)
        setState((previous) => ({
          ...previous,
          loading: false,
          error: errorMessage(error, t)
        }));
    });
    return () => {
      active = false;
      controller.abort();
    };
  }, [enabled, host.api, itemKey, key, reload, serializedInput, t]);
  const refresh = React.useCallback(() => setReload((value) => value + 1), []);
  return { ...state, refresh };
}
function createAbortableOperation() {
  let active = null;
  let generation = 0;
  let disposed = false;
  const cancel = () => {
    generation += 1;
    active?.controller.abort();
    active = null;
  };
  return {
    begin() {
      if (disposed) throw new DOMException("Operation aborted", "AbortError");
      cancel();
      const controller = new AbortController();
      const operationGeneration = generation;
      active = { controller, generation: operationGeneration };
      const isCurrent = () => !disposed && active?.controller === controller && active.generation === operationGeneration && !controller.signal.aborted;
      return {
        signal: controller.signal,
        isCurrent,
        finish() {
          if (!isCurrent()) return false;
          active = null;
          return true;
        }
      };
    },
    cancel,
    dispose() {
      if (disposed) return;
      disposed = true;
      cancel();
    }
  };
}
function useAbortableOperation(host) {
  const { React } = host;
  const operation = React.useRef(null);
  if (!operation.current) operation.current = createAbortableOperation();
  React.useEffect(() => () => operation.current?.dispose(), []);
  return operation.current;
}
function isAbortError(reason) {
  return record2(reason).name === "AbortError";
}
function icon(h, name) {
  const paths = {
    watch: "M3 12s3.2-5 9-5 9 5 9 5-3.2 5-9 5-9-5-9-5Zm9 3a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z",
    back: "m15 18-6-6 6-6"
  };
  return h(
    "svg",
    {
      viewBox: "0 0 24 24",
      width: 18,
      height: 18,
      fill: "none",
      stroke: "currentColor",
      strokeWidth: 1.8,
      strokeLinecap: "round",
      strokeLinejoin: "round",
      "aria-hidden": true
    },
    h("path", { d: paths[name] ?? paths.back })
  );
}
function pullRequestStateIcon(host, pullRequest) {
  const normalized = pullRequest.state.toLowerCase();
  const merged = normalized === "merged";
  const closed = normalized === "declined" || normalized === "closed";
  return host.jsx(host.ui.IntegrationIcon, {
    name: merged ? "merged" : closed ? "pull-request-closed" : "pull-request",
    className: `h-4 w-4 ${merged ? "text-purple-600 dark:text-purple-400" : closed ? "text-red-600 dark:text-red-400" : "text-emerald-600 dark:text-emerald-400"}`
  });
}
function Badge(host, label, tone = "neutral") {
  return host.jsx(
    host.ui.Badge,
    { className: `bb-badge bb-badge-${tone}` },
    label
  );
}
function EmptyState(host, title, detail, actionLabel, onAction) {
  const { jsx: h, ui } = host;
  return h(
    "section",
    { className: "bb-empty", role: "status" },
    h("h2", null, title),
    h("p", null, detail),
    actionLabel && onAction ? h(
      ui.Button,
      { type: "button", className: "min-h-11", onClick: onAction },
      actionLabel
    ) : null
  );
}

// ui/src/actions.ts
var action = {
  connectionGet: "connection.get",
  connectionSave: "connection.save",
  connectionDisconnect: "connection.disconnect",
  oauthStart: "oauth.start",
  repositoriesList: "repositories.list",
  repositoriesBranches: "repositories.branches",
  repositoriesInspect: "repositories.inspect",
  pullRequestsGet: "pullrequests.get",
  pullRequestsAssociations: "pullrequests.associations",
  pullRequestsInspect: "pullrequests.inspect",
  pullRequestsLink: "pullrequests.link",
  pullRequestsCreate: "pullrequests.create",
  pullRequestsUnlink: "pullrequests.unlink",
  reviewsAction: "reviews.action",
  tasksLaunch: "tasks.launch",
  watchesGet: "watches.get",
  watchesUpdate: "watches.update",
  watchesRun: "watches.run",
  watchesPause: "watches.pause",
  watchesResume: "watches.resume",
  watchesPreviewReset: "watches.preview_reset",
  watchesPreviewDelete: "watches.preview_delete",
  watchesReset: "watches.reset",
  watchesDelete: "watches.delete"
};

// ui/src/disconnect-confirmation.ts
function DisconnectConfirmation({
  host,
  workspaceId,
  onSuccess,
  onCancel
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const [disconnecting, setDisconnecting] = React.useState(false);
  const [error, setError] = React.useState(null);
  const mutation = useAbortableOperation(host);
  const disconnect = async () => {
    setDisconnecting(true);
    setError(null);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.connectionDisconnect,
        disconnectConnectionInput(workspaceId),
        {
          signal: request.signal
        }
      );
      if (request.isCurrent()) onSuccess();
    } catch (reason) {
      if (request.isCurrent() && !isAbortError(reason))
        setError(errorMessage(reason, t));
    } finally {
      if (request.finish()) setDisconnecting(false);
    }
  };
  return h(
    "section",
    { className: "bb-disconnect-confirm" },
    h("p", null, t("disconnectTitle")),
    h("p", { className: "bb-capability-note" }, t("disconnectDescription")),
    error ? h("p", { className: "bb-error", role: "alert" }, error) : null,
    h(
      "div",
      { className: "bb-card-actions" },
      h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          disabled: disconnecting,
          onClick: onCancel
        },
        t("cancel")
      ),
      h(
        ui.Button,
        {
          type: "button",
          variant: "destructive",
          className: "min-h-11",
          disabled: disconnecting,
          onClick: () => void disconnect()
        },
        disconnecting ? t("disconnecting") : t("disconnectBitbucket")
      )
    )
  );
}

// ui/src/connection-view.ts
function ConnectionHealth({
  host,
  workspaceId: scopedWorkspaceId
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const responsive = host.useResponsiveBreakpoint();
  const connection = usePluginQuery(
    host,
    action.connectionGet,
    scopedWorkspaceId ? { workspaceId: scopedWorkspaceId } : void 0,
    Boolean(scopedWorkspaceId)
  );
  const [saving, setSaving] = React.useState(false);
  const [message, setMessage] = React.useState(null);
  const [product, setProduct] = React.useState("cloud");
  const [baseUrl, setBaseUrl] = React.useState("");
  const [cloudWorkspace, setCloudWorkspace] = React.useState("");
  const [authMethod, setAuthMethod] = React.useState("api_token");
  const [token, setToken] = React.useState("");
  const details = record2(connection.data);
  const [identity, setIdentity] = React.useState("");
  const oauthRegistration = connectionOAuthRegistration(details);
  const [oauthClientId, setOAuthClientId] = React.useState("");
  const [oauthClientSecret, setOAuthClientSecret] = React.useState("");
  const oauthCallbackUrl = deriveOAuthCallbackURL(
    host.api.baseUrl,
    window.location.origin
  );
  const [disconnectOpen, setDisconnectOpen] = React.useState(false);
  const mutation = useAbortableOperation(host);
  React.useEffect(() => {
    mutation.cancel();
    setSaving(false);
    setMessage(null);
    setProduct("cloud");
    setBaseUrl("");
    setCloudWorkspace("");
    setAuthMethod("api_token");
    setToken("");
    setIdentity("");
    setOAuthClientId("");
    setOAuthClientSecret("");
    setDisconnectOpen(false);
  }, [scopedWorkspaceId]);
  React.useEffect(() => {
    if (!connection.data) return;
    setProduct(text(details.product, "cloud"));
    setBaseUrl(text(details.base_url));
    setCloudWorkspace(text(details.cloud_workspace));
    setAuthMethod(text(details.auth_method, "api_token"));
    setIdentity(text(details.auth_identity));
  }, [connection.data]);
  const state = connectionState(details);
  const identityField = connectionIdentity(product, authMethod, t);
  const formInput = () => ({
    product,
    baseUrl,
    cloudWorkspace,
    authMethod,
    token,
    identity,
    oauthRegistrationConfigured: oauthRegistration.configured,
    oauthClientId,
    oauthClientSecret,
    oauthCallbackUrl
  });
  const connectionValidationError = () => validateCloudWorkspace(formInput(), t) ?? validateConnectionIdentity(formInput(), t) ?? validateOAuthRegistration(formInput(), t);
  const oauthReady = authMethod === "oauth" && !connectionValidationError();
  const label = {
    unconfigured: t("notConfigured"),
    checking: t("checkingConnection"),
    connected: t("connected"),
    auth_required: t("authenticationRequired"),
    unavailable: t("unavailable")
  };
  const productDescription = details.product === "cloud" ? t("bitbucketCloud") : details.product === "data_center" ? t("bitbucketDataCenter") : t("connectDescription");
  const saveConnection = async () => {
    if (!scopedWorkspaceId) return;
    const validationError = connectionValidationError();
    if (validationError) {
      setMessage(validationError);
      return;
    }
    setSaving(true);
    setMessage(null);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.connectionSave,
        {
          workspaceId: scopedWorkspaceId,
          body: {
            ...connectionSaveBody(formInput()),
            probe: true
          }
        },
        { signal: request.signal }
      );
      if (!request.isCurrent()) return;
      setToken("");
      setOAuthClientSecret("");
      setMessage(t("connectionSaved"));
      connection.refresh();
    } catch (error) {
      if (request.isCurrent() && !isAbortError(error))
        setMessage(errorMessage(error, t));
    } finally {
      if (request.finish()) setSaving(false);
    }
  };
  const startOauth = async () => {
    if (!scopedWorkspaceId) return;
    const validationError = connectionValidationError();
    if (validationError) {
      setMessage(validationError);
      return;
    }
    setSaving(true);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.connectionSave,
        {
          workspaceId: scopedWorkspaceId,
          body: {
            ...connectionSaveBody(formInput()),
            probe: false
          }
        },
        { signal: request.signal }
      );
      if (!request.isCurrent()) return;
      setOAuthClientSecret("");
      const result = await host.api.invokeAction(
        action.oauthStart,
        oauthStartInput(scopedWorkspaceId),
        { signal: request.signal }
      );
      if (!request.isCurrent()) return;
      const href = text(record2(result).url) || text(record2(result).authorization_url);
      if (href) window.location.assign(href);
      else setMessage(t("oauthReady"));
    } catch (error) {
      if (request.isCurrent() && !isAbortError(error))
        setMessage(errorMessage(error, t));
    } finally {
      if (request.finish()) setSaving(false);
    }
  };
  const completeDisconnect = () => {
    setToken("");
    setOAuthClientSecret("");
    setProduct("cloud");
    setBaseUrl("");
    setCloudWorkspace("");
    setAuthMethod("api_token");
    setIdentity("");
    setOAuthClientId("");
    setDisconnectOpen(false);
    setMessage(t("disconnected"));
    connection.refresh();
  };
  const openDisconnectConfirmation = () => {
    if (!scopedWorkspaceId) return;
    if (responsive.isMobile) {
      setDisconnectOpen(true);
      return;
    }
    let modal;
    modal = host.openModal({
      title: t("disconnectBitbucket"),
      size: "sm",
      content: () => h(DisconnectConfirmation, {
        host,
        workspaceId: scopedWorkspaceId,
        onSuccess: () => {
          completeDisconnect();
          modal?.close();
        },
        onCancel: () => modal?.close()
      })
    });
  };
  return h(
    ui.Card,
    {
      className: "bb-connection",
      "data-testid": "bitbucket-connection-health"
    },
    h(
      ui.CardHeader,
      null,
      h(
        "div",
        { className: "bb-title-row" },
        h(ui.CardTitle, null, t("connection")),
        Badge(
          host,
          label[state],
          state === "connected" ? "success" : state === "auth_required" ? "warning" : "neutral"
        )
      ),
      h(ui.CardDescription, null, productDescription)
    ),
    h(
      ui.CardContent,
      { className: "bb-settings-form" },
      connection.loading ? h(ui.Spinner, { "aria-label": t("checkingBitbucketConnection") }) : null,
      connection.error ? h("p", { className: "bb-error", role: "alert" }, connection.error) : null,
      message ? h("p", { className: "bb-message", role: "status" }, message) : null,
      h(ui.Label, { htmlFor: "bitbucket-product" }, t("bitbucketProduct")),
      h(
        ui.Select,
        {
          value: product,
          onValueChange: (next) => {
            setProduct(next);
            setAuthMethod(next === "cloud" ? "api_token" : "user_pat");
            setCloudWorkspace("");
            setIdentity("");
            setToken("");
            setOAuthClientId("");
            setOAuthClientSecret("");
          }
        },
        h(
          ui.SelectTrigger,
          { id: "bitbucket-product", className: "min-h-11" },
          h(ui.SelectValue, null)
        ),
        h(
          ui.SelectContent,
          null,
          h(ui.SelectItem, { value: "cloud" }, t("bitbucketCloud")),
          h(ui.SelectItem, { value: "data_center" }, t("bitbucketDataCenter"))
        )
      ),
      product === "cloud" ? h(
        "div",
        { className: "bb-field" },
        h(
          ui.Label,
          { htmlFor: "bitbucket-cloud-workspace" },
          t("bitbucketWorkspace")
        ),
        h(ui.Input, {
          id: "bitbucket-cloud-workspace",
          "data-testid": "bitbucket-cloud-workspace",
          className: "min-h-11",
          autoComplete: "organization",
          value: cloudWorkspace,
          onChange: (event) => setCloudWorkspace(event.target.value),
          placeholder: "workspace-slug",
          "aria-describedby": "bitbucket-cloud-workspace-help"
        }),
        h(
          "p",
          {
            id: "bitbucket-cloud-workspace-help",
            className: "bb-capability-note"
          },
          t("workspaceHelp")
        )
      ) : null,
      product === "data_center" ? h(
        "div",
        { className: "bb-field" },
        h(ui.Label, { htmlFor: "bitbucket-base-url" }, t("dataCenterUrl")),
        h(ui.Input, {
          id: "bitbucket-base-url",
          className: "min-h-11",
          value: baseUrl,
          onChange: (event) => setBaseUrl(event.target.value),
          placeholder: "https://bitbucket.example.com/bitbucket"
        })
      ) : null,
      h(ui.Label, { htmlFor: "bitbucket-auth-method" }, t("authentication")),
      h(
        ui.Select,
        {
          value: authMethod,
          onValueChange: (next) => {
            setAuthMethod(next);
            if (next === "oauth") setToken("");
            if (next !== "oauth") {
              setOAuthClientId("");
              setOAuthClientSecret("");
            }
          }
        },
        h(
          ui.SelectTrigger,
          { id: "bitbucket-auth-method", className: "min-h-11" },
          h(ui.SelectValue, null)
        ),
        h(
          ui.SelectContent,
          null,
          product === "cloud" ? [
            h(ui.SelectItem, { value: "api_token" }, t("apiToken")),
            h(ui.SelectItem, { value: "access_token" }, t("cloudAccessToken")),
            h(ui.SelectItem, { value: "oauth" }, t("oauth"))
          ] : [
            h(
              ui.SelectItem,
              { value: "user_pat" },
              t("personalAccessToken")
            ),
            h(
              ui.SelectItem,
              { value: "project_token" },
              t("projectAccessToken")
            ),
            h(
              ui.SelectItem,
              { value: "repository_token" },
              t("repositoryAccessToken")
            ),
            h(ui.SelectItem, { value: "oauth" }, t("oauth"))
          ]
        )
      ),
      authMethod !== "oauth" ? h(
        "div",
        { className: "bb-field" },
        h(ui.Label, { htmlFor: "bitbucket-token" }, t("accessToken")),
        h(ui.Input, {
          id: "bitbucket-token",
          type: "password",
          className: "min-h-11",
          autoComplete: "off",
          value: token,
          onChange: (event) => setToken(event.target.value),
          placeholder: t("tokenPlaceholder")
        })
      ) : null,
      identityField ? h(
        "div",
        { className: "bb-field" },
        h(
          ui.Label,
          { htmlFor: "bitbucket-connection-identity" },
          identityField.label
        ),
        h(ui.Input, {
          id: "bitbucket-connection-identity",
          "data-testid": "bitbucket-connection-identity",
          type: identityField.inputType,
          className: "min-h-11",
          autoComplete: identityField.inputType === "email" ? "email" : "username",
          value: identity,
          onChange: (event) => setIdentity(event.target.value),
          "aria-describedby": "bitbucket-connection-identity-help"
        }),
        h(
          "p",
          {
            id: "bitbucket-connection-identity-help",
            className: "bb-capability-note"
          },
          identityField.help
        )
      ) : null,
      authMethod === "oauth" ? h(
        "section",
        {
          className: "bb-oauth-registration",
          "aria-label": t("oauthClientRegistration")
        },
        oauthRegistration.configured ? h(
          "p",
          { className: "bb-capability-note", role: "status" },
          t("oauthRegistrationConfigured")
        ) : null,
        h(
          "div",
          { className: "bb-field" },
          h(
            ui.Label,
            { htmlFor: "bitbucket-oauth-client-id" },
            oauthRegistration.configured ? t("oauthClientIdReplace") : t("oauthClientId")
          ),
          h(ui.Input, {
            id: "bitbucket-oauth-client-id",
            "data-testid": "bitbucket-oauth-client-id",
            className: "min-h-11",
            autoComplete: "off",
            value: oauthClientId,
            onChange: (event) => setOAuthClientId(event.target.value)
          })
        ),
        h(
          "div",
          { className: "bb-field" },
          h(
            ui.Label,
            { htmlFor: "bitbucket-oauth-client-secret" },
            oauthRegistration.configured ? t("oauthClientSecretReplace") : t("oauthClientSecret")
          ),
          h(ui.Input, {
            id: "bitbucket-oauth-client-secret",
            "data-testid": "bitbucket-oauth-client-secret",
            type: "password",
            className: "min-h-11",
            autoComplete: "off",
            value: oauthClientSecret,
            onChange: (event) => setOAuthClientSecret(event.target.value),
            "aria-describedby": "bitbucket-oauth-client-secret-help"
          }),
          h(
            "p",
            {
              id: "bitbucket-oauth-client-secret-help",
              className: "bb-capability-note"
            },
            t("oauthSecretHelp")
          )
        ),
        h(
          "div",
          { className: "bb-field" },
          h(
            ui.Label,
            { htmlFor: "bitbucket-oauth-callback-url" },
            t("oauthCallbackUrl")
          ),
          h(ui.Input, {
            id: "bitbucket-oauth-callback-url",
            "data-testid": "bitbucket-oauth-callback-url",
            type: "url",
            className: "min-h-11",
            readOnly: true,
            value: oauthCallbackUrl,
            "aria-describedby": "bitbucket-oauth-callback-url-help"
          }),
          h(
            "p",
            {
              id: "bitbucket-oauth-callback-url-help",
              className: "bb-capability-note"
            },
            t("oauthCallbackHelp")
          )
        )
      ) : null,
      authMethod === "oauth" ? h(
        ui.Button,
        {
          type: "button",
          className: "min-h-11",
          disabled: saving || !oauthReady,
          onClick: startOauth
        },
        t("connectWithOauth")
      ) : null,
      h(ui.Separator, null),
      h(
        "div",
        { className: "bb-settings-actions" },
        h(
          ui.Button,
          {
            type: "button",
            variant: "outline",
            className: "min-h-11",
            disabled: saving,
            onClick: saveConnection
          },
          t("checkConnection")
        ),
        h(
          ui.Button,
          {
            type: "button",
            variant: "destructive",
            className: "bb-settings-disconnect min-h-11",
            disabled: saving || !scopedWorkspaceId,
            onClick: openDisconnectConfirmation
          },
          t("disconnectBitbucket")
        )
      ),
      responsive.isMobile && scopedWorkspaceId ? h(
        ui.Drawer,
        {
          open: disconnectOpen,
          onOpenChange: (open) => setDisconnectOpen(open)
        },
        h(
          ui.DrawerContent,
          { className: "bb-disconnect-drawer" },
          h(
            ui.DrawerHeader,
            null,
            h(ui.DrawerTitle, null, t("disconnectBitbucket")),
            h(
              ui.DrawerDescription,
              null,
              t("disconnectWorkspaceDescription")
            )
          ),
          h(
            "div",
            { className: "bb-drawer-scroll" },
            h(DisconnectConfirmation, {
              host,
              workspaceId: scopedWorkspaceId,
              onSuccess: completeDisconnect,
              onCancel: () => setDisconnectOpen(false)
            })
          )
        )
      ) : null
    )
  );
}

// ui/src/dashboard-task-list.ts
function DashboardPullRequestList({
  host,
  pullRequests,
  loading,
  error,
  tasksByReview,
  onStartTask
}) {
  const { jsx: h, ui } = host;
  const { t } = usePluginTranslation(host);
  const presets = taskLaunchPresets(t);
  return h(
    "div",
    { "data-testid": "bitbucket-pr-queue" },
    h(
      ui.ChangeRequestList,
      {
        loading,
        error,
        emptyMessage: t("noMatchingPullRequests"),
        isEmpty: pullRequests.length === 0
      },
      ...pullRequests.map((pullRequest) => {
        const author = displayPullRequestAuthor(pullRequest.author);
        const opened = pullRequest.createdAt ? host.utils.formatRelativeTime(pullRequest.createdAt) : void 0;
        const metadata = h(
          "span",
          { className: "bb-change-request-metadata" },
          h("span", null, pullRequest.key),
          author ? h("span", null, ` \xB7 ${t("byAuthor", { values: { author } })}`) : null,
          opened ? h(
            "span",
            null,
            ` \xB7 ${t("openedAgo", { values: { value: opened } })}`
          ) : null,
          pullRequest.sourceBranch && pullRequest.destinationBranch ? h(
            "span",
            null,
            ` \xB7 ${pullRequest.sourceBranch} \u2192 ${pullRequest.destinationBranch}`
          ) : null,
          h("span", null, " \xB7 "),
          Badge(
            host,
            pullRequest.statusLabel ?? pullRequest.state,
            pullRequest.statusTone
          )
        );
        const tasks = tasksForPullRequest(tasksByReview, pullRequest);
        return h(ui.ChangeRequestRow, {
          key: pullRequest.key,
          stateIcon: pullRequestStateIcon(host, pullRequest),
          title: pullRequest.title,
          href: pullRequest.url,
          metadata,
          taskIndicator: h(ui.TaskRowIndicator, {
            tasks,
            testIdPrefix: `bitbucket-pr-${pullRequest.number}-task`
          }),
          action: pullRequest.capabilities.includes("launch_task") ? h(ui.IntegrationStartTaskMenu, {
            presets,
            onSelect: (selected) => {
              const preset = presets.find(
                (candidate) => candidate.id === selected.id
              );
              if (preset) onStartTask(pullRequest, preset);
            },
            triggerTestId: "bitbucket-start-task-trigger",
            itemTestId: "bitbucket-start-task-preset"
          }) : null,
          testId: "bitbucket-pr-row",
          dataAttributes: { "data-pr-number": pullRequest.number }
        });
      })
    )
  );
}
function tasksForPullRequest(tasksByReview, pullRequest) {
  const identity = pullRequestAssociationIdentity(
    pullRequest.providerScope ?? connectionScopeFromURL(pullRequest.url),
    pullRequest.repositoryId,
    pullRequest.number
  );
  if (identity) {
    return tasksByReview[identity] ?? pullRequest.tasks;
  }
  return tasksByReview[pullRequest.key] ?? pullRequest.tasks;
}
function connectionScopeFromURL(rawURL) {
  try {
    return new URL(rawURL).origin;
  } catch {
    return void 0;
  }
}

// ui/src/dashboard-review.ts
function ReviewDetailPanel({
  host,
  workspaceId: scopedWorkspaceId,
  taskId,
  reviewKey,
  connectionScope,
  repositoryId,
  changeRequestNumber,
  presentation
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const review = usePluginQuery(
    host,
    action.pullRequestsGet,
    {
      workspaceId: scopedWorkspaceId,
      taskId,
      body: {
        review_key: reviewKey,
        provider_scope: connectionScope,
        repository_id: repositoryId,
        number: changeRequestNumber,
        include: ["files", "participants", "threads", "status", "viewer"]
      }
    },
    Boolean(
      scopedWorkspaceId && taskId && repositoryId && changeRequestNumber > 0
    )
  );
  const detail = normalizeReviewDetail(review.data, t);
  const [busyActionId, setBusyActionId] = React.useState(null);
  const [actionError, setActionError] = React.useState(null);
  const mutation = useAbortableOperation(host);
  const runAction = async (requestValue) => {
    if (!detail || busyActionId) return;
    const kind = requestValue.actionId === "comment" ? "add_comment" : requestValue.actionId;
    setBusyActionId(requestValue.actionId);
    setActionError(null);
    const request = mutation.begin();
    try {
      await host.api.invokeAction(
        action.reviewsAction,
        workspaceReviewAction(
          scopedWorkspaceId,
          taskId,
          {
            reviewKey: detail.key,
            connectionScope: detail.providerScope ?? connectionScope,
            repositoryId: detail.repositoryId,
            changeRequestNumber: detail.number
          },
          detail.id,
          kind,
          {
            ...requestValue.body ? { comment: requestValue.body } : {},
            ...requestValue.threadId ? { parentCommentId: requestValue.threadId } : {}
          }
        ),
        { signal: request.signal }
      );
      if (request.isCurrent()) review.refresh();
    } catch (reason) {
      if (request.isCurrent() && !isAbortError(reason))
        setActionError(errorMessage(reason, t));
    } finally {
      if (request.finish()) setBusyActionId(null);
    }
  };
  return h(ui.ChangeRequestDetail, {
    detail: detail ? changeRequestDetailModel(detail, t) : null,
    presentation: presentation ?? "desktop",
    loading: review.loading,
    error: review.error,
    onRefresh: review.refresh,
    onRetry: review.refresh,
    actions: detail ? changeRequestDetailActions(detail, t) : [],
    busyActionId,
    onAction: runAction,
    notice: actionError ? h("p", { className: "bb-error", role: "alert" }, actionError) : null
  });
}

// ui/src/dashboard-watches.ts
function watchPreviewTaskCount(value) {
  const response = record2(value);
  const taskIDs = response.task_ids ?? response.TaskIDs;
  return Array.isArray(taskIDs) ? taskIDs.length : 0;
}
function WatchRow({
  host,
  watch,
  disabled,
  run,
  preview
}) {
  const { jsx: h, ui } = host;
  const { t } = usePluginTranslation(host);
  const toggleKey = watch.status === "running" ? action.watchesPause : action.watchesResume;
  return h(
    "li",
    { className: "bb-watch-row", key: watch.id },
    h(
      "div",
      null,
      h("strong", null, watch.id),
      Badge(
        host,
        t(watch.status === "running" ? "running" : "paused"),
        watch.status === "running" ? "success" : "neutral"
      ),
      watch.lastPolled ? h(
        "span",
        null,
        t("lastPolled", {
          values: {
            value: host.utils.formatRelativeTime(watch.lastPolled)
          }
        })
      ) : null
    ),
    h(
      "div",
      { className: "bb-secondary-actions" },
      h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          disabled,
          onClick: () => run(action.watchesRun, watch.id)
        },
        t("runNow")
      ),
      h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          disabled,
          onClick: () => run(toggleKey, watch.id)
        },
        watch.status === "running" ? t("pause") : t("resume")
      ),
      h(
        ui.Button,
        {
          type: "button",
          variant: "ghost",
          className: "min-h-11",
          disabled,
          onClick: () => preview("reset", watch.id)
        },
        t("reset")
      ),
      h(
        ui.Button,
        {
          type: "button",
          variant: "destructive",
          className: "min-h-11",
          disabled,
          onClick: () => preview("delete", watch.id)
        },
        t("delete")
      )
    )
  );
}
function WatchConfirmation({
  host,
  pending,
  disabled,
  confirm,
  cancel
}) {
  const { jsx: h, ui } = host;
  const { t } = usePluginTranslation(host);
  const warningKey = pending.kind === "delete" ? "watchDeleteWarning" : "watchResetWarning";
  return h(
    "div",
    { className: "bb-watch-confirm", role: "alert" },
    h("p", null, t(warningKey, { count: pending.taskCount })),
    h(
      "div",
      { className: "bb-secondary-actions" },
      h(
        ui.Button,
        {
          type: "button",
          variant: "destructive",
          className: "min-h-11",
          disabled,
          onClick: confirm
        },
        t(pending.kind === "delete" ? "confirmDelete" : "confirmReset")
      ),
      h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          disabled,
          onClick: cancel
        },
        t("cancel")
      )
    )
  );
}
function Watches({
  host,
  workspaceId: scopedWorkspaceId,
  filter,
  showCreate = true
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const watches = usePluginQuery(
    host,
    action.watchesGet,
    scopedWorkspaceId ? { workspaceId: scopedWorkspaceId } : void 0,
    Boolean(scopedWorkspaceId)
  );
  const [working, setWorking] = React.useState(null);
  const [error, setError] = React.useState(null);
  const [pending, setPending] = React.useState(null);
  const mutation = useAbortableOperation(host);
  React.useEffect(() => {
    mutation.cancel();
    setWorking(null);
    setError(null);
    setPending(null);
  }, [scopedWorkspaceId]);
  const invoke = async (key, body) => {
    if (!scopedWorkspaceId) return null;
    setWorking(key);
    setError(null);
    const request = mutation.begin();
    try {
      const response = await host.api.invokeAction(
        key,
        { workspaceId: scopedWorkspaceId, body },
        { signal: request.signal }
      );
      if (!request.isCurrent()) return null;
      watches.refresh();
      return response;
    } catch (reason) {
      if (request.isCurrent() && !isAbortError(reason))
        setError(errorMessage(reason, t));
      return null;
    } finally {
      if (request.finish()) setWorking(null);
    }
  };
  const run = (key, watchId) => {
    void invoke(key, { watch_id: watchId });
  };
  const preview = async (kind, watchId) => {
    const key = kind === "reset" ? action.watchesPreviewReset : action.watchesPreviewDelete;
    const response = await invoke(key, { watch_id: watchId });
    if (response)
      setPending({ watchId, kind, taskCount: watchPreviewTaskCount(response) });
  };
  const confirm = async () => {
    if (!pending) return;
    const key = pending.kind === "reset" ? action.watchesReset : action.watchesDelete;
    const response = await invoke(key, { watch_id: pending.watchId });
    if (response) setPending(null);
  };
  const watchItems = normalizeWatches(watches.data);
  return h(
    ui.Card,
    { className: "bb-watches" },
    h(
      ui.CardHeader,
      null,
      h(ui.CardTitle, null, t("watches")),
      h(ui.CardDescription, null, t("watchesDescription"))
    ),
    h(
      ui.CardContent,
      { className: "bb-card-actions" },
      watches.error ? h("p", { className: "bb-error", role: "alert" }, watches.error) : null,
      error ? h("p", { className: "bb-error", role: "alert" }, error) : null,
      watchItems.length ? h(
        "ul",
        { className: "bb-watch-list" },
        ...watchItems.map(
          (watch) => h(WatchRow, {
            host,
            watch,
            disabled: Boolean(working),
            run,
            preview: (kind, watchId) => void preview(kind, watchId)
          })
        )
      ) : h("p", null, t("noSavedWatches")),
      pending ? h(WatchConfirmation, {
        host,
        pending,
        disabled: Boolean(working),
        confirm: () => void confirm(),
        cancel: () => setPending(null)
      }) : null,
      showCreate ? h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          disabled: Boolean(working),
          onClick: () => void invoke(action.watchesUpdate, { enabled: true, filter })
        },
        icon(h, "watch"),
        t("addFilterWatch")
      ) : null
    )
  );
}

// ui/src/dashboard-scope.ts
function repositoryFilter(host, repositories, repository, setRepository) {
  const { jsx: h, ui } = host;
  const t = pluginTranslate(host);
  return h(ui.IntegrationRepositoryFilter, {
    value: repository,
    onValueChange: setRepository,
    options: repositories.map((candidate) => {
      const label = `${candidate.ownerOrProject}/${candidate.repositoryName}`;
      return { value: candidate.repositoryId, label, keywords: [label] };
    }),
    ariaLabel: t("filterRepositoryAria"),
    allLabel: t("allRepositories"),
    searchPlaceholder: t("filterRepositories"),
    emptyMessage: t("noRepositories"),
    triggerClassName: "min-h-11 w-full border border-input bg-background px-2 py-1.5 text-xs/relaxed hover:bg-secondary/50 md:h-8 md:min-h-0 md:w-[220px]",
    className: "md:min-w-[360px]",
    testId: "bitbucket-repository-filter",
    dropdownTestId: "bitbucket-repository-filter-dropdown"
  });
}
function StateScopeBar({
  host,
  selection,
  savedQueries,
  onSelect,
  onDeleteSaved,
  canSaveCurrent,
  onSaveCurrent
}) {
  const { jsx: h, ui } = host;
  const { t } = usePluginTranslation(host);
  const presets = [
    {
      value: "open",
      label: t("open"),
      iconName: "pull-request",
      group: "inbox"
    },
    { value: "all", label: t("all"), iconName: "filter", group: "inbox" },
    {
      value: "merged",
      label: t("merged"),
      iconName: "merged",
      group: "created"
    },
    {
      value: "declined",
      label: t("declined"),
      iconName: "pull-request-closed",
      group: "created"
    }
  ];
  return h(ui.IntegrationScopeBar, {
    testId: "bitbucket-scope-bar",
    savedMenuTestId: "bitbucket-saved-filters",
    kinds: [{ value: "pull_requests", label: t("pullRequests") }],
    selected: selection,
    onSelect,
    presetsByKind: () => presets,
    savedPresets: savedQueries.map((query) => ({
      id: query.id,
      kind: "pull_requests",
      label: query.label
    })),
    onDeleteSaved,
    canSaveCurrent,
    onSaveCurrent
  });
}
function MobileFilters({
  host,
  repositories,
  repository,
  setRepository,
  ...scope
}) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const [open, setOpen] = React.useState(false);
  const mobileScope = {
    ...scope,
    onSelect(selection) {
      scope.onSelect(selection);
      setOpen(false);
    },
    onSaveCurrent() {
      setOpen(false);
      scope.onSaveCurrent();
    }
  };
  return h(
    ui.Sheet,
    { open, onOpenChange: setOpen },
    h(
      ui.SheetTrigger,
      { asChild: true },
      h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          "aria-label": t("openFilters")
        },
        h(ui.IntegrationIcon, { name: "filter", className: "h-4 w-4" }),
        t("filters")
      )
    ),
    h(
      ui.SheetContent,
      { side: "left", className: "bb-filter-sheet" },
      h(
        ui.SheetHeader,
        null,
        h(ui.SheetTitle, null, t("bitbucketFilters")),
        h(ui.SheetDescription, null, t("filtersDescription"))
      ),
      h(
        "div",
        { className: "bb-mobile-filter-fields" },
        h(StateScopeBar, { host, ...mobileScope }),
        h(
          ui.Label,
          { htmlFor: "bitbucket-repository-filter" },
          t("repository")
        ),
        repositoryFilter(host, repositories, repository, setRepository)
      )
    )
  );
}
function ConnectionNotice({
  host,
  connection,
  workspaceId
}) {
  const { jsx: h, ui } = host;
  const { t } = usePluginTranslation(host);
  if (connection.loading)
    return h(
      "div",
      { className: "bb-connection-loading" },
      h(ui.Spinner, { "aria-label": t("checkingBitbucketConnection") })
    );
  const state = connectionState(record2(connection.data));
  if (state === "connected") return null;
  const checking = state === "checking";
  const message = connection.error ?? (checking ? t("verifyingConnection") : t("connectToLoad"));
  return h(
    ui.Alert,
    { className: "bb-connection-notice" },
    h(
      ui.AlertTitle,
      null,
      checking ? t("checkingBitbucketConnection") : t("bitbucketNeedsAttention")
    ),
    h(
      ui.AlertDescription,
      { className: "bb-notice-content" },
      h("span", null, message),
      checking ? null : h(
        ui.Button,
        {
          type: "button",
          variant: "outline",
          className: "min-h-11",
          onClick: () => host.navigate(integrationSettingsHref(workspaceId))
        },
        t("configureBitbucket")
      )
    )
  );
}

// ui/src/bitbucket-page.ts
function BitbucketPage({ host }) {
  const { jsx: h, ui, React } = host;
  const { t } = usePluginTranslation(host);
  const responsive = host.useResponsiveBreakpoint();
  const activeWorkspaceId = useActiveWorkspaceId(host);
  const initialQuery = pullRequestScopeQuery("open");
  const [searchDraft, setSearchDraft] = React.useState(initialQuery);
  const [search, setSearch] = React.useState(initialQuery);
  const [state, setState] = React.useState("open");
  const [repository, setRepository] = React.useState("");
  const [scopeSelection, setScopeSelection] = React.useState({
    kind: "pull_requests",
    source: "preset",
    id: "open"
  });
  const [saveDialogOpen, setSaveDialogOpen] = React.useState(false);
  const savedQueries = useSavedQueries(host, activeWorkspaceId);
  const [launch, setLaunch] = React.useState(null);
  const pluginCreatedTaskIDs = React.useRef(/* @__PURE__ */ new Set());
  const taskLaunchMutation = useAbortableOperation(host);
  const taskLinkMutation = useAbortableOperation(host);
  React.useEffect(() => {
    taskLaunchMutation.cancel();
    taskLinkMutation.cancel();
    setLaunch(null);
  }, [activeWorkspaceId]);
  const connection = usePluginQuery(
    host,
    action.connectionGet,
    activeWorkspaceId ? { workspaceId: activeWorkspaceId } : void 0,
    Boolean(activeWorkspaceId)
  );
  const connected = connectionState(record2(connection.data)) === "connected";
  const repositoriesQuery = usePagedPluginQuery(
    host,
    action.repositoriesList,
    activeWorkspaceId ? { workspaceId: activeWorkspaceId, body: { limit: 100 } } : void 0,
    "repositories",
    Boolean(activeWorkspaceId && connected)
  );
  const repositories = normalizeRepositories(repositoriesQuery.data);
  const selectedRepository = repositories.find((candidate) => candidate.repositoryId === repository) ?? null;
  const queueScopeKey = JSON.stringify([
    activeWorkspaceId ?? "",
    selectedRepository?.repositoryId ?? "",
    search,
    state
  ]);
  const [queuePagination, setQueuePagination] = React.useState(() => ({ scopeKey: queueScopeKey, page: 1, cursors: [""] }));
  const activeQueuePagination = queuePagination.scopeKey === queueScopeKey ? queuePagination : { scopeKey: queueScopeKey, page: 1, cursors: [""] };
  const queueCursor = activeQueuePagination.cursors[activeQueuePagination.page - 1] ?? "";
  const queueRequest = pullRequestListRequest(
    selectedRepository,
    search,
    state,
    queueCursor,
    25
  );
  const queue = usePluginQuery(
    host,
    queueRequest.actionKey,
    activeWorkspaceId ? { workspaceId: activeWorkspaceId, body: queueRequest.body } : void 0,
    Boolean(activeWorkspaceId && connected)
  );
  const pullRequests = normalizePullRequests(queue.data, t);
  const nextQueueCursor = text(record2(queue.data).next_cursor);
  const associations = usePluginQuery(
    host,
    action.pullRequestsAssociations,
    activeWorkspaceId ? {
      workspaceId: activeWorkspaceId,
      body: {
        review_keys: pullRequests.map((pullRequest) => pullRequest.key)
      }
    } : void 0,
    Boolean(activeWorkspaceId && connected && pullRequests.length)
  );
  const tasksByReview = normalizePullRequestAssociations(associations.data, t);
  const createContext = useTaskCreationContext(host, activeWorkspaceId);
  const noWorkspace = !activeWorkspaceId;
  const commitSearch = () => {
    const committed = searchDraft.trim();
    setSearch(committed);
    setState(parsePullRequestListQuery(committed, state).state);
  };
  const selectScopeState = (nextState) => {
    const query = pullRequestScopeQuery(nextState);
    setState(nextState);
    setSearchDraft(query);
    setSearch(query);
    setScopeSelection({
      kind: "pull_requests",
      source: "preset",
      id: nextState
    });
  };
  const selectDashboardScope = (selection) => {
    if (selection.source === "preset") {
      selectScopeState(selection.id);
      return;
    }
    const saved = savedQueries.queries.find(
      (query) => query.id === selection.id
    );
    if (!saved) return;
    setScopeSelection(selection);
    setSearchDraft(saved.query);
    setSearch(saved.query);
    setState(saved.state);
    setRepository(saved.repositoryId);
  };
  const deleteSavedQuery = (id) => {
    savedQueries.remove(id);
    if (scopeSelection.source === "saved" && scopeSelection.id === id)
      selectScopeState("open");
  };
  const saveCurrentQuery = async (label, defaultRepositoryId) => {
    const query = searchDraft.trim();
    const parsed = parsePullRequestListQuery(query, state);
    const created = await savedQueries.save({
      label,
      query,
      repositoryId: defaultRepositoryId,
      state: parsed.state
    });
    setSearch(query);
    setState(parsed.state);
    setScopeSelection({
      kind: "pull_requests",
      source: "saved",
      id: created.id
    });
    setRepository(defaultRepositoryId);
  };
  const canSaveCurrent = canSaveDashboardQuery(searchDraft, repository);
  const scopeProps = {
    selection: scopeSelection,
    savedQueries: savedQueries.queries,
    onSelect: selectDashboardScope,
    onDeleteSaved: deleteSavedQuery,
    canSaveCurrent,
    onSaveCurrent: () => {
      if (canSaveCurrent) setSaveDialogOpen(true);
    }
  };
  const finishTaskCreation = async (taskValue) => {
    const taskId = text(record2(taskValue).id);
    if (!activeWorkspaceId || !launch || !taskId) return;
    const linkedByLaunch = pluginCreatedTaskIDs.current.delete(taskId);
    const request = taskLinkMutation.begin();
    try {
      if (!linkedByLaunch) {
        await host.api.invokeAction(
          action.pullRequestsLink,
          {
            workspaceId: activeWorkspaceId,
            taskId,
            body: pullRequestLookupBody(launch.pullRequest, t)
          },
          { signal: request.signal }
        );
      }
      if (request.isCurrent()) associations.refresh();
    } catch (reason) {
      if (isAbortError(reason) || !request.isCurrent()) return;
    } finally {
      if (request.finish()) {
        setLaunch(null);
        host.navigate(`/tasks/${encodeURIComponent(taskId)}`);
      }
    }
  };
  const selectedHostRepositoryId = launch && activeWorkspaceId && launch.pullRequest.providerScope ? host.context.resolveRepositoryId({
    workspaceId: activeWorkspaceId,
    providerId: "bitbucket",
    providerScope: launch.pullRequest.providerScope,
    providerRepositoryId: launch.pullRequest.repositoryId
  }) : void 0;
  const launchRemoteRepository = launch ? repositories.find(
    (candidate) => candidate.repositoryId === launch.pullRequest.repositoryId
  ) : void 0;
  const createBitbucketTask = async (payload) => {
    if (!activeWorkspaceId || !launch)
      throw new Error(t("taskLaunchUnavailable"));
    const request = taskLaunchMutation.begin();
    try {
      const result = await host.api.invokeAction(
        action.tasksLaunch,
        {
          workspaceId: activeWorkspaceId,
          body: taskLaunchBody(
            launch.pullRequest,
            payload,
            launch.launchId,
            t
          )
        },
        { signal: request.signal }
      );
      if (!request.isCurrent())
        throw new DOMException("Task launch aborted", "AbortError");
      const task = taskFromLaunchResult(result, t);
      const taskId = text(task.id);
      if (task.bitbucketLinked === true)
        pluginCreatedTaskIDs.current.add(taskId);
      associations.refresh();
      return task;
    } finally {
      request.finish();
    }
  };
  const taskDialog = launch && createContext ? h(ui.TaskCreateDialog, {
    open: true,
    onOpenChange: (open) => {
      if (!open) {
        taskLaunchMutation.cancel();
        taskLinkMutation.cancel();
        setLaunch(null);
      }
    },
    mode: "create",
    workspaceId: activeWorkspaceId ?? null,
    workflowId: createContext.workflowId,
    defaultStepId: createContext.defaultStepId,
    steps: createContext.steps,
    initialValues: taskDialogInitialValues(
      launch.pullRequest,
      launch.preset,
      selectedHostRepositoryId,
      launchRemoteRepository
    ),
    createTask: usePluginTaskCreation(selectedHostRepositoryId) ? createBitbucketTask : void 0,
    onSuccess: (task) => void finishTaskCreation(task)
  }) : null;
  const filter = responsive.isMobile ? h(MobileFilters, {
    host,
    repositories,
    repository,
    setRepository,
    ...scopeProps
  }) : repositoryFilter(host, repositories, repository, setRepository);
  const workbenchBody = !connected ? null : [
    responsive.isMobile ? null : h(StateScopeBar, { host, ...scopeProps }),
    h(ui.IntegrationListToolbar, {
      title: t("pullRequests"),
      count: pullRequests.length,
      loading: queue.loading,
      lastFetchedAt: queue.lastFetchedAt,
      customQuery: searchDraft,
      committedQuery: search,
      onCustomQueryChange: setSearchDraft,
      onCommitCustomQuery: commitSearch,
      onRefresh: queue.refresh,
      filter,
      queryPlaceholder: t("customQueryPlaceholder"),
      titleTestId: "bitbucket-list-toolbar",
      queryTestId: "bitbucket-list-query",
      refreshTestId: "bitbucket-list-refresh"
    }),
    h(
      "section",
      {
        className: "bb-results",
        "data-testid": "bitbucket-results",
        "aria-label": t("pullRequestResults")
      },
      h(DashboardPullRequestList, {
        host,
        pullRequests,
        loading: queue.loading,
        error: queue.error,
        tasksByReview,
        onStartTask: (pullRequest, preset) => setLaunch({
          pullRequest,
          preset,
          launchId: host.utils.generateUUID()
        })
      })
    ),
    h(ui.IntegrationCursorPagination, {
      page: activeQueuePagination.page,
      itemCount: pullRequests.length,
      hasPrevious: activeQueuePagination.page > 1,
      hasNext: Boolean(nextQueueCursor),
      loading: queue.loading,
      onPrevious: () => {
        if (activeQueuePagination.page <= 1) return;
        setQueuePagination({
          ...activeQueuePagination,
          page: activeQueuePagination.page - 1
        });
      },
      onNext: () => {
        if (!nextQueueCursor) return;
        const cursors = activeQueuePagination.cursors.slice(
          0,
          activeQueuePagination.page
        );
        cursors.push(nextQueueCursor);
        setQueuePagination({
          scopeKey: queueScopeKey,
          page: activeQueuePagination.page + 1,
          cursors
        });
      },
      testId: "bitbucket-results-pagination"
    }),
    taskDialog,
    h(ui.IntegrationSaveQueryDialog, {
      open: saveDialogOpen,
      onOpenChange: setSaveDialogOpen,
      description: t("saveQueryDescription"),
      suggestedLabel: searchDraft.trim() || (repository ? t("repositoryPullRequests") : t("savedQuery")),
      query: searchDraft,
      repositoryId: repository,
      repositoryOptions: repositories.map((candidate) => ({
        value: candidate.repositoryId,
        label: `${candidate.ownerOrProject}/${candidate.repositoryName}`
      })),
      onSave: saveCurrentQuery
    })
  ];
  return h(
    "main",
    {
      className: `bb-workbench ${responsive.isMobile ? "bb-mobile" : "bb-desktop"}`,
      "data-testid": "bitbucket-workbench"
    },
    noWorkspace ? EmptyState(host, t("chooseWorkspace"), t("chooseWorkspaceDescription")) : [
      h(ConnectionNotice, {
        host,
        connection,
        workspaceId: activeWorkspaceId
      }),
      workbenchBody
    ]
  );
}

// ui/src/task-review-status.ts
var STATUS_HYDRATION_CONCURRENCY = 4;
async function loadTaskPullRequestDetails(invokeAction, context, signal, t = translateEnglish) {
  const scope = {
    ...context.workspaceId ? { workspaceId: context.workspaceId } : {},
    taskId: context.taskId
  };
  const linkedResponse = await invokeAction(
    "pullrequests.get",
    { ...scope, body: { view: "task" } },
    { signal }
  );
  const linked = normalizePullRequests(linkedResponse, t);
  return mapWithConcurrency(
    linked,
    STATUS_HYDRATION_CONCURRENCY,
    signal,
    async (pullRequest) => {
      const detailResponse = await invokeAction(
        "pullrequests.get",
        {
          ...scope,
          body: {
            review_key: pullRequest.key,
            provider_scope: pullRequest.providerScope ?? pullRequestConnectionScope(pullRequest.url),
            repository_id: pullRequest.repositoryId,
            number: pullRequest.number,
            pull_request_id: pullRequest.id,
            include: ["participants", "status"]
          }
        },
        { signal }
      );
      return normalizeReviewDetail(detailResponse, t) ?? pullRequest;
    }
  );
}
async function mapWithConcurrency(values, concurrency, signal, map) {
  const results = new Array(values.length);
  let nextIndex = 0;
  let failed = false;
  let failure;
  const worker = async () => {
    while (!failed) {
      if (signal.aborted) {
        failed = true;
        failure = signal.reason ?? new DOMException("Operation aborted", "AbortError");
        return;
      }
      const index = nextIndex;
      nextIndex += 1;
      if (index >= values.length) return;
      try {
        results[index] = await map(values[index]);
      } catch (reason) {
        failed = true;
        failure = reason;
      }
    }
  };
  await Promise.all(
    Array.from(
      { length: Math.min(Math.max(1, concurrency), values.length) },
      () => worker()
    )
  );
  if (failed) throw failure;
  return results;
}
function normalizePipelineState(value) {
  const state = value.trim().toUpperCase();
  if (["SUCCESS", "SUCCESSFUL", "PASSED", "COMPLETED"].includes(state))
    return "success";
  if (["FAILED", "FAILURE", "ERROR", "STOPPED"].includes(state))
    return "failure";
  if (["PENDING", "INPROGRESS", "IN_PROGRESS", "RUNNING"].includes(state))
    return "pending";
  return "neutral";
}
function pullRequestState(value) {
  const state = value.trim().toUpperCase();
  if (state === "MERGED") return "merged";
  if (state === "DRAFT") return "draft";
  if (["DECLINED", "CLOSED", "SUPERSEDED"].includes(state)) return "closed";
  return "open";
}
function changeRequestStatusView(pullRequest, refreshedAt = Date.now()) {
  const checks = (pullRequest.statuses ?? []).map((status) => ({
    id: status.key,
    label: status.name,
    state: normalizePipelineState(status.state),
    ...status.target ? { detail: status.target } : {},
    ...status.url ? { url: status.url } : {}
  }));
  const states = checks.map((row) => row.state);
  const reviewers = "participants" in pullRequest && Array.isArray(pullRequest.participants) ? pullRequest.participants.filter(
    (participant) => ["REVIEWER", "APPROVER"].includes(
      participant.role?.toUpperCase() ?? "REVIEWER"
    )
  ) : [];
  const approved = reviewers.filter(
    (participant) => participant.verdict === "approved" || participant.approved
  ).length;
  const changesRequested = reviewers.filter(
    (participant) => participant.verdict === "changes_requested"
  ).length;
  const requested = reviewers.filter(
    (participant) => participant.verdict !== "changes_requested" && participant.verdict !== "approved" && !participant.approved
  ).length;
  const unresolvedComments = pullRequest.unresolvedThreadCount ?? ("threads" in pullRequest && Array.isArray(pullRequest.threads) ? pullRequest.threads.filter((thread) => !thread.resolved).length : 0);
  const providerUpdatedAt = pullRequest.updatedAt ? Date.parse(pullRequest.updatedAt) : Number.NaN;
  const updatedAt = Number.isFinite(providerUpdatedAt) ? providerUpdatedAt : refreshedAt;
  const pipelineState = states.includes("failure") ? "failure" : states.includes("pending") ? "pending" : states.length > 0 && states.every((state) => state === "success") ? "success" : "neutral";
  return {
    number: pullRequest.number,
    state: pullRequestState(pullRequest.state),
    pipelineState,
    checks,
    ...reviewers.length > 0 ? {
      review: {
        state: changesRequested > 0 ? "changes_requested" : approved > 0 ? "approved" : "pending",
        approved,
        ...requested > 0 ? { requested } : {}
      }
    } : {},
    ...unresolvedComments > 0 ? { unresolvedComments } : {},
    updatedAt
  };
}
function reviewSummaryForPullRequest(pullRequest, refreshedAt = Date.now()) {
  return {
    providerId: "bitbucket",
    reviewKey: pullRequest.key,
    title: pullRequest.title,
    url: pullRequest.url,
    repositoryId: pullRequest.repositoryId,
    connectionScope: pullRequest.providerScope ?? pullRequestConnectionScope(pullRequest.url),
    changeRequestNumber: pullRequest.number,
    state: pullRequest.state,
    ...pullRequest.statusLabel ? {
      statusBadge: {
        label: pullRequest.statusLabel,
        ...pullRequest.statusTone ? { tone: pullRequest.statusTone } : {}
      }
    } : {},
    taskStatus: changeRequestStatusView(pullRequest, refreshedAt)
  };
}
function pullRequestConnectionScope(rawURL) {
  try {
    return new URL(rawURL).origin;
  } catch {
    return "bitbucket";
  }
}

// ui/src/review-store.ts
var reviewStore = /* @__PURE__ */ (() => {
  const snapshots = /* @__PURE__ */ new Map();
  const listeners = /* @__PURE__ */ new Map();
  const refreshes = /* @__PURE__ */ new Map();
  let epoch = 0;
  return {
    get(taskId) {
      return snapshots.get(taskId) ?? [];
    },
    set(taskId, pullRequests) {
      snapshots.set(
        taskId,
        pullRequests.map(
          (pullRequest) => reviewSummaryForPullRequest(pullRequest)
        )
      );
      listeners.get(taskId)?.forEach((listener) => listener());
    },
    beginRefresh(taskId) {
      const version = (refreshes.get(taskId) ?? 0) + 1;
      refreshes.set(taskId, version);
      return { epoch, version };
    },
    isCurrentRefresh(taskId, token) {
      return token.epoch === epoch && refreshes.get(taskId) === token.version;
    },
    subscribe(taskId, listener) {
      const taskListeners = listeners.get(taskId) ?? /* @__PURE__ */ new Set();
      taskListeners.add(listener);
      listeners.set(taskId, taskListeners);
      return () => {
        taskListeners.delete(listener);
        if (taskListeners.size === 0) listeners.delete(taskId);
      };
    },
    clear() {
      epoch += 1;
      refreshes.clear();
      snapshots.clear();
      listeners.forEach(
        (taskListeners) => taskListeners.forEach((listener) => listener())
      );
      listeners.clear();
    }
  };
})();
var associationStore = /* @__PURE__ */ (() => {
  const snapshots = /* @__PURE__ */ new Map();
  const listeners = /* @__PURE__ */ new Map();
  const refreshes = /* @__PURE__ */ new Map();
  let epoch = 0;
  return {
    get(workspaceId) {
      return snapshots.get(workspaceId) ?? [];
    },
    set(workspaceId, value, t = translateEnglish) {
      const associations = normalizePullRequestAssociations(value, t);
      snapshots.set(
        workspaceId,
        Object.entries(associations).flatMap(
          ([reviewKey, tasks]) => reviewKey.startsWith("repository:") || reviewKey.startsWith("connection:") ? [] : tasks.flatMap(
            (task) => task.connectionScope && task.repositoryId && task.changeRequestNumber ? [
              {
                providerId: "bitbucket",
                taskId: task.taskId,
                reviewKey,
                connectionScope: task.connectionScope,
                repositoryId: task.repositoryId,
                changeRequestNumber: task.changeRequestNumber
              }
            ] : []
          )
        )
      );
      listeners.get(workspaceId)?.forEach((listener) => listener());
    },
    beginRefresh(workspaceId) {
      const version = (refreshes.get(workspaceId) ?? 0) + 1;
      refreshes.set(workspaceId, version);
      return { epoch, version };
    },
    isCurrentRefresh(workspaceId, token) {
      return token.epoch === epoch && refreshes.get(workspaceId) === token.version;
    },
    subscribe(workspaceId, listener) {
      const workspaceListeners = listeners.get(workspaceId) ?? /* @__PURE__ */ new Set();
      workspaceListeners.add(listener);
      listeners.set(workspaceId, workspaceListeners);
      return () => {
        workspaceListeners.delete(listener);
        if (workspaceListeners.size === 0) listeners.delete(workspaceId);
      };
    },
    clear() {
      epoch += 1;
      refreshes.clear();
      snapshots.clear();
      listeners.forEach(
        (workspaceListeners) => workspaceListeners.forEach((listener) => listener())
      );
      listeners.clear();
    }
  };
})();
async function refreshAssociationStore(host, workspaceId, signal) {
  const refresh = associationStore.beginRefresh(workspaceId);
  const response = await host.api.invokeAction(
    action.pullRequestsAssociations,
    { workspaceId },
    { signal }
  );
  if (!signal.aborted && associationStore.isCurrentRefresh(workspaceId, refresh))
    associationStore.set(workspaceId, response, pluginTranslate(host));
}
async function refreshReviewStore(host, taskId, signal, workspaceId) {
  const refresh = reviewStore.beginRefresh(taskId);
  const pullRequests = await loadTaskPullRequestDetails(
    (key, input, options) => host.api.invokeAction(key, input, options),
    { taskId, ...workspaceId ? { workspaceId } : {} },
    signal,
    pluginTranslate(host)
  );
  if (!signal.aborted && reviewStore.isCurrentRefresh(taskId, refresh))
    reviewStore.set(taskId, pullRequests);
}

// ui/src/native-integrations.ts
function isCredentialFreeHTTPSURL(rawURL) {
  try {
    const parsed = new URL(rawURL);
    return parsed.protocol === "https:" && !parsed.username && !parsed.password;
  } catch {
    return false;
  }
}
function registerNativeIntegrations(registry, host, bitbucketIcon) {
  const t = pluginTranslate(host);
  registry.registerRepositoryProvider({
    id: "bitbucket",
    get label() {
      return t("bitbucket");
    },
    icon: bitbucketIcon,
    async listRepositories({
      workspaceId: scopedWorkspaceId,
      query,
      cursor,
      limit,
      signal
    }) {
      const response = await host.api.invokeAction(
        action.repositoriesList,
        {
          workspaceId: scopedWorkspaceId,
          body: {
            query: query ?? "",
            cursor: cursor ?? "",
            limit: limit ?? 100
          }
        },
        { signal }
      );
      if (signal.aborted) return { repositories: [] };
      return {
        repositories: normalizeRepositories(response),
        nextCursor: text(record2(response).next_cursor) || void 0
      };
    },
    async listBranches({ workspaceId: scopedWorkspaceId, repository, signal }) {
      const response = await host.api.invokeAction(
        action.repositoriesBranches,
        {
          workspaceId: scopedWorkspaceId,
          body: { repository: pluginRepositoryInput(repository) }
        },
        { signal }
      );
      if (signal.aborted) return [];
      const branches = record2(response).branches;
      return Array.isArray(branches) ? branches.flatMap((entry) => {
        const name = text(record2(entry).name);
        return name ? [{ name }] : [];
      }) : [];
    },
    async inspectURL({ workspaceId: scopedWorkspaceId, url, signal }) {
      if (!isCredentialFreeHTTPSURL(url)) return null;
      const response = await host.api.invokeAction(
        action.repositoriesInspect,
        { workspaceId: scopedWorkspaceId, body: { url } },
        { signal }
      );
      return signal.aborted ? null : normalizeRepositoryInspection(response);
    },
    supportsDraft: false,
    async createChangeRequest({
      workspaceId,
      taskId,
      sessionId,
      repositoryId,
      title,
      body,
      baseBranch,
      signal
    }) {
      const response = await host.api.invokeAction(
        action.pullRequestsCreate,
        {
          workspaceId,
          taskId,
          sessionId,
          repositoryId,
          body: {
            title,
            description: body,
            ...baseBranch ? { destination: baseBranch } : {}
          }
        },
        { signal }
      );
      await Promise.all([
        refreshReviewStore(host, taskId, signal, workspaceId),
        refreshAssociationStore(host, workspaceId, signal)
      ]).catch(() => void 0);
      return {
        url: text(response.url),
        provider: "bitbucket",
        ...typeof response.linked === "boolean" ? { linked: response.linked } : {},
        ...text(response.association_error) ? { associationError: text(response.association_error) } : {}
      };
    }
  });
  registry.registerTaskAction({
    id: "link-pull-request",
    get label() {
      return t("linkPullRequestMenu");
    },
    icon: bitbucketIcon,
    placement: "link",
    async run(context) {
      host.openTaskLinkDialog({
        title: t("linkPullRequestTitle"),
        description: t("linkPullRequestDescription"),
        inputLabel: t("pullRequest"),
        placeholder: "workspace/repository#42",
        emptyError: t("linkPullRequestEmpty"),
        failureMessage: t("linkPullRequestFailure"),
        successMessage: t("linkPullRequestSuccess"),
        inputTestId: "bitbucket-review-reference",
        errorTestId: "bitbucket-review-reference-error",
        submitTestId: "bitbucket-review-reference-submit",
        async onSubmit(reference, signal) {
          const body = linkPullRequestBody(reference);
          if (!body) throw new Error(t("linkPullRequestEmpty"));
          await host.api.invokeAction(
            action.pullRequestsLink,
            {
              workspaceId: context.workspaceId,
              taskId: context.taskId,
              body
            },
            { signal }
          );
          await Promise.all([
            refreshReviewStore(
              host,
              context.taskId,
              signal,
              context.workspaceId
            ),
            refreshAssociationStore(host, context.workspaceId, signal)
          ]).catch(() => void 0);
        }
      });
    }
  });
  registry.registerReviewProvider({
    id: "bitbucket",
    get label() {
      return t("bitbucket");
    },
    icon: bitbucketIcon,
    get changeRequestNoun() {
      return t("pullRequestNoun");
    },
    order: 30,
    getSnapshot: (taskId) => reviewStore.get(taskId),
    subscribe: (taskId, listener) => reviewStore.subscribe(taskId, listener),
    async refresh(taskId, signal) {
      await refreshReviewStore(host, taskId, signal);
    },
    getAssociationSnapshot: (workspaceId) => associationStore.get(workspaceId),
    subscribeAssociations: (workspaceId, listener) => associationStore.subscribe(workspaceId, listener),
    async refreshAssociations(workspaceId, signal) {
      await refreshAssociationStore(host, workspaceId, signal);
    },
    async unlink({
      workspaceId,
      taskId,
      reviewKey,
      connectionScope,
      repositoryId,
      changeRequestNumber,
      signal
    }) {
      await host.api.invokeAction(
        action.pullRequestsUnlink,
        {
          workspaceId,
          taskId,
          body: {
            review_key: reviewKey,
            provider_scope: connectionScope,
            repository_id: repositoryId,
            number: changeRequestNumber
          }
        },
        { signal }
      );
    },
    ReviewPanel: (props) => host.jsx(ReviewDetailPanel, {
      host,
      workspaceId: text(props.workspaceId),
      taskId: text(props.taskId),
      reviewKey: text(props.reviewKey),
      connectionScope: text(props.connectionScope),
      repositoryId: text(props.repositoryId),
      changeRequestNumber: Number(props.changeRequestNumber),
      presentation: text(props.presentation) === "mobile" ? "mobile" : "desktop"
    })
  });
}

// ui/src/bitbucket-icon.ts
function createBitbucketIcon(host) {
  return function BitbucketIcon({ className, "aria-hidden": ariaHidden }) {
    return host.jsx(
      "svg",
      {
        xmlns: "http://www.w3.org/2000/svg",
        width: 24,
        height: 24,
        viewBox: "0 0 24 24",
        fill: "none",
        stroke: "currentColor",
        strokeWidth: 2,
        strokeLinecap: "round",
        strokeLinejoin: "round",
        className,
        "aria-hidden": ariaHidden ?? true
      },
      host.jsx("path", {
        d: "M3.648 4a.64 .64 0 0 0 -.64 .744l3.14 14.528c.07 .417 .43 .724 .852 .728h10a.644 .644 0 0 0 .642 -.539l3.35 -14.71a.641 .641 0 0 0 -.64 -.744l-16.704 -.007"
      }),
      host.jsx("path", { d: "M14 15h-4l-1 -6h6l-1 6" })
    );
  };
}

// ui/src/bundle.ts
var PLUGIN_ID = "kandev-plugin-bitbucket";
function makeIntegrationSettings(host) {
  return function IntegrationSettings(props = {}) {
    const activeWorkspaceId = useActiveWorkspaceId(host);
    const scopedWorkspaceId = text(props.workspaceId) || activeWorkspaceId;
    return host.jsx(
      "div",
      { className: "bb-plugin-settings" },
      host.jsx(ConnectionHealth, {
        key: scopedWorkspaceId || "unscoped",
        host,
        workspaceId: scopedWorkspaceId
      }),
      host.jsx(Watches, {
        key: `watches:${scopedWorkspaceId || "unscoped"}`,
        host,
        workspaceId: scopedWorkspaceId,
        filter: {},
        showCreate: false
      })
    );
  };
}
function makeTopbarActions(host) {
  return function TopbarActions() {
    const activeWorkspaceId = useActiveWorkspaceId(host);
    const { t } = usePluginTranslation(host);
    return host.jsx(
      host.ui.Button,
      {
        type: "button",
        variant: "ghost",
        size: "sm",
        className: "bb-topbar-settings",
        "aria-label": t("openSettings"),
        onClick: () => host.navigate(integrationSettingsHref(activeWorkspaceId))
      },
      t("settings")
    );
  };
}
window.registerKandevPlugin(PLUGIN_ID, {
  initialize(registry, host) {
    registerTranslations(registry);
    const t = pluginTranslate(host);
    const bitbucketIcon = createBitbucketIcon(host);
    registry.registerNavItem({
      id: "bitbucket",
      get label() {
        return t("bitbucket");
      },
      path: "/bitbucket",
      icon: bitbucketIcon,
      section: "integrations"
    });
    registry.registerRoute(
      "/bitbucket",
      () => host.jsx(BitbucketPage, { host }),
      {
        topbar: {
          get title() {
            return t("bitbucket");
          },
          get subtitle() {
            return t("pullRequests");
          },
          icon: bitbucketIcon,
          actions: makeTopbarActions(host)
        }
      }
    );
    registry.registerIntegrationSettings({
      id: "bitbucket",
      get label() {
        return t("bitbucket");
      },
      get description() {
        return t("connectDescription");
      },
      icon: bitbucketIcon,
      Component: makeIntegrationSettings(host)
    });
    registerNativeIntegrations(registry, host, bitbucketIcon);
  },
  destroy() {
    reviewStore.clear();
    associationStore.clear();
  }
});

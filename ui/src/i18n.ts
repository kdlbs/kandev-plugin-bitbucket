import type {
  PluginHost,
  PluginRegistry,
  PluginTranslationOptions,
} from "./host-contract";

const english = {
  bitbucket: "Bitbucket",
  settings: "Settings",
  openSettings: "Open Bitbucket settings",
  pullRequests: "Pull requests",
  pullRequest: "Pull request",
  pullRequestNumber: "Pull request {{number}}",
  connectDescription:
    "Connect Bitbucket Cloud or Data Center for this workspace.",
  connection: "Connection",
  notConfigured: "Not configured",
  checkingConnection: "Checking connection",
  connected: "Connected",
  authenticationRequired: "Authentication required",
  unavailable: "Unavailable",
  connectionSaved: "Connection saved and health check started.",
  oauthReady:
    "OAuth authorization is ready. Continue in the connection settings.",
  disconnected: "Bitbucket disconnected. Stored credentials cleared.",
  disconnectBitbucket: "Disconnect Bitbucket",
  disconnectTitle: "Disconnect Bitbucket connection?",
  disconnectDescription:
    "Stored Bitbucket credentials and connection settings for this workspace will be removed.",
  disconnectWorkspaceDescription: "Remove this workspace Bitbucket connection.",
  disconnecting: "Disconnecting…",
  cancel: "Cancel",
  bitbucketProduct: "Bitbucket product",
  bitbucketCloud: "Bitbucket Cloud",
  bitbucketDataCenter: "Bitbucket Data Center",
  bitbucketWorkspace: "Bitbucket workspace",
  workspaceHelp:
    "Workspace slug or ID from bitbucket.org/workspace; required to list repositories.",
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
  atlassianEmailHelp:
    "Used with this API token for Bitbucket Cloud REST. Git uses x-bitbucket-api-token-auth.",
  bitbucketUsername: "Bitbucket username",
  bitbucketUsernameHelp:
    "Used for HTTPS Git with this Data Center PAT or OAuth credential.",
  enterIdentity: "Enter {{identity}}.",
  invalidAtlassianEmail: "Enter a valid Atlassian account email.",
  enterCloudWorkspace: "Enter Bitbucket Cloud workspace slug or ID.",
  oauthClientRegistration: "OAuth client registration",
  oauthRegistrationConfigured:
    "OAuth app registration is configured. Enter both values only to replace it.",
  oauthClientId: "OAuth client ID",
  oauthClientIdReplace: "OAuth client ID (optional to replace)",
  oauthClientSecret: "OAuth client secret",
  oauthClientSecretReplace: "OAuth client secret (optional to replace)",
  oauthSecretHelp:
    "Stored only by Bitbucket secret handling. Existing secrets are never displayed.",
  oauthCallbackUrl: "OAuth callback URL",
  oauthCallbackHelp:
    "Copy this Kandev callback URL into your OAuth app. It is derived from this Kandev origin.",
  connectWithOauth: "Connect with OAuth",
  checkConnection: "Check connection",
  linkPullRequestMenu: "Bitbucket Pull Request",
  linkPullRequestTitle: "Link Bitbucket pull request",
  linkPullRequestDescription:
    "Use a Bitbucket pull request URL or canonical key for this task.",
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
  customQueryPlaceholder:
    'Custom query: press Enter, for example "state:open fix login"',
  pullRequestResults: "Pull request results",
  saveQueryDescription:
    "Save this Bitbucket pull-request search for the current workspace.",
  repositoryPullRequests: "Repository pull requests",
  savedQuery: "Saved query",
  chooseWorkspace: "Choose a workspace",
  chooseWorkspaceDescription:
    "Open Bitbucket from a workspace to connect and browse pull requests.",
  noMatchingPullRequests: "No pull requests match this filter.",
  byAuthor: "by {{author}}",
  openedAgo: "opened {{value}}",
  watches: "Watches",
  watchesDescription:
    "Poll saved pull-request criteria and create only plugin-owned tasks.",
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
  watchResetWarning:
    "Resetting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchResetWarning_one:
    "Resetting this watch will remove {{count}} plugin-owned task. Adopted and manual tasks stay untouched.",
  watchResetWarning_other:
    "Resetting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchDeleteWarning:
    "Deleting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  watchDeleteWarning_one:
    "Deleting this watch will remove {{count}} plugin-owned task. Adopted and manual tasks stay untouched.",
  watchDeleteWarning_other:
    "Deleting this watch will remove {{count}} plugin-owned tasks. Adopted and manual tasks stay untouched.",
  reviewPreset: "Review",
  reviewPresetHint: "Read the diff, flag issues",
  reviewPresetPrompt:
    "Review Bitbucket pull request {{reference}}. Inspect the changes, run relevant tests, and report concrete findings.",
  addressFeedbackPreset: "Address feedback",
  addressFeedbackPresetHint: "Apply review comments",
  addressFeedbackPresetPrompt:
    "Address the review feedback on Bitbucket pull request {{reference}}. Make the requested changes, verify them, and summarize what changed.",
  fixCiPreset: "Fix CI",
  fixCiPresetHint: "Diagnose failing checks",
  fixCiPresetPrompt:
    "Fix the failing CI checks on Bitbucket pull request {{reference}}. Reproduce the failures, implement the smallest correct fix, and run the relevant checks.",
  removeApproval: "Remove approval",
  removingApproval: "Removing approval…",
  approve: "Approve",
  approving: "Approving…",
  merge: "Merge",
  merging: "Merging…",
  decline: "Decline",
  declining: "Declining…",
  comment: "Comment",
  reply: "Reply",
  unknown: "Unknown",
  bitbucketTask: "Bitbucket task",
  taskLaunchUnavailable: "Bitbucket task launch is unavailable.",
  pullRequestIdentityUnavailable:
    "Bitbucket pull request identity is unavailable. Refresh and try again.",
  taskLaunchMissingId: "Bitbucket task launch returned no task id.",
  enterOauthClientId: "Enter OAuth client ID.",
  enterOauthClientSecret: "Enter OAuth client secret.",
  oauthCallbackUnavailable: "OAuth callback URL is unavailable.",
  requestFailed: "Bitbucket request failed. Try again.",
} as const;

export type TranslationKey = keyof typeof english;
export type Translate = (
  key: TranslationKey,
  options?: PluginTranslationOptions,
) => string;

const pseudoGlyphs: Record<string, string> = {
  a: "à",
  b: "ƀ",
  c: "ç",
  d: "ď",
  e: "é",
  f: "ƒ",
  g: "ğ",
  h: "ĥ",
  i: "í",
  j: "ĵ",
  k: "ķ",
  l: "ļ",
  m: "ḿ",
  n: "ñ",
  o: "ö",
  p: "þ",
  q: "q",
  r: "ř",
  s: "š",
  t: "ţ",
  u: "ü",
  v: "ṽ",
  w: "ŵ",
  x: "ẋ",
  y: "ý",
  z: "ž",
};

function pseudoMessage(message: string): string {
  const transformed = message
    .split(/(\{\{[^}]+\}\})/g)
    .map((part) =>
      part.startsWith("{{")
        ? part
        : part.replace(/[A-Za-z]/g, (letter) => {
            const glyph = pseudoGlyphs[letter.toLowerCase()] ?? letter;
            return letter === letter.toUpperCase()
              ? glyph.toUpperCase()
              : glyph;
          }),
    )
    .join("");
  return `[${transformed}${"~".repeat(Math.ceil(message.length * 0.3))}]`;
}

const pseudo = Object.fromEntries(
  Object.entries(english).map(([key, message]) => [
    key,
    pseudoMessage(message),
  ]),
);

export const translationCatalogs = { en: english, pseudo };

function interpolate(
  message: string,
  options?: PluginTranslationOptions,
): string {
  const values: Readonly<Record<string, string | number | undefined>> = {
    count: options?.count,
    ...options?.values,
  };
  return message.replace(/\{\{([^}]+)\}\}/g, (_match, key: string) =>
    values[key] === undefined ? `{{${key}}}` : String(values[key]),
  );
}

export const translateEnglish: Translate = (key, options) => {
  const pluralKey =
    `${key}_${options?.count === 1 ? "one" : "other"}` as TranslationKey;
  const message =
    options?.count !== undefined && pluralKey in english
      ? english[pluralKey]
      : english[key];
  return interpolate(message, options);
};

export function usePluginTranslation(host: PluginHost) {
  const translation = host.i18n.useTranslation();
  return {
    locale: translation.locale,
    t: translation.t as Translate,
  };
}

export function pluginTranslate(host: PluginHost): Translate {
  return (key, options) => host.i18n.t(key, options);
}

export function registerTranslations(registry: PluginRegistry): void {
  registry.registerTranslations(translationCatalogs);
}

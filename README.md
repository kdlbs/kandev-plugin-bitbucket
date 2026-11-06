# kandev-plugin-bitbucket

The Bitbucket runtime plugin for Kandev. It connects one Kandev workspace to
Bitbucket Cloud or Bitbucket Data Center and adds native repository discovery,
pull-request/task links, review surfaces, task actions, and pull-request
watches.

This repository produces a Kandev plugin package; it is not a standalone web
service or an integration built into the Kandev host.

## Capabilities

- Bitbucket Cloud: API-token, repository/project/workspace access-token, or OAuth 2.0 connections.
- Bitbucket Data Center: personal, project, repository, or OAuth credentials.
- Native Kandev repository provider, task-menu actions, and review panel.
- Native Create PR transport after Kandev pushes the task branch, with exact
  persisted-repository selection for multi-repository tasks.
- Shared task-list PR indicators, CI/review status, and unlink controls on
  desktop and mobile.
- Workspace-persisted dashboard queries using Kandev's shared saved-query UI.
- Authenticated manifest-declared actions for connection management,
  repositories, pull requests, reviews, task links, and watches.
- Workspace-scoped connection state and encrypted plugin-owned secrets.
- Transient, provider-scoped Git credential resolution with a non-secret
  binding revision so rotated or disconnected credentials fail closed.
- A dynamic `#` pull-request reference source, reauthorized against the live
  provider before Kandev accepts it in a prompt.

The plugin requests `api_read` for Kandev task/workspace/workflow/agent-profile
and repository data, and `api_write: ["tasks"]` for its task workflow. Review
the manifest before installing: plugin code is privileged inside a Kandev
installation and capabilities are API permission gates, not a sandbox.

## Requirements

- Kandev 0.99.0 or newer: the SDK-wide minimum host in the published
  `sdk-release.json` records for both SDKs. The plugin retains manifest API v1.
- Go 1.26.0 and Node 24 (lockfile regenerated with npm 11.17.0).
- Public SDKs pinned to `github.com/kdlbs/kandev/pluginsdk v0.1.0` and
  `@kdlbs/plugin-sdk` exactly `0.1.0`. These share an independent SDK version;
  the plugin remains version 0.4.0.

SDK compilation, typechecking, and tests work from a standalone checkout.
Packaging additionally needs backend-owned `cmd/plugin-pack` from an explicit
Kandev tooling checkout pinned by `.kandev-tooling-ref` to
`f0e2e774cfe2df8a0d15e4c21dea7b48580e38fb`. This is a packaging input,
separate from the public SDK dependencies and the runtime host floor.

## Build and verify

From a fresh plugin checkout, without sibling sources:

```sh
npm ci --ignore-scripts
go mod tidy
git diff --exit-code -- go.mod go.sum
make sdk-check
make check-format
make vet
make test
make build
```

`make sdk-check` runs the installed-dependency compatibility checker:

```sh
go run github.com/kdlbs/kandev/pluginsdk/cmd/kandev-sdk-check@v0.1.0 \
  --manifest manifest.yaml --frontend-sdk node_modules/@kdlbs/plugin-sdk
```

Upgrade SDKs deliberately: install exact Go/npm versions, regenerate and commit
both locks, inspect their published release records, set the manifest minimum
to at least the highest SDK floor, rerun the checker and tests, then smoke-test
the declared minimum host. See the [public authoring guide](https://github.com/kdlbs/kandev/blob/main/docs/public/plugins-authoring.md#install-and-migrate-sdk-dependencies).

To package, prepare the tooling source outside the plugin repository:

```sh
git clone https://github.com/kdlbs/kandev.git ../kandev-tooling
git -C ../kandev-tooling checkout "$(cat .kandev-tooling-ref)"
make package-host verify-package-host
make package verify-package
# For another checkout location:
make package-host verify-package-host KANDEV_TOOLING=/absolute/path/to/kandev/apps/backend
```

A sparse tooling checkout must include both `apps/backend` and root
`pluginsdk/`, which the backend module replaces locally. It is never used to
resolve this plugin's SDK imports. Compilation independence does not imply
checkout-free packaging.

`make package-host` creates a package for the current host platform; `make
package` cross-compiles all five executables declared in `manifest.yaml`. Both
produce `kandev-plugin-bitbucket-<version>.tar.gz` and generate its internal
`checksums.txt`. The verification targets check the package file inventory and
every checksum. Do not author that file by hand. `make test` also runs negative
checks for missing or unexpected package files and inconsistent release
versions.

For the packaged-host contract, use a separate disposable checkout of Kandev
`v0.99.0` (the manifest minimum). Install its dependencies with pnpm 9.15.9,
then follow CI's `build-backend`, `build-web-e2e` (with `VERSION=0.99.0`), and
`pnpm e2e:raw --project=chromium tests/plugins/bitbucket-packaged-plugin.spec.ts`
commands. The desktop/mobile contract uses fake provider data and needs no
Bitbucket credentials.

`make e2e` requires `KANDEV_PLUGIN_E2E_URL` to name a fresh disposable
compatible host that accepts test package uploads. The runner installs and
activates the freshly built host package, then checks native desktop and mobile
surfaces. CI runs this additional external-host contract when its secret is
configured and emits a notice otherwise. The minimum-host packaged contract
remains required before release.

### Configured live acceptance

`make e2e-live` installs the package into the same kind of disposable Kandev
host, saves one real Bitbucket connection through the authenticated plugin
action, and exercises health, repository discovery, branches, pull-request
inspection, and full review loading. It is opt-in and fail-fast; normal tests
never contact Bitbucket. Run it once for Cloud and once for Data Center.

Provide these non-secret coordinates and secret values through the task or CI
environment—never commit them and never paste tokens into logs:

```text
KANDEV_PLUGIN_E2E_URL                         disposable compatible Kandev host
KANDEV_BITBUCKET_LIVE_PRODUCT                cloud | data_center
KANDEV_BITBUCKET_LIVE_AUTH_METHOD            api_token | access_token | user_pat | project_token | repository_token
KANDEV_BITBUCKET_LIVE_TOKEN                  secret API token/PAT/scoped token
KANDEV_BITBUCKET_LIVE_AUTH_IDENTITY          Cloud email for api_token; DC username for user_pat
KANDEV_BITBUCKET_LIVE_CLOUD_WORKSPACE        Cloud workspace slug/ID
KANDEV_BITBUCKET_LIVE_BASE_URL               DC HTTPS base URL including context path
KANDEV_BITBUCKET_LIVE_REPOSITORY_NAMESPACE   Cloud workspace or DC project key
KANDEV_BITBUCKET_LIVE_REPOSITORY_SLUG        disposable repository slug
KANDEV_BITBUCKET_LIVE_PR_NUMBER              existing PR with review data
KANDEV_BITBUCKET_LIVE_KANDEV_WORKSPACE_ID    required only if host has != 1 workspace
```

Optional live review mutations require explicit disposable-target permission:

```text
KANDEV_BITBUCKET_LIVE_REVIEW_WRITES=1
KANDEV_BITBUCKET_LIVE_APPROVAL_PR_NUMBER     open PR authored by another identity
KANDEV_BITBUCKET_LIVE_DECLINE_PR_NUMBER      open disposable PR that may be declined
```

That mode adds a marked test comment to `KANDEV_BITBUCKET_LIVE_PR_NUMBER`,
approves then unapproves the approval PR, and permanently declines the decline
PR. Tracing, screenshots, and video are disabled for the live runner so secret
request bodies cannot enter Playwright artifacts. The runner disconnects the
workspace in a `finally` cleanup. It does not replace the separate Docker/SSH
credential-broker gate, which additionally needs a public HTTPS broker URL
reachable from those executors.

## Install and configure

Use **Settings > Plugins** in a compatible Kandev host to upload the generated
tarball, then enable **Bitbucket**. Open the Bitbucket navigation entry and
configure it for the active Kandev workspace:

- For Cloud, provide the Bitbucket workspace slug or ID. Select API token,
  access token, or OAuth 2.0.
- For an Atlassian account API token, provide the account email.
  This mode uses Basic authentication for REST. The token needs `read:user:bitbucket`,
  `read:repository:bitbucket`, `write:repository:bitbucket`,
  `read:pullrequest:bitbucket`, and `write:pullrequest:bitbucket` for the full
  workflow.
- For a repository, project, or workspace access token, select
  **Repository, project or workspace access token** (`access_token`).
  This mode uses Bearer authentication for REST and `x-token-auth` for Git.
  The token needs `repository`, `repository:write`, `pullrequest`, and
  `pullrequest:write` for the full workflow.
  The plugin does not require an account email or an OAuth registration.
- OAuth consumers need Account read plus Repository and Pull request
  read/write permissions.
- For Data Center, provide the full HTTPS base URL, then use the narrowest
  supported personal, project, repository, or OAuth credential.
- Data Center OAuth requires `REPO_READ` and `REPO_WRITE`; PAT and scoped-token
  identities need equivalent repository read/write authority for the full
  workflow.
- For OAuth, copy the callback URL shown by the plugin into the Bitbucket OAuth
  consumer before starting the browser flow. Callback URLs must use HTTPS,
  except that local development callbacks may use HTTP on `localhost` or a
  loopback IP address.

Connection settings are workspace-specific. Tokens, OAuth refresh tokens, and
client secrets are held as plugin secrets; they are never shown after save.
Disconnecting removes the workspace connection and its stored plugin
credentials. The public Kandev guide covers the detailed setup, security model,
and task Git limitations in [Integrations](https://kandev.dev/docs/integrations).

## Package and release policy

Pull-request CI installs the exact public Go/frontend SDKs and checks their
compatibility before building. It validates formatting, Go tests/vet, UI
typecheck/build/tests, exact archive contents, and generated checksums. A required, credential-free job checks out
the released minimum host, installs the real package, and exercises its
desktop/mobile lifecycle and shared review/status surfaces.
`KANDEV_PLUGIN_E2E_URL` adds an optional external-host smoke test; it is not the
compatibility gate. The release workflow checks that the tag, manifest,
Makefile, and packaged manifest versions match. Manual releases run the tests
and packaged-host contract before committing release metadata or pushing a tag;
pushed tags run them before GitHub publishes the archive and checksums.

The initial release intentionally follows Kandev's current unsigned marketplace
contract. Its generated internal `checksums.txt` remains mandatory, but the
package does not claim a verified signature or publisher provenance and Kandev
will report it as unsigned. Plugin signing is deferred until Kandev has a
host-wide verifier, trust policy, and key lifecycle; it is not a Bitbucket-only
release gate.

Version v0.2.0 is the first public plugin release and requires Kandev v0.88.0
or newer. Source builds now inherit the SDK-wide Kandev 0.99.0 minimum; the
release workflow validates against the current manifest minimum before publishing. The plugin is listed in Kandev's official marketplace,
which
resolves the latest GitHub Release containing the required
`kandev-plugin-bitbucket-<version>.tar.gz` asset.

## Troubleshooting

- If packaging cannot find `cmd/plugin-pack`, prepare the pinned tooling
  checkout or pass `KANDEV_TOOLING` pointing to its `apps/backend` directory.
- If `make sdk-check` fails, inspect the installed SDK release records and
  manifest minimum/API version; remove SDK replacements or file/link aliases.
- If `npm ci` fails, use Node 24 and keep `package-lock.json`; do not replace
  the lockfile with a floating install.
- If package verification fails, rebuild with `make package` or
  `make package-host`. The verifier rejects missing files, extra files,
  incomplete checksums, and damaged contents.
- `make e2e` needs a fresh disposable Kandev host in
  `KANDEV_PLUGIN_E2E_URL`. It does not use Bitbucket credentials. Live Bitbucket
  acceptance is a separate opt-in workflow described above.

## License

MIT. See [LICENSE](LICENSE).

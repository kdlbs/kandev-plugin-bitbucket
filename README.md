# kandev-plugin-bitbucket

The Bitbucket runtime plugin for Kandev. It connects one Kandev workspace to
Bitbucket Cloud or Bitbucket Data Center and adds native repository discovery,
pull-request/task links, review surfaces, task actions, and pull-request
watches.

This repository produces a Kandev plugin package; it is not a standalone web
service or an integration built into the Kandev host.

## Capabilities

- Bitbucket Cloud: API-token or OAuth 2.0 connections.
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

- Kandev 0.88.0 or newer. This is the first release line that contains the
  declared-action, provider-registration, reference-authorization,
  credential-broker, and live `api_write` contracts required by this plugin.
- Go version from `go.mod` and Node 24 for the UI toolchain.
- A sibling Kandev checkout while developing, because the Go and frontend SDKs
  are resolved from that exact host source:

  ```text
  parent-directory/
  ├── kandev/                         # apps/backend Go module
  └── kandev-plugin-bitbucket/        # this repository
  ```

## Build and verify

```sh
npm ci
make fmt
make vet
make test
make build
make package-host
make verify-package-host
make package
make verify-package
```

`make package-host` creates a package for the current host platform; `make
package` cross-compiles every executable declared in `manifest.yaml`. Both
produce `kandev-plugin-bitbucket-<version>.tar.gz` and generate its internal
`checksums.txt`. Do not author that file by hand.

`make e2e` is deliberately fail-fast: it requires `KANDEV_PLUGIN_E2E_URL` to
name a fresh, disposable compatible Kandev host that accepts test package
uploads. The runner installs and activates the freshly built host package,
then checks the native desktop and mobile plugin surfaces. Pull-request CI runs
this external-host contract when that repository secret is configured and
emits an explicit notice otherwise; package, checksum, type, unit, and build
gates always run. Tag releases remain fail-fast and cannot publish without the
packaged-host contract.

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
KANDEV_BITBUCKET_LIVE_AUTH_METHOD            api_token | user_pat | project_token | repository_token
KANDEV_BITBUCKET_LIVE_TOKEN                  secret API token/PAT/scoped token
KANDEV_BITBUCKET_LIVE_AUTH_IDENTITY          Cloud account email; DC username for user_pat
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

- For Cloud, provide the Bitbucket workspace slug or ID plus either an
  Atlassian-account email/API token or an OAuth consumer registration.
- A scoped Cloud API token needs `read:user:bitbucket`,
  `read:repository:bitbucket`, `write:repository:bitbucket`,
  `read:pullrequest:bitbucket`, and `write:pullrequest:bitbucket` for the full
  workflow. OAuth consumers need Account read plus Repository and Pull request
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

Pull-request CI validates formatting, Go tests/vet, UI typecheck/build/tests,
archive contents, and generated checksums. A required, credential-free job
checks out the exact reviewed Kandev host, builds both heads, installs the real
package, and exercises its desktop/mobile lifecycle and shared review/status
surfaces. `KANDEV_PLUGIN_E2E_URL` adds an optional external-host smoke test; it
is not the compatibility gate. The tag release workflow always runs the
packaged-plugin contract before it uploads
`<id>-<version>.tar.gz` on a matching `v<version>` tag.

The initial release intentionally follows Kandev's current unsigned marketplace
contract. Its generated internal `checksums.txt` remains mandatory, but the
package does not claim a verified signature or publisher provenance and Kandev
will report it as unsigned. Plugin signing is deferred until Kandev has a
host-wide verifier, trust policy, and key lifecycle; it is not a Bitbucket-only
release gate.

Version v0.2.0 is the first public plugin release and requires Kandev v0.88.0
or newer. The release workflow validates against that exact minimum host tag
before publishing. The plugin is listed in Kandev's official marketplace, which
resolves the latest GitHub Release containing the required
`kandev-plugin-bitbucket-<version>.tar.gz` asset.

## License

MIT. See [LICENSE](LICENSE).

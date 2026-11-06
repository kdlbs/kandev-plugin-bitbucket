# Public SDK 0.1.0 migration validation

Dates: 2026-11-05–06. Plugin base: `cee20a6` (current `main` when refreshed).
Platform: Linux amd64; Go 1.26.6; Node 24.19.0; npm 11.17.0; pnpm 9.15.9.

## Release and scope

- [PR #4534](https://github.com/kdlbs/kandev/pull/4534) was merged and the
  [release recovery run, attempt 3](https://github.com/kdlbs/kandev/actions/runs/42089596215/attempts/3)
  verified public consumers before this migration was authorized.
- [SDK release](https://github.com/kdlbs/kandev/releases/tag/pluginsdk/v0.1.0):
  source `f0e2e774cfe2df8a0d15e4c21dea7b48580e38fb`. The public Go and npm
  `sdk-release.json` records both specify SDK `0.1.0`, host release/floor
  `0.99.0`, and API versions `[1, 2]`.
- Plugin version remains `0.4.0`; manifest API remains `1`. The only manifest
  change is its SDK-wide minimum host. No releases, catalog edits, or author
  notifications were performed.
- There was no repository `AGENTS.md` at the base revision. Added guidance uses
  the parent's current create-kandev-plugin workflow and public authoring guide.
  No generated protocol imports or recipe subprojects exist here.

## Standalone resolution

Created a separate clone with no `../kandev` or `../kandev-tooling`. Public
resolution used empty dedicated caches and no Go workspace:

```sh
git clone --no-local "$(git rev-parse --show-toplevel)" /tmp/bitbucket-sdk-migration/standalone/plugin
cd /tmp/bitbucket-sdk-migration/standalone/plugin
export GOWORK=off
export GOPROXY=https://proxy.golang.org
export GOSUMDB=sum.golang.org
export GOMODCACHE=/tmp/bitbucket-sdk-migration/public-modcache
export npm_config_cache=/tmp/bitbucket-sdk-migration/public-npm-cache
npm ci --ignore-scripts
go mod download
go mod verify
go mod tidy
git diff --exit-code -- go.mod go.sum package-lock.json
go list -m all > /tmp/bitbucket-sdk-migration/go-graph.txt
! rg '^github.com/kandev/kandev ' /tmp/bitbucket-sdk-migration/go-graph.txt
npm ls @kdlbs/plugin-sdk
make sdk-check check-format vet test build
```

Dependency installation, module verification, tidy/lock comparison and SDK
compatibility passed. npm selected registry `@kdlbs/plugin-sdk@0.1.0`; Go
selected `github.com/kdlbs/kandev/pluginsdk v0.1.0` without a replacement or
backend module in the graph. Both compatibility records report:
`SDK 0.1.0: compatible (minimum Kandev 0.99.0)`.

## Repository checks

The working checkout passed `make check-format vet test build`, including
`go vet ./...`, `go test ./...` (all 9 packages), `npm test` (typecheck, esbuild,
59 Vitest tests across 8 files), and both package/release verifier negative
suites. The migration contract test was first run against the old configuration
and failed on the old floor/source checkout, then passed after migration.

The standalone clone was fast-forwarded to the verifier fixes and both verifier
suites passed again. New regression cases failed first against the old
verifiers, then passed with canonical full/host manifests. They also reject a
host-only archive offered as a full package and mismatched unquoted versions.
The changes remove the old awk escaped-quote warning.

The following workflow syntax check passed:

```sh
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7 -shellcheck= -pyflakes= \
  .github/workflows/build.yml .github/workflows/ci.yml .github/workflows/release.yml
```

npm reports 8 advisories (1 moderate, 5 high, 2 critical) in the existing
toolchain dependency graph; no unrelated npm upgrades were included.

## Packaging and minimum host

The packaging input is a separately cloned Kandev `v0.99.0` source at
`f0e2e774cfe2df8a0d15e4c21dea7b48580e38fb`, equal to `.kandev-tooling-ref`.
From the standalone plugin checkout:

```sh
make verify-package-host KANDEV_TOOLING=/tmp/bitbucket-sdk-migration/kandev-min/apps/backend
cp kandev-plugin-bitbucket-0.4.0.tar.gz /tmp/bitbucket-sdk-migration/bitbucket-host.tar.gz
CGO_ENABLED=0 make verify-package KANDEV_TOOLING=/tmp/bitbucket-sdk-migration/kandev-min/apps/backend
sh scripts/verify-release-version.sh v0.4.0 kandev-plugin-bitbucket-0.4.0.tar.gz
tar -tzf kandev-plugin-bitbucket-0.4.0.tar.gz
sha256sum kandev-plugin-bitbucket-0.4.0.tar.gz /tmp/bitbucket-sdk-migration/bitbucket-host.tar.gz
```

Host-only and full construction passed, including exact archive inventory,
every generated checksum and release-version matching. Full package contents:
manifest, icon/NOTICE, all five declared platform executables, UI bundle/style,
and checksums. SHA-256:

- Full: `ec5cf1918cba1384656fbbdd839527f54bebdf707fd07641a6b90d49ef643300`
- Host: `10e5155902435f004e5e214e0be6dd4444042140600884a5e30063396980d039`

Minimum-host smoke passed locally on 2026-11-06: **3 passed (11.6s)**.
The suite covers real package installation/activation, unconfigured desktop
RPC/lifecycle behavior, touch-sized mobile workbench, and sidebar PR summary.
Provider data is mocked by the maintained suite; real Cloud/Data Center accounts
and optional provider writes were not exercised.

The disposable host and fixture preparation commands were:

```sh
git clone --depth 1 --branch v0.99.0 https://github.com/kdlbs/kandev.git /tmp/bitbucket-sdk-migration/kandev-min
pnpm install --frozen-lockfile --dir /tmp/bitbucket-sdk-migration/kandev-min/apps
make -C /tmp/bitbucket-sdk-migration/kandev-min build-backend build-web-e2e
make -C /tmp/bitbucket-sdk-migration/kandev-min/apps/backend e2e-plugin-package build-e2e-delivery-fixture build-e2e-send-now-fixture
make -C /tmp/bitbucket-sdk-migration/kandev-min/apps/backend build-kandev VERSION=0.99.0
# After interrupted long builds, completed the outstanding targets:
make -C /tmp/bitbucket-sdk-migration/kandev-min build-web-e2e
make -C /tmp/bitbucket-sdk-migration/kandev-min/apps/backend build-e2e-send-now-fixture
/tmp/bitbucket-sdk-migration/kandev-min/apps/backend/bin/kandev --version --json
```

Binary identity was `0.99.0`, commit
`f0e2e774cfe2df8a0d15e4c21dea7b48580e38fb`. Explicit `VERSION` avoids
`git describe` selecting `pluginsdk/v0.1.0` on the shared release commit.
CI stamps the version during its initial minimum-host build.

Ran this command from the disposable host's `apps/web`:

```sh
pnpm e2e:raw --project=chromium tests/plugins/bitbucket-packaged-plugin.spec.ts
```

The command ran in a sanitized Python child environment: removed every inherited
`KANDEV_*` variable plus `GH_TOKEN`/`GITHUB_TOKEN`; set task-owned
`HOME=/tmp/bitbucket-sdk-migration/runtime/home`, `XDG_CONFIG_HOME=.../config`,
`XDG_DATA_HOME=.../data`, `XDG_CACHE_HOME=.../cache`, and `TMPDIR=.../tmp`.
The maintained fixture creates its own isolated database/data/home below that
temporary root. Before isolating `HOME`, select/install the browser cache from `apps/web`:

```sh
BROWSER_CACHE="${PLAYWRIGHT_BROWSERS_PATH:-$HOME/.cache/ms-playwright}"
PLAYWRIGHT_BROWSERS_PATH="$BROWSER_CACHE" pnpm exec playwright install chromium
```

The recorded run used that invoking user's existing default cache. Preserve
`BROWSER_CACHE` when constructing the sanitized child. Explicit inputs:

```text
KANDEV_SERVER_HOST=127.0.0.1
KANDEV_BITBUCKET_PLUGIN_PACKAGE=/tmp/bitbucket-sdk-migration/bitbucket-host.tar.gz
E2E_PORT_OFFSET=23
PLAYWRIGHT_BROWSERS_PATH=$BROWSER_CACHE
```

Loaded the installed Nix browser libraries via `LD_LIBRARY_PATH` (the existing
`kandev-builtin-playwright-libs`, `libxcb`, and `systemd` library directories).
The initial browser launch without that path failed for missing libglib;
Chromium 149.0.7827.55 launched after supplying it. No shared configuration was
changed.

A temporary, additional `test.beforeEach` in the disposable clone queried
`apiClient.rawRequest("GET", "/health")`, logged JSON, and asserted
`health.version === "0.99.0"` before each of the three unchanged tests. All three
responses were:

```json
{"mode":"websocket+http","service":"kandev","status":"ok","version":"0.99.0"}
```

Restored the disposable spec afterward. The plugin and shared Kandev checkouts
were not modified for this assertion.

## Pull request checks

[PR #19](https://github.com/kdlbs/kandev-plugin-bitbucket/pull/19) targets the
actual default branch, `main`. Initial implementation head
`b5b0eb06086262aeb286db91e97aeccbf0810c53` passed all three GitHub checks:

- [Build plugin packages](https://github.com/kdlbs/kandev-plugin-bitbucket/actions/runs/42104259194): success.
- [CI](https://github.com/kdlbs/kandev-plugin-bitbucket/actions/runs/42104259304):
  tidy/format/vet/tests success; packaged minimum-host desktop/mobile job success,
  **3 passed (18.3s)** on `v0.99.0` source with explicit `VERSION=0.99.0`.

Final-head CI/review readiness is recorded on the PR after this evidence commit.


The first package verifier run exposed newer `plugin-pack` behavior: it
canonicalizes YAML and narrows a host manifest to one platform. Updated
verifiers preserve inventory/checksum safeguards while accepting that output.
The first full cross-build failed because this executor's persisted Go config
forces `CGO_ENABLED=1` even for ARM; the retry explicitly uses `CGO_ENABLED=0`.
No shared Go configuration was changed.


## Automated review follow-up

Addressed Cubic's six findings: guard the required Test step and every tooling
checkout's pinned ref; reject malformed executable entries even after blank or
comment lines; quote packaging paths; make checkout/browser-cache examples
portable. The malformed-entry regression failed against the old verifier before
its fix. All verifier suites and `make sdk-check check-format vet test build`
passed afterward (59 UI tests and all 9 Go packages).

Both actual archive forms passed with whitespace in the tooling override:

```sh
ln -s /tmp/bitbucket-sdk-migration/kandev-min '/tmp/bitbucket-sdk-migration/tooling source'
make verify-package-host KANDEV_TOOLING='/tmp/bitbucket-sdk-migration/tooling source/apps/backend'
CGO_ENABLED=0 make verify-package KANDEV_TOOLING='/tmp/bitbucket-sdk-migration/tooling source/apps/backend'
```

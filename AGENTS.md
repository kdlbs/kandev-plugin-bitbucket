# Bitbucket plugin

This repository owns only the Bitbucket runtime plugin. Follow Kandev's
[create-kandev-plugin workflow](https://github.com/kdlbs/kandev/blob/main/.agents/skills/create-kandev-plugin/SKILL.md)
and [public authoring guide](https://github.com/kdlbs/kandev/blob/main/docs/public/plugins-authoring.md).

- Keep Go `github.com/kdlbs/kandev/pluginsdk` and npm `@kdlbs/plugin-sdk`
  exactly pinned to a verified public SDK release. SDK versions are independent
  of plugin and host versions. Do not add local SDK replacements or file links.
- Read both installed `sdk-release.json` records when upgrading. Keep
  `min_kandev_version` at least the highest SDK-wide floor and retain a supported
  manifest API version. Preserve declared surfaces and installed-plugin behavior.
- After `npm ci`, run `make sdk-check check-format vet test build`.
- Packaging uses backend-owned `cmd/plugin-pack`, not an SDK command.
  `.kandev-tooling-ref` pins that separate input. Use `KANDEV_TOOLING` for a
  nondefault backend path. Sparse tooling checkouts need `apps/backend` and
  root `pluginsdk/`; they do not supply this plugin's dependencies.
- Run host/full package verification and minimum-host desktop/mobile smoke
  before release. Record exact commands, results, and unavailable checks.
- Do not publish releases, change the catalog, or contact authors unless asked.

import { readFile } from "node:fs/promises";
import { describe, expect, it } from "vitest";

describe("Bitbucket manifest", () => {
  it("declares authenticated UI actions and native provider registrations", async () => {
    const manifest = await readFile(new URL("../../manifest.yaml", import.meta.url), "utf8");

    expect(manifest).toContain('key: "connection.get"');
    expect(manifest).toMatch(/key: "connection\.disconnect", scope: "workspace"/);
    expect(manifest).not.toContain("resource_scope:");
    expect(manifest).toContain('key: "pullrequests.queue"');
    expect(manifest).toContain('key: "pullrequests.link"');
    expect(manifest).toContain('api_write: ["tasks"]');
    expect(manifest.match(/^  api_write: \["tasks"\]$/gm)).toHaveLength(1);
    expect(manifest).not.toContain('key: "pullrequests.launch"');
    expect(manifest).not.toContain('key: "pullrequests.update"');
    expect(manifest).toContain('repository_providers: ["bitbucket"]');
    expect(manifest).toContain('source: "bitbucket"');
    expect(manifest).toContain('min_kandev_version: "0.88.0"');
  });

  it("materializes both Kandev SDKs in pull request packaging workflows", async () => {
    const workflows = await Promise.all(
      ["build.yml", "ci.yml"].map((name) =>
        readFile(new URL(`../../.github/workflows/${name}`, import.meta.url), "utf8"),
      ),
    );

    for (const workflow of workflows) {
      expect(workflow).toContain("apps/backend");
      expect(workflow).toContain("apps/packages/plugin-sdk");
    }
  });

  it("checks out the complete minimum host for release package validation", async () => {
    const workflow = await readFile(
      new URL("../../.github/workflows/release.yml", import.meta.url),
      "utf8",
    );

    expect(workflow).toContain("Checkout minimum supported Kandev host");
    expect(workflow).toContain("path: kandev");
    expect(workflow).not.toContain("sparse-checkout:");
  });

  it("tests pull requests against the declared minimum Kandev release", async () => {
    const manifest = await readFile(new URL("../../manifest.yaml", import.meta.url), "utf8");
    const minimumVersion = manifest.match(/^min_kandev_version: "([^"]+)"$/m)?.[1];
    const workflows = await Promise.all(
      ["build.yml", "ci.yml"].map((name) =>
        readFile(new URL(`../../.github/workflows/${name}`, import.meta.url), "utf8"),
      ),
    );

    expect(minimumVersion).toBe("0.88.0");
    for (const workflow of workflows) {
      expect(workflow).toContain(`ref: v${minimumVersion}`);
    }
  });

  it("gates releases on the packaged desktop and mobile host contract", async () => {
    const workflow = await readFile(
      new URL("../../.github/workflows/release.yml", import.meta.url),
      "utf8",
    );

    expect(workflow).toContain("tests/plugins/bitbucket-packaged-plugin.spec.ts");
    expect(workflow).toContain("KANDEV_BITBUCKET_PLUGIN_PACKAGE:");
    expect(workflow).toMatch(/Exercise optional external Kandev host[\s\S]*if:/);
  });
});

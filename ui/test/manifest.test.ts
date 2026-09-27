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

  it("uses the fixed Kandev SDK source pin in build and release workflows", async () => {
    const sdkRef = await readFile(new URL("../../.kandev-sdk-ref", import.meta.url), "utf8");
    const workflows = await Promise.all(
      ["build.yml", "ci.yml", "release.yml"].map((name) =>
        readFile(new URL(`../../.github/workflows/${name}`, import.meta.url), "utf8"),
      ),
    );

    expect(sdkRef.trim()).toBe("570600439036e81f8e9e1c63f15c4abce8a6c846");
    for (const workflow of workflows) {
      expect(workflow).toContain("apps/backend");
      expect(workflow).toContain("apps/packages/plugin-sdk");
      expect(workflow).toContain('sdk_ref="$(cat .kandev-sdk-ref)"');
    }
    expect(workflows[0]).toContain("ref: ${{ steps.sdk.outputs.ref }}");
    expect(workflows[1]).toContain("ref: ${{ steps.versions.outputs.sdk_ref }}");
    expect(workflows[2]).toContain("ref: ${{ steps.versions.outputs.sdk_ref }}");
  });

  it("checks out the complete minimum host for release package validation", async () => {
    const workflow = await readFile(
      new URL("../../.github/workflows/release.yml", import.meta.url),
      "utf8",
    );

    expect(workflow).toContain("name: Check out minimum supported Kandev host");
    const minimumHostCheckout = workflow
      .split("name: Check out minimum supported Kandev host")[1]
      ?.split("\n      - name:")[0];
    expect(minimumHostCheckout).toContain("ref: v${{ steps.versions.outputs.min_version }}");
    expect(minimumHostCheckout).toContain("path: kandev-min");
    expect(minimumHostCheckout).not.toContain("sparse-checkout:");
    expect(workflow).toContain("path: kandev\n");
  });

  it("runs packaged tests against the manifest minimum Kandev release", async () => {
    const manifest = await readFile(new URL("../../manifest.yaml", import.meta.url), "utf8");
    const minimumVersion = manifest.match(/^min_kandev_version: "([^"]+)"$/m)?.[1];
    const [ci, release] = await Promise.all(
      ["ci.yml", "release.yml"].map((name) =>
        readFile(new URL(`../../.github/workflows/${name}`, import.meta.url), "utf8"),
      ),
    );

    expect(minimumVersion).toBe("0.88.0");
    for (const workflow of [ci, release]) {
      expect(workflow).toContain("min_kandev_version:");
      expect(workflow).toContain("ref: v${{ steps.versions.outputs.min_version }}");
      expect(workflow).not.toContain(`ref: v${minimumVersion}`);
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

  it("resolves the pull-request package from release metadata", async () => {
    const workflow = await readFile(
      new URL("../../.github/workflows/ci.yml", import.meta.url),
      "utf8",
    );

    expect(workflow).toContain("id: package");
    expect(workflow).toContain("manifest.yaml");
    expect(workflow).toContain(
      "KANDEV_BITBUCKET_PLUGIN_PACKAGE: ${{ steps.package.outputs.path }}",
    );
    expect(workflow).not.toMatch(/kandev-plugin-bitbucket-\d+\.\d+\.\d+\.tar\.gz/);
  });
});

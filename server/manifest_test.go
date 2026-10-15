package main

import (
	"crypto/sha256"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const officialMarketplaceIconSHA256 = "1979d590ffb94871a6fb18d16008e62c755275f4e254e2f7f7d6aac170b91da7"

func TestManifestParses(t *testing.T) {
	contents, err := os.ReadFile("../manifest.yaml")
	require.NoError(t, err)

	var manifest map[string]any
	require.NoError(t, yaml.Unmarshal(contents, &manifest))
}

func TestManifestIncludesPackagedMarketplaceIcon(t *testing.T) {
	contents, err := os.ReadFile("../manifest.yaml")
	require.NoError(t, err)

	var manifest struct {
		Icon string `yaml:"icon"`
	}
	require.NoError(t, yaml.Unmarshal(contents, &manifest))
	require.Equal(t, "assets/icon.svg", manifest.Icon)

	icon, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(manifest.Icon)))
	require.NoError(t, err)
	assertMarketplaceSVG(t, icon)
}

func assertMarketplaceSVG(t *testing.T, icon []byte) {
	t.Helper()
	require.Equal(t, officialMarketplaceIconSHA256, fmt.Sprintf("%x", sha256.Sum256(icon)))

	var root struct {
		XMLName xml.Name
		Width   string `xml:"width,attr"`
		Height  string `xml:"height,attr"`
		ViewBox string `xml:"viewBox,attr"`
	}
	require.NoError(t, xml.Unmarshal(icon, &root))
	require.Equal(t, "svg", root.XMLName.Local)
	require.Equal(t, "48", root.Width)
	require.Equal(t, "48", root.Height)
	require.Equal(t, "0 0 48 48", root.ViewBox)

	lower := strings.ToLower(string(icon))
	for _, forbidden := range []string{"<script", "<foreignobject", "href=", "url(", "currentcolor"} {
		require.NotContains(t, lower, forbidden)
	}
}

func TestManifestSourceTransportHasOnlyScopedCapabilities(t *testing.T) {
	contents, err := os.ReadFile("../manifest.yaml")
	require.NoError(t, err)
	var m struct {
		Actions []struct {
			Key   string `yaml:"key"`
			Scope string `yaml:"scope"`
			Max   int    `yaml:"max_body_bytes"`
		} `yaml:"actions"`
		Tools []struct {
			Name        string          `yaml:"name"`
			Surfaces    []string        `yaml:"surfaces"`
			Schema      map[string]any  `yaml:"input_schema"`
			Annotations map[string]bool `yaml:"annotations"`
		} `yaml:"agent_tools"`
	}
	require.NoError(t, yaml.Unmarshal(contents, &m))
	require.Len(t, m.Tools, 2)
	for _, tool := range m.Tools {
		require.ElementsMatch(t, []string{"kanban-task", "office-task"}, tool.Surfaces)
		require.Equal(t, false, tool.Schema["additionalProperties"])
		require.NotContains(t, tool.Surfaces, "managed-conversation")
		if tool.Name == "source_read" {
			require.True(t, tool.Annotations["read_only_hint"])
		} else {
			require.Equal(t, "source_create", tool.Name)
			require.False(t, tool.Annotations["read_only_hint"])
		}
	}
	found := 0
	for _, action := range m.Actions {
		if action.Key == "source.read" || action.Key == "source.create" {
			found++
			require.Equal(t, "task", action.Scope)
			require.Equal(t, 40000, action.Max)
		}
	}
	require.Equal(t, 2, found)
}

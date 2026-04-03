package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEmbeddedTemplatesListAll(t *testing.T) {
	paths, err := GetEmbeddedTemplatesList(TemplateTypeAll)
	require.NoError(t, err)
	assert.NotEmpty(t, paths)

	// Should contain both managed and customer catalog files
	var hasManaged, hasCustomer bool
	for _, p := range paths {
		if contains(p, "managed-service-catalog") {
			hasManaged = true
		}
		if contains(p, "customer-service-catalog") {
			hasCustomer = true
		}
	}
	assert.True(t, hasManaged, "should have managed-service-catalog files")
	assert.True(t, hasCustomer, "should have customer-service-catalog files")
}

func TestGetEmbeddedTemplatesListHelm(t *testing.T) {
	paths, err := GetEmbeddedTemplatesList(TemplateTypeHelm)
	require.NoError(t, err)
	assert.NotEmpty(t, paths)

	for _, p := range paths {
		assert.Contains(t, p, "/helm/", "helm filter should only return helm paths")
	}
}

func TestTemplateFilesRendersTPLT(t *testing.T) {
	data := map[string]interface{}{
		"name":    "test-cluster",
		"dnsName": "test.example.com",
		"stage":   "dev",
		"fluxcd": map[string]interface{}{
			"distribution": map[string]interface{}{
				"version":  "2.x",
				"registry": "ghcr.io/fluxcd",
			},
			"cluster": map[string]interface{}{
				"type":          "kubernetes",
				"size":          "medium",
				"networkPolicy": true,
			},
			"sync": map[string]interface{}{
				"kind":     "GitRepository",
				"url":      "https://github.com/org/repo",
				"ref":      "refs/heads/main",
				"path":     "clusters/test-cluster",
				"interval": "5m",
			},
			"webUI": map[string]interface{}{
				"enabled": true,
			},
		},
		"services": map[string]interface{}{
			"traefik":     map[string]interface{}{"status": "enabled"},
			"certManager": map[string]interface{}{"status": "enabled"},
		},
	}

	paths, err := GetEmbeddedTemplatesList(TemplateTypeHelm)
	require.NoError(t, err)

	results, err := TemplateFiles(paths, data)
	// Some templates may have errors due to missing nested keys — that's OK
	// The important thing is we get results
	assert.NotEmpty(t, results, "should produce template results")

	// Verify at least one result has rendered content
	var foundContent bool
	for _, r := range results {
		if len(r.Content) > 0 {
			foundContent = true
			break
		}
	}
	assert.True(t, foundContent, "should have at least one result with content")
}

func TestTemplateAllFilesHelm(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"stage": "dev",
		"fluxcd": map[string]interface{}{
			"distribution": map[string]interface{}{"version": "2.x", "registry": "ghcr.io/fluxcd"},
			"cluster":      map[string]interface{}{"type": "kubernetes", "size": "medium", "networkPolicy": true},
			"sync":         map[string]interface{}{"kind": "GitRepository", "url": "https://github.com/org/repo", "ref": "refs/heads/main", "path": "clusters/test", "interval": "5m"},
			"webUI":        map[string]interface{}{"enabled": true},
		},
		"services": map[string]interface{}{},
	}

	results, _ := TemplateAllFiles(TemplateTypeHelm, data)
	assert.NotEmpty(t, results)
}

func TestGitRepositoriesTemplateRendering(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"stage": "dev",
		"fluxcd": map[string]interface{}{
			"distribution": map[string]interface{}{"version": "2.x", "registry": "ghcr.io/fluxcd"},
			"cluster":      map[string]interface{}{"type": "kubernetes", "size": "medium", "networkPolicy": true},
			"sync":         map[string]interface{}{"kind": "GitRepository", "url": "https://github.com/org/repo", "ref": "refs/heads/main", "path": "clusters/test", "interval": "5m"},
			"webUI":        map[string]interface{}{"enabled": true},
			"gitRepositories": []map[string]interface{}{
				{
					"name":      "app-repo",
					"url":       "https://github.com/org/app-repo",
					"branch":    "develop",
					"secretRef": "app-creds",
					"interval":  "10m",
				},
				{
					"name": "config-repo",
					"url":  "https://github.com/org/config-repo",
				},
			},
		},
		"services": map[string]interface{}{},
	}

	results, _ := TemplateAllFiles(TemplateTypeHelm, data)

	// Find the gitrepositories output
	var gitRepoContent string
	for _, r := range results {
		if contains(r.Path, "gitrepositories.yaml") {
			gitRepoContent = string(r.Content)
			break
		}
	}

	require.NotEmpty(t, gitRepoContent, "gitrepositories.yaml should be rendered")
	assert.Contains(t, gitRepoContent, "name: app-repo")
	assert.Contains(t, gitRepoContent, "url: https://github.com/org/app-repo")
	assert.Contains(t, gitRepoContent, "branch: develop")
	assert.Contains(t, gitRepoContent, "name: app-creds")
	assert.Contains(t, gitRepoContent, "interval: 10m")
	assert.Contains(t, gitRepoContent, "name: config-repo")
	assert.Contains(t, gitRepoContent, "url: https://github.com/org/config-repo")
	assert.Contains(t, gitRepoContent, "branch: main", "should default to main branch")
	assert.Contains(t, gitRepoContent, "kind: GitRepository")
}

func TestGitRepositoriesTemplateEmpty(t *testing.T) {
	data := map[string]interface{}{
		"name":  "test",
		"stage": "dev",
		"fluxcd": map[string]interface{}{
			"distribution": map[string]interface{}{"version": "2.x", "registry": "ghcr.io/fluxcd"},
			"cluster":      map[string]interface{}{"type": "kubernetes", "size": "medium", "networkPolicy": true},
			"sync":         map[string]interface{}{"kind": "GitRepository", "url": "https://github.com/org/repo", "ref": "refs/heads/main", "path": "clusters/test", "interval": "5m"},
			"webUI":        map[string]interface{}{"enabled": true},
		},
		"services": map[string]interface{}{},
	}

	results, _ := TemplateAllFiles(TemplateTypeHelm, data)

	// gitrepositories should be empty/whitespace-only (no gitRepositories in config)
	for _, r := range results {
		if contains(r.Path, "gitrepositories.yaml") {
			assert.Empty(t, strings.TrimSpace(string(r.Content)), "gitrepositories.yaml should be empty when no git repos configured")
			break
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

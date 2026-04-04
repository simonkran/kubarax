package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigManagerLoadAndValidate(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yamlContent := `clusters:
  - name: test-cluster
    stage: dev
    type: controlplane
    dnsName: test.example.com
    fluxcd:
      distribution:
        version: "2.x"
        registry: ghcr.io/fluxcd
      cluster:
        type: kubernetes
        size: medium
        networkPolicy: true
      sync:
        kind: GitRepository
        url: https://github.com/org/repo
        ref: refs/heads/main
        path: clusters/test-cluster
        interval: 5m
      webUI:
        enabled: true
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      externalDns:
        status: enabled
      externalSecrets:
        status: enabled
      kubePrometheusStack:
        status: enabled
      loki:
        status: enabled
      metricsServer:
        status: enabled
      kyverno:
        status: disabled
      kyvernoPolicies:
        status: disabled
      kyvernoPolicyReporter:
        status: disabled
      oauth2Proxy:
        status: disabled
      longhorn:
        status: disabled
      metallb:
        status: disabled
      homerDashboard:
        status: enabled
      forgejo:
        status: disabled
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0600)
	require.NoError(t, err)

	cm := NewConfigManager(configPath)
	err = cm.Load()
	require.NoError(t, err)

	cfg := cm.GetConfig()
	require.Len(t, cfg.Clusters, 1)

	cluster := cfg.Clusters[0]
	assert.Equal(t, "test-cluster", cluster.Name)
	assert.Equal(t, "dev", cluster.Stage)
	assert.Equal(t, "controlplane", cluster.Type)
	assert.Equal(t, "test.example.com", cluster.DNSName)

	// FluxCD config
	assert.Equal(t, "2.x", cluster.FluxCD.Distribution.Version)
	assert.Equal(t, "ghcr.io/fluxcd", cluster.FluxCD.Distribution.Registry)
	assert.Equal(t, "kubernetes", cluster.FluxCD.Cluster.Type)
	assert.Equal(t, "medium", cluster.FluxCD.Cluster.Size)
	assert.True(t, cluster.FluxCD.Cluster.NetworkPolicy)
	assert.Equal(t, "GitRepository", cluster.FluxCD.Sync.Kind)
	assert.Equal(t, "https://github.com/org/repo", cluster.FluxCD.Sync.URL)
	assert.Equal(t, "refs/heads/main", cluster.FluxCD.Sync.Ref)
	assert.Equal(t, "clusters/test-cluster", cluster.FluxCD.Sync.Path)
	assert.True(t, cluster.FluxCD.WebUI.Enabled)

	// Services
	assert.Equal(t, ServiceEnabled, cluster.Services.Traefik.Status)
	assert.Equal(t, ServiceEnabled, cluster.Services.CertManager.Status)
	assert.Equal(t, ServiceDisabled, cluster.Services.Kyverno.Status)

	assert.Equal(t, ServiceDisabled, cluster.Services.Forgejo.Status)
}

func TestConfigManagerSaveToFile(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "output", "config.yaml")

	cm := NewConfigManager(configPath)
	cm.GetConfig().Clusters = []Cluster{
		{
			Name:    "save-test",
			Stage:   "prod",
			Type:    "controlplane",
			DNSName: "save.example.com",
			FluxCD: FluxConfig{
				Distribution: FluxDistribution{Version: "2.x", Registry: "ghcr.io/fluxcd"},
				Cluster:      FluxCluster{Type: "kubernetes", Size: "small"},
				Sync:         FluxSync{Kind: "GitRepository", URL: "https://github.com/org/repo"},
			},
			Services: DefaultServices(),
		},
	}

	err := cm.SaveToFile()
	require.NoError(t, err)

	// Verify the file was created
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	// Re-load and verify
	cm2 := NewConfigManager(configPath)
	err = cm2.Load()
	require.NoError(t, err)

	assert.Equal(t, "save-test", cm2.GetConfig().Clusters[0].Name)
	assert.Equal(t, "prod", cm2.GetConfig().Clusters[0].Stage)
}

func TestConfigManagerLoadNonexistentFile(t *testing.T) {
	cm := NewConfigManager("/nonexistent/path/config.yaml")
	err := cm.Load()
	assert.Error(t, err)
}

func TestGenerateSchema(t *testing.T) {
	schema, err := GenerateSchema()
	require.NoError(t, err)
	require.NotNil(t, schema)

	// Schema should reference Config type
	assert.NotNil(t, schema.Ref)
}

func TestConfigWithGitRepositories(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yamlContent := `clusters:
  - name: test-cluster
    stage: dev
    type: controlplane
    dnsName: test.example.com
    fluxcd:
      distribution:
        version: "2.x"
        registry: ghcr.io/fluxcd
      sync:
        kind: GitRepository
        url: https://github.com/org/repo
        ref: refs/heads/main
        path: clusters/test-cluster
        interval: 5m
      gitRepositories:
        - name: app-repo
          url: https://github.com/org/app-repo
          branch: develop
          secretRef: app-repo-creds
          interval: 10m
        - name: config-repo
          url: https://github.com/org/config-repo
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      externalDns:
        status: enabled
      externalSecrets:
        status: enabled
      kubePrometheusStack:
        status: enabled
      loki:
        status: enabled
      metricsServer:
        status: enabled
      kyverno:
        status: disabled
      kyvernoPolicies:
        status: disabled
      kyvernoPolicyReporter:
        status: disabled
      oauth2Proxy:
        status: disabled
      longhorn:
        status: disabled
      metallb:
        status: disabled
      homerDashboard:
        status: enabled
      forgejo:
        status: disabled
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0600)
	require.NoError(t, err)

	cm := NewConfigManager(configPath)
	err = cm.Load()
	require.NoError(t, err)

	cfg := cm.GetConfig()
	cluster := cfg.Clusters[0]

	require.Len(t, cluster.FluxCD.GitRepositories, 2)

	assert.Equal(t, "app-repo", cluster.FluxCD.GitRepositories[0].Name)
	assert.Equal(t, "https://github.com/org/app-repo", cluster.FluxCD.GitRepositories[0].URL)
	assert.Equal(t, "develop", cluster.FluxCD.GitRepositories[0].Branch)
	assert.Equal(t, "app-repo-creds", cluster.FluxCD.GitRepositories[0].SecretRef)
	assert.Equal(t, "10m", cluster.FluxCD.GitRepositories[0].Interval)

	assert.Equal(t, "config-repo", cluster.FluxCD.GitRepositories[1].Name)
	assert.Equal(t, "https://github.com/org/config-repo", cluster.FluxCD.GitRepositories[1].URL)
	assert.Empty(t, cluster.FluxCD.GitRepositories[1].Branch)
	assert.Empty(t, cluster.FluxCD.GitRepositories[1].SecretRef)
}

func TestConfigWithApplications(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	yamlContent := `clusters:
  - name: test-cluster
    stage: dev
    type: controlplane
    dnsName: test.example.com
    fluxcd:
      distribution:
        version: "2.x"
        registry: ghcr.io/fluxcd
      sync:
        kind: GitRepository
        url: https://github.com/org/repo
        ref: refs/heads/main
        path: clusters/test-cluster
        interval: 5m
    services:
      traefik:
        status: enabled
      certManager:
        status: enabled
      externalDns:
        status: enabled
      externalSecrets:
        status: enabled
      kubePrometheusStack:
        status: enabled
      loki:
        status: enabled
      metricsServer:
        status: enabled
      kyverno:
        status: disabled
      kyvernoPolicies:
        status: disabled
      kyvernoPolicyReporter:
        status: disabled
      oauth2Proxy:
        status: disabled
      longhorn:
        status: disabled
      metallb:
        status: disabled
      homerDashboard:
        status: enabled
      forgejo:
        status: disabled
    applications:
      - name: node-red
        type: kustomization
        sourceRef:
          kind: GitRepository
          name: app-repo
        path: ./apps/node-red
        targetNamespace: home-automation
        dependsOn:
          - cert-manager
      - name: grafana
        type: helmrelease
        sourceRef:
          kind: HelmRepository
          name: grafana-charts
        chart: grafana
        chartVersion: "7.x"
        targetNamespace: monitoring
        createNamespace: true
        values:
          replicas: 1
      - name: edge-agent
        type: kustomization
        sourceRef:
          kind: GitRepository
          name: app-repo
        path: ./apps/edge-agent
        targetNamespace: edge
        serviceAccountName: flux-edge
        kubeConfig:
          secretRef: worker-kubeconfig
`
	err := os.WriteFile(configPath, []byte(yamlContent), 0600)
	require.NoError(t, err)

	cm := NewConfigManager(configPath)
	err = cm.Load()
	require.NoError(t, err)

	cfg := cm.GetConfig()
	cluster := cfg.Clusters[0]

	require.Len(t, cluster.Applications, 3)

	// Kustomization app
	app0 := cluster.Applications[0]
	assert.Equal(t, "node-red", app0.Name)
	assert.Equal(t, "kustomization", app0.Type)
	assert.Equal(t, "GitRepository", app0.SourceRef.Kind)
	assert.Equal(t, "app-repo", app0.SourceRef.Name)
	assert.Equal(t, "./apps/node-red", app0.Path)
	assert.Equal(t, "home-automation", app0.TargetNamespace)
	assert.Equal(t, []string{"cert-manager"}, app0.DependsOn)

	// HelmRelease app
	app1 := cluster.Applications[1]
	assert.Equal(t, "grafana", app1.Name)
	assert.Equal(t, "helmrelease", app1.Type)
	assert.Equal(t, "HelmRepository", app1.SourceRef.Kind)
	assert.Equal(t, "grafana-charts", app1.SourceRef.Name)
	assert.Equal(t, "grafana", app1.Chart)
	assert.Equal(t, "7.x", app1.ChartVersion)
	assert.Equal(t, "monitoring", app1.TargetNamespace)
	assert.True(t, app1.CreateNamespace)
	assert.NotNil(t, app1.Values)

	// App with kubeConfig
	app2 := cluster.Applications[2]
	assert.Equal(t, "edge-agent", app2.Name)
	assert.Equal(t, "flux-edge", app2.ServiceAccount)
	require.NotNil(t, app2.KubeConfig)
	assert.Equal(t, "worker-kubeconfig", app2.KubeConfig.SecretRef)
}

func TestDefaultServices(t *testing.T) {
	services := DefaultServices()

	assert.Equal(t, ServiceEnabled, services.Traefik.Status)
	assert.Equal(t, ServiceEnabled, services.CertManager.Status)
	assert.Equal(t, ServiceEnabled, services.ExternalDNS.Status)
	assert.Equal(t, ServiceEnabled, services.ExternalSecrets.Status)
	assert.Equal(t, ServiceEnabled, services.KubePrometheusStack.Status)
	assert.Equal(t, ServiceEnabled, services.Loki.Status)
	assert.Equal(t, ServiceEnabled, services.MetricsServer.Status)
	assert.Equal(t, ServiceDisabled, services.Kyverno.Status)
	assert.Equal(t, ServiceDisabled, services.KyvernoPolicies.Status)
	assert.Equal(t, ServiceDisabled, services.KyvernoPolicyReporter.Status)
	assert.Equal(t, ServiceDisabled, services.OAuth2Proxy.Status)
	assert.Equal(t, ServiceDisabled, services.Longhorn.Status)
	assert.Equal(t, ServiceDisabled, services.MetalLB.Status)
	assert.Equal(t, ServiceEnabled, services.HomerDashboard.Status)
	assert.Equal(t, ServiceDisabled, services.Forgejo.Status)
}

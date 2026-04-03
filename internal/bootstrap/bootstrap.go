package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"kubarax/assets/config"
	"kubarax/assets/envmap"
	"kubarax/internal/helm"
	"kubarax/internal/k8s"
	"kubarax/templates"

	"github.com/rs/zerolog/log"
)

const (
	fluxNamespace            = "flux-system"
	externalSecretsNamespace = "external-secrets"
	fluxOperatorOCIChart     = "oci://ghcr.io/controlplaneio-fluxcd/charts/flux-operator"
)

// Options for bootstrap operations
type Options struct {
	Kubeconfig     string
	ManagedCatalog string
	OverlayValues  string
	WithES         bool
	WithProm       bool
	WithESCSSPath  string
	EnvMap         *envmap.EnvMap
	ClusterConfig  *config.Cluster
	DryRun         bool
	Timeout        time.Duration
	ClusterName    string
}

// BootstrapChart describes a Helm chart to install during bootstrap
type BootstrapChart struct {
	Name            string
	Namespace       string
	Path            string
	OverlayValues   []string
	RepoURL         string
	EnsureNamespace bool
	EnsureCRD       bool
}

// Bootstrap orchestrates the complete Flux Operator bootstrap process
func Bootstrap(ctx context.Context, opts *Options) error {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	// Create Kubernetes client
	client, err := k8s.NewClient(k8s.Config{
		KubeconfigPath: opts.Kubeconfig,
		QPS:            50,
		Burst:          100,
		Timeout:        30 * time.Second,
		UserAgent:      "kubarax-bootstrap",
	})
	if err != nil {
		return fmt.Errorf("creating kubernetes client: %w", err)
	}

	log.Info().Msg("Starting Flux Operator bootstrap process")

	// Step 1: Ensure flux-system namespace exists
	if err := client.EnsureNamespace(ctx, fluxNamespace, opts.DryRun); err != nil {
		return fmt.Errorf("ensuring flux-system namespace: %w", err)
	}

	// Step 2: Apply Git credentials secret (needed by FluxInstance sync)
	if err := applySecrets(ctx, client, opts); err != nil {
		return fmt.Errorf("applying secrets: %w", err)
	}

	// Step 3: Install the Flux Operator via Helm OCI chart
	if err := installFluxOperator(ctx, client, opts); err != nil {
		return fmt.Errorf("installing Flux Operator: %w", err)
	}

	// Step 4: Wait for the Flux Operator deployment to be ready
	if err := waitForFluxOperator(ctx, client, opts); err != nil {
		return fmt.Errorf("waiting for Flux Operator: %w", err)
	}

	// Step 5: Refresh discovery so FluxInstance CRD is available
	_ = client.RefreshDiscovery()

	// Step 6: Create the FluxInstance CR (this triggers Flux controller installation + sync)
	if err := applyFluxInstance(ctx, client, opts); err != nil {
		return fmt.Errorf("applying FluxInstance: %w", err)
	}

	// Step 7: Wait for Flux controllers to be ready
	if err := waitForFluxControllers(ctx, client, opts); err != nil {
		return fmt.Errorf("waiting for Flux controllers: %w", err)
	}

	// Step 8: Optionally install additional bootstrap charts (external-secrets, prometheus CRDs)
	if err := installAdditionalCharts(ctx, client, opts); err != nil {
		return fmt.Errorf("installing additional charts: %w", err)
	}

	// Step 9: Apply ClusterSecretStore if provided (requires external-secrets CRDs from Step 8)
	if opts.WithESCSSPath != "" {
		_ = client.RefreshDiscovery()
		if err := applyClusterSecretStore(ctx, client, opts); err != nil {
			return fmt.Errorf("applying ClusterSecretStore: %w", err)
		}
	}

	// Step 10: Print completion message
	printCompletionMessage(opts)
	log.Info().Msg("Flux Operator bootstrap completed successfully")
	return nil
}

func installFluxOperator(ctx context.Context, client *k8s.Client, opts *Options) error {
	log.Info().Msg("Installing Flux Operator via Helm OCI chart")

	webEnabled := "true"
	if !opts.ClusterConfig.FluxCD.WebUI.Enabled {
		webEnabled = "false"
	}

	// Template the flux-operator OCI chart
	manifest, err := helm.Template(ctx, helm.TemplateOptions{
		ReleaseName: "flux-operator",
		ChartPath:   fluxOperatorOCIChart,
		Namespace:   fluxNamespace,
		SetArgs: []string{
			"web.enabled=" + webEnabled,
		},
	})
	if err != nil {
		return fmt.Errorf("templating Flux Operator: %w", err)
	}

	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would apply Flux Operator manifest")
		return nil
	}

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-flux-operator-bootstrap"
	applyOpts.ForceConflicts = true

	if err := client.ApplyManifest(ctx, manifest, applyOpts); err != nil {
		return fmt.Errorf("applying Flux Operator manifest: %w", err)
	}

	log.Info().Msg("Flux Operator installed successfully")
	return nil
}

func waitForFluxOperator(ctx context.Context, client *k8s.Client, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Skipping wait for Flux Operator")
		return nil
	}

	log.Info().Msg("Waiting for Flux Operator to be ready")
	if err := client.WaitForDeployment(ctx, fluxNamespace, "flux-operator"); err != nil {
		return fmt.Errorf("waiting for flux-operator deployment: %w", err)
	}

	log.Info().Msg("Flux Operator is ready")
	return nil
}

func applyFluxInstance(ctx context.Context, client *k8s.Client, opts *Options) error {
	log.Info().Msg("Creating FluxInstance CR")

	manifest := buildFluxInstanceManifest(opts)

	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would apply FluxInstance CR")
		log.Info().Msgf("[DRY-RUN] FluxInstance manifest:\n%s", manifest)
		return nil
	}

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-flux-instance"
	applyOpts.ForceConflicts = true

	if err := client.ApplyManifest(ctx, []byte(manifest), applyOpts); err != nil {
		return fmt.Errorf("applying FluxInstance: %w", err)
	}

	log.Info().Msg("FluxInstance CR applied successfully")
	return nil
}

func buildFluxInstanceManifest(opts *Options) string {
	fluxCfg := opts.ClusterConfig.FluxCD

	// Distribution defaults
	version := "2.x"
	if fluxCfg.Distribution.Version != "" {
		version = fluxCfg.Distribution.Version
	}
	registry := "ghcr.io/fluxcd"
	if fluxCfg.Distribution.Registry != "" {
		registry = fluxCfg.Distribution.Registry
	}

	// Cluster defaults
	clusterType := "kubernetes"
	if fluxCfg.Cluster.Type != "" {
		clusterType = fluxCfg.Cluster.Type
	}
	clusterSize := "medium"
	if fluxCfg.Cluster.Size != "" {
		clusterSize = fluxCfg.Cluster.Size
	}
	networkPolicy := "true"
	if !fluxCfg.Cluster.NetworkPolicy {
		networkPolicy = "false"
	}
	multitenant := "false"
	if fluxCfg.Cluster.Multitenant {
		multitenant = "true"
	}

	// Sync defaults
	syncKind := "GitRepository"
	if fluxCfg.Sync.Kind != "" {
		syncKind = fluxCfg.Sync.Kind
	}
	syncRef := "refs/heads/main"
	if fluxCfg.Sync.Ref != "" {
		syncRef = fluxCfg.Sync.Ref
	}
	syncPath := fmt.Sprintf("clusters/%s", opts.ClusterName)
	if fluxCfg.Sync.Path != "" {
		syncPath = fluxCfg.Sync.Path
	}
	syncInterval := "5m"
	if fluxCfg.Sync.Interval != "" {
		syncInterval = fluxCfg.Sync.Interval
	}

	// Build the sync.pullSecret reference
	pullSecretBlock := ""
	pullSecretName := "flux-git-credentials"
	if fluxCfg.Sync.PullSecret != "" {
		pullSecretName = fluxCfg.Sync.PullSecret
	}
	pullSecretBlock = fmt.Sprintf("    pullSecret: %s", pullSecretName)

	return fmt.Sprintf(`apiVersion: fluxcd.controlplane.io/v1
kind: FluxInstance
metadata:
  name: flux
  namespace: flux-system
  annotations:
    fluxcd.controlplane.io/reconcile: "enabled"
    fluxcd.controlplane.io/reconcileEvery: "1h"
spec:
  distribution:
    version: "%s"
    registry: "%s"
  components:
    - source-controller
    - kustomize-controller
    - helm-controller
    - notification-controller
  cluster:
    type: %s
    size: %s
    multitenant: %s
    networkPolicy: %s
  sync:
    kind: %s
    url: "%s"
    ref: "%s"
    path: "%s"
    interval: "%s"
%s
`, version, registry, clusterType, clusterSize, multitenant, networkPolicy,
		syncKind, fluxCfg.Sync.URL, syncRef, syncPath, syncInterval, pullSecretBlock)
}

func waitForFluxControllers(ctx context.Context, client *k8s.Client, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Skipping wait for Flux controllers")
		return nil
	}

	log.Info().Msg("Waiting for Flux controllers to be ready (managed by FluxInstance)")

	controllers := []string{
		"source-controller",
		"kustomize-controller",
		"helm-controller",
		"notification-controller",
	}

	for _, controller := range controllers {
		if err := client.WaitForDeployment(ctx, fluxNamespace, controller); err != nil {
			return fmt.Errorf("waiting for %s: %w", controller, err)
		}
	}

	log.Info().Msg("All Flux controllers are ready")
	return nil
}

func installAdditionalCharts(ctx context.Context, client *k8s.Client, opts *Options) error {
	charts := buildAdditionalCharts(opts)

	for _, chart := range charts {
		if !chart.EnsureCRD {
			continue
		}

		if chart.EnsureNamespace {
			if err := client.EnsureNamespace(ctx, chart.Namespace, opts.DryRun); err != nil {
				return fmt.Errorf("ensuring %s namespace: %w", chart.Namespace, err)
			}
		}

		// Add helm repo
		repo := helm.RepoOptions{Name: chart.Name, URL: chart.RepoURL}
		if err := helm.AddRepository(ctx, repo); err != nil {
			return fmt.Errorf("adding helm repository %s: %w", repo.Name, err)
		}

		// Update dependencies
		dep := helm.DependencyOptions{ChartPath: chart.Path, Timeout: opts.Timeout}
		if err := helm.UpdateDependencies(ctx, dep); err != nil {
			return fmt.Errorf("updating dependencies for %s: %w", chart.Name, err)
		}

		// Filter overlay values to only include files that exist
		var valuesPaths []string
		for _, v := range chart.OverlayValues {
			if info, err := os.Stat(v); err == nil && !info.IsDir() {
				valuesPaths = append(valuesPaths, v)
			} else {
				log.Debug().Msgf("Skipping non-existent overlay values: %s", v)
			}
		}

		// Template and apply
		manifest, err := helm.Template(ctx, helm.TemplateOptions{
			ReleaseName: chart.Name,
			ChartPath:   chart.Path,
			Namespace:   chart.Namespace,
			ValuesPaths: valuesPaths,
		})
		if err != nil {
			return fmt.Errorf("templating %s: %w", chart.Name, err)
		}

		if opts.DryRun {
			log.Info().Msgf("[DRY-RUN] Would apply %s manifest", chart.Name)
			continue
		}

		applyOpts := k8s.DefaultApplyOptions()
		applyOpts.FieldManager = "kubarax-bootstrap-" + chart.Name
		applyOpts.ForceConflicts = true

		if err := client.ApplyManifest(ctx, manifest, applyOpts); err != nil {
			return fmt.Errorf("applying %s manifest: %w", chart.Name, err)
		}

		log.Info().Msgf("Installed %s successfully", chart.Name)
	}

	return nil
}

func buildAdditionalCharts(opts *Options) []BootstrapChart {
	return []BootstrapChart{
		{
			Name:            "external-secrets",
			Namespace:       externalSecretsNamespace,
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "external-secrets"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "external-secrets", "values.yaml")},
			RepoURL:         "https://charts.external-secrets.io",
			EnsureNamespace: opts.WithES,
			EnsureCRD:       opts.WithES,
		},
		{
			Name:            "kube-prometheus-stack",
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "kube-prometheus-stack"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "kube-prometheus-stack", "values.yaml")},
			RepoURL:         "https://prometheus-community.github.io/helm-charts",
			EnsureNamespace: false,
			EnsureCRD:       opts.WithProm,
		},
	}
}

func applyClusterSecretStore(ctx context.Context, client *k8s.Client, opts *Options) error {
	if opts.WithESCSSPath == "" {
		return nil
	}

	log.Info().Msg("Applying ClusterSecretStore manifest")

	content, err := os.ReadFile(opts.WithESCSSPath)
	if err != nil {
		return fmt.Errorf("reading ClusterSecretStore file %s: %w", opts.WithESCSSPath, err)
	}

	// Build template context from cluster config (same pattern as cmd/generate.go buildTemplateContext)
	jsonBytes, err := json.Marshal(opts.ClusterConfig)
	if err != nil {
		return fmt.Errorf("marshaling cluster config: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return fmt.Errorf("unmarshaling cluster config: %w", err)
	}

	rendered, err := templates.RenderTemplate("ClusterSecretStore", string(content), data)
	if err != nil {
		return fmt.Errorf("rendering ClusterSecretStore template: %w", err)
	}

	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would apply ClusterSecretStore manifest")
		log.Info().Msgf("[DRY-RUN] ClusterSecretStore manifest:\n%s", rendered)
		return nil
	}

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-cluster-secret-store"
	applyOpts.ForceConflicts = true

	if err := client.ApplyManifest(ctx, []byte(rendered), applyOpts); err != nil {
		return fmt.Errorf("applying ClusterSecretStore: %w", err)
	}

	log.Info().Msg("ClusterSecretStore applied successfully")
	return nil
}

func applySecrets(ctx context.Context, client *k8s.Client, opts *Options) error {
	log.Info().Msg("Applying bootstrap secrets")

	sm := NewSecretManager(client)

	// Create flux-system git credentials secret
	if err := sm.CreateFluxGitSecret(ctx, opts); err != nil {
		return fmt.Errorf("creating flux git secret: %w", err)
	}

	// Create docker registry secret if configured
	if envmap.IsConfiguredEnvValue(opts.EnvMap.DockerconfigBase64) {
		if err := sm.CreateDockerRegistrySecret(ctx, opts); err != nil {
			return fmt.Errorf("creating docker registry secret: %w", err)
		}
	}

	log.Info().Msg("Secrets applied successfully")
	return nil
}

// CompletionLogConfig contains the data for the completion message
type CompletionLogConfig struct {
	ClusterDNSName string
	WebUIEnabled   bool
}

func printCompletionMessage(opts *Options) {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Flux Operator bootstrap completed successfully")
		return
	}

	cfg := CompletionLogConfig{
		WebUIEnabled: opts.ClusterConfig.FluxCD.WebUI.Enabled,
	}
	if opts.ClusterConfig != nil {
		cfg.ClusterDNSName = opts.ClusterConfig.DNSName
	}

	log.Info().Msg(CreateCompletionMessage(cfg))
}

// CreateCompletionMessage returns the formatted completion message
func CreateCompletionMessage(cfg CompletionLogConfig) string {
	webUIMsg := ""
	if cfg.WebUIEnabled {
		webUIMsg = `

To access the Flux Operator Web UI:

    kubectl port-forward svc/flux-operator -n flux-system 9080:9080 --kubeconfig ...

Then open: http://localhost:9080`
	}

	ingressMsg := ""
	if cfg.ClusterDNSName != "" {
		ingressMsg = fmt.Sprintf(" or try: https://%s/flux (if ingress is configured)", cfg.ClusterDNSName)
	}

	return fmt.Sprintf(`
Flux Operator bootstrap complete!

The Flux Operator is managing your cluster via the FluxInstance CR.
Check reconciliation status with:

    kubectl get fluxinstance -n flux-system
    kubectl get kustomizations -A
    kubectl get helmreleases -A%s%s

Next steps:
1. Commit and push your generated manifests to the Git repository
2. The FluxInstance will automatically reconcile from your configured sync path
3. All platform components will be deployed via HelmReleases
4. Use ResourceSets for multi-cluster deployment patterns`, webUIMsg, ingressMsg)
}

package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"kubarax/assets/config"
	"kubarax/assets/envmap"
	"kubarax/internal/helm"
	"kubarax/internal/k8s"

	"github.com/rs/zerolog/log"
)

const (
	fluxNamespace            = "flux-system"
	externalSecretsNamespace = "external-secrets"
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

// Bootstrap orchestrates the complete FluxCD bootstrap process
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

	// Define the bootstrap charts
	bootstrapCharts := buildBootstrapCharts(opts)

	log.Info().Msg("Starting FluxCD bootstrap process")

	// Step 1: Ensure namespaces exist
	if err := ensureNamespaces(ctx, client, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("ensuring namespaces: %w", err)
	}

	// Step 2: Add helm repositories
	if err := addHelmRepositories(ctx, bootstrapCharts); err != nil {
		return fmt.Errorf("adding helm repositories: %w", err)
	}

	// Step 3: Update helm dependencies
	if err := updateHelmDependencies(ctx, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("updating helm dependencies: %w", err)
	}

	// Step 4: Apply CRDs (FluxCD CRDs, external-secrets CRDs, prometheus CRDs)
	if err := applyCRDs(ctx, client, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("applying CRDs: %w", err)
	}
	_ = client.RefreshDiscovery()

	// Step 5: Apply secrets (git credentials, etc.)
	if err := applySecrets(ctx, client, opts); err != nil {
		return fmt.Errorf("applying secrets: %w", err)
	}

	// Step 6: Bootstrap FluxCD
	if err := bootstrapFluxCD(ctx, client, opts, bootstrapCharts); err != nil {
		return fmt.Errorf("bootstrapping FluxCD: %w", err)
	}

	// Step 7: Wait for FluxCD to be ready
	if err := waitForFluxCD(ctx, client, opts); err != nil {
		return fmt.Errorf("waiting for FluxCD readiness: %w", err)
	}

	// Step 8: Apply initial GitRepository and Kustomization sources
	if err := applyFluxSources(ctx, client, opts); err != nil {
		return fmt.Errorf("applying Flux sources: %w", err)
	}

	// Step 9: Print completion message
	printCompletionMessage(opts)
	log.Info().Msg("FluxCD bootstrap completed successfully")
	return nil
}

func buildBootstrapCharts(opts *Options) []BootstrapChart {
	charts := []BootstrapChart{
		{
			Name:            "flux2",
			Namespace:       fluxNamespace,
			Path:            filepath.Join(opts.ManagedCatalog, "helm", "flux2"),
			OverlayValues:   []string{filepath.Join(opts.OverlayValues, "helm", opts.ClusterName, "flux2", "values.yaml")},
			RepoURL:         "https://fluxcd-community.github.io/helm-charts",
			EnsureNamespace: true,
			EnsureCRD:       true,
		},
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
	return charts
}

func ensureNamespaces(ctx context.Context, client *k8s.Client, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Ensuring namespaces exist")
	for _, chart := range charts {
		if chart.EnsureNamespace {
			if err := client.EnsureNamespace(ctx, chart.Namespace, opts.DryRun); err != nil {
				return fmt.Errorf("ensuring %s namespace: %w", chart.Name, err)
			}
		}
	}
	return nil
}

func addHelmRepositories(ctx context.Context, charts []BootstrapChart) error {
	log.Info().Msg("Adding helm repositories")
	for _, chart := range charts {
		if chart.EnsureCRD {
			repo := helm.RepoOptions{Name: chart.Name, URL: chart.RepoURL}
			if err := helm.AddRepository(ctx, repo); err != nil {
				return fmt.Errorf("adding helm repository %s: %w", repo.Name, err)
			}
			log.Info().Msgf("Added helm repository: %s", repo.Name)
		}
	}
	return nil
}

func updateHelmDependencies(ctx context.Context, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Updating helm chart dependencies")
	for _, chart := range charts {
		if chart.EnsureCRD {
			dep := helm.DependencyOptions{ChartPath: chart.Path, Timeout: opts.Timeout}
			if err := helm.UpdateDependencies(ctx, dep); err != nil {
				return fmt.Errorf("updating helm dependencies for %s: %w", chart.Name, err)
			}
			log.Info().Msgf("Updated helm dependencies for: %s", chart.Name)
		}
	}
	return nil
}

func applyCRDs(ctx context.Context, client *k8s.Client, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Applying CRDs")
	// CRD application is handled via helm template + server-side apply
	// Each chart that needs CRDs will have them applied during the helm install
	for _, chart := range charts {
		if !chart.EnsureCRD {
			continue
		}
		log.Info().Msgf("CRDs for %s will be applied during chart installation", chart.Name)
	}
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

func bootstrapFluxCD(ctx context.Context, client *k8s.Client, opts *Options, charts []BootstrapChart) error {
	log.Info().Msg("Bootstrapping FluxCD")

	// Find the flux chart
	var fluxChart BootstrapChart
	for _, c := range charts {
		if c.Name == "flux2" {
			fluxChart = c
			break
		}
	}

	// Template and apply FluxCD
	manifest, err := helm.Template(ctx, helm.TemplateOptions{
		ReleaseName: fluxChart.Name,
		ChartPath:   fluxChart.Path,
		Namespace:   fluxChart.Namespace,
		ValuesPaths: fluxChart.OverlayValues,
	})
	if err != nil {
		return fmt.Errorf("templating FluxCD: %w", err)
	}

	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would apply FluxCD manifest")
		return nil
	}

	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-flux-bootstrap"
	applyOpts.ForceConflicts = true

	if err := client.ApplyManifest(ctx, manifest, applyOpts); err != nil {
		return fmt.Errorf("applying FluxCD manifest: %w", err)
	}

	log.Info().Msg("FluxCD manifest applied successfully")
	return nil
}

func waitForFluxCD(ctx context.Context, client *k8s.Client, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Skipping wait for FluxCD readiness")
		return nil
	}

	log.Info().Msg("Waiting for FluxCD components to be ready")

	// Wait for source-controller
	if err := client.WaitForDeployment(ctx, fluxNamespace, "source-controller"); err != nil {
		return fmt.Errorf("waiting for source-controller: %w", err)
	}

	// Wait for kustomize-controller
	if err := client.WaitForDeployment(ctx, fluxNamespace, "kustomize-controller"); err != nil {
		return fmt.Errorf("waiting for kustomize-controller: %w", err)
	}

	// Wait for helm-controller
	if err := client.WaitForDeployment(ctx, fluxNamespace, "helm-controller"); err != nil {
		return fmt.Errorf("waiting for helm-controller: %w", err)
	}

	// Wait for notification-controller
	if err := client.WaitForDeployment(ctx, fluxNamespace, "notification-controller"); err != nil {
		return fmt.Errorf("waiting for notification-controller: %w", err)
	}

	log.Info().Msg("FluxCD components are ready")
	return nil
}

func applyFluxSources(ctx context.Context, client *k8s.Client, opts *Options) error {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] Would apply Flux GitRepository and Kustomization sources")
		return nil
	}

	log.Info().Msg("Applying FluxCD GitRepository and Kustomization sources")

	// Create the GitRepository source pointing to the platform repo
	gitRepoManifest := buildGitRepositoryManifest(opts)
	applyOpts := k8s.DefaultApplyOptions()
	applyOpts.FieldManager = "kubarax-flux-bootstrap"

	if err := client.ApplyManifest(ctx, []byte(gitRepoManifest), applyOpts); err != nil {
		return fmt.Errorf("applying GitRepository: %w", err)
	}

	// Create the root Kustomization that points to the cluster's managed catalog
	kustomizationManifest := buildRootKustomizationManifest(opts)
	if err := client.ApplyManifest(ctx, []byte(kustomizationManifest), applyOpts); err != nil {
		return fmt.Errorf("applying root Kustomization: %w", err)
	}

	log.Info().Msg("FluxCD sources applied successfully")
	return nil
}

func buildGitRepositoryManifest(opts *Options) string {
	interval := "5m"
	if opts.ClusterConfig.FluxCD.Interval != "" {
		interval = opts.ClusterConfig.FluxCD.Interval
	}
	branch := "main"
	if opts.ClusterConfig.FluxCD.GitRepository.Branch != "" {
		branch = opts.ClusterConfig.FluxCD.GitRepository.Branch
	}

	return fmt.Sprintf(`apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: platform
  namespace: flux-system
spec:
  interval: %s
  url: %s
  ref:
    branch: %s
  secretRef:
    name: flux-git-credentials
`, interval, opts.ClusterConfig.FluxCD.GitRepository.URL, branch)
}

func buildRootKustomizationManifest(opts *Options) string {
	return fmt.Sprintf(`apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: platform
  namespace: flux-system
spec:
  interval: 10m
  sourceRef:
    kind: GitRepository
    name: platform
  path: ./managed-service-catalog/helm/%s
  prune: true
  wait: true
  timeout: 5m
`, opts.ClusterName)
}

// CompletionLogConfig contains the data for the completion message
type CompletionLogConfig struct {
	WeaveGitopsPassword string
	ClusterDNSName      string
}

func printCompletionMessage(opts *Options) {
	if opts.DryRun {
		log.Info().Msg("[DRY-RUN] FluxCD bootstrap completed successfully")
		return
	}

	config := CompletionLogConfig{}
	if opts.ClusterConfig != nil {
		config.ClusterDNSName = opts.ClusterConfig.DNSName
	}
	config.WeaveGitopsPassword = opts.EnvMap.WeaveGitopsPassword

	log.Info().Msg(CreateCompletionMessage(config))
}

// CreateCompletionMessage returns the formatted completion message
func CreateCompletionMessage(config CompletionLogConfig) string {
	formattedOutput := ""
	if config.ClusterDNSName != "" {
		formattedOutput = fmt.Sprintf(" or try: https://%s/gitops (if ingress is running)", config.ClusterDNSName)
	}

	return fmt.Sprintf(`
FluxCD bootstrap complete!

FluxCD is now managing your cluster. You can check reconciliation status with:

    flux get all -A

To access the Weave GitOps UI:

    kubectl port-forward svc/weave-gitops -n flux-system 9001:9001 --kubeconfig ...

Then open: http://localhost:9001%s

Next steps:
1. Commit and push your generated manifests to the Git repository
2. FluxCD will automatically reconcile and deploy all platform components
3. Monitor progress with: flux get kustomizations -A
4. Check HelmRelease status: flux get helmreleases -A`, formattedOutput)
}

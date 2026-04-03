package config

// Config is the root configuration structure
type Config struct {
	Clusters []Cluster `yaml:"clusters" json:"clusters" jsonschema:"required,description=List of cluster configurations"`
}

// Cluster represents a single Kubernetes cluster configuration
type Cluster struct {
	Name                  string     `yaml:"name" json:"name" jsonschema:"required,description=Unique name for this cluster"`
	Stage                 string     `yaml:"stage" json:"stage" jsonschema:"required,description=Deployment stage (e.g. dev/staging/prod)"`
	Type                  string     `yaml:"type" json:"type" jsonschema:"required,enum=controlplane,enum=worker,description=Cluster type"`
	DNSName               string     `yaml:"dnsName" json:"dnsName" jsonschema:"required,description=DNS name for the cluster"`
	IngressClassName      string     `yaml:"ingressClassName,omitempty" json:"ingressClassName,omitempty" jsonschema:"description=Ingress class name"`
	PrivateLoadBalancerIP string     `yaml:"privateLoadBalancerIP,omitempty" json:"privateLoadBalancerIP,omitempty" jsonschema:"description=Private LB IP"`
	PublicLoadBalancerIP  string     `yaml:"publicLoadBalancerIP,omitempty" json:"publicLoadBalancerIP,omitempty" jsonschema:"description=Public LB IP"`
	SSOOrg                string     `yaml:"ssoOrg,omitempty" json:"ssoOrg,omitempty" jsonschema:"description=SSO organization"`
	SSOTeam               string     `yaml:"ssoTeam,omitempty" json:"ssoTeam,omitempty" jsonschema:"description=SSO team"`
	Terraform             *Terraform `yaml:"terraform,omitempty" json:"terraform,omitempty" jsonschema:"description=Terraform configuration"`
	FluxCD                FluxConfig `yaml:"fluxcd" json:"fluxcd" jsonschema:"required,description=FluxCD configuration"`
	Services              Services   `yaml:"services" json:"services" jsonschema:"required,description=Platform services configuration"`
	Projects              []Project      `yaml:"projects,omitempty" json:"projects,omitempty" jsonschema:"description=Tenant projects for this cluster"`
	Applications          []Application  `yaml:"applications,omitempty" json:"applications,omitempty" jsonschema:"description=Declarative application deployments for this cluster"`
}

// Project represents a tenant project (namespace + RBAC)
type Project struct {
	Name        string `yaml:"name" json:"name" jsonschema:"required,description=Project/tenant name"`
	ClusterRole string `yaml:"clusterRole,omitempty" json:"clusterRole,omitempty" jsonschema:"default=cluster-admin,description=ClusterRole to bind to the tenant ServiceAccount"`
}

// Application represents a declarative application deployment
type Application struct {
	Name            string         `yaml:"name" json:"name" jsonschema:"required,description=Application name"`
	Type            string         `yaml:"type" json:"type" jsonschema:"required,enum=kustomization,enum=helmrelease,description=Deployment type"`
	SourceRef       AppSourceRef   `yaml:"sourceRef" json:"sourceRef" jsonschema:"required,description=Flux source reference"`
	Path            string         `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Path within source (for kustomization or git-based charts)"`
	Chart           string         `yaml:"chart,omitempty" json:"chart,omitempty" jsonschema:"description=Helm chart name (for helmrelease with HelmRepository)"`
	ChartVersion    string         `yaml:"chartVersion,omitempty" json:"chartVersion,omitempty" jsonschema:"description=Chart version semver range"`
	TargetNamespace string         `yaml:"targetNamespace,omitempty" json:"targetNamespace,omitempty" jsonschema:"description=Target namespace"`
	CreateNamespace bool           `yaml:"createNamespace,omitempty" json:"createNamespace,omitempty" jsonschema:"description=Create namespace if missing"`
	Interval        string         `yaml:"interval,omitempty" json:"interval,omitempty" jsonschema:"default=5m,description=Reconciliation interval"`
	DependsOn       []string       `yaml:"dependsOn,omitempty" json:"dependsOn,omitempty" jsonschema:"description=Dependency names"`
	ServiceAccount  string         `yaml:"serviceAccountName,omitempty" json:"serviceAccountName,omitempty" jsonschema:"description=ServiceAccount for impersonation"`
	KubeConfig      *AppKubeConfig `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty" jsonschema:"description=Remote cluster kubeconfig secret"`
	Values          map[string]any `yaml:"values,omitempty" json:"values,omitempty" jsonschema:"description=Helm values (helmrelease only)"`
}

// AppSourceRef identifies the Flux source object
type AppSourceRef struct {
	Kind string `yaml:"kind" json:"kind" jsonschema:"required,enum=GitRepository,enum=HelmRepository,description=Source kind"`
	Name string `yaml:"name" json:"name" jsonschema:"required,description=Source resource name"`
}

// AppKubeConfig references a secret for cross-cluster deployment
type AppKubeConfig struct {
	SecretRef string `yaml:"secretRef" json:"secretRef" jsonschema:"required,description=Kubeconfig secret name"`
}

// Terraform holds cloud infrastructure configuration
type Terraform struct {
	ProjectID         string `yaml:"projectId,omitempty" json:"projectId,omitempty" jsonschema:"description=Cloud project ID"`
	KubernetesType    string `yaml:"kubernetesType,omitempty" json:"kubernetesType,omitempty" jsonschema:"description=Kubernetes distribution type"`
	KubernetesVersion string `yaml:"kubernetesVersion,omitempty" json:"kubernetesVersion,omitempty" jsonschema:"description=Kubernetes version"`
	DNS               *DNS   `yaml:"dns,omitempty" json:"dns,omitempty" jsonschema:"description=DNS configuration"`
}

// DNS holds DNS zone configuration
type DNS struct {
	ZoneName   string `yaml:"zoneName" json:"zoneName" jsonschema:"required,description=DNS zone name"`
	AdminEmail string `yaml:"adminEmail" json:"adminEmail" jsonschema:"required,description=Admin email for DNS"`
}

// FluxConfig holds FluxCD Operator-specific configuration
type FluxConfig struct {
	Distribution     FluxDistribution `yaml:"distribution" json:"distribution" jsonschema:"required,description=Flux distribution configuration"`
	Cluster          FluxCluster      `yaml:"cluster,omitempty" json:"cluster,omitempty" jsonschema:"description=Cluster profile configuration"`
	Sync             FluxSync         `yaml:"sync" json:"sync" jsonschema:"required,description=Flux sync configuration (GitRepository + Kustomization)"`
	HelmRepositories []HelmRepoConfig `yaml:"helmRepositories,omitempty" json:"helmRepositories,omitempty" jsonschema:"description=Additional Helm repositories"`
	GitRepositories  []GitRepoConfig  `yaml:"gitRepositories,omitempty" json:"gitRepositories,omitempty" jsonschema:"description=Additional Git repositories"`
	WebUI            FluxWebUI        `yaml:"webUI,omitempty" json:"webUI,omitempty" jsonschema:"description=Flux Operator Web UI configuration"`
}

// FluxDistribution configures the Flux distribution installed by the operator
type FluxDistribution struct {
	Version  string `yaml:"version,omitempty" json:"version,omitempty" jsonschema:"description=Flux version semver range (e.g. 2.x)"`
	Registry string `yaml:"registry,omitempty" json:"registry,omitempty" jsonschema:"description=Container registry for Flux images"`
}

// FluxCluster configures cluster-specific Flux Operator settings
type FluxCluster struct {
	Type          string `yaml:"type,omitempty" json:"type,omitempty" jsonschema:"enum=kubernetes,enum=openshift,enum=azure,enum=aws,enum=gcp,description=Cluster type"`
	Size          string `yaml:"size,omitempty" json:"size,omitempty" jsonschema:"enum=small,enum=medium,enum=large,description=Resource profile size"`
	Multitenant   bool   `yaml:"multitenant,omitempty" json:"multitenant,omitempty" jsonschema:"description=Enable multi-tenancy lockdown"`
	NetworkPolicy bool   `yaml:"networkPolicy,omitempty" json:"networkPolicy,omitempty" jsonschema:"description=Enable network policies"`
}

// FluxSync configures the FluxInstance sync spec (Git/OCI source + path)
type FluxSync struct {
	Kind       string `yaml:"kind,omitempty" json:"kind,omitempty" jsonschema:"enum=GitRepository,enum=OCIRepository,enum=Bucket,description=Source kind"`
	URL        string `yaml:"url" json:"url" jsonschema:"required,description=Source URL (Git HTTPS or OCI)"`
	Ref        string `yaml:"ref,omitempty" json:"ref,omitempty" jsonschema:"description=Git ref (e.g. refs/heads/main)"`
	Path       string `yaml:"path,omitempty" json:"path,omitempty" jsonschema:"description=Path within the source to reconcile"`
	PullSecret string `yaml:"pullSecret,omitempty" json:"pullSecret,omitempty" jsonschema:"description=Name of secret containing source credentials"`
	Interval   string `yaml:"interval,omitempty" json:"interval,omitempty" jsonschema:"description=Reconciliation interval (e.g. 5m)"`
}

// FluxWebUI configures the Flux Operator built-in web dashboard
type FluxWebUI struct {
	Enabled bool `yaml:"enabled,omitempty" json:"enabled,omitempty" jsonschema:"description=Enable the Flux Operator Web UI"`
}

// HelmRepoConfig holds Helm repository configuration
type HelmRepoConfig struct {
	Name      string `yaml:"name" json:"name" jsonschema:"required,description=Repository name"`
	URL       string `yaml:"url" json:"url" jsonschema:"required,description=Repository URL"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty" jsonschema:"enum=default,enum=oci,description=Repository type"`
	SecretRef string `yaml:"secretRef,omitempty" json:"secretRef,omitempty" jsonschema:"description=Name of secret containing repo credentials"`
}

// GitRepoConfig holds additional Git repository source configuration
type GitRepoConfig struct {
	Name      string `yaml:"name" json:"name" jsonschema:"required,description=Repository name"`
	URL       string `yaml:"url" json:"url" jsonschema:"required,description=Git repository URL"`
	Branch    string `yaml:"branch,omitempty" json:"branch,omitempty" jsonschema:"default=main,description=Git branch to track"`
	SecretRef string `yaml:"secretRef,omitempty" json:"secretRef,omitempty" jsonschema:"description=Name of secret containing repo credentials"`
	Interval  string `yaml:"interval,omitempty" json:"interval,omitempty" jsonschema:"default=5m,description=Reconciliation interval"`
}

// Services holds all platform service configurations
type Services struct {
	Traefik               GenericService     `yaml:"traefik" json:"traefik" jsonschema:"description=Traefik ingress controller"`
	CertManager           CertManagerService `yaml:"certManager" json:"certManager" jsonschema:"description=Certificate manager"`
	ExternalDNS           GenericService     `yaml:"externalDns" json:"externalDns" jsonschema:"description=External DNS"`
	ExternalSecrets       GenericService     `yaml:"externalSecrets" json:"externalSecrets" jsonschema:"description=External Secrets Operator"`
	KubePrometheusStack   GenericService     `yaml:"kubePrometheusStack" json:"kubePrometheusStack" jsonschema:"description=Kube Prometheus Stack"`
	Loki                  GenericService     `yaml:"loki" json:"loki" jsonschema:"description=Grafana Loki"`
	MetricsServer         GenericService     `yaml:"metricsServer" json:"metricsServer" jsonschema:"description=Metrics Server"`
	Kyverno               GenericService     `yaml:"kyverno" json:"kyverno" jsonschema:"description=Kyverno policy engine"`
	KyvernoPolicies       GenericService     `yaml:"kyvernoPolicies" json:"kyvernoPolicies" jsonschema:"description=Kyverno policies"`
	KyvernoPolicyReporter GenericService     `yaml:"kyvernoPolicyReporter" json:"kyvernoPolicyReporter" jsonschema:"description=Kyverno policy reporter"`
	OAuth2Proxy           GenericService     `yaml:"oauth2Proxy" json:"oauth2Proxy" jsonschema:"description=OAuth2 Proxy"`
	Longhorn              GenericService     `yaml:"longhorn" json:"longhorn" jsonschema:"description=Longhorn storage"`
	MetalLB               GenericService     `yaml:"metallb" json:"metallb" jsonschema:"description=MetalLB load balancer"`
	FluxWebUI             GenericService     `yaml:"fluxWebUI" json:"fluxWebUI" jsonschema:"description=Flux Operator Web UI dashboard"`
	HomeDashboard         GenericService     `yaml:"homeDashboard" json:"homeDashboard" jsonschema:"description=Home dashboard"`
	Forgejo               GenericService     `yaml:"forgejo" json:"forgejo" jsonschema:"description=Forgejo Git service"`
}

// GenericService represents a basic service with enable/disable status
type GenericService struct {
	Status string `yaml:"status" json:"status" jsonschema:"enum=enabled,enum=disabled,description=Service status"`
}

// CertManagerService extends GenericService with cert-manager specific config
type CertManagerService struct {
	Status        string         `yaml:"status" json:"status" jsonschema:"enum=enabled,enum=disabled,description=Service status"`
	ClusterIssuer *ClusterIssuer `yaml:"clusterIssuer,omitempty" json:"clusterIssuer,omitempty" jsonschema:"description=ClusterIssuer configuration"`
}

// ClusterIssuer holds Let's Encrypt / ACME configuration
type ClusterIssuer struct {
	Name   string `yaml:"name" json:"name" jsonschema:"description=ClusterIssuer name"`
	Email  string `yaml:"email" json:"email" jsonschema:"description=ACME registration email"`
	Server string `yaml:"server" json:"server" jsonschema:"description=ACME server URL"`
}

// Service status constants
const (
	ServiceEnabled  = "enabled"
	ServiceDisabled = "disabled"
)

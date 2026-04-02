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

// FluxConfig holds FluxCD-specific configuration
type FluxConfig struct {
	GitRepository    GitRepoConfig    `yaml:"gitRepository" json:"gitRepository" jsonschema:"required,description=Git repository for FluxCD sources"`
	HelmRepositories []HelmRepoConfig `yaml:"helmRepositories,omitempty" json:"helmRepositories,omitempty" jsonschema:"description=Additional Helm repositories"`
	Interval         string           `yaml:"interval,omitempty" json:"interval,omitempty" jsonschema:"description=Default reconciliation interval (e.g. 5m)"`
}

// GitRepoConfig holds Git repository configuration for FluxCD
type GitRepoConfig struct {
	URL            string `yaml:"url" json:"url" jsonschema:"required,description=Git repository URL"`
	Branch         string `yaml:"branch,omitempty" json:"branch,omitempty" jsonschema:"description=Git branch to track"`
	TargetRevision string `yaml:"targetRevision,omitempty" json:"targetRevision,omitempty" jsonschema:"description=Target revision (tag/commit)"`
	SecretRef      string `yaml:"secretRef,omitempty" json:"secretRef,omitempty" jsonschema:"description=Name of secret containing Git credentials"`
}

// HelmRepoConfig holds Helm repository configuration
type HelmRepoConfig struct {
	Name      string `yaml:"name" json:"name" jsonschema:"required,description=Repository name"`
	URL       string `yaml:"url" json:"url" jsonschema:"required,description=Repository URL"`
	Type      string `yaml:"type,omitempty" json:"type,omitempty" jsonschema:"enum=default,enum=oci,description=Repository type"`
	SecretRef string `yaml:"secretRef,omitempty" json:"secretRef,omitempty" jsonschema:"description=Name of secret containing repo credentials"`
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
	WeaveGitops           GenericService     `yaml:"weaveGitops" json:"weaveGitops" jsonschema:"description=Weave GitOps UI (FluxCD dashboard)"`
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

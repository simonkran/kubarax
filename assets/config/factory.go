package config

import (
	"kubarax/assets/envmap"
)

// NewClusterFromEnv creates a new Cluster configuration from environment variables
func NewClusterFromEnv(env *envmap.EnvMap) Cluster {
	return Cluster{
		Name:    env.ProjectName,
		Stage:   env.ProjectStage,
		Type:    "controlplane",
		DNSName: env.DomainName,
		FluxCD: FluxConfig{
			Sync: FluxSync{
				URL:  env.FluxGitHTTPSUrl,
				Ref:  "refs/heads/main",
				Path: "clusters/" + env.ProjectName,
			},
			WebUI: FluxWebUI{
				Enabled: true,
			},
		},
		Services: DefaultServices(),
	}
}

// DefaultServices returns a Services config with sensible defaults
func DefaultServices() Services {
	return Services{
		Traefik:               GenericService{Status: ServiceEnabled},
		CertManager:           CertManagerService{Status: ServiceEnabled},
		ExternalDNS:           GenericService{Status: ServiceEnabled},
		ExternalSecrets:       GenericService{Status: ServiceEnabled},
		KubePrometheusStack:   GenericService{Status: ServiceEnabled},
		Loki:                  GenericService{Status: ServiceEnabled},
		MetricsServer:         GenericService{Status: ServiceEnabled},
		Kyverno:               GenericService{Status: ServiceDisabled},
		KyvernoPolicies:       GenericService{Status: ServiceDisabled},
		KyvernoPolicyReporter: GenericService{Status: ServiceDisabled},
		OAuth2Proxy:           GenericService{Status: ServiceDisabled},
		Longhorn:              GenericService{Status: ServiceDisabled},
		MetalLB:               GenericService{Status: ServiceDisabled},
		HomerDashboard:         GenericService{Status: ServiceEnabled},
		Forgejo:               GenericService{Status: ServiceDisabled},
	}
}

// CreateOrUpdateClusterFromEnv updates the first cluster in config with env values
func CreateOrUpdateClusterFromEnv(cfg *Config, env *envmap.EnvMap) {
	if len(cfg.Clusters) == 0 {
		cluster := NewClusterFromEnv(env)
		cfg.Clusters = append(cfg.Clusters, cluster)
		return
	}

	c := &cfg.Clusters[0]
	if env.ProjectName != "" && env.ProjectName != "<...>" {
		c.Name = env.ProjectName
	}
	if env.ProjectStage != "" && env.ProjectStage != "<...>" {
		c.Stage = env.ProjectStage
	}
	if env.DomainName != "" && env.DomainName != "<...>" {
		c.DNSName = env.DomainName
	}
	if env.FluxGitHTTPSUrl != "" && env.FluxGitHTTPSUrl != "<...>" {
		c.FluxCD.Sync.URL = env.FluxGitHTTPSUrl
	}
}

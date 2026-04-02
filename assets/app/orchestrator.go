package app

import (
	"kubarax/assets/config"
	"kubarax/assets/envmap"
)

// CreateOrUpdateClusterFromEnv delegates to config package
func CreateOrUpdateClusterFromEnv(cfg *config.Config, env *envmap.EnvMap) {
	config.CreateOrUpdateClusterFromEnv(cfg, env)
}

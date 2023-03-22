package aggregator

type obj = map[string]interface{}

// ApplicationConfig describes the cofiguration in an application namespace
type ApplicationConfig struct {
	Service        string `json:"service"`
	MetricsConfigs []obj  `json:"metrics_configs"`
	UnifiedConfigs []obj  `json:"unified_configs"`
}

// GlobalConfig describe the global configuration in the centralized namespace
type GlobalConfig struct {
	Configs []struct {
		Namespace string `json:"namespace"`
		ApplicationConfig
	} `json:"configs"`
}

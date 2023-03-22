package aggregator

type obj = map[string]interface{}
type ApplicationConfig = obj

// GlobalConfig describe the global configuration in the centralized namespace
type GlobalConfig struct {
	Configs []struct {
		Namespace string `json:"namespace"`
		ApplicationConfig
	} `json:"configs"`
}

package aggregator

const (
	Namespace = "namespace"
)

type obj = map[string]interface{}

// GlobalConfig describe the global configuration in the centralized namespace
type GlobalConfig struct {
	Configs []obj `json:"configs"`
}

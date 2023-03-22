package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

var (
	applicationConfigStr = `metric_configs:
- composite_keys:
  - ck11
  - ck12
  metric: m1
  static_threshold: 1
- composite_keys:
  - ck21
  - ck22
  metric: m2
  static_threshold: 2
service: test
unified_configs:
- unified_metric_name: umn1
  unified_metrics:
  - um11
  - um12
- unified_metric_name: umn2
  unified_metrics:
  - um21
  - um22
`
)

func fakeApplicationConfig(t *testing.T) ApplicationConfig {
	t.Helper()
	return ApplicationConfig{
		"service": "test",
		"metric_configs": []obj{
			{
				"metric":           "m1",
				"composite_keys":   []string{"ck11", "ck12"},
				"static_threshold": 1,
			},
			{
				"metric":           "m2",
				"composite_keys":   []string{"ck21", "ck22"},
				"static_threshold": 2,
			},
		},
		"unified_configs": []obj{
			{
				"unified_metric_name": "umn1",
				"unified_metrics":     []string{"um11", "um12"},
			},
			{
				"unified_metric_name": "umn2",
				"unified_metrics":     []string{"um21", "um22"},
			},
		},
	}
}

func fakeGlobalConfig(t *testing.T) GlobalConfig {
	return GlobalConfig{
		Configs: []struct {
			Namespace string `json:"namespace"`
			ApplicationConfig
		}{
			{Namespace: "ns1", ApplicationConfig: fakeApplicationConfig(t)},
		},
	}
}

func Test_ApplicationConfig(t *testing.T) {
	fakeAppConfig := fakeApplicationConfig(t)
	yamlBytes, err := yaml.Marshal(&fakeAppConfig)
	assert.NoError(t, err)
	assert.Equal(t, applicationConfigStr, string(yamlBytes))
	a1 := ApplicationConfig{}
	err = yaml.Unmarshal(yamlBytes, &a1)
	assert.NoError(t, err)
}

func Test_GlobalConfig(t *testing.T) {
	fakeConfig := fakeGlobalConfig(t)
	yamlBytes, err := yaml.Marshal(&fakeConfig)
	assert.NoError(t, err)
	g1 := GlobalConfig{}
	err = yaml.Unmarshal(yamlBytes, &g1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(g1.Configs))
	assert.Equal(t, "ns1", g1.Configs[0].Namespace)
	mc, ok := g1.Configs[0].ApplicationConfig["metric_configs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(mc))
	ck, ok := mc[0].(obj)
	assert.True(t, ok)
	l, ok := ck["composite_keys"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(l))
}

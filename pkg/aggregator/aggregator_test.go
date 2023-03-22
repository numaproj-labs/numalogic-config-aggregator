package aggregator

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func Test_NewAggregator(t *testing.T) {
	k8sCli := k8sfake.NewSimpleClientset()

	t.Run("default", func(t *testing.T) {
		path, err := os.Getwd()
		assert.NoError(t, err)
		a := NewAggregator(k8sCli, "ns", "cm", WithSchemaFileDir(path+"/../../manifests/install/base"))
		assert.NotNil(t, a)
		assert.Equal(t, defaultSettings.interval, a.interval)
		assert.Equal(t, defaultSettings.configMapKey, a.configMapKey)
		assert.Equal(t, defaultSettings.appConfigMapLabel, a.appConfigLabel)
	})

	t.Run("customized", func(t *testing.T) {
		path, err := os.Getwd()
		assert.NoError(t, err)
		a := NewAggregator(k8sCli, "ns", "cm", WithInterval(time.Second*100), WithAppConfigLabel("a=b"), WithConfigMapKey("a.yaml"), WithSchemaFileDir(path+"/../../manifests/install/base"))
		assert.NotNil(t, a)
		assert.Equal(t, time.Second*100, a.interval)
		assert.Equal(t, "a.yaml", a.configMapKey)
		assert.Equal(t, "a=b", a.appConfigLabel)
	})
}

func Test_runOnce(t *testing.T) {
	namespace := "test-ns"
	cm := "test-cm"
	cm1 := fakeAppConfigMap(t, "ns1", "n1")
	cm2 := fakeAppConfigMap(t, "ns2", "n2")
	k8sCli := k8sfake.NewSimpleClientset()
	_, _ = k8sCli.CoreV1().ConfigMaps("ns1").Create(context.Background(), cm1, metav1.CreateOptions{})
	_, _ = k8sCli.CoreV1().ConfigMaps("ns2").Create(context.Background(), cm2, metav1.CreateOptions{})
	path, err := os.Getwd()
	assert.NoError(t, err)
	a := NewAggregator(k8sCli, namespace, cm, WithSchemaFileDir(path+"/../../manifests/install/base"))
	err = a.runOnce(context.Background())
	assert.NoError(t, err)
	configMap, err := k8sCli.CoreV1().ConfigMaps(namespace).Get(context.Background(), cm, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(configMap.Data))
	conf, existing := configMap.Data[defaultSettings.configMapKey]
	assert.True(t, existing)
	assert.NotEmpty(t, conf)
	var c GlobalConfig
	err = yaml.Unmarshal([]byte(conf), &c)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(c.Configs))
	assert.Equal(t, "ns1", c.Configs[0][Namespace])
	assert.Equal(t, "ns2", c.Configs[1][Namespace])
	assert.Equal(t, 4, len(c.Configs[0]))
	assert.Equal(t, 4, len(c.Configs[1]))
	mc0, ok := c.Configs[0]["metric_configs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(mc0))
	mc1, ok := c.Configs[1]["metric_configs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(mc1))
	uc0, ok := c.Configs[0]["unified_configs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(uc0))
	uc1, ok := c.Configs[1]["unified_configs"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(uc1))
}

func fakeAppConfigMap(t *testing.T, ns, name string) *corev1.ConfigMap {
	t.Helper()
	l, _ := labels.ConvertSelectorToLabelsMap(defaultSettings.appConfigMapLabel)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
			Labels:    l,
		},
		Data: map[string]string{
			"hello": applicationConfigStr,
		},
	}
}

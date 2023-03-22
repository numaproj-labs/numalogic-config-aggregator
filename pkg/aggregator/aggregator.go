package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/numaproj-labs/numalogic-config-aggregator/pkg/logging"
)

var defaultSettings struct {
	interval          time.Duration
	configMapKey      string
	appConfigMapLabel string
	schemaFileDir     string
}

func init() {
	defaultSettings.interval = time.Second * 180
	defaultSettings.configMapKey = "config.yaml"
	defaultSettings.appConfigMapLabel = "numaprom.numaproj.io/component=argo-rollouts"
	defaultSettings.schemaFileDir = "/etc/config/config-aggregator"
}

type aggregator struct {
	k8sclient kubernetes.Interface
	// The namespace of the centralized configuration is located
	namespace string
	// Centralized config map name
	configMap string
	// The key of the config in the centralized config map
	configMapKey string
	// The label of the config in application namespace
	appConfigLabel string
	// The dir of the schema.json file for validation
	schemaFileDir string
	// Interval of each run
	interval time.Duration
	logger   *zap.SugaredLogger

	schemaLoader gojsonschema.JSONLoader
}

// NewAggregator returns an aggregator instance
func NewAggregator(k8sclient kubernetes.Interface, namespace, configMap string, opts ...Option) *aggregator {
	a := &aggregator{
		k8sclient:      k8sclient,
		namespace:      namespace,
		configMap:      configMap,
		configMapKey:   defaultSettings.configMapKey,
		interval:       defaultSettings.interval,
		appConfigLabel: defaultSettings.appConfigMapLabel,
		schemaFileDir:  defaultSettings.schemaFileDir,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}
	if a.logger == nil {
		a.logger = logging.NewLogger()
	}
	a.loadConfig()
	return a
}

// Auto reload schema.json
func (a *aggregator) loadConfig() {
	v := viper.New()
	v.SetConfigName("schema")
	v.SetConfigType("json")
	v.AddConfigPath(a.schemaFileDir)
	v.WatchConfig()
	f := fmt.Sprintf("file://%s/schema.json", a.schemaFileDir)
	v.OnConfigChange(func(e fsnotify.Event) {
		a.schemaLoader = gojsonschema.NewReferenceLoader(f)
	})
	a.schemaLoader = gojsonschema.NewReferenceLoader(f)
}

// Run starts an infinite for loop to aggregate the config from applications namespaces,
// it accepts a cancellable context as a parameter.
func (a *aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := a.runOnce(ctx)
			if err != nil {
				a.logger.Error(err)
			}
		case <-ctx.Done():
			a.logger.Info("Shutting down...")
			return
		}
	}
}

func (a *aggregator) runOnce(ctx context.Context) error {
	cmList, err := a.k8sclient.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{LabelSelector: a.appConfigLabel})
	if err != nil {
		return fmt.Errorf("failed to list configmaps, %w", err)
	}
	config := GlobalConfig{
		Configs: []obj{},
	}
	for _, cm := range cmList.Items {
		for key, data := range cm.Data { // Iterate all the key/value pairs in the configmap
			// TODO: merge multiple entries in one ConfigMap.
			// TODO: merge entries from multiple ConfinMaps in one namespace.
			appConfig, err := a.convert(data)
			if err != nil {
				a.logger.Errorw("Invalid application config", zap.String("namespace", cm.Namespace), zap.String("configmap", cm.Name), zap.String("configmapKey", key), zap.Error(err))
				continue
			}
			if len(appConfig) == 0 {
				a.logger.Warnw("Empty application config", zap.String("namespace", cm.Namespace), zap.String("configmap", cm.Name), zap.String("configmapKey", key))
				continue
			}
			appConfig[Namespace] = cm.Namespace
			config.Configs = append(config.Configs, appConfig)
		}
	}
	configBytes, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration, %w", err)
	}
	cm, err := a.k8sclient.CoreV1().ConfigMaps(a.namespace).Get(ctx, a.configMap, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: a.namespace,
					Name:      a.configMap,
					Annotations: map[string]string{
						"app.kubernetes.io/managed-by": "numalogic-config-aggregator",
					},
				},
				Data: map[string]string{
					a.configMapKey: string(configBytes),
				},
			}
			if _, err := a.k8sclient.CoreV1().ConfigMaps(a.namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
				return fmt.Errorf("failed to create aggregated configmap, %w", err)
			}
		} else {
			return fmt.Errorf("failed to get aggregated configmap, %w", err)
		}
	}
	if string(configBytes) != cm.Data[a.configMapKey] {
		cm.Data[a.configMapKey] = string(configBytes)
		if _, err := a.k8sclient.CoreV1().ConfigMaps(a.namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update aggregated configmap, %w", err)
		} else {
			a.logger.Info("Config changes saved successfully.")
		}
	} else {
		a.logger.Info("No config changes.")
	}
	return nil
}

// Validate the user configured YAML string, and convert to an object
func (a *aggregator) convert(config string) (obj, error) {
	// Validation
	jsonBytes, err := yaml.YAMLToJSON([]byte(config))
	if err != nil {
		return nil, fmt.Errorf("invalid config, %w", err)
	}
	jsonLoader := gojsonschema.NewBytesLoader(jsonBytes)
	result, err := gojsonschema.Validate(a.schemaLoader, jsonLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to validate application config, %w", err)
	}
	if !result.Valid() {
		return nil, fmt.Errorf("invalid application config, %v", result.Errors())
	}
	var appConfig obj
	if err := yaml.Unmarshal([]byte(config), &appConfig); err != nil {
		return nil, fmt.Errorf("failed to marshal application config, %w", err)
	}
	return appConfig, nil
}

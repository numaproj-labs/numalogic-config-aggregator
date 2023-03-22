package main

import (
	"context"
	"flag"
	"os"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/numaproj-labs/numalogic-config-aggregator/pkg/aggregator"
	"github.com/numaproj-labs/numalogic-config-aggregator/pkg/leaderelection"
	"github.com/numaproj-labs/numalogic-config-aggregator/pkg/logging"
)

func main() {
	logger := logging.NewLogger()

	var (
		configMapName  string
		configMapKey   string
		appConfigLabel string
		interval       time.Duration
	)

	flag.StringVar(&configMapName, "configmap-name", "", "Aggregated ConfigMap name")
	flag.StringVar(&configMapKey, "configmap-key", "", "Key of the aggregated ConfigMap name")
	flag.StringVar(&appConfigLabel, "app-config-label", "", "Label of the ConfigMap in the application namespaces")
	flag.DurationVar(&interval, "interval", time.Second*180, "Interval of each run")
	flag.Parse()

	if configMapName == "" {
		logger.Fatal("The name of the centralized ConfigMap is missing.")
	}

	namespace, existing := os.LookupEnv("NAMESPACE")
	if !existing {
		logger.Fatal("Required environment variable \"NAMESPACE\" is missing")
	}

	hostname, existing := os.LookupEnv("POD_NAME")
	if !existing {
		logger.Fatal("Required environment variable \"POD_NAME\" is missing")
	}

	config, err := getClientConfig()
	if err != nil {
		logger.Fatalw("Failed to retrieve kubernetes config", zap.Error(err))
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatalw("Failed to create kubernetes client", zap.Error(err))
	}

	opts := []aggregator.Option{aggregator.WithInterval(interval), aggregator.WithLogger(logger)}
	if appConfigLabel != "" {
		opts = append(opts, aggregator.WithAppConfigLabel(appConfigLabel))
	}
	if configMapKey != "" {
		opts = append(opts, aggregator.WithConfigMapKey(configMapKey))
	}
	a := aggregator.NewAggregator(client, namespace, configMapName, opts...)
	elector := leaderelection.NewK8sLeaderElector(client, namespace, "numalogic-config-aggregator-lock", hostname)
	ctx := ctrl.SetupSignalHandler()
	elector.RunOrDie(ctx, leaderelection.LeaderCallbacks{
		OnStartedLeading: func(_ context.Context) {
			a.Run(ctx)
		},
		OnStoppedLeading: func() {
			logger.Fatalf("Leader lost: %s", hostname)
		},
	})
}

func getClientConfig() (*rest.Config, error) {
	kubeconfig, _ := os.LookupEnv("KUBECONFIG")
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

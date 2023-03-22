# Numalogic Configuration Aggregator

This is a configuration aggregator for Numalogic, it aggregates the configuration from application namespaces, and saves to a ConfigMap in a centralized namespace.

## Why and How

A Numalogic pipeline needs to load an aggregated configuration from multiple applicatiohn namespaces for inferences. The configuration distributed in application namespaces use `ConfigMap`. The key of the config data in the `ConfigMap` could be any string, but the config value needs to follow a format. A valid configuration looks like below.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  labels:
    numaprom.numaproj.io/component: argo-rollouts # Required
  name: my-app-config
  namespace: my-namespace
data:
  config: |
    service: test1
    metric_configs:
      - metric: "rollout_error_rate"
        composite_keys: ["namespace", "name", "hash_id"]
        static_threshold: 3
      - metric: "rollout_latency"
        composite_keys: ["namespace", "name", "hash_id"]
        static_threshold: 3
    unified_configs:
      - unified_metric_name: "unified_anomaly"
        unified_metrics: ["rollout_error_rate", "rollout_latency"]
```

**Notes**

- There's no requirement on the ConfigMap name and the key of the data, they could be any string as long as it's valid for a ConfigMap.
- Questions:
  - Should we allow multiple ConfigMaps in one application namespace?
  - Should we allow multiple entries in one ConfigMap?
  - If allowed, should we merge them into one entry when doing aggregation?
- A label `numaprom.numaproj.io/component: argo-rollouts` is requried for the application ConfigMap.

The aggregated configuration is located in the same namespace of the inference pipeline, for example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: numalogic-argorollouts-config
  namespace: numalogic-argorollouts
data:
  config.yaml: |
    configs:
    - namespace: "sandbox_numalogic_demo1"
      service: test1
      metric_configs:
        - metric: "rollout_error_rate"
          composite_keys: [ "namespace", "name", "hash_id" ]
          static_threshold: 3
        - metric: "rollout_latency"
          composite_keys: [ "namespace", "name", "hash_id" ]
          static_threshold: 3
      unified_configs:
        - unified_metric_name: "unified_anomaly"
          unified_metrics: ["rollout_error_rate", "rollout_latency"]
    - namespace: "sandbox_numalogic_demo2"
      service: test2
      metric_configs:
        - metric: "rollout_error_rate"
          composite_keys: [ "namespace", "name", "hash_id" ]
          static_threshold: 3
        - metric: "rollout_latency"
          composite_keys: [ "namespace", "name", "hash_id" ]
          static_threshold: 3
      unified_configs:
        - unified_metric_name: "unified_anomaly"
          unified_metrics: [ "rollout_error_rate", "rollout_latency" ]
```

The configuration aggregator is deployed in the Numalogic inference pipeline namespace, periodically lists all the ConfigMaps with the same label in the Kubernetes cluster, validates the configuration, and saves to the aggregated ConfigMap.

## Deployment

The deployment manifests are defined in [manifests/install](manifests/install), after making changes to the manifests, remember to run `make manifests` to make sure there's no error and regenerate [install.yaml](manifests/install.yaml).

### Configuration

The deployment spec accepts following arguments.

- `--configmap-name`

  (Required) The name of the aggregated ConfigMap.

- `--configmap-key`

  (Optional) The `key` of the aggregated ConfigMap, defaults to `config.yaml`.

- `--app-config-label`

  (Optional) The label of the ConfigMap in the application namespace, defaults to `numaprom.numaproj.io/component: argo-rollouts`.

- `interval`

  (Optional) The interval of the periodical job, accepts format like `30s`, `2m10s`, defaults to `180s`.

`Active-Passive` HA is available for the deployment by default, which means multiple replica for hight availablity is supported.

## Application Configuration Validation

The application configuration is supposed to be in YAML format, a `schema.json` is used for validation. The `schema.json` is stored in a ConfigMap named `application-config-schema`. Don't forget to overwrite it with the real schema for deployment.

```yaml
apiVersion: v1
data:
  schema.json: |+
    {
      "$schema": "https://json-schema.org/draft/2020-12/schema",
      "title": "Numalogic application configuration",
      "type": "object",
      "properties": {
        "service": {
          "type": "string"
        },
        "metrics_configs": {
          "type": "array",
          "items": {
            "type": "object"
          }
        },
        "unified_configs": {
          "type": "array",
          "items": {
            "type": "object"
          }
        }
      },
      "required": [
        "service",
        "metrics_configs",
        "unified_configs"
      ]
    }

kind: ConfigMap
metadata:
  name: application-config-schema
```

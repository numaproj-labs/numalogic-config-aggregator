package aggregator

import (
	"time"

	"go.uber.org/zap"
)

type Option func(*aggregator)

// WithInterval sets the interval of each run of checking configurations.
func WithInterval(f time.Duration) Option {
	return func(o *aggregator) {
		o.interval = f
	}
}

// WithLogger sets the logger to be used.
func WithLogger(l *zap.SugaredLogger) Option {
	return func(o *aggregator) {
		o.logger = l
	}
}

// WithAppConfigLabel sets the config label in the application namespace.
func WithAppConfigLabel(l string) Option {
	return func(o *aggregator) {
		o.appConfigLabel = l
	}
}

// WithConfigMapKey sets the key of the aggregated config map.
func WithConfigMapKey(k string) Option {
	return func(o *aggregator) {
		o.configMapKey = k
	}
}

// WithSchemaFileDir sets the dir of the json-schema file for validation.
func WithSchemaFileDir(p string) Option {
	return func(o *aggregator) {
		o.schemaFileDir = p
	}
}

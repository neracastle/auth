package config

// Trace настройки трейсинга
type Trace struct {
	BatchTimeout      int    `yaml:"batch_timeout" env:"TRACE_BATCH_TIMEOUT" env-default:"1"`
	JaegerGRPCAddress string `yaml:"jaeger_grpc_address" env:"TRACE_JAEGER_GRPC_ADDRESS" env-default:"localhost:4317"`
}

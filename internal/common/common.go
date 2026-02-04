package common

type TestConfig struct {
	AutoScalingSupported         bool
	BastionSupported             bool
	TeleportSupported            bool
	ExternalDnsSupported         bool
	ControlPlaneMetricsSupported bool
	MinimalCluster               bool
}

func NewTestConfigWithDefaults() *TestConfig {
	return &TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
		MinimalCluster:               false,
	}
}

func Run(cfg *TestConfig) {
	RunApps(cfg.MinimalCluster)
	runBasic()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runMetrics(cfg.ControlPlaneMetricsSupported)
	runTeleport(cfg.TeleportSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

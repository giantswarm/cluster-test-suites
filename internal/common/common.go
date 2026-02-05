package common

type TestConfig struct {
	AutoScalingSupported         bool
	BastionSupported             bool
	TeleportSupported            bool
	ExternalDnsSupported         bool
	ControlPlaneMetricsSupported bool
	ObservabilityBundleInstalled bool
	SecurityBundleInstalled      bool
}

func NewTestConfigWithDefaults() *TestConfig {
	return &TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		ControlPlaneMetricsSupported: true,
		ObservabilityBundleInstalled: true,
		SecurityBundleInstalled:      true,
	}
}

func Run(cfg *TestConfig) {
	RunApps(cfg)
	runBasic()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runMetrics(cfg.ControlPlaneMetricsSupported)
	runTeleport(cfg.TeleportSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

package common

type TestConfig struct {
	AutoScalingSupported         bool
	BastionSupported             bool
	TeleportSupported            bool
	ExternalDnsSupported         bool
	CertManagerSupported         bool
	ControlPlaneMetricsSupported bool
	ObservabilityBundleInstalled bool
	SecurityBundleInstalled      bool
	GatewayAPISupported          bool
	ARMNodePoolEnabled           bool
}

func NewTestConfigWithDefaults() *TestConfig {
	return &TestConfig{
		AutoScalingSupported:         true,
		BastionSupported:             false,
		TeleportSupported:            true,
		ExternalDnsSupported:         true,
		CertManagerSupported:         true,
		ControlPlaneMetricsSupported: true,
		ObservabilityBundleInstalled: true,
		SecurityBundleInstalled:      true,
		GatewayAPISupported:          true,
		ARMNodePoolEnabled:           false,
	}
}

func Run(cfg *TestConfig) {
	RunApps(cfg)
	runBasic(cfg)
	runCertManager(cfg.CertManagerSupported)
	runDNS(cfg.BastionSupported)
	runMetrics(cfg)
	runTeleport(cfg.TeleportSupported)
	runHelloWorldGateway(cfg.GatewayAPISupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

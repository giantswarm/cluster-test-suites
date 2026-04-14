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
	IngressNginxSupported        bool
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
		IngressNginxSupported:        false,
	}
}

func Run(cfg *TestConfig) {
	RunApps(cfg)
	runBasic()
	runCertManager(cfg.CertManagerSupported)
	runDNS(cfg.BastionSupported)
	runMetrics(cfg)
	runTeleport(cfg.TeleportSupported)
	runHelloWorld(cfg.ExternalDnsSupported && cfg.IngressNginxSupported)
	runHelloWorldGateway(cfg.GatewayAPISupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

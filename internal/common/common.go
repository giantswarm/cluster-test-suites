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
	// APIServerDNSRecordSupported indicates whether our DNS controllers set up
	// an A record for the Kubernetes API endpoint. Managed control planes (e.g.
	// AKS) provide their own API endpoint, so no such record is created.
	APIServerDNSRecordSupported bool
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
		APIServerDNSRecordSupported:  true,
	}
}

func Run(cfg *TestConfig) {
	RunApps(cfg)
	runBasic(cfg)
	runCertManager(cfg.CertManagerSupported)
	runDNS(cfg)
	runMetrics(cfg)
	runTeleport(cfg.TeleportSupported)
	runHelloWorldGateway(cfg.GatewayAPISupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

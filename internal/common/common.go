package common

type TestConfig struct {
	AutoScalingSupported         bool
	BastionSupported             bool
	TeleportSupported            bool
	ExternalDnsSupported         bool
	ControlPlaneMetricsSupported bool
}

func Run(cfg *TestConfig) {
	runApps()
	runBasic()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runMetrics(cfg.ControlPlaneMetricsSupported)
	runTeleport(cfg.TeleportSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

package common

type TestConfig struct {
	AutoScalingSupported bool
	BastionSupported     bool
	ExternalDnsSupported bool
}

func Run(cfg *TestConfig) {
	runApps()
	runBasic()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runScale(cfg.AutoScalingSupported)
	runStorage()
}

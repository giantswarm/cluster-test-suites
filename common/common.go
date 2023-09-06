package common

type TestConfig struct {
	BastionSupported     bool
	ExternalDnsSupported bool
}

func Run(cfg *TestConfig) {
	runApps()
	runBasic()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runStorage()
}

package common

type TestConfig struct {
	BastionSupported     bool
	ExternalDnsSupported bool
}

func Run(cfg *TestConfig) {
	runApps()
	runBasic()
	runDNS(cfg.BastionSupported)
	runHelloWorld(cfg.ExternalDnsSupported)
	runStorage()
}

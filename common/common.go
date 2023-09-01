package common

type TestConfig struct {
	BastionSupported bool
}

func Run(cfg *TestConfig) {
	runBasic()
	runApps()
	runCertManager()
	runDNS(cfg.BastionSupported)
	runStorage()
}

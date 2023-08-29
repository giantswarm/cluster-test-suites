package common

type TestConfig struct {
	BastionSupported bool
}

func Run(cfg *TestConfig) {
	runApps()
	runBasic()
	runDNS(cfg.BastionSupported)
	runHelloWorld()
	runStorage()
}

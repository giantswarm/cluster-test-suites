package common

type TestConfig struct {
	BastionSupported bool
}

func Run(cfg *TestConfig) {
	helloWorld()
	runBasic()
	runApps()
	runDNS(cfg.BastionSupported)
	runStorage()
}

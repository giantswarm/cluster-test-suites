package common

type TestConfig struct {
	BastionSupported bool
}

func Run(cfg *TestConfig) {
	runBasic()
	runApps()
	runDNS(cfg.BastionSupported)
	runStorage()
}

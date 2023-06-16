package common

import (
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
)

type TestConfig struct {
	BastionSupported bool
}

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run(cfg *TestConfig) {
	runBasic()
	runDNS(cfg.BastionSupported)
	runStorage()
}

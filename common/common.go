package common

import (
	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
)

var (
	Framework *clustertest.Framework
	Cluster   *application.Cluster
)

func Run() {
	runBasic()
	runDNS()
}

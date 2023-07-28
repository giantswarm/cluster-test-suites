package state

import (
	"context"
	"sync"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
)

var lock = &sync.Mutex{}

type state struct {
	framework *clustertest.Framework
	cluster   *application.Cluster
	ctx       context.Context
}

var singleInstance *state

func Get() *state {
	if singleInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if singleInstance == nil {
			singleInstance = &state{}
		}
	}

	return singleInstance
}

func (s *state) SetContext(ctx context.Context) {
	s.ctx = ctx
}

func (s *state) GetContext() context.Context {
	return s.ctx
}

func (s *state) SetFramework(framework *clustertest.Framework) {
	s.framework = framework
}

func (s *state) GetFramework() *clustertest.Framework {
	return s.framework
}

func (s *state) SetCluster(framework *application.Cluster) {
	s.cluster = framework
}

func (s *state) GetCluster() *application.Cluster {
	return s.cluster
}

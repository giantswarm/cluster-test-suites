package state

import (
	"context"
	"sync"
	"time"

	"github.com/giantswarm/cluster-test-suites/v2/internal/timeout"

	"github.com/giantswarm/clustertest/v2"
	"github.com/giantswarm/clustertest/v2/pkg/application"
)

var lock = &sync.Mutex{}

type state struct {
	framework *clustertest.Framework
	cluster   *application.Cluster
	ctx       context.Context
}

var singleInstance *state

func get() *state {
	if singleInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if singleInstance == nil {
			singleInstance = &state{}
		}
	}

	return singleInstance
}

func SetContext(ctx context.Context) {
	s := get()
	s.ctx = ctx
}

func GetContext() context.Context {
	return get().ctx
}

func SetFramework(framework *clustertest.Framework) {
	s := get()
	s.framework = framework
}

func GetFramework() *clustertest.Framework {
	return get().framework
}

func SetCluster(framework *application.Cluster) {
	s := get()
	s.cluster = framework
}

func GetCluster() *application.Cluster {
	return get().cluster
}

// SetTestTImeout sets the provided timeout against the given TestKey in the current state context to be used by tests
func SetTestTimeout(testKey timeout.TestKey, timeout time.Duration) {
	s := get()
	ctx := context.WithValue(s.ctx, testKey, timeout)
	SetContext(ctx)
}

// GetTestTimeout returns the timeout from the context for the given TestKey or the defaultTimeout if not found
func GetTestTimeout(testKey timeout.TestKey, defaultTimeout time.Duration) time.Duration {
	s := get()
	val, ok := s.ctx.Value(testKey).(time.Duration)
	if ok {
		return val
	}

	return defaultTimeout
}

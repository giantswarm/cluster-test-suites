package standard

import (
	"context"
	"fmt"

	"github.com/giantswarm/clustertest/pkg/client"
	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/wait"

	"github.com/giantswarm/cluster-test-suites/common"
)

type ClusterValues struct {
	ControlPlane application.ControlPlane `yaml:"controlPlane"`
	NodePools    []application.NodePool   `yaml:"nodePools"`
}

var _ = Describe("Common tests", func() {
	common.Run(&common.TestConfig{
		BastionSupported: true,
	})
})

func CheckWorkerNodesReady(wcClient *client.Client, values *ClusterValues) func() error {
	minNodes := 0
	maxNodes := 0
	for _, pool := range values.NodePools {
		if pool.Replicas > 0 {
			minNodes += pool.Replicas
			maxNodes += pool.Replicas
			continue
		}

		minNodes += pool.MinSize
		maxNodes += pool.MaxSize
	}
	expectedNodes := wait.Range{
		Min: minNodes,
		Max: maxNodes,
	}

	workersFunc := wait.AreNumNodesReadyWithinRange(context.Background(), wcClient, expectedNodes, client.DoesNotHaveLabels{"node-role.kubernetes.io/control-plane"})

	return func() error {
		ok, err := workersFunc()
		if !ok {
			return fmt.Errorf("unexpected number of nodes")
		}
		return err
	}
}

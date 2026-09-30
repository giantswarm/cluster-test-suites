package timeout

import "time"

// AKS clusters are noticeably slower to provision and to delete than the other providers:
// creating the managed control plane and its node pool, and tearing the whole thing down
// again, regularly takes longer than the shared defaults allow. The constants below are the
// AKS-specific replacements for those defaults and are applied by the AKS test suites.
const (
	// AKSNodesReady bounds the wait for the System node pool to come up during cluster standup.
	AKSNodesReady = 40 * time.Minute
	// AKSClusterReady replaces the default ClusterReadyTimeout for the AKS suites.
	AKSClusterReady = 40 * time.Minute
	// AKSWorkerNodes bounds the post-upgrade wait for the worker nodes to be back and ready.
	AKSWorkerNodes = 30 * time.Minute
	// AKSTeardown bounds the AfterSuite cleanup, including deletion of the cluster.
	AKSTeardown = 90 * time.Minute
)

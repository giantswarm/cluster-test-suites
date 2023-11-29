package common

import (
	"github.com/giantswarm/clustertest/pkg/application"
)

// ClusterValues holds common values for cluster-<provider> charts. These are
// the provider independent values and are present for all the charts
type ClusterValues struct {
	Global Global `yaml:"global"`
}

func (v *ClusterValues) ToClusterValuesWithOldSchema() *application.ClusterValues {
	return &application.ClusterValues{
		BaseDomain:   v.Global.Connectivity.BaseDomain,
		ControlPlane: application.ControlPlane(v.Global.ControlPlane),
		NodePools:    v.Global.NodePools.ToNodePoolsWithOldSchema(),
	}
}

type Global struct {
	Metadata     Metadata     `yaml:"metadata"`
	Connectivity Connectivity `yaml:"connectivity"`
	ControlPlane ControlPlane `yaml:"controlPlane"`
	NodePools    NodePools    `yaml:"nodePools"`
}

type Metadata struct {
	Name         string `yaml:"name"`
	Organization string `yaml:"organization"`
}

type Connectivity struct {
	BaseDomain string `yaml:"baseDomain"`
}

type ControlPlane struct {
	Replicas int `yaml:"replicas"`
}

type NodePools map[string]NodePool

func (np *NodePools) ToNodePoolsWithOldSchema() application.NodePools {
	nodePoolsWithOldSchema := application.NodePools{}
	for nodePoolName, nodePool := range *np {
		nodePoolsWithOldSchema[nodePoolName] = application.NodePool(nodePool)
	}
	return nodePoolsWithOldSchema
}

type NodePool struct {
	Replicas int     `yaml:"replicas"`
	MaxSize  int     `yaml:"maxSize"`
	MinSize  int     `yaml:"minSize"`
	Name     *string `yaml:"name"`
}

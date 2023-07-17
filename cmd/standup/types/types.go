package types

type StandupResult struct {
	Provider       string `json:"provider"`
	ClusterName    string `json:"clusterName"`
	OrgName        string `json:"orgName"`
	Namespace      string `json:"namespace"`
	ClusterVersion string `json:"clusterVersion"`
	KubeconfigPath string `json:"kubeconfigPath"`
}

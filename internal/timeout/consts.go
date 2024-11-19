package timeout

const (
	// DeployApps is used by "all default apps are deployed without issues"
	DeployApps TestKey = "deployAppsTimeout"
	// ClusterReadyTimeout is used by "has Cluster Ready condition with Status='True'"
	ClusterReadyTimeout TestKey = "clusterReadyTimeout"
)

package timeout

const (
	// DeployApps is used by "all default apps are deployed without issues"
	DeployApps TestKey = "deployAppsTimeout"
	// UpgradeClusterReadyTimeout is used by "upgrade cluster" has Cluster Ready condition with Status='True'
	UpgradeClusterReadyTimeout TestKey = "upgradeClusterReadyTimeout"
)

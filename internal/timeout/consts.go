package timeout

const (
	// DeployApps is used by "all default apps are deployed without issues"
	DeployApps TestKey = "deployAppsTimeout"
	// ClusterReadyTimeout is used by "has Cluster Ready condition with Status='True'"
	ClusterReadyTimeout TestKey = "clusterReadyTimeout"
	// MimirMetrics is used by "ensure key metrics are available on mimir"
	MimirMetrics TestKey = "mimirMetricsTimeout"
	// PVCBinding is used by "binds the PVC"
	PVCBinding TestKey = "pvcBindingTimeout"
	// CertManager is used by "cert-manager default ClusterIssuers are present and ready"
	CertManager TestKey = "certManagerTimeout"
	// BundleApps is used by observability-bundle and security-bundle app detection
	BundleApps TestKey = "bundleAppsTimeout"
	// ClusterConnection is used by "should be able to connect to the management/workload cluster"
	ClusterConnection TestKey = "clusterConnectionTimeout"
	// GatewayAppReady is used by the hello-world gateway app readiness checks
	GatewayAppReady TestKey = "gatewayAppReadyTimeout"
)

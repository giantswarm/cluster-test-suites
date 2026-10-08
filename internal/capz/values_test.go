package capz

import (
	"testing"

	"github.com/giantswarm/clustertest/v5/pkg/env"

	"github.com/giantswarm/cluster-test-suites/v7/internal/suite"
)

func TestWithLegacyNodePoolReplicas(t *testing.T) {
	tests := []struct {
		name       string
		isUpgrade  bool
		release    string
		preUpgrade string
		want       string
	}{
		{name: "no release", want: ""},
		{name: "v34", release: "v34.2.0", want: legacyNodePoolValues},
		{name: "v35", release: "v35.0.0", want: ""},
		{name: "v35 rc", release: "v35.0.0-rc.1", want: ""},
		{name: "unparseable", release: "latest", want: ""},
		{name: "upgrade from v34", isUpgrade: true, release: "v35.0.0", preUpgrade: "v34.2.0", want: legacyNodePoolValues},
		{name: "upgrade from v35", isUpgrade: true, release: "v35.1.0", preUpgrade: "v35.0.0", want: ""},
		{name: "non-upgrade ignores pre-upgrade", release: "v35.0.0", preUpgrade: "v34.2.0", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(env.ReleaseVersion, tc.release)
			t.Setenv(env.ReleasePreUpgradeVersion, tc.preUpgrade)

			o := &suite.Options{}
			WithLegacyNodePoolReplicas(tc.isUpgrade)(o)
			got, err := o.ExtraClusterValuesFn()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAutoScalingSupported(t *testing.T) {
	tests := []struct {
		release string
		want    bool
	}{
		{release: "", want: true},
		{release: "latest", want: true},
		{release: "v34.2.0", want: false},
		{release: "v35.0.0-rc.1", want: true},
		{release: "v35.0.0", want: true},
		{release: "v36.1.0", want: true},
	}
	for _, tc := range tests {
		t.Run(tc.release, func(t *testing.T) {
			t.Setenv(env.ReleaseVersion, tc.release)
			if got := AutoScalingSupported(); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

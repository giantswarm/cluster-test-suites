package common

import (
	"os"
	"testing"

	"github.com/giantswarm/clustertest/v4/pkg/env"
)

func TestReleaseVersionIsHelmReleaseBased(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		// Old releases: strictly less than the cutover.
		{"34.9.0", false},
		{"34.9.9", false},
		{"v34.0.0", false},
		{"0.1.0", false},
		{"", false},
		{"not-a-semver", false},

		// Cutover itself and above.
		{"35.0.0-0", true},
		{"35.0.0-0a9s8d09as0a8sd", true},
		{"35.0.0-alpha.1", true},
		{"35.0.0", true},
		{"v35.0.0", true},
		{"35.1.0", true},
		{"36.1.0", true},
	}

	for _, tc := range tests {
		got := releaseVersionIsHelmReleaseBased(tc.in)
		if got != tc.want {
			t.Errorf("releaseVersionIsHelmReleaseBased(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestUsesHelmReleaseBasedDefaultApps_PhaseSwitch(t *testing.T) {
	t.Setenv(env.ReleasePreUpgradeVersion, "34.1.0")
	t.Setenv(env.ReleaseVersion, "35.0.0-abc")

	// Ensure we start in the pre-upgrade phase regardless of test ordering.
	ResetUpgradeApplied()
	t.Cleanup(ResetUpgradeApplied)

	if UsesHelmReleaseBasedDefaultApps() {
		t.Fatalf("pre-upgrade: expected App-CR based (from=34.1.0), got HelmRelease-based")
	}

	MarkUpgradeApplied()

	if !UsesHelmReleaseBasedDefaultApps() {
		t.Fatalf("post-upgrade: expected HelmRelease-based (to=35.0.0-abc), got App-CR based")
	}
}

func TestUsesHelmReleaseBasedDefaultApps_StandardOnlyReleaseVersion(t *testing.T) {
	// Simulate a standard (non-upgrade) test: only env.ReleaseVersion is set.
	_ = os.Unsetenv(env.ReleasePreUpgradeVersion)
	t.Setenv(env.ReleaseVersion, "35.0.0-0a9s8d09as0a8sd")

	ResetUpgradeApplied()
	t.Cleanup(ResetUpgradeApplied)

	if !UsesHelmReleaseBasedDefaultApps() {
		t.Fatalf("standard test with v35 prerelease: expected HelmRelease-based, got App-CR based")
	}
}

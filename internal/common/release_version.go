package common

import (
	"os"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/giantswarm/clustertest/v4/pkg/env"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
)

// helmReleaseAppsSinceRelease is the GiantSwarm Release version at which the
// cluster helm charts switched from deploying default apps as App CRs (reconciled
// by app-operator on the MC) to deploying them as Flux HelmReleases in the
// organization namespace.
//
// The `-0` suffix is the semver trick that makes every prerelease of 35.0.0
// (e.g. `v35.0.0-0a9s8d09as0a8sd`) compare >= this constant. In semver,
// `35.0.0-<any>` sorts higher than `35.0.0-0` for any non-numeric-zero
// prerelease, and `35.0.0` (no prerelease) is the highest of all. So using
// `35.0.0-0` as the cutover treats all of v35 (prereleases and final) as
// HelmRelease-based.
const helmReleaseAppsSinceRelease = "35.0.0-0"

var (
	upgradeAppliedMu sync.RWMutex
	upgradeApplied   bool
)

// MarkUpgradeApplied signals that the upgrade test has finished re-applying the
// cluster at the target version. Subsequent calls to UsesHelmReleaseBasedDefaultApps
// will consult the "to" release version (env.ReleaseVersion) instead of the
// "from" version (env.ReleasePreUpgradeVersion).
func MarkUpgradeApplied() {
	upgradeAppliedMu.Lock()
	defer upgradeAppliedMu.Unlock()
	upgradeApplied = true
}

// ResetUpgradeApplied reverts the phase flag. Intended for tests of this helper.
func ResetUpgradeApplied() {
	upgradeAppliedMu.Lock()
	defer upgradeAppliedMu.Unlock()
	upgradeApplied = false
}

// UsesHelmReleaseBasedDefaultApps returns true when the release currently under
// test deploys default apps as Flux HelmReleases rather than App CRs.
//
// Resolution order for the "current release":
//  1. For upgrade tests before the upgrade has been applied, use
//     env.ReleasePreUpgradeVersion (the "from" version).
//  2. Otherwise, use env.ReleaseVersion (the "to" version for upgrades, or the
//     only version for standard tests).
//
// If neither env var is set or the value can't be parsed as semver, the default
// is false (treat as an old App-CR release) — this is the safe choice because
// the existing App-based assertions continue to run.
func UsesHelmReleaseBasedDefaultApps() bool {
	return releaseVersionIsHelmReleaseBased(currentReleaseVersion())
}

func currentReleaseVersion() string {
	upgradeAppliedMu.RLock()
	applied := upgradeApplied
	upgradeAppliedMu.RUnlock()

	pre := strings.TrimSpace(os.Getenv(env.ReleasePreUpgradeVersion))
	to := strings.TrimSpace(os.Getenv(env.ReleaseVersion))

	if !applied && pre != "" {
		return pre
	}
	return to
}

func releaseVersionIsHelmReleaseBased(raw string) bool {
	if raw == "" {
		return false
	}

	cutover, err := semver.NewVersion(helmReleaseAppsSinceRelease)
	if err != nil {
		// Unreachable for a hardcoded constant, but fail safe.
		logger.Log("Failed to parse HelmRelease cutover constant %q: %v", helmReleaseAppsSinceRelease, err)
		return false
	}

	v, err := semver.NewVersion(stripProviderPrefix(raw))
	if err != nil {
		logger.Log("Could not parse release version %q as semver; treating as App-CR based: %v", raw, err)
		return false
	}

	return !v.LessThan(cutover)
}

// stripProviderPrefix removes a leading non-digit provider prefix (e.g.
// "aws-", "cloud-director-") from a release version string. GiantSwarm
// Release versions are often formatted as "<provider>-<semver>", for example
// "aws-35.0.0-t.umz0zjc0xx". This function finds the first digit and returns
// everything from that position onward, so that the remainder is parseable as
// semver. A leading "v" (as in "v35.0.0") is also handled since the digit
// scan starts after it.
func stripProviderPrefix(raw string) string {
	for i, c := range raw {
		if c >= '0' && c <= '9' {
			return raw[i:]
		}
	}
	return raw
}

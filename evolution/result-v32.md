# Result V32

O05 removes the legacy-process assumption from Aofei's bootstrap boundary.
Bootstrap is now a generic first activation from an absent release selection
and absent systemd unit. It snapshots the target-owned base configs, projects
release asset paths, installs and starts the exact unit, verifies loaded state
and health, and returns to an uninstalled state on failure. Target migrations
from any pre-existing service are explicit private maintenance operations, not
generic product behavior. Ordinary deploy still requires a verified healthy
prior release and retains atomic rollback.

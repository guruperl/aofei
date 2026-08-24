#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
REPO_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)

cd "$REPO_ROOT"

GOWORK=off go test -race ./publisherauth \
  -run 'Test(RequestProofBindsBodyFreshnessScopeAndSharedReplay|RequestProofConcurrentReplayHasOneWinner|CredentialLifecycleIsScopedAuditedAndRotatable)$' \
  -count=1

GOWORK=off go test ./dsp \
  -run 'TestServeSSP(VersionedInventoryTokenMigration|PolicyAllowsSDKWithoutBrowserHeaders|AuthenticatedSDKRequiresFreshOneUseBodyProof|AuthenticationFailuresPrecedeAuctionSideEffects|AuthenticatedSDKCannotOverrideIndependentEnforcement)$' \
  -count=1

GOWORK=off go test ./dsp \
  -run 'TestTrafficGate(RateLimitsConfiguredPartnersIndependently|RejectsConcurrentOverloadWithoutBlocking)$' \
  -count=1

echo "P03 repository proof passed: token migration, SDK authentication, replay, lifecycle, and SSP admission controls."

# Principal Provenance And Maintenance Identity

This document is the S05 contract for authenticated HTTP actors and offline
maintenance principals. A role label, account id, permission, scope, or MFA
claim supplied by a request or command line is never authentication.

## HTTP Session Chain

Genelet removes all inbound `X-Forwarded-*` identity/session headers before
routing. Its signed role cookie identifies the account candidate, and the
paired opaque database session must validate for that exact role and account.
Genelet then resolves the configured action permission and resource scope,
checks required TOTP and recent reauthentication, performs database-grant
authorization, and only then binds a verified principal to the request context.

The application-visible boundary is server-owned:

- Genelet deletes every caller-supplied `_g*` value from `Form`, `PostForm`, and
  multipart value views before routing, and repeats the scrub for direct
  `Controller.Handle` callers. JSON responses also remove internal `_g*`
  fields.
- `_grole`, the account-id field, and `_gpermission` are compatibility dispatch
  values derived from the verified route, session, and component configuration;
  they are not authentication evidence.
- The `genelet.VerifiedPrincipal` request capability records the authorized
  role, account, component, action, permission, resource role/id, MFA state,
  and server reauthentication deadline. Its provenance bit and request-context
  keys are private to Genelet, so callers cannot construct or attach a verified
  value through headers, form data, or a direct Summer filter call.

Summer's management-API credential, publisher credential, traffic-quality,
hosted-payment, and account-security filters fail closed unless that exact
component/action capability is present and still matches the compatibility
dispatch fields. They read recent MFA only from its typed session state and
future deadline; action names and caller values cannot manufacture it.
Credential and traffic-quality filters also compare the selected resource to
the authorized resource. Advertiser and publisher scopes remain locked to
their verified account id, while administrator/delegated resources remain
subject to Genelet's server configuration and database-grant check.

## Offline Commands

`cmd/traffic-quality` and `cmd/hosted-payment` have no actor flag. They derive
`admin:unix-uid:<effective-uid>` from the kernel-reported effective Unix UID
and receive only the exact read/retention permission required by the selected
action. The service boundary recognizes that identity only for aggregate
traffic-quality health, quality-evidence retention, or hosted-event retention.
It carries no recent-MFA claim or wildcard permission and cannot authorize a
quality mutation, reconciliation, provider call, or money movement.

`../pzdesign/cmd/identity-admin` needs a numeric administrator foreign key for
the existing identity/audit schema. The restricted Summer configuration must
therefore contain an explicit mapping:

```json
"Identity": {
  "MaintenanceActors": {
    "1001": "42"
  }
}
```

The key is the effective Unix UID and the value is the reviewed administrator
account id. The command has no `-actor-admin-id` override and fails closed when
the running UID is not mapped. Every stored reason is prefixed with
`launcher=unix-uid:<uid>;`, preserving both the mapped database actor and the
authenticated launcher principal in audit evidence. Keep the binary,
maintenance database configuration, identity key, and config write access
restricted to named, non-shared OS accounts; record and independently approve
mapping changes.

## Maker/Checker Boundary

- Identity grant, revoke, recovery, and retention changes require an external
  ticket/checker in addition to the mapped launcher; the command cannot map or
  choose its own actor.
- A01 accounting confirmation and settlement reject the creating/confirming
  principal as the next checker and already derive CLI actors from effective
  Unix UIDs.
- A02 binding approval and operation approval/execution reject the maker or
  prior checker as applicable. Only verified Genelet-session actors reach
  these service methods; offline hosted-payment maintenance is retention-only.
- Traffic-quality billing recommendation and approval use separate service
  transitions and actors. The offline principal cannot invoke either.

## Verification

Repository tests cover inbound reserved-field scrubbing, request-context
principal binding, exact action/resource authorization, direct-Handle and
direct-filter spoof denial, action/header MFA spoof resistance, UID mapping,
launcher audit attribution, exact offline permissions, and denial of
money/quality mutations to maintenance principals. Run the three repository
test/vet/staticcheck/race suites and documentation/public-data guards before
changing this boundary.

# Result V20

M45 makes repository publication a deliberate data boundary.

Implemented direction:

- `etc/step4_init.sql` owns schema and non-sensitive reference catalogs only;
  `etc/demand.sql` owns one documented synthetic development fixture.
- Production database/traffic/configuration snapshots, customer source
  documents, uploaded-media metadata, and third-party geodata are not Git
  inputs, including under `backup/`.
- Missing SMTP configuration disables public account mail actions before any
  account mutation while leaving login and authenticated portals available.
- Both public repositories scan tracked data and full Git history in CI, and
  their affected branches/tags were rewritten after credential containment.

Unchanged direction:

- Exported Go APIs, the active MySQL schema, Redis/cache contracts, DSP/SSP HTTP
  contracts, and deployed advertiser/publisher records are unchanged.

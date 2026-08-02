# Historical Artifacts

Production database dumps, runtime configurations, service units, traffic
captures, uploaded media metadata, and third-party geodata must not be stored in
this repository, including under `backup/`.

Keep operational backups in access-controlled, encrypted storage with an
explicit retention policy. Recreate development state from `etc/step4_init.sql`
and the synthetic fixture in `etc/demand.sql`.

The single-region backup inventory, restore order, retention objectives, and
evidence requirements live in
[`docs/single-region-availability.md`](../docs/single-region-availability.md).
`scripts/aofei-recovery-drill.sh` is an isolated disposable rehearsal, not an
authorization to place its temporary dump or any production backup under this
directory.

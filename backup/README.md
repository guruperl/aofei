# Historical Artifacts

Production database dumps, runtime configurations, service units, traffic
captures, uploaded media metadata, and third-party geodata must not be stored in
this repository, including under `backup/`.

Keep operational backups in access-controlled, encrypted storage with an
explicit retention policy. Recreate development state from `etc/step4_init.sql`
and the synthetic fixture in `etc/demand.sql`.

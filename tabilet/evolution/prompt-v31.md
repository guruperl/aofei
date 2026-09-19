# Prompt V31

Make immutable Aofei website deployment reusable without publishing private
host policy or keeping the deployment algorithm in one target repository.

- write a generic versioned Aofei/Pzdesign/Genelet release format;
- retain legacy W8M release read compatibility only for rollback;
- validate a strict target-owned environment manifest before effects;
- provide tested preflight, bootstrap, atomic deploy, health, rollback, and
  credential-free history behavior; and
- keep exact host names, paths, dependency identities, credentials, and
  activation history in a private infrastructure repository.

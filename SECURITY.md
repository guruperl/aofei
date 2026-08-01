# Security Policy

Do not report vulnerabilities, leaked credentials, or private customer data in
a public issue. Use GitHub's private vulnerability reporting for this
repository so maintainers can investigate without publishing sensitive detail.

Never commit production configuration, credentials, database or traffic dumps,
customer source documents, uploaded media metadata, or real user identifiers.
Run these checks before opening a pull request:

```bash
./scripts/aofei-public-data-check.sh
gitleaks git --redact .
```

If a live credential is exposed, revoke or rotate it before attempting a Git
history cleanup. Treat every value reachable in a public commit as compromised.

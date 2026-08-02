# OpenRTB 2.5 compatibility fixtures

These credential-free fixtures exercise the W8M partner profile without real
partner endpoints, headers, identifiers, or creative assets. Tests compress
`request-multiimp.json` in memory so no opaque generated gzip payload is
tracked. `timeout.json` supplies only a simulated delay and deadline.

- `request-multiimp.json`: two independently identified Banner/Video imps.
- `request-native.json`: strict Native 1.2 request assets.
- `malformed-request.json`: intentionally truncated JSON.
- `response-currency-eur.json`: unsupported response currency.
- `response-native.json`: strict Native response payload.
- `response-video.json`: HTTPS VAST response payload.
- `response-unsafe-callback.json`: encoded active callback scheme.
- `timeout.json`: deterministic local timeout parameters.

All URLs use IANA example domains and all IP addresses use documentation
ranges. Never replace these files with captured production traffic.

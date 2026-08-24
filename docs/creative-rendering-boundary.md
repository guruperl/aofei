# Creative Rendering And Consumer Boundary

This document inventories every first-party surface that receives creative
source or materialized auction output. It separates an executable renderer
from a protocol response: returning OpenRTB, VAST, Native JSON, or structured
`/pz` JSON does not mean Aofei owns the caller's rendering implementation.

## Consumer Inventory

| Surface | Owner and entry point | Creative form | Execution/fetch contract |
|---|---|---|---|
| Advertiser, publisher, and agent management/review | pzdesign Summer templates, composed by Genelet | Stored URL or Native source JSON shown in `creative-source` text blocks | Contextually escaped source only. No frame, image, script, object, preview, or server fetch is created from the stored value; no administrator creative preview exists. |
| Direct-SSP browser HTML | pzdesign `www/js/ads.js`, consuming omitted/`html` `POST /pz` output | Ordered Banner HTML strings; empty string is no-fill | The only first-party executable HTML sink. It creates one `srcdoc` iframe with an opaque origin, no referrer, fixed sandbox, and a fixed sensitive-feature deny policy. It never writes creative markup into the publisher page. |
| Direct-SSP structured API | Aofei `POST /pz` with `responseFormat:"json"` | Fill metadata, `adm`, and extracted structured Native JSON | Serialized protocol data only. No checked-in first-party code executes it. An external integration owns media-specific parsing and rendering. |
| Direct-SSP OpenRTB | Aofei `POST /pz` with `responseFormat:"openrtb"` | OpenRTB `BidResponse` containing Banner `adm`, VAST, or Native JSON | Serialized protocol data only. There is no checked-in WebView, VAST player, or Native component renderer. |
| DSP/OpenRTB exchange response | Aofei `/bid` | OpenRTB Banner `adm`, VAST, or Native JSON | Serialized protocol data for the exchange/SDK that made the request; Aofei does not execute it. |
| Audit, reporting, cache, logs, and P03 App integration output | Aofei and pzdesign | Stored source records, redacted delivery metadata, identifiers, or textual request examples | Never a creative execution surface. P03's App output is a signed-request example, not a mobile SDK or renderer. |

Genelet has no creative trusted-HTML conversion. Its sole raw HTML boundary is
the fixed CSRF input described in
[template-rendering-security.md](template-rendering-security.md). Repository
search and the current I02 status confirm that no maintained Android, iOS,
WebView, or WKWebView package exists.

Stored site review URLs and creative source/image URLs are management metadata,
not server fetch targets. Syntax and media-shape validation parse them without
DNS resolution or HTTP. Focused private-host tripwire tests run the actual
Summer filters against a loopback HTTP server and require zero requests, while
a source guard rejects outbound HTTP clients, transports, and convenience
fetch functions in the source-only campaign/item/site/creative packages. A
private-host URL on one of these source-only surfaces is therefore not an SSRF
path. Any future preview, crawler, verifier, or metadata-enrichment fetch is a
new outbound trust boundary and must use the reviewed special-use/rebinding
transport before it is introduced.

## Browser Renderer

`ads.js` clears the chosen host container and, for a non-empty string, appends
exactly one iframe. The sandbox deliberately permits scripts, forms, and popup
landing behavior, including escape of a newly opened popup from its sandbox.
It deliberately omits `allow-same-origin` and every top-navigation permission,
so creative code receives an opaque origin and cannot become same-origin with
the publisher page. `referrerpolicy="no-referrer"` suppresses the publisher URL.
The iframe `allow` policy denies camera, microphone, geolocation, payment, USB,
serial, Bluetooth, and clipboard features. Hostile markup remains an inert
string at the publisher-page boundary and is assigned only to `srcdoc`.

This is containment, not a claim that arbitrary markup has been sanitized.
D02 validates local creatives as URL-only Banner/Video or structured Native
source and validates middleman media, size, MIME, secure-inventory, callback,
VAST, Native asset, and contained-markup contracts before response selection.
S05 extends contained-markup checks to fetching `srcset`, `ping`, and legacy
`background` attributes and to entity-decoded event-handler attempts to address
the parent/top context. Scripts and ordinary event behavior remain supported
because removing them would change the approved advertising language.

A universal Content Security Policy is not added here. Publisher CSP is
inherited by `srcdoc`, and a restrictive script/style/network policy would
silently break currently approved third-party creatives. Any stricter CSP or
script-stripping policy requires measured compatibility, a versioned migration,
canary rejection/renderer evidence, and rollback. The fixed iframe permissions
policy is the compatible strengthening in this milestone.

## Required Native Consumer Contract

I02 remains demand-gated and unimplemented. Before a named Android or iOS
integration may consume the structured JSON or OpenRTB forms, its milestone
must provide and test all of the following:

- Banner HTML runs only in a dedicated, ephemeral WebView/WKWebView data store
  that has no app origin, cookies, persistent storage, file/content access,
  universal file-URL access, or JavaScript/native bridge. It must not share the
  authenticated application WebView or execute through a first-party HTML
  string sink.
- Top-level navigation is intercepted. Only a user-activated absolute HTTP(S)
  landing URL that passes the delivery URL contract may leave the ad view, and
  it opens through the platform browser. Custom schemes require a separately
  named allowlist and threat review. New windows, downloads, permission
  requests, mixed content, and unsolicited redirects fail closed.
- VAST is parsed as bounded XML and played through a media component. A VAST
  `HTMLResource`, if supported, uses the same isolated ad WebView; XML or URLs
  are never treated as application HTML. Trackers and media URLs retain the
  D02 HTTPS/MIME/event rules.
- Native JSON is decoded strictly by requested asset id/type and rendered with
  native text/image/media controls. Text never becomes HTML; image/media
  fetches accept only validated HTTP(S) URLs and bounded declared types/sizes;
  clicks use the intercepted landing flow.
- Renderer lifetime is bounded to the ad placement. Stop media, revoke any
  temporary resource, clear ephemeral state, and detach handlers on reuse or
  destruction. Hostile fixtures must cover scripts/events, parent/top access,
  storage, bridges, file URLs, redirects, popup/window requests, custom schemes,
  mixed content, malformed VAST, and Native asset/type mismatches.

Until that implementation and evidence exist, the App/SDK examples are server
API integration guidance only and must not be advertised as a maintained
mobile renderer.

## Verification

- Aofei creative tests cover local URL-only materialization, strict Native and
  VAST handling, hostile middleman markup, URL attributes, entity-decoded event
  escapes, secure inventory, and every `/pz` response format.
- pzdesign source and Node fixtures prove there is one `srcdoc` sink, no host
  `innerHTML`, no `allow-same-origin`, fixed sandbox/referrer/permission values,
  hostile string containment, and deterministic fill/no-fill/error states.
- pzdesign hostile rendering fixtures prove all management/review surfaces show
  stored creatives as escaped source. Genelet tests preserve its single fixed
  trusted-HTML boundary.

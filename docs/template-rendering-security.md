# Template Rendering Security Boundary

Aofei supplies account, campaign, publisher, reporting, and delivery data to
the sibling pzdesign Summer/Genelet UI. The renderer, templates, and static
assets are owned by `../pzdesign`; their detailed inventory and review rules are
in
[`../pzdesign/docs/rendering-security.md`](../../pzdesign/docs/rendering-security.md).

## Repository Boundary

- Aofei owns schema fields, cached domain values, auction output, tracking
  identifiers, and the public `/bid` and `/pz` delivery contracts.
- pzdesign owns the Summer filters/models that prepare UI data, all page/mail
  templates, the unified HTTP command, and browser assets.
- Genelet owns template selection/composition and the sole trusted-HTML helper:
  a fixed, contextually escaped CSRF hidden input, injected once into every
  otherwise-unprotected POST form including identity/logout forms.

All control-plane data crosses this boundary as ordinary Go strings, numbers,
or collections. Aofei does not promise that stored account names, campaign
names, publisher metadata, route labels, report dimensions, URLs, or creative
content are safe HTML. The UI must retain Go `html/template` contextual
escaping.

## Creative Execution Boundary

There are two deliberately different surfaces:

1. Authenticated management and review pages display stored creative markup or
   URLs only as escaped source. They do not create frames, images, scripts,
   objects, embeds, or other fetching elements from stored creative values.
2. Auction delivery may intentionally materialize an approved creative in an
   OpenRTB or `/pz` response. That is a product delivery contract rather than a
   trusted template helper. D02 enforces media type, exact size, MIME, source,
   landing/tracker URL, secure-inventory, structured Native assets, and
   hostile middleman-markup validation before this path. The exact contract is
   [auction-pricing-creatives.md](auction-pricing-creatives.md).

Never reuse delivery markup as `template.HTML` in a management page. Previewing
actual advertising behavior, if later required, needs an isolated origin or
sandbox contract and its own milestone-level threat review.

## URL And Redirect Ownership

Genelet validates login/return redirects as local single-slash paths beneath
the configured script root. pzdesign templates use local reviewed assets and
contextually escaped URL attributes. Aofei and Summer controllers remain
responsible for semantic validation of external bidder, callback, and creative
landing endpoints; displaying an endpoint does not authorize the browser or
server to fetch it.

## Change Checklist

When an Aofei change adds a value to a page, mail, report, chart, generated tag,
or control-plane preview:

- classify its source and whether an account or partner controls it;
- keep it as an ordinary type through the Summer/Genelet boundary;
- add a hostile rendering fixture in pzdesign for every new HTML, attribute,
  URL, JavaScript, or CSS context;
- update the pzdesign rendering inventory if the entry point or trust boundary
  changes;
- keep actual creative execution and fetch behavior in the delivery-security
  milestone that owns it.

Closeout runs both pzdesign template parsers and security fixtures, Genelet's
trusted-boundary tests, Aofei documentation/public-data checks, and repository
diff hygiene.

S02 TOTP enrollment secrets, standard `otpauth` URIs, and newly issued recovery
codes are particularly sensitive ordinary strings. They are rendered only on
the authenticated account-security response, covered by hostile fixtures for
all five role families, and must never become `template.HTML`, a data URL,
script input, log field, audit field, or stored page value.

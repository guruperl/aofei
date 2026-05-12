# Genelet Manual

Genelet is the local web/admin framework used by Summer and `cmd/unify`. It
owns route parsing, role/cookie auth, component-driven CRUD setup, templates,
uploads, CORS, and framework error rendering. It does not own DSP bidding,
schema migrations, Redis payload contracts, or production secrets.

## Configuration

Genelet reads the `SUMMER` config through `genelet.NewConfig`. The local helper
generates `etc/summer.local.json`; checked-in examples live in `etc/summer.json`.
The config uses upper-case keys such as `ConnectArray`, `Template`,
`UploadDir`, `ProjectRoot`, `Script`, `Roles`, and `Chartags`.
The local Summer template path points at the sibling `../pzdesign/tmpls` tree,
and static UI assets are served from `../pzdesign/www`.

`cmd/unify` usually obtains the database, Redis, NATS, and server defaults from
the DSP controller, then fills missing Genelet server fields from the DSP
config. Admin tests that need a database should use:

```bash
GOWORK=off SUMMER="$PWD/etc/summer.local.json" go test ./genelet
```

## Routes

Genelet handles paths under `Config.Script`:

```text
/{script}/{role}/{chartag}/{object}
/{script}/{role}/{chartag}/{object}/{id}
```

Requests outside `Config.Script` are static-file requests served from
`DocumentRoot`. Static paths are cleaned and must resolve under `DocumentRoot`.
The URL segments select the role, response format (`chartag`), object/module,
and optional object id.

HTTP methods map to actions through `DefaultActions`. The default map is:

```text
GET -> dashboard
GET_item -> edit
POST -> insert
PUT -> update
DELETE -> delete
```

The query/form `ActionName` can override the method action when the component
allows it.

## Roles And Auth

Roles define id fields, attributes, cookie surface names, issuers, logout
targets, admin status, and optional id ciphering. Gate handling verifies the
role cookie and populates forwarded headers:

```text
X-Forwarded-User
X-Forwarded-Group
X-Forwarded-Time
X-Forwarded-Duration
```

Controller handling copies those values into request args. The group header
must contain exactly one value for each role attribute after the login/id
attribute. A mismatch returns a framework error instead of indexing past the
attribute list.

Login flows are handled by OAuth2, OAuth1, or procedure issuers. Login/logout
status handling only treats `genelet.Gerror` as a framework status error; other
errors are returned as server errors with their diagnostic string.

## Components

Each Summer module has a `component.json`. Components declare actions, role
groups, foreign-key signing rules, current table/key metadata, CRUD field lists,
joined table metadata, select labels, next pages, and request parameter names.

Use `genelet.LoadComponent(path)` for active setup. It returns actionable file,
JSON, and component-validation errors. `genelet.NewComponent(path)` remains as a
legacy panic wrapper for older tests and callers.

Component metadata that reaches SQL construction is validated centrally:
tables, aliases, keys, field lists, join types, join fields, join conditions,
select labels, and order fields must pass the Genelet SQL identifier helpers.
Component select expressions may contain known SQL functions, but statement
separators and comments are rejected.

## Model And Filter Lifecycle

Controller handling follows this order:

1. Set model defaults with request args, output lists, `other`, and storage.
2. Set filter base/action/component state.
3. Read the action rule and foreign-key rule from `Filter.GetAll`.
4. Copy role/auth metadata into request args.
5. Enforce action group access and foreign-key signatures.
6. Run `Filter.Preset`.
7. Set the model DB unless action options include `no_db`.
8. Run `Filter.Before`.
9. Run the model action unless action options suppress the model method.
10. Assign foreign-key signatures for returned lists.
11. Run `Filter.After`, send optional mail blocks, then render JSON/template.

Reflection dispatch is guarded by `TryInvoke`, `InvokeVoid`, and
`InvokeError`. Missing methods, wrong arity, wrong argument types, and panics
return framework errors. Embedded Summer/Genelet model adapters are resolved for
promoted base methods.

## CRUD And Query Rules

Use Genelet CRUD helpers instead of ad hoc SQL when the input includes request
or component metadata. Active helpers validate identifiers before building SQL:

- `TableStringSafe`
- `SelectLabelStringSafe`
- `SelectConditionStringSafe`
- `SingleConditionStringSafe`
- `Model.GetOrderStringChecked`

Request-derived `_gsql` condition fragments are not allowed in active condition
building. Query values remain parameterized; only validated identifiers are
interpolated.

## Templates, Uploads, And CORS

HTML responses are rendered with Go `html/template`. Genelet loads templates
from `Template/{role}/{object}/{action}.{tag}` and shared role templates
matching `Template/{role}/*.{tag}`. JSON chartags marshal the `Tmpl` payload
directly.

Multipart uploads are written under `UploadDir`. Field names must be present,
field values are bounded, and file names must be clean basenames. Summer may
move validated uploads into role/object subdirectories after Genelet receives
them.

CORS allows the exact `ServerURL` origin plus exact origins in `CORSOrigins`.
Rejected origins receive HTTP 403 before action handling.

## Testing

Fast framework and admin package checks:

```bash
GOWORK=off go test ./genelet ./summer/... ./cmd/unify
```

DB-backed admin compatibility, after local Docker config exists:

```bash
GOWORK=off SUMMER="$PWD/etc/summer.local.json" \
  go test ./genelet ./summer ./summer/pub ./summer/slot ./summer/weight
```

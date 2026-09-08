# Result V31

O04 changes the deployment boundary from a reusable release builder plus a
target-owned implementation to a reusable release builder and deployment
engine plus a target-owned manifest and thin adapter.

The public repository owns strict schemas and effect ordering. Private
realizations own all exact host values, installation inputs, and activation
history. New bundles write the generic v2 contract; legacy W8M v1 bundles are
accepted only as a read/rollback bridge. Deployment remains distinct from
schema, cache, feature, dependency, provider, browser, and retention changes.

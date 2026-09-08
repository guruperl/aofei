# Prompt V32

Make generic website bootstrap describe a clean first activation rather than a
one-off migration from W8M's former direct binary.

- require an absent release selection and uninstalled/unloaded service unit;
- retain owner-provided base configs and secret files as target-owned inputs;
- install, start, verify, and record one immutable release transactionally;
- restore an uninstalled state when activation fails; and
- preserve ordinary verified-release deployment and rollback behavior.

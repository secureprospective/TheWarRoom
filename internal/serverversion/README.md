# Server compatibility

This package owns client admission and read-only league-file preflight. See
[the upgrade contract](../../deploy/server/UPGRADE.md) for ordering, recovery and the
slice 44 update-required signal. Append a release row when the schema, protocol,
command version or security floor changes; never edit a shipped row. The contract
test compares the latest row with the history schema and durable command version.

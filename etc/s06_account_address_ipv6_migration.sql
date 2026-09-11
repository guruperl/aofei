-- S06 account-address IPv6 compatibility migration.
--
-- This is a populated-system migration for deployments whose add_address.ip
-- column still has the legacy IPv4-only width. Take a verified backup and
-- rehearse against a representative restore before applying it. MySQL DDL is
-- not transactional; a failed or partial run must be reviewed rather than
-- guessed through. The widening is compatible with existing IPv4 values and
-- with binaries that already pass canonical IPv4 or IPv6 address strings.

-- Require the exact reviewed legacy source shape before durable mutation. A
-- rerun against the widened target fails closed instead of silently claiming
-- that this migration performed the change.
SELECT COUNT(*)=1 AND SUM(CASE
  WHEN data_type='varchar' AND character_maximum_length=15
       AND is_nullable='YES' THEN 1
  ELSE 0 END)=1 INTO @s06_address_ip_source_ok
FROM information_schema.columns
WHERE table_schema=DATABASE()
  AND table_name='add_address'
  AND column_name='ip';

SET @s06_address_ip_preflight_sql = IF(@s06_address_ip_source_ok,
  'DO 0',
  'SELECT s06_account_address_ipv6_migration_requires_legacy_varchar_15');
PREPARE s06_address_ip_preflight_statement
  FROM @s06_address_ip_preflight_sql;
EXECUTE s06_address_ip_preflight_statement;
DEALLOCATE PREPARE s06_address_ip_preflight_statement;

ALTER TABLE add_address
  MODIFY COLUMN ip VARCHAR(45) DEFAULT NULL;

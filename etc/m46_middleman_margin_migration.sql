-- M46 offline migration for populated databases that already use the A03
-- exact-money schema but do not yet enforce route percentage fractions.
--
-- Stop route and management writers, take a verified backup, and resolve every
-- invalid row before applying this file. MySQL DDL is not transactional; a
-- failed or partial run must be restored and reviewed rather than retried.

-- Require the exact reviewed source shape and no prior/partial constraint
-- application before any durable mutation.
SELECT COUNT(*)=2 AND SUM(CASE
  WHEN table_name='mid_route_group' AND is_nullable='NO'
       AND data_type='decimal' AND numeric_precision=7 AND numeric_scale=4 THEN 1
  WHEN table_name='mid_route_bidder' AND is_nullable='YES'
       AND data_type='decimal' AND numeric_precision=7 AND numeric_scale=4 THEN 1
  ELSE 0 END)=2 INTO @m46_margin_source_ok
FROM information_schema.columns
WHERE table_schema=DATABASE() AND column_name='margin_pct'
  AND table_name IN ('mid_route_group','mid_route_bidder');
SET @m46_margin_preflight_ok = @m46_margin_source_ok AND
  (SELECT COUNT(*) FROM information_schema.table_constraints
   WHERE constraint_schema=DATABASE() AND constraint_name IN
     ('mid_route_group_margin_fraction_chk','mid_route_bidder_margin_fraction_chk'))=0;
SET @m46_margin_preflight_sql = IF(@m46_margin_preflight_ok, 'DO 0',
  'SELECT m46_middleman_margin_migration_requires_an_unmodified_unconstrained_source');
PREPARE m46_margin_preflight_statement FROM @m46_margin_preflight_sql;
EXECUTE m46_margin_preflight_statement;
DEALLOCATE PREPARE m46_margin_preflight_statement;

-- Do not infer percent-vs-fraction intent, clamp, deactivate, or skip a row.
-- The operator must reconcile invalid configuration against its commercial
-- agreement before rerunning from the restored source.
SELECT
  (SELECT COUNT(*) FROM mid_route_group
   WHERE margin_pct < 0.0000 OR margin_pct > 1.0000) +
  (SELECT COUNT(*) FROM mid_route_bidder
   WHERE margin_pct IS NOT NULL AND (margin_pct < 0.0000 OR margin_pct > 1.0000))
INTO @m46_invalid_margin_rows;
SET @m46_margin_data_sql = IF(@m46_invalid_margin_rows=0, 'DO 0',
  'SELECT m46_middleman_margin_migration_requires_zero_invalid_rows');
PREPARE m46_margin_data_statement FROM @m46_margin_data_sql;
EXECUTE m46_margin_data_statement;
DEALLOCATE PREPARE m46_margin_data_statement;

ALTER TABLE mid_route_group
  ADD CONSTRAINT mid_route_group_margin_fraction_chk
  CHECK (margin_pct BETWEEN 0.0000 AND 1.0000);
ALTER TABLE mid_route_bidder
  ADD CONSTRAINT mid_route_bidder_margin_fraction_chk
  CHECK (margin_pct IS NULL OR margin_pct BETWEEN 0.0000 AND 1.0000);

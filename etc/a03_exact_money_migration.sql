-- A03 offline migration for an existing pre-v3 database.
--
-- Preconditions: stop all writers, take and verify a frozen backup, and run
-- the inventory queries in docs/exact-money.md. This script deliberately does
-- not DROP the evidence table on retry. MySQL DDL is not transactional; a
-- failed run must be restored from the frozen backup before retry.

-- Fail before any durable mutation unless this is the one reviewed v2 source
-- shape. A rerun against v3, a partial prior attempt, or a different legacy
-- schema must be restored/reviewed instead of guessed through.
SELECT COUNT(*)=23 AND SUM(CASE
  WHEN table_name='adv_item' AND column_name='cost' AND data_type='double' THEN 1
  WHEN table_name='pub_slot' AND column_name='bidfloor' AND data_type='float' THEN 1
  WHEN table_name='adv_balance' AND column_name IN ('limit_spend','current_spend') AND data_type='float' THEN 1
  WHEN table_name='his_balance' AND column_name IN ('budget_old','budget_add','budget_new') AND data_type='float' THEN 1
  WHEN table_name IN ('ledger_log','ledger_adv','ledger_pub','ledger_pub_adv','daily_log','daily_adv','daily_pub','daily_pub_adv') AND column_name='spend' AND data_type='float' THEN 1
  WHEN table_name IN ('ledger_mid','daily_mid') AND column_name IN ('charge_spend','pay_spend','margin_spend') AND data_type='float' THEN 1
  WHEN table_name IN ('mid_route_group','mid_route_bidder') AND column_name='min_margin_cpm'
       AND data_type='decimal' AND numeric_precision=10 AND numeric_scale=4 THEN 1
  ELSE 0 END)=23 INTO @a03_source_types_ok
FROM information_schema.columns
WHERE table_schema=DATABASE() AND (
  (table_name='adv_item' AND column_name='cost') OR
  (table_name='pub_slot' AND column_name='bidfloor') OR
  (table_name='adv_balance' AND column_name IN ('limit_spend','current_spend')) OR
  (table_name='his_balance' AND column_name IN ('budget_old','budget_add','budget_new')) OR
  (table_name IN ('ledger_log','ledger_adv','ledger_pub','ledger_pub_adv','daily_log','daily_adv','daily_pub','daily_pub_adv') AND column_name='spend') OR
  (table_name IN ('ledger_mid','daily_mid') AND column_name IN ('charge_spend','pay_spend','margin_spend')) OR
  (table_name IN ('mid_route_group','mid_route_bidder') AND column_name='min_margin_cpm'));
SET @a03_preflight_ok =
  @a03_source_types_ok AND
  (SELECT COUNT(*) FROM acct_contract WHERE contract_id=1 AND unit_version='usd-cpm-impression-v2')=1 AND
  (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='money_migration_evidence')=0 AND
  (SELECT column_default='usd-cpm-impression-v2' FROM information_schema.columns
    WHERE table_schema=DATABASE() AND table_name='report_delivery' AND column_name='accounting_version');
SET @a03_preflight_sql = IF(@a03_preflight_ok, 'DO 0',
  'SELECT a03_exact_money_migration_requires_an_unmodified_frozen_v2_source');
PREPARE a03_preflight_statement FROM @a03_preflight_sql;
EXECUTE a03_preflight_statement;
DEALLOCATE PREPARE a03_preflight_statement;

CREATE TABLE IF NOT EXISTS `money_migration_evidence` (
  `evidence_id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `contract_version` varchar(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `source_table` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `source_pk` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `source_column` varchar(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  `legacy_value` varchar(255) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  `converted_value` decimal(20,9) DEFAULT NULL,
  `conversion_rule` enum('LegacyRenderedHalfAway','AlreadyExact','Quarantined') NOT NULL,
  `discrepancy` decimal(20,9) DEFAULT NULL,
  `created_at` datetime(6) NOT NULL,
  PRIMARY KEY (`evidence_id`),
  UNIQUE KEY `money_migration_source` (`contract_version`,`source_table`,`source_pk`,`source_column`),
  KEY `money_migration_disposition` (`conversion_rule`,`source_table`),
  CONSTRAINT `money_migration_discrepancy_chk` CHECK (((`conversion_rule` = 'Quarantined') AND (`converted_value` IS NULL)) OR (`conversion_rule` <> 'Quarantined'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Capture every database-observable legacy monetary value before its column is
-- altered. The original human-entered decimal behind an IEEE-754 value is not
-- recoverable; discrepancy records only the difference between that legacy
-- value and the reviewed target scale. NULL is already an exact absence.
DROP PROCEDURE IF EXISTS a03_capture_legacy_money;
DELIMITER ;;
CREATE PROCEDURE a03_capture_legacy_money(
  IN source_table_name VARCHAR(64),
  IN source_pk_name VARCHAR(64),
  IN source_column_name VARCHAR(64),
  IN target_scale TINYINT UNSIGNED,
  IN require_nonnegative BOOLEAN,
  IN source_already_exact BOOLEAN
)
BEGIN
  IF source_table_name NOT REGEXP '^[a-z0-9_]+$' OR
     source_pk_name NOT REGEXP '^[a-z0-9_]+$' OR
     source_column_name NOT REGEXP '^[a-z0-9_]+$' OR
     target_scale NOT IN (6,9)
  THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='invalid A03 evidence source'; END IF;
  SET @a03_target_type = IF(target_scale=6, 'DECIMAL(12,6)', 'DECIMAL(20,9)');
  SET @a03_source = CONCAT('`',source_column_name,'`');
  SET @a03_minimum = IF(require_nonnegative, '0',
    IF(target_scale=9, '-9223372036.854775808', '-999999.999999'));
  SET @a03_maximum = IF(target_scale=9, '9223372036.854775807', '999999.999999');
  SET @a03_invalid = CONCAT(@a03_source,' < ',@a03_minimum,
    ' OR ',@a03_source,' > ',@a03_maximum);
  SET @a03_evidence_sql = CONCAT(
    'INSERT INTO money_migration_evidence ',
    '(contract_version,source_table,source_pk,source_column,legacy_value,',
    'converted_value,conversion_rule,discrepancy,created_at) SELECT ',
    QUOTE('usd-cpm-impression-v3'),',',QUOTE(source_table_name),',',
    'CAST(`',source_pk_name,'` AS CHAR),',QUOTE(source_column_name),',',
    'CAST(',@a03_source,' AS CHAR),',
    'CASE WHEN ',@a03_source,' IS NULL OR ',@a03_invalid,' THEN NULL ELSE CAST(',
      @a03_source,' AS ',@a03_target_type,') END,',
    'CASE WHEN ',@a03_source,' IS NULL THEN ',QUOTE('AlreadyExact'),
      ' WHEN ',@a03_invalid,' THEN ',QUOTE('Quarantined'),
      ' WHEN ',IF(source_already_exact,'TRUE','FALSE'),' THEN ',QUOTE('AlreadyExact'),
      ' ELSE ',QUOTE('LegacyRenderedHalfAway'),' END,',
    'CASE WHEN ',@a03_source,' IS NULL OR ',@a03_invalid,' THEN NULL WHEN ',
      IF(source_already_exact,'TRUE','FALSE'),' THEN 0 ELSE CAST(CAST(',
      @a03_source,' AS ',@a03_target_type,')-',@a03_source,' AS DECIMAL(20,9)) END,NOW(6) ',
    'FROM `',source_table_name,'` ON DUPLICATE KEY UPDATE evidence_id=evidence_id');
  PREPARE a03_evidence_statement FROM @a03_evidence_sql;
  EXECUTE a03_evidence_statement;
  DEALLOCATE PREPARE a03_evidence_statement;
END ;;
DELIMITER ;

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'adv_item', CAST(item_id AS CHAR), 'cost',
       CAST(cost AS CHAR),
       CASE WHEN cost_type='CPM' AND cost BETWEEN 0.000001 AND 999999.999999
            THEN CAST(cost AS DECIMAL(12,6)) ELSE NULL END,
       CASE WHEN cost_type='CPM' AND cost BETWEEN 0.000001 AND 999999.999999
            THEN 'LegacyRenderedHalfAway' ELSE 'Quarantined' END,
       CASE WHEN cost_type='CPM' AND cost BETWEEN 0.000001 AND 999999.999999
            THEN CAST(CAST(cost AS DECIMAL(12,6))-cost AS DECIMAL(20,9)) ELSE NULL END,
       NOW(6)
FROM adv_item
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'pub_slot', CAST(slot_id AS CHAR), 'bidfloor',
       CAST(bidfloor AS CHAR),
	   CASE WHEN bidfloor IS NULL THEN NULL
	        WHEN bidfloor BETWEEN 0 AND 999999.999999
            THEN CAST(bidfloor AS DECIMAL(12,6)) ELSE NULL END,
	   CASE WHEN bidfloor IS NULL THEN 'AlreadyExact'
	        WHEN bidfloor BETWEEN 0 AND 999999.999999
            THEN 'LegacyRenderedHalfAway' ELSE 'Quarantined' END,
	   CASE WHEN bidfloor IS NULL THEN NULL
	        WHEN bidfloor BETWEEN 0 AND 999999.999999
            THEN CAST(CAST(bidfloor AS DECIMAL(12,6))-bidfloor AS DECIMAL(20,9)) ELSE NULL END,
       NOW(6)
FROM pub_slot
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

CALL a03_capture_legacy_money('adv_balance','balance_id','limit_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('adv_balance','balance_id','current_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('his_balance','his_balance_id','budget_old',9,TRUE,FALSE);
CALL a03_capture_legacy_money('his_balance','his_balance_id','budget_add',9,FALSE,FALSE);
CALL a03_capture_legacy_money('his_balance','his_balance_id','budget_new',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_log','log_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_adv','la_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_pub','lp_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_pub_adv','lpa_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_mid','lm_id','charge_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_mid','lm_id','pay_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('ledger_mid','lm_id','margin_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_log','log_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_adv','la_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_pub','lp_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_pub_adv','lpa_id','spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_mid','lm_id','charge_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_mid','lm_id','pay_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('daily_mid','lm_id','margin_spend',9,TRUE,FALSE);
CALL a03_capture_legacy_money('mid_route_group','group_id','min_margin_cpm',6,TRUE,TRUE);
CALL a03_capture_legacy_money('mid_route_bidder','route_bidder_id','min_margin_cpm',6,TRUE,TRUE);
DROP PROCEDURE a03_capture_legacy_money;

-- Evidence is deliberately durable, but a quarantined source cannot be
-- converted, paused, or promoted to v3. Restore the frozen backup, resolve the
-- source, and start the reviewed migration again.
SET @a03_quarantine_sql = IF(
  (SELECT COUNT(*) FROM money_migration_evidence WHERE contract_version='usd-cpm-impression-v3' AND conversion_rule='Quarantined')=0,
  'DO 0', 'SELECT a03_exact_money_migration_requires_zero_quarantined_sources');
PREPARE a03_quarantine_statement FROM @a03_quarantine_sql;
EXECUTE a03_quarantine_statement;
DEALLOCATE PREPARE a03_quarantine_statement;

ALTER TABLE adv_item MODIFY cost DECIMAL(12,6) NULL;
ALTER TABLE pub_slot MODIFY bidfloor DECIMAL(12,6) NULL DEFAULT 0.000000;
ALTER TABLE adv_balance
  MODIFY limit_spend DECIMAL(20,9) NULL,
  MODIFY current_spend DECIMAL(20,9) NULL DEFAULT 0.000000000;
ALTER TABLE his_balance
  MODIFY budget_old DECIMAL(20,9) NOT NULL,
  MODIFY budget_add DECIMAL(20,9) NOT NULL,
  MODIFY budget_new DECIMAL(20,9) NOT NULL;
ALTER TABLE ledger_log MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE ledger_adv MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE ledger_pub MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE ledger_pub_adv MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE ledger_mid
  MODIFY charge_spend DECIMAL(20,9) NULL DEFAULT 0.000000000,
  MODIFY pay_spend DECIMAL(20,9) NULL DEFAULT 0.000000000,
  MODIFY margin_spend DECIMAL(20,9) NULL DEFAULT 0.000000000;
ALTER TABLE daily_log MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE daily_adv MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE daily_pub MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE daily_pub_adv MODIFY spend DECIMAL(20,9) NULL;
ALTER TABLE daily_mid
  MODIFY charge_spend DECIMAL(20,9) NULL DEFAULT 0.000000000,
  MODIFY pay_spend DECIMAL(20,9) NULL DEFAULT 0.000000000,
  MODIFY margin_spend DECIMAL(20,9) NULL DEFAULT 0.000000000;
ALTER TABLE mid_route_group MODIFY min_margin_cpm DECIMAL(12,6) NOT NULL DEFAULT 0.000000;
ALTER TABLE mid_route_bidder MODIFY min_margin_cpm DECIMAL(12,6) NULL;
ALTER TABLE report_delivery ALTER accounting_version SET DEFAULT 'usd-cpm-impression-v3';

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'acct_contract', CAST(contract_id AS CHAR),
       'unit_version', unit_version, NULL, 'AlreadyExact', NULL, NOW(6)
FROM acct_contract WHERE contract_id=1 AND unit_version='usd-cpm-impression-v2'
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;
UPDATE acct_contract
SET unit_version='usd-cpm-impression-v3', effective_at=UTC_TIMESTAMP(),
    notes='Exact micro-USD CPM and nano-USD impression aggregation; statements round once to micro-USD'
WHERE contract_id=1 AND unit_version='usd-cpm-impression-v2';

DROP TRIGGER IF EXISTS acct_statement_protected_update;
DROP TRIGGER IF EXISTS acct_statement_immutable_delete;
DROP TRIGGER IF EXISTS money_migration_evidence_immutable_update;
DROP TRIGGER IF EXISTS money_migration_evidence_immutable_delete;
DELIMITER ;;
CREATE TRIGGER acct_statement_protected_update BEFORE UPDATE ON acct_statement FOR EACH ROW
BEGIN
  DECLARE expected_adjustment DECIMAL(20,6);
  IF NOT (NEW.request_key <=> OLD.request_key) OR
     NOT (NEW.party_type <=> OLD.party_type) OR NOT (NEW.party_id <=> OLD.party_id) OR
     NOT (NEW.cadence <=> OLD.cadence) OR NOT (NEW.period_start <=> OLD.period_start) OR
     NOT (NEW.period_end <=> OLD.period_end) OR NOT (NEW.currency <=> OLD.currency) OR
     NOT (NEW.source_amount <=> OLD.source_amount) OR NOT (NEW.supersedes_id <=> OLD.supersedes_id) OR
     NOT (NEW.created_by <=> OLD.created_by) OR NOT (NEW.created_at <=> OLD.created_at)
  THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='accounting statement identity and source are immutable'; END IF;
  IF NOT (NEW.adjustment_amount <=> OLD.adjustment_amount) OR NOT (NEW.total_amount <=> OLD.total_amount) THEN
    IF OLD.status NOT IN ('Draft','Held') OR NEW.status <> OLD.status
    THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='accounting amounts change only through draft adjustments'; END IF;
    SELECT COALESCE(SUM(amount),0) INTO expected_adjustment FROM acct_adjustment WHERE statement_id=OLD.statement_id;
    IF NEW.adjustment_amount <> expected_adjustment OR NEW.total_amount <> NEW.source_amount + expected_adjustment
    THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='accounting amounts must match immutable adjustments'; END IF;
  END IF;
END ;;
CREATE TRIGGER acct_statement_immutable_delete BEFORE DELETE ON acct_statement FOR EACH ROW
BEGIN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='accounting statements use immutable corrections'; END ;;
CREATE TRIGGER money_migration_evidence_immutable_update BEFORE UPDATE ON money_migration_evidence FOR EACH ROW
BEGIN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='money migration evidence is immutable'; END ;;
CREATE TRIGGER money_migration_evidence_immutable_delete BEFORE DELETE ON money_migration_evidence FOR EACH ROW
BEGIN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='money migration evidence is immutable'; END ;;
DELIMITER ;

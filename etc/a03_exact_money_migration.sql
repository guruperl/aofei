-- A03 offline migration for an existing pre-v3 database.
--
-- Preconditions: stop all writers, take and verify a frozen backup, and run
-- the inventory queries in docs/exact-money.md. This script deliberately does
-- not DROP the evidence table on retry. MySQL DDL is not transactional; a
-- failed run must be restored from the frozen backup before retry.

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

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'adv_item', CAST(item_id AS CHAR), 'cost',
       CAST(cost AS CHAR),
       CASE WHEN cost_type='CPM' AND cost BETWEEN 0.000001 AND 999999.999999
            THEN CAST(cost AS DECIMAL(12,6)) ELSE NULL END,
       CASE WHEN cost_type='CPM' AND cost BETWEEN 0.000001 AND 999999.999999
            THEN 'LegacyRenderedHalfAway' ELSE 'Quarantined' END,
       NULL, NOW(6)
FROM adv_item
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

-- Unsupported commercial models remain excluded and cannot enter the v3
-- active cache. Their rendered legacy value is retained above.
UPDATE adv_item
SET active='Pause'
WHERE active IN ('Yes','New','Pass2')
  AND (cost_type <> 'CPM' OR cost IS NULL OR cost < 0.000001 OR cost > 999999.999999);

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'adv_balance', CAST(balance_id AS CHAR), 'spend',
       CONCAT('limit=',COALESCE(CAST(limit_spend AS CHAR),'NULL'),';current=',COALESCE(CAST(current_spend AS CHAR),'NULL')),
       CASE WHEN COALESCE(limit_spend,0) >= 0 AND COALESCE(current_spend,0) >= 0
            THEN CAST(current_spend AS DECIMAL(20,9)) ELSE NULL END,
       CASE WHEN COALESCE(limit_spend,0) >= 0 AND COALESCE(current_spend,0) >= 0
            THEN 'LegacyRenderedHalfAway' ELSE 'Quarantined' END,
       NULL, NOW(6)
FROM adv_balance
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'pub_slot', CAST(slot_id AS CHAR), 'bidfloor',
       CAST(bidfloor AS CHAR),
       CASE WHEN COALESCE(bidfloor,0) BETWEEN 0 AND 999999.999999
            THEN CAST(bidfloor AS DECIMAL(12,6)) ELSE NULL END,
       CASE WHEN COALESCE(bidfloor,0) BETWEEN 0 AND 999999.999999
            THEN 'LegacyRenderedHalfAway' ELSE 'Quarantined' END,
       NULL, NOW(6)
FROM pub_slot
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

-- Preserve interval and daily row renderings as a single evidence record per
-- table row. Each component is converted independently by the ALTER below.
INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'ledger_pub_adv', CAST(lpa_id AS CHAR), 'spend',
       CAST(spend AS CHAR), CAST(spend AS DECIMAL(20,9)),
       'LegacyRenderedHalfAway', NULL, NOW(6)
FROM ledger_pub_adv
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

INSERT INTO money_migration_evidence
  (contract_version, source_table, source_pk, source_column, legacy_value,
   converted_value, conversion_rule, discrepancy, created_at)
SELECT 'usd-cpm-impression-v3', 'daily_pub_adv', CAST(lpa_id AS CHAR), 'spend',
       CAST(spend AS CHAR), CAST(spend AS DECIMAL(20,9)),
       'LegacyRenderedHalfAway', NULL, NOW(6)
FROM daily_pub_adv
ON DUPLICATE KEY UPDATE evidence_id=evidence_id;

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

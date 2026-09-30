-- S07 one-way account-identifier plaintext retirement.
--
-- This migration is deliberately separate from the additive S07 migration.
-- Before execution, stop every account writer, verify a frozen backup and its
-- restore, run cmd/account-data -mode=verify with a limit covering every row
-- under the exact current key, and record the private canary/rollback,
-- outstanding-link, proxy-log, and rotation evidence. The verifier proves
-- ciphertext/digest/current-key/plaintext parity; SQL cannot prove that
-- cryptographic relation without the protected key.
--
-- Configure PlaintextRetired=true before restarting protected writers after
-- this migration. MySQL DDL is not transactional. A partial/failed execution
-- requires restore and review; never retry this file against a partially
-- changed schema. This file does not modify passwords or token proofs.

-- Require the complete additive schema and its original plaintext indexes.
-- Any drift fails before the first durable operation.
SELECT COUNT(*)=27 INTO @s07_retirement_source_shape_ok
FROM information_schema.columns
WHERE table_schema=DATABASE() AND (
  (table_name IN ('adv','pub') AND (
    (column_name='email' AND data_type='varchar' AND character_maximum_length=255 AND is_nullable='NO') OR
    (column_name='email_hmac' AND data_type='binary' AND character_maximum_length=32 AND is_nullable='YES') OR
    (column_name='email_cipher' AND data_type='varbinary' AND character_maximum_length=512 AND is_nullable='YES') OR
    (column_name IN ('activation_token_digest','reset_token_digest') AND data_type='binary' AND character_maximum_length=32 AND is_nullable='YES') OR
    (column_name IN ('activation_token_expires','reset_token_expires') AND data_type='datetime' AND datetime_precision=6 AND is_nullable='YES'))) OR
  (table_name IN ('admin','agent','analyst') AND (
    (column_name='login' AND data_type='varchar' AND character_maximum_length=255 AND is_nullable='NO') OR
    (column_name='login_hmac' AND data_type='binary' AND character_maximum_length=32 AND is_nullable='YES') OR
    (column_name='login_cipher' AND data_type='varbinary' AND character_maximum_length=512 AND is_nullable='YES'))) OR
  (table_name IN ('adv_ip','pub_ip') AND (
    (column_name='email' AND data_type='varchar' AND character_maximum_length=255 AND is_nullable='NO') OR
    (column_name='email_hmac' AND data_type='binary' AND character_maximum_length=32 AND is_nullable='YES'))));

SELECT COUNT(*)=5 AND SUM(CASE
  WHEN table_name='adv' AND index_name='email' AND column_name='email' AND
       seq_in_index=1 AND non_unique=0 AND sub_part=20 THEN 1
  WHEN table_name='pub' AND index_name='email' AND column_name='email' AND
       seq_in_index=1 AND non_unique=0 AND sub_part=20 THEN 1
  WHEN table_name='admin' AND index_name='login' AND column_name='login' AND
       seq_in_index=1 AND non_unique=1 AND sub_part=8 THEN 1
  WHEN table_name='agent' AND index_name='login' AND column_name='login' AND
       seq_in_index=1 AND non_unique=0 AND sub_part=8 THEN 1
  WHEN table_name='analyst' AND index_name='analyst_login' AND column_name='login' AND
       seq_in_index=1 AND non_unique=0 AND sub_part IS NULL THEN 1
  ELSE 0 END)=5 INTO @s07_retirement_indexes_ok
FROM information_schema.statistics
WHERE table_schema=DATABASE() AND (
  (table_name='adv' AND index_name='email') OR
  (table_name='pub' AND index_name='email') OR
  (table_name='admin' AND index_name='login') OR
  (table_name='agent' AND index_name='login') OR
  (table_name='analyst' AND index_name='analyst_login'));

SELECT COUNT(*)=5 AND SUM(CASE
  WHEN table_name='adv' AND index_name='adv_email_hmac' AND column_name='email_hmac' AND
       non_unique=0 AND seq_in_index=1 AND sub_part IS NULL THEN 1
  WHEN table_name='pub' AND index_name='pub_email_hmac' AND column_name='email_hmac' AND
       non_unique=0 AND seq_in_index=1 AND sub_part IS NULL THEN 1
  WHEN table_name='admin' AND index_name='admin_login_hmac' AND column_name='login_hmac' AND
       non_unique=0 AND seq_in_index=1 AND sub_part IS NULL THEN 1
  WHEN table_name='agent' AND index_name='agent_login_hmac' AND column_name='login_hmac' AND
       non_unique=0 AND seq_in_index=1 AND sub_part IS NULL THEN 1
  WHEN table_name='analyst' AND index_name='analyst_login_hmac' AND column_name='login_hmac' AND
       non_unique=0 AND seq_in_index=1 AND sub_part IS NULL THEN 1
  ELSE 0 END)=5 INTO @s07_retirement_protected_indexes_ok
FROM information_schema.statistics
WHERE table_schema=DATABASE() AND (
  (table_name='adv' AND index_name='adv_email_hmac') OR
  (table_name='pub' AND index_name='pub_email_hmac') OR
  (table_name='admin' AND index_name='admin_login_hmac') OR
  (table_name='agent' AND index_name='agent_login_hmac') OR
  (table_name='analyst' AND index_name='analyst_login_hmac'));

SELECT COUNT(*)=3 AND SUM(CASE
  WHEN event_object_table='adv' AND trigger_name='trig_adv' AND
       event_manipulation='UPDATE' AND action_timing='AFTER' AND
       LOWER(action_statement) NOT LIKE '%email%' AND
       LOWER(action_statement) NOT LIKE '%login%' THEN 1
  WHEN event_object_table='pub' AND trigger_name='trig_pub' AND
       event_manipulation='UPDATE' AND action_timing='AFTER' AND
       LOWER(action_statement) NOT LIKE '%email%' AND
       LOWER(action_statement) NOT LIKE '%login%' THEN 1
  WHEN event_object_table='pub' AND trigger_name='trig_pub_seller_approval' AND
       event_manipulation='UPDATE' AND action_timing='BEFORE' AND
       LOWER(action_statement) NOT LIKE '%email%' AND
       LOWER(action_statement) NOT LIKE '%login%' THEN 1
  ELSE 0 END)=3 INTO @s07_retirement_trigger_shape_ok
FROM information_schema.triggers
WHERE trigger_schema=DATABASE() AND event_object_table IN (
  'adv','pub','admin','agent','analyst','adv_ip','pub_ip');

SELECT COUNT(*)=0 INTO @s07_retirement_dependent_views_absent
FROM information_schema.view_table_usage
WHERE view_schema=DATABASE() AND table_name IN (
  'adv','pub','admin','agent','analyst','adv_ip','pub_ip');

SELECT COUNT(*)=4 INTO @s07_retirement_legacy_procedures_ok
FROM information_schema.routines
WHERE routine_schema=DATABASE() AND routine_type='PROCEDURE' AND
  routine_name IN ('proc_adv','proc_adv_as','proc_pub','proc_pub_as');

SELECT
  (SELECT COUNT(*)=0 FROM adv WHERE email_hmac IS NULL OR email_cipher IS NULL) AND
  (SELECT COUNT(*)=0 FROM pub WHERE email_hmac IS NULL OR email_cipher IS NULL) AND
  (SELECT COUNT(*)=0 FROM admin WHERE login_hmac IS NULL OR login_cipher IS NULL) AND
  (SELECT COUNT(*)=0 FROM agent WHERE login_hmac IS NULL OR login_cipher IS NULL) AND
  (SELECT COUNT(*)=0 FROM analyst WHERE login_hmac IS NULL OR login_cipher IS NULL)
INTO @s07_retirement_rows_complete;

SELECT COUNT(*)=0 INTO @s07_password_digest_absent
FROM information_schema.columns
WHERE table_schema=DATABASE() AND column_name='passwd_hmac';

SET @s07_retirement_preflight_sql = IF(
  @s07_retirement_source_shape_ok AND @s07_retirement_indexes_ok AND
  @s07_retirement_protected_indexes_ok AND
  @s07_retirement_trigger_shape_ok AND
  @s07_retirement_dependent_views_absent AND
  @s07_retirement_legacy_procedures_ok AND @s07_retirement_rows_complete AND
  @s07_password_digest_absent,
  'DO 0',
  'SELECT s07_account_identifier_retirement_requires_reviewed_complete_source');
PREPARE s07_retirement_preflight_statement FROM @s07_retirement_preflight_sql;
EXECUTE s07_retirement_preflight_statement;
DEALLOCATE PREPARE s07_retirement_preflight_statement;

-- Remove obsolete plaintext credential entry points before dropping their
-- source columns. Protected application paths use the HMAC/cipher pair.
DROP PROCEDURE proc_adv;
DROP PROCEDURE proc_adv_as;
DROP PROCEDURE proc_pub;
DROP PROCEDURE proc_pub_as;

ALTER TABLE adv
  DROP INDEX email,
  DROP COLUMN email,
  MODIFY COLUMN email_hmac BINARY(32) NOT NULL,
  MODIFY COLUMN email_cipher VARBINARY(512) NOT NULL;
ALTER TABLE pub
  DROP INDEX email,
  DROP COLUMN email,
  MODIFY COLUMN email_hmac BINARY(32) NOT NULL,
  MODIFY COLUMN email_cipher VARBINARY(512) NOT NULL;
ALTER TABLE admin
  DROP INDEX login,
  DROP COLUMN login,
  MODIFY COLUMN login_hmac BINARY(32) NOT NULL,
  MODIFY COLUMN login_cipher VARBINARY(512) NOT NULL;
ALTER TABLE agent
  DROP INDEX login,
  DROP COLUMN login,
  MODIFY COLUMN login_hmac BINARY(32) NOT NULL,
  MODIFY COLUMN login_cipher VARBINARY(512) NOT NULL;
ALTER TABLE analyst
  DROP INDEX analyst_login,
  DROP COLUMN login,
  MODIFY COLUMN login_hmac BINARY(32) NOT NULL,
  MODIFY COLUMN login_cipher VARBINARY(512) NOT NULL;

-- Keep non-identity login history while removing its retained plaintext
-- address. email_hmac remains available for protected correlation if needed.
ALTER TABLE adv_ip DROP COLUMN email;
ALTER TABLE pub_ip DROP COLUMN email;

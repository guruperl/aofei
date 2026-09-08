-- S07 additive account-identifier protection migration.
--
-- Preconditions: stop account writers, take and verify a frozen backup, and
-- run the inventory in docs/account-identifier-protection.md. MySQL DDL is not
-- transactional; a failed or partial run must be restored and reviewed rather
-- than retried. This migration does not backfill, cut login reads over, drop a
-- plaintext column, or enable account protection.

-- Require the exact reviewed plaintext source shape and no prior/partial S07
-- columns before the first durable mutation.
SELECT COUNT(*)=12 AND SUM(CASE
  WHEN table_name IN ('adv','pub') AND column_name='email'
       AND data_type='varchar' AND is_nullable='NO' THEN 1
  WHEN table_name IN ('admin','agent','analyst') AND column_name='login'
       AND data_type='varchar' AND is_nullable='NO' THEN 1
  WHEN table_name IN ('adv','pub','admin','agent','analyst') AND column_name='passwd'
       AND data_type='varchar' AND is_nullable='NO' THEN 1
  WHEN table_name IN ('adv_ip','pub_ip') AND column_name='email'
       AND data_type='varchar' AND is_nullable='NO' THEN 1
  ELSE 0 END)=12 INTO @s07_source_shape_ok
FROM information_schema.columns
WHERE table_schema=DATABASE() AND (
  (table_name IN ('adv','pub','adv_ip','pub_ip') AND column_name='email') OR
  (table_name IN ('admin','agent','analyst') AND column_name='login') OR
  (table_name IN ('adv','pub','admin','agent','analyst') AND column_name='passwd'));

SELECT COUNT(*)=0 INTO @s07_target_absent
FROM information_schema.columns
WHERE table_schema=DATABASE() AND (
  (table_name IN ('adv','pub') AND column_name IN ('email_hmac','email_cipher','activation_token_digest','activation_token_expires','reset_token_digest','reset_token_expires')) OR
  (table_name IN ('admin','agent','analyst') AND column_name IN ('login_hmac','login_cipher')) OR
  (table_name IN ('adv_ip','pub_ip') AND column_name='email_hmac') OR
  column_name='passwd_hmac');

SET @s07_preflight_sql = IF(@s07_source_shape_ok AND @s07_target_absent,
  'DO 0',
  'SELECT s07_account_identifier_migration_requires_an_unmodified_plaintext_source');
PREPARE s07_preflight_statement FROM @s07_preflight_sql;
EXECUTE s07_preflight_statement;
DEALLOCATE PREPARE s07_preflight_statement;

ALTER TABLE adv
  ADD COLUMN email_hmac BINARY(32) NULL AFTER email,
  ADD COLUMN email_cipher VARBINARY(512) NULL AFTER email_hmac,
  ADD COLUMN activation_token_digest BINARY(32) NULL AFTER email_cipher,
  ADD COLUMN activation_token_expires DATETIME(6) NULL AFTER activation_token_digest,
  ADD COLUMN reset_token_digest BINARY(32) NULL AFTER activation_token_expires,
  ADD COLUMN reset_token_expires DATETIME(6) NULL AFTER reset_token_digest,
  ADD UNIQUE KEY adv_email_hmac (email_hmac),
  ADD UNIQUE KEY adv_activation_token_digest (activation_token_digest),
  ADD UNIQUE KEY adv_reset_token_digest (reset_token_digest);
ALTER TABLE pub
  ADD COLUMN email_hmac BINARY(32) NULL AFTER email,
  ADD COLUMN email_cipher VARBINARY(512) NULL AFTER email_hmac,
  ADD COLUMN activation_token_digest BINARY(32) NULL AFTER email_cipher,
  ADD COLUMN activation_token_expires DATETIME(6) NULL AFTER activation_token_digest,
  ADD COLUMN reset_token_digest BINARY(32) NULL AFTER activation_token_expires,
  ADD COLUMN reset_token_expires DATETIME(6) NULL AFTER reset_token_digest,
  ADD UNIQUE KEY pub_email_hmac (email_hmac),
  ADD UNIQUE KEY pub_activation_token_digest (activation_token_digest),
  ADD UNIQUE KEY pub_reset_token_digest (reset_token_digest);
ALTER TABLE admin
  ADD COLUMN login_hmac BINARY(32) NULL AFTER login,
  ADD COLUMN login_cipher VARBINARY(512) NULL AFTER login_hmac,
  ADD UNIQUE KEY admin_login_hmac (login_hmac);
ALTER TABLE agent
  ADD COLUMN login_hmac BINARY(32) NULL AFTER login,
  ADD COLUMN login_cipher VARBINARY(512) NULL AFTER login_hmac,
  ADD UNIQUE KEY agent_login_hmac (login_hmac);
ALTER TABLE analyst
  ADD COLUMN login_hmac BINARY(32) NULL AFTER login,
  ADD COLUMN login_cipher VARBINARY(512) NULL AFTER login_hmac,
  ADD UNIQUE KEY analyst_login_hmac (login_hmac);
ALTER TABLE adv_ip ADD COLUMN email_hmac BINARY(32) NULL AFTER email;
ALTER TABLE pub_ip ADD COLUMN email_hmac BINARY(32) NULL AFTER email;

-- Intentionally absent: passwd_hmac. Password verification remains bcrypt in
-- Go; adding a fast deterministic password verifier would weaken the stored
-- credential boundary.

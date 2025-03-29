ALTER TABLE pub
	ADD domain VARCHAR(255) DEFAULT NULL AFTER lastname,
	ADD `daily_balance_id` INT UNSIGNED DEFAULT NULL AFTER domain,
	ADD `total_balance_id` INT UNSIGNED DEFAULT NULL AFTER daily_balance_id,
	ADD FOREIGN KEY (daily_balance_id) REFERENCES adv_balance(balance_id) ON DELETE RESTRICT ON UPDATE CASCADE,
	ADD FOREIGN KEY (total_balance_id) REFERENCES adv_balance(balance_id) ON DELETE RESTRICT ON UPDATE CASCADE;
ALTER TABLE adv
	ADD `domain` VARCHAR(255) AFTER lastname;

ALTER TABLE adv_campaign
	ADD `startx` datetime DEFAULT NULL AFTER foreign_id, 
	ADD `endx` datetime DEFAULT NULL AFTER startx, 
	ADD `target_type` enum("App","Web") DEFAULT NULL AFTER endx, 
	ADD `description` TEXT DEFAULT NULL AFTER target_type,
	ADD `iurl` TEXT DEFAULT NULL AFTER description;

ALTER TABLE adv_item
	DROP COLUMN size_id;
ALTER TABLE adv_creative
	ADD size_id int unsigned NOT NULL AFTER item_id;
ALTER TABLE adv_creative
	ADD key `size_id` (size_id);

ALTER TABLE pub_site
	ADD `site_type` enum("App","Web") DEFAULT NULL AFTER site_url, 
	ADD `foreign_id` VARCHAR(255) DEFAULT NULL AFTER site_url, 
	ADD `description` TEXT DEFAULT NULL AFTER site_type;
ALTER TABLE pub
	ADD `domain` VARCHAR(255) AFTER lastname;
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


DROP VIEW IF EXISTS WHITEfalsefalse;
DROP VIEW IF EXISTS WHITEfalsetrue;
DROP VIEW IF EXISTS WHITEtruefalse;
DROP VIEW IF EXISTS WHITEtruetrue;
DROP VIEW IF EXISTS VIEWfalsefalse;
DROP VIEW IF EXISTS VIEWfalsefalsefalsefalse;
DROP VIEW IF EXISTS VIEWfalsefalsefalsetrue;
DROP VIEW IF EXISTS VIEWfalsefalsetruefalse;
DROP VIEW IF EXISTS VIEWfalsefalsetruetrue;
DROP VIEW IF EXISTS VIEWfalsetrue;
DROP VIEW IF EXISTS VIEWfalsetruefalsefalse;
DROP VIEW IF EXISTS VIEWfalsetruefalsetrue;
DROP VIEW IF EXISTS VIEWfalsetruetruefalse;
DROP VIEW IF EXISTS VIEWfalsetruetruetrue;
DROP VIEW IF EXISTS VIEWtruefalse;
DROP VIEW IF EXISTS VIEWtruefalsefalsefalse;
DROP VIEW IF EXISTS VIEWtruefalsefalsetrue;
DROP VIEW IF EXISTS VIEWtruefalsetruefalse;
DROP VIEW IF EXISTS VIEWtruefalsetruetrue;
DROP VIEW IF EXISTS VIEWtruetrue;
DROP VIEW IF EXISTS VIEWtruetruefalsefalse;
DROP VIEW IF EXISTS VIEWtruetruefalsetrue;
DROP VIEW IF EXISTS VIEWtruetruetruefalse;
DROP VIEW IF EXISTS VIEWtruetruetruetrue;
DROP VIEW IF EXISTS ViewGroupSlot;
DROP VIEW IF EXISTS ViewSlot;
DROP VIEW IF EXISTS ViewSlotOpen;
DROP VIEW IF EXISTS ViewTaoSlot;

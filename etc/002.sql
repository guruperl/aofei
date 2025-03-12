ALTER TABLE adv_campaign
	ADD `startx` datetime DEFAULT NULL AFTER foreign_id, 
	ADD `endx` datetime DEFAULT NULL AFTER startx, 
	ADD `target_type` enum("App","Web") DEFAULT NULL AFTER endx, 
	ADD `description` TEXT DEFAULT NULL AFTER target_type,
	ADD `extlink` TEXT DEFAULT NULL AFTER description;

ALTER TABLE adv_item
	DROP COLUMN size_id;
ALTER TABLE adv_creative
	ADD size_id int unsigned NOT NULL AFTER item_id;
ALTER TABLE adv_creative
	ADD key `size_id` (size_id);

ALTER TABLE pub_site
	ADD `site_type` enum("App","Web") DEFAULT NULL AFTER site_url, 
	ADD `description` TEXT DEFAULT NULL AFTER site_type;


DROP VIEW IF EXISTS WHITEfalsefalse          |
DROP VIEW IF EXISTS WHITEfalsetrue           |
DROP VIEW IF EXISTS WHITEtruefalse           |
DROP VIEW IF EXISTS WHITEtruetrue            |
| VIEWfalsefalse           |
| VIEWfalsefalsefalsefalse |
| VIEWfalsefalsefalsetrue  |
| VIEWfalsefalsetruefalse  |
| VIEWfalsefalsetruetrue   |
| VIEWfalsetrue            |
| VIEWfalsetruefalsefalse  |
| VIEWfalsetruefalsetrue   |
| VIEWfalsetruetruefalse   |
| VIEWfalsetruetruetrue    |
| VIEWtruefalse            |
| VIEWtruefalsefalsefalse  |
| VIEWtruefalsefalsetrue   |
| VIEWtruefalsetruefalse   |
| VIEWtruefalsetruetrue    |
| VIEWtruetrue             |
| VIEWtruetruefalsefalse   |
| VIEWtruetruefalsetrue    |
| VIEWtruetruetruefalse    |
| VIEWtruetruetruetrue   

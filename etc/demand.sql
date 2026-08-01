-- Deterministic local-only sample data. These identities and credentials are
-- public development fixtures and must never be copied into production.
START TRANSACTION;

INSERT INTO add_address
  (address_id, company, contact, contact_email, url, created)
VALUES
  (1, 'Example Advertiser', 'Local Advertiser', 'advertiser@example.test', 'https://advertiser.example.test', NOW()),
  (2, 'Example Publisher', 'Local Publisher', 'publisher@example.test', 'https://publisher.example.test', NOW());

INSERT INTO admin (admin_id, login, passwd, active, created) VALUES
  (1, 'admin_local', '$2y$10$VL/dlk47CYtyQtkXhY4kVOo8An/ksArTA5b6togvq0W8YEnvuWZbG', 'Yes', NOW());

INSERT INTO adv
  (adv_id, email, passwd, firstname, lastname, domain, address_id, active, access_order, created)
VALUES
  (1, 'advertiser@example.test', '$2y$10$VL/dlk47CYtyQtkXhY4kVOo8An/ksArTA5b6togvq0W8YEnvuWZbG',
   'Local', 'Advertiser', 'advertiser.example.test', 1, 'Yes', 'Black', NOW());

INSERT INTO adv_campaign
  (campaign_id, adv_id, campaign_name, foreign_id, target_type, description, access_order, active, created)
VALUES
  (1, 1, 'Example Web Campaign', 'example-web', 'Web', 'Synthetic local web campaign', 'Inherit', 'Yes', NOW()),
  (2, 1, 'Example App Campaign', 'example-app', 'App', 'Synthetic local app campaign', 'Inherit', 'Yes', NOW());

INSERT INTO adv_item
  (item_id, campaign_id, item_name, item_click, cost_type, cost, startx, page_cap,
   fl_sitetypes, access_order, qa_item, fl_slot, fl_language, fl_device,
   fl_position, qa_creative, qa_expnd, qa_mime, channel_order, active, created)
VALUES
  (1, 1, 'Example Web Ad Group', 'https://advertiser.example.test/landing', 'CPM', 1.0, NOW(), 2,
   'App,Web', 'Inherit', 149796, 2996629, 'EN', '0,1,2,3,4,5,6,7',
   '0,1,2,3,4,5,6,7', '0', '0', '4', 'Black', 'Yes', NOW()),
  (2, 2, 'Example App Ad Group', 'https://advertiser.example.test/app', 'CPM', 6.0, NOW(), 2,
   'App,Web', 'Inherit', 149796, 2996629, 'EN', '0,1,2,3,4,5,6,7',
   '0,1,2,3,4,5,6,7', '0', '0', '4', 'Black', 'Yes', NOW());

INSERT INTO adv_creative
  (creative_id, creative_name, item_id, size_id, media_type, content, weight, active, created)
VALUES
  (1, 'Example Web Creative', 1, 4194368, 'Banner', 'https://cdn.example.test/creative-web-64.png', 1, 'Yes', NOW()),
  (2, 'Example App Creative', 2, 4194368, 'Banner', 'https://cdn.example.test/creative-app-64.png', 1, 'Yes', NOW());

INSERT INTO pub
  (pub_id, email, passwd, firstname, lastname, domain, address_id, active, access_order, created)
VALUES
  (2, 'publisher@example.test', '$2y$10$VL/dlk47CYtyQtkXhY4kVOo8An/ksArTA5b6togvq0W8YEnvuWZbG',
   'Local', 'Publisher', 'default', 2, 'Yes', 'Black', NOW());

INSERT INTO pub_site
  (site_id, pub_id, site_name, site_url, foreign_id, site_type, description, access_order, active, created)
VALUES
  (1, 2, 'Example App', 'com.example.publisher', 'com.example.publisher', 'App', 'Synthetic local app', 'Inherit', 'Yes', NOW()),
  (2, 2, 'Example Website', 'https://publisher.example.test', 'publisher.example.test', 'Web', 'Synthetic local website', 'Inherit', 'Yes', NOW());

INSERT INTO pub_slot
  (slot_id, site_id, slot_name, bidfloor, size_id, qa_slot, fl_item, qa_language,
   qa_device, qa_position, fl_creative, fl_expnd, fl_mime, channel_order, active, created)
VALUES
  (1, 1, 'defaultSlot', 0, 4194368, 0, 0, 'EN', '0', '0', '0', '0,1,2,3,4,5', '0,1,2,3,4', 'Black', 'Yes', NOW()),
  (2, 2, 'defaultSlot', 0, 4194368, 0, 0, 'EN', '0', '0', '0', '0,1,2,3,4,5', '0,1,2,3,4', 'Black', 'Yes', NOW());

COMMIT;

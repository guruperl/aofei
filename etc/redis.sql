SADD ifa:pubID:value
SADD openid:pubID:value
SADD h5:pubID:value

create table fcap (
	ts timestamp,
	user_id timestamp,
	act tinyint,
	total tinyint,
	ym tinyint,
	dhm smallint,
	ls smallint
) TAGS (campaign_id int);
HADD fcap:value:itemID:value

-- SELECT tn.campaign_id, an.attrname_id, tv.value_id
-- FROM adv_targetvalue tv
-- INNER JOIN adv_targetname tn USING (targetname_id)
-- INNER JOIN adv_attrname an USING (attrname_id)
HADD target:campaignID:attrnameID:valueID

-- SELECT * FROM ViewTaoSlot
HADD slot:release:packed
CREATE TABLE slot (
	ts timestamp,
	slot_id int,
	adv_id int,
	campaign_id int,
	item_id int,
	cost_type tinyint,
	price float,
	endx int,
	cpm_fc smallint,
	cpm_length int,
	cpm_throttle int,
	cpc_fc smallint,
	cpc_length	int
) TAGS (release int);
HADD slot:release:packed

-- SELECT item_id, creative_id, content, weight
-- FROM adv_creative c
-- WHERE active='Yes'
create table item (
	ts timestamp,
	creative_id	int,
	weight	float,
	item_click   binary(64),
	cpc_fc smallint,
	cpc_length int,
	content binary(504)
) TAGS (item_id int);
HADD item:itemID:creativeID:weight

create table rawclick (
	click_id timestamp,
	imp_id	timestamp,
	raw_id	timestamp,
	user_id timestamp,
	slot_id int,
	creative_id int,
	item_id int,
	cost_type tinyint,
	price float,
	campaign_id int,
	adv_id int
) TAGS (pub_id int, site_id int);

create table rawimp (
	imp_id timestamp,
	raw_id	timestamp,
	slot_id int,
	creative_id int,
	item_id int,
	cost_type tinyint,
	price float,
	campaign_id int,
	adv_id int
) TAGS (pub_id int, site_id int);

create table rawlog (
	raw_id timestamp,
	user_id	timestamp,
	ip32 int,
	pzua int,
	tag0 int,
	tag1 int,
	tag2 int,
	tag3 int,
	tag4 int,
	tag5 int,
	tag6 int,
	tag7 int,
	tag8 int,
	tag9 int
) TAGS (pub_id int, site_id int);

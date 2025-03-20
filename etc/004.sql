-- possible ads for a slot, including the black-white list by campaign
DROP VIEW IF EXISTS SimpleRedisWeb;
DROP VIEW IF EXISTS SimpleRedisApp;

CREATE VIEW SimpleRedisApp AS
SELECT a.adv_id, c.campaign_id, t.slot_id, v.creative_id, v.weight, i.item_id, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_creative v USING (size_id)
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)
WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
AND   a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (  i.endx >= NOW() OR (  i.endx IS NULL))
AND s.site_type = 'App' AND c.target_type = 'App';

CREATE VIEW SimpleRedisWeb AS
SELECT a.adv_id, c.campaign_id, t.slot_id, v.creative_id, v.weight, i.item_id, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length
FROM pub_slot t
INNER JOIN pub_site s USING (site_id)
INNER JOIN pub      p USING (pub_id)
INNER JOIN adv_creative v USING (size_id)
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
INNER JOIN adv      a USING (adv_id)
WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
AND   a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
AND (i.startx <= NOW() OR (i.startx IS NULL))
AND (  i.endx >= NOW() OR (  i.endx IS NULL))
AND s.site_type = 'Web' AND c.target_type = 'Web';
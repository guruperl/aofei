DROP PROCEDURE IF EXISTS proc_creative;
-- proc_creative returns the list of slots which host the creative
--
-- if i.access_order='Inherit' and c.access_order='Inherit', slots are from adv
-- if i.access_order='Inherit' and c.access_order!='Inherit', slots are from adv_campaign
-- otherwise, slots are from pub_slot, slots are from adv_item
--
-- the logic is if not inherited, it overrides adv or adv_campaing
--
DELIMITER ;;
CREATE PROCEDURE proc_creative(IN creativeID INT UNSIGNED)
BEGIN
  DECLARE advID, campaignID, itemID, sizeID, cpmLength, cpmThrottle, cpcLength INT UNSIGNED;
  DECLARE weight FLOAT;
  DECLARE costType ENUM('ROI','CPM','CPC','CPA');
  DECLARE cost DOUBLE;
  DECLARE cpmFc, cpcFc SMALLINT UNSIGNED;
  DECLARE aclSiteTypes SET("App","Web");
  DECLARE aAccessOrder ENUM('White','Black');
  DECLARE cAccessOrder. iAccessOrder ENUM('White','Black','Inherit');

  SELECT a.adv_id, c.campaign_id, i.item_id, v.size_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length, a.access_order, c.access_order, i.fl_sitetypes, i.access_order
  INTO advID, campaignID, itemID, sizeID, weight, costType, cost, cpmFc, cpmLength, cpmThrottle, cpcFc, cpcLength, aAccessOrder, cAccessOrder, aclSiteTypes, iAccessOrder
  FROM adv_creative       v
  INNER JOIN adv_item     i USING (item_id)
  INNER JOIN adv_campaign c USING (campaign_id)
  INNER JOIN adv          a USING (adv_id)
  WHERE a.active="Yes" AND c.active="Yes" AND v.creative_id = creativeID
  AND (i.startx <= NOW() OR (i.startx IS NULL))
  AND (  i.endx >= NOW() OR (  i.endx IS NULL));

  IF (iAccessOrder='Inherit' AND cAccessOrder='Inherit' AND aAccessOrder="Black")
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=advID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  ELSEIF (iAccessOrder='Inherit' AND cAccessOrder='Inherit' AND aAccessOrder="White")
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=advID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NOT NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  ELSEIF (iAccessOrder='Inherit' AND cAccessOrder='Black')
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=campaignID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  ELSEIF (iAccessOrder='Inherit' AND cAccessOrder='White')
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=campaignID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NOT NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  ELSEIF (iAccessOrder='Black')
  THEN
    SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=42 AND ac.entity_id=itemID AND (ac.othertype_id=31 AND ac.other_id=s.site_id))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  ELSEIF (iAccessOrder='White')
  THEN
    SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=42 AND ac.entity_id=itemID AND (ac.othertype_id=31 AND ac.other_id=s.site_id))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NOT NULL
    AND FIND_IN_SET(s.site_type, aclSiteTypes) > 0;
  END IF;
END ;;
DELIMITER ;
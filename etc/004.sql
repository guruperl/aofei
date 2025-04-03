DROP PROCEDURE IF EXISTS proc_creative;
DELIMITER ;;
CREATE PROCEDURE proc_creative(IN creativeID INT UNSIGNED)
BEGIN
  DECLARE advID, campaignID, itemID, sizeID, cpmLength, cpmThrottle, cpcLength INT UNSIGNED;
  DECLARE weight FLOAT;
  DECLARE costType ENUM('ROI','CPM','CPC','CPA');
  DECLARE cost DOUBLE;
  DECLARE cpmFc, cpcFc SMALLINT UNSIGNED;
  DECLARE targetType ENUM("App","Web");
  DECLARE aAccessOrder ENUM('White','Black');
  DECLARE cAccessOrder ENUM('White','Black','Inherit');

  SELECT a.adv_id, c.campaign_id, i.item_id, v.size_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length, a.access_order, c.access_order, c.target_type
  INTO advID, campaignID, itemID, sizeID, weight, costType, cost, cpmFc, cpmLength, cpmThrottle, cpcFc, cpcLength, aAccessOrder, cAccessOrder, targetType
  FROM adv_creative       v
  INNER JOIN adv_item     i USING (item_id)
  INNER JOIN adv_campaign c USING (campaign_id)
  INNER JOIN adv          a USING (adv_id)
  WHERE a.active="Yes" AND c.active="Yes" AND v.creative_id = creativeID
  AND (i.startx <= NOW() OR (i.startx IS NULL))
  AND (  i.endx >= NOW() OR (  i.endx IS NULL));

  IF (cAccessOrder='Inherit' AND aAccessOrder="Black")
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=advID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NULL
    AND s.site_type = targetType;
  ELSEIF (cAccessOrder='Inherit' AND aAccessOrder="White")
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=advID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NOT NULL
    AND s.site_type = targetType;
  ELSEIF (cAccessOrder='Black')
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=campaignID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NULL
    AND s.site_type = targetType;
  ELSEIF (cAccessOrder='White')
  THEN
	  SELECT p.pub_id, s.site_id, t.slot_id, sizeID, advID, campaignID, itemID, creativeID, weight, costType, cost, cpmFC, cpmLength, cpmThrottle, cpcFc, cpcLength
    FROM pub_slot t
    INNER JOIN pub_site s USING (site_id)
    INNER JOIN pub      p USING (pub_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=campaignID AND ((ac.othertype_id=3 AND ac.other_id=p.pub_id) OR (ac.othertype_id=31 AND ac.other_id=s.site_id)))
    WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes"
    AND ac.entity_id IS NOT NULL
    AND s.site_type = targetType;
  END IF;
END ;;
DELIMITER ;
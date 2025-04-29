DROP PROCEDURE IF EXISTS proc_slot;
-- proc_slot provides the list of ads for a given slot and size, taking into account
-- demands may have blocked or allowed the supply;
--
-- if i.access_order='Inherit' and c.access_order='Inherit', creatives are from adv
-- if i.access_order='Inherit' and c.access_order!='Inherit', creatives are from adv_campaign
-- otherwise, slots are from pub_slot, creatives are from adv_item
-- we union them together
--
-- currently adv and campaign can block pub (3) and site (31) 
-- item can only block site (31)
-- 
-- since block/white listing demands by the supply is determined in bid request,
-- we should not take care of it here.  
--
DELIMITER ;;
CREATE PROCEDURE proc_slot(IN slotID INT UNSIGNED, IN sizeID INT UNSIGNED)
BEGIN
  DECLARE pubID INT UNSIGNED;
  DECLARE siteID INT UNSIGNED;
  DECLARE siteType ENUM("App","Web");
  SELECT s.site_type, p.pub_id, s.site_id INTO siteType, pubID, siteID
  FROM pub_slot t
  INNER JOIN pub_site s USING (site_id)
  INNER JOIN pub      p USING (pub_id)
  WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes" AND t.slot_id = slotID;
  IF (siteID>0)
  THEN
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=a.adv_id AND ((ac.othertype_id=3 AND ac.other_id=pubID) OR (ac.othertype_id=31 AND ac.other_id=siteID)))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( c.access_order ='Inherit' AND i.access_order = 'Inherit' AND ((a.access_order="Black" AND ac.entity_id IS NULL) OR (a.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND (i.startx <= NOW() OR (i.startx IS NULL))
    AND (  i.endx >= NOW() OR (  i.endx IS NULL))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID
    UNION
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=c.campaign_id AND ((ac.othertype_id=3 AND ac.other_id=pubID) OR (ac.othertype_id=31 AND ac.other_id=siteID)))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( c.access_order !='Inherit' AND i.access_order = 'Inherit' AND ((c.access_order="Black" AND ac.entity_id IS NULL) OR (c.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND (i.startx <= NOW() OR (i.startx IS NULL))
    AND (  i.endx >= NOW() OR (  i.endx IS NULL))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID
    UNION
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=42 AND ac.entity_id=i.item_id AND (ac.othertype_id=31 AND ac.other_id=siteID))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( i.access_order !='Inherit' AND ((i.access_order="Black" AND ac.entity_id IS NULL) OR (i.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND (i.startx <= NOW() OR (i.startx IS NULL))
    AND (  i.endx >= NOW() OR (  i.endx IS NULL))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID;
  END IF;
END ;;
DELIMITER ;


DROP PROCEDURE IF EXISTS proc_slotall;
-- proc_slotall the same as proc_slot, but without the time constraints
DELIMITER ;;
CREATE PROCEDURE proc_slotall(IN slotID INT UNSIGNED, IN sizeID INT UNSIGNED)
BEGIN
  DECLARE pubID INT UNSIGNED;
  DECLARE siteID INT UNSIGNED;
  DECLARE siteType ENUM("App","Web");
  SELECT s.site_type, p.pub_id, s.site_id INTO siteType, pubID, siteID
  FROM pub_slot t
  INNER JOIN pub_site s USING (site_id)
  INNER JOIN pub      p USING (pub_id)
  WHERE p.active="Yes" AND s.active="Yes" AND t.active="Yes" AND t.slot_id = slotID;
  IF (siteID>0)
  THEN
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length, i.startx, i.endx
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=4 AND ac.entity_id=a.adv_id AND ((ac.othertype_id=3 AND ac.other_id=pubID) OR (ac.othertype_id=31 AND ac.other_id=siteID)))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( c.access_order ='Inherit' AND i.access_order = 'Inherit' AND ((a.access_order="Black" AND ac.entity_id IS NULL) OR (a.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID
    UNION
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length, i.startx, i.endx
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=41 AND ac.entity_id=c.campaign_id AND ((ac.othertype_id=3 AND ac.other_id=pubID) OR (ac.othertype_id=31 AND ac.other_id=siteID)))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( c.access_order !='Inherit' AND i.access_order = 'Inherit' AND ((c.access_order="Black" AND ac.entity_id IS NULL) OR (c.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID
    UNION
    SELECT a.adv_id, c.campaign_id, i.item_id, v.creative_id, v.weight, i.cost_type, i.cost, i.cpm_fc, i.cpm_length, i.cpm_throttle, i.cpc_fc, i.cpc_length, i.startx, i.endx
    FROM adv_creative v
    INNER JOIN adv_item i USING (item_id)
    INNER JOIN adv_campaign c USING (campaign_id)
    INNER JOIN adv      a USING (adv_id)
    LEFT JOIN ac ON (ac.entitytype_id=42 AND ac.entity_id=i.item_id AND (ac.othertype_id=31 AND ac.other_id=siteID))
    WHERE a.active="Yes" AND c.active="Yes" AND i.active="Yes" AND v.active="Yes"
    AND ( i.access_order !='Inherit' AND ((i.access_order="Black" AND ac.entity_id IS NULL) OR (i.access_order="White" AND ac.entity_id IS NOT NULL)))
    AND FIND_IN_SET(siteType,i.fl_sitetypes) > 0
    AND v.size_id=sizeID;
  END IF;
END ;;
DELIMITER ;

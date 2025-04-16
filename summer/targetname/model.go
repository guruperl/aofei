package targetname

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) getItemID(extra ...url.Values) string {
	itemID := self.ARGS.Get("item_id")
	if itemID == "" {
		itemID = self.ProperValue("item_id", extra[0])
	}
	return itemID
}

func (self *Model) TopicsDmas(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT t.city_id, t.city_name, d.dma_id, d.metro_code, tmp.value_id
FROM def_dma d
INNER JOIN def_city t USING (city_id)
INNER JOIN def_state s USING (state_id)
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.item_id=? AND an.attrname='dma'
) tmp ON (d.dma_id=tmp.value_id)
WHERE c.active="Yes"`, self.getItemID(extra...))
}

func (self *Model) TopicsCities(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT s.state_id, s.state_name, t.city_id, t.city_name, tmp.value_id
FROM def_city t
INNER JOIN def_state s USING (state_id)
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.item_id=? AND an.attrname='city'
) tmp ON (t.city_id=tmp.value_id)
WHERE c.active="Yes"`, self.getItemID(extra...))
}

func (self *Model) TopicsCountries(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT c.country_id, c.country_name, tmp.value_id
FROM def_country c
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.item_id=? AND an.attrname='country'
) tmp ON (c.country_id=tmp.value_id)`, self.getItemID(extra...))
}

func (self *Model) TopicsStates(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT c.country_id, c.country_name, s.state_id, s.state_code, s.state_name, tmp.value_id
FROM def_state s
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.item_id=? AND an.attrname='state'
) tmp ON (s.state_id=tmp.value_id)
WHERE c.active="Yes"`, self.getItemID(extra...))
}

func (self *Model) EditACL(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT fl_sitetypes, access_order
FROM adv_item
WHERE item_id=?`, self.getItemID(extra...))
}

func (self *Model) TopicsACL(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT s.site_id, s.site_type, p.domain, s.foreign_id, ac.other_id 
FROM pub_site s
INNER JOIN pub p USING (pub_id)
LEFT JOIN ac ac ON (ac.entitytype_id=42 AND ac.entity_id=? AND ac.othertype_id=31 AND s.site_id=ac.other_id)
WHERE domain != "default" AND SUBSTR(foreign_id,1,7) != "default"`, self.getItemID(extra...))
}

func (self *Model) TopicsIsps(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT s.isp_id, s.isp_name, tmp.value_id
FROM def_isp s
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.item_id=? AND an.attrname='isp'
) tmp ON (s.isp_id=tmp.value_id)
WHERE s.counts>=100 and isp_name!=''`, self.getItemID(extra...))
}

func (self *Model) TopicsCustom(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT an.attrname_id, an.attrname, av.attrvalue_id, av.value, ta.value_id
FROM adv_attrname an
INNER JOIN adv_attrvalue av USING (attrname_id)
LEFT JOIN adv_targetname tn 
	ON (an.attrname_id=tn.attrname_id AND tn.item_id=?)
LEFT JOIN adv_targetvalue ta 
	ON (tn.targetname_id=ta.targetname_id AND av.attrvalue_id=ta.value_id)
WHERE an.adv_id=? AND an.attrname_id>=10000`, self.getItemID(extra...), self.ARGS.Get("adv_id"))
}

func (self *Model) Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	itemID := self.getItemID(extra...)

	// for acl
	if ARGS.Get("fl_sitetypes") == "" {
		ARGS.Set("fl_sitetypes", "App,Web")
	}
	if ARGS.Get("access_order") == "" {
		ARGS.Set("access_order", "Inherit")
	}
	err := self.DoSQL(`UPDATE adv_item 
SET fl_sitetypes=?, access_order=?
WHERE item_id=?`, ARGS.Get("fl_sitetypes"), ARGS.Get("access_order"), itemID)
	if err != nil {
		return err
	}
	err = self.DoSQL(`
DELETE FROM ac WHERE entitytype_id=42 AND othertype_id=31 AND entity_id=?`, itemID)
	if err != nil {
		return err
	}
	if ARGS.Get("site_id") != "" && ARGS.Get("access_order") != "Inherit" {
		for _, siteID := range ARGS["site_id"] {
			if err = self.DoSQL(`
INSERT INTO ac (entitytype_id, othertype_id, entity_id, other_id)
VALUES (42, 31, ?, ?)`, itemID, siteID); err != nil {
				return err
			}
		}
		delete(ARGS, "other_id")
	}
	// end acl

	// Delete all the old targetname
	err = self.DoSQL(`
DELETE FROM adv_targetname WHERE item_id=?`, itemID)
	if err != nil {
		return err
	}

	hash := make(map[string]string)
	for attrname, attrnameID := range summer.AttrValue {
		if _, ok := ARGS[attrname]; ok {
			if ARGS.Get(attrname) == "" {
				continue
			}
			hash[attrname] = strconv.FormatUint(uint64(attrnameID), 10)
		}
	}
	for k := range ARGS {
		parts := strings.Split(k, "_")
		if len(parts) < 2 {
			continue
		}
		id := parts[len(parts)-1]
		if summer.IsDigit(id) {
			hash[k] = id
		}
	}

	data := ``
	for attrname, attrnameID := range hash {
		err = self.DoSQL(`
INSERT INTO adv_targetname (item_id, attrname_id) VALUES (?, ?)`, itemID, attrnameID)
		if err != nil {
			return err
		}
		targetnameID := strconv.FormatInt(self.LastID, 10)
		total := 0
		for _, id := range ARGS[attrname] {
			if summer.IsDigit(id) {
				d, _ := strconv.Atoi(id)
				id = strconv.FormatInt(int64(uint32(d)), 10)
				data += `(` + targetnameID + `, ` + id + `),`
				total++
				*self.LISTS = append(*self.LISTS, map[string]interface{}{"item_id": itemID, "attrname_id": attrnameID, "value_id": id})
			}
		}
		if total == 0 {
			err = self.DoSQL(`
DELETE FROM adv_targetname WHERE targetname_id=?`, targetnameID)
			if err != nil {
				return err
			}
		}
	}
	length := len(data)
	if length == 0 {
		return nil
	}

	return self.DoSQL(`
INSERT INTO adv_targetvalue (targetname_id, value_id) VALUES ` + data[:length-1])
}

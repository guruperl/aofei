package targetname

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) TopicsDmas(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT t.city_id, t.city_name, d.dma_id, d.metro_code, tmp.value_id
FROM def_dma d
INNER JOIN def_city t USING (city_id)
INNER JOIN def_state s USING (state_id)
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.campaign_id=? AND an.attrname='dma'
) tmp ON (d.dma_id=tmp.value_id)
WHERE c.active="Yes"`, self.ProperValue("campaign_id", extra[0]))
}

func (self *Model) TopicsCities(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT s.state_id, s.state_name, t.city_id, t.city_name, tmp.value_id
FROM def_city t
INNER JOIN def_state s USING (state_id)
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.campaign_id=? AND an.attrname='city'
) tmp ON (t.city_id=tmp.value_id)
WHERE c.active="Yes"`, self.ProperValue("campaign_id", extra[0]))
}

func (self *Model) TopicsStates(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT c.country_id, c.country_name, s.state_id, s.state_code, s.state_name, tmp.value_id
FROM def_state s
INNER JOIN def_country c USING (country_id)
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.campaign_id=? AND an.attrname='state'
) tmp ON (s.state_id=tmp.value_id)
WHERE c.active="Yes"`, self.ProperValue("campaign_id", extra[0]))
}

func (self *Model) TopicsIsps(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT s.isp_id, s.isp_name, tmp.value_id
FROM def_isp s
LEFT JOIN (
	SELECT tn.targetname_id, tn.attrname_id, tv.value_id
	FROM adv_targetname tn
	INNER JOIN adv_targetvalue tv USING (targetname_id)
	INNER JOIN adv_attrname an USING (attrname_id)
	WHERE tn.campaign_id=? AND an.attrname='isp'
) tmp ON (s.isp_id=tmp.value_id)
WHERE s.counts>=100 and isp_name!=''`, self.ProperValue("campaign_id", extra[0]))
}

func (self *Model) TopicsCustom(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS,
		`SELECT an.attrname_id, an.attrname, av.attrvalue_id, av.value, ta.value_id
FROM adv_attrname an
INNER JOIN adv_attrvalue av USING (attrname_id)
LEFT JOIN adv_targetname tn 
	ON (an.attrname_id=tn.attrname_id AND tn.campaign_id=?)
LEFT JOIN adv_targetvalue ta 
	ON (tn.targetname_id=ta.targetname_id AND av.attrvalue_id=ta.value_id)
WHERE an.adv_id=? AND an.attrname_id>=10000`,
		self.ProperValue("campaign_id", extra[0]), self.ARGS.Get("adv_id"))
}

func (self *Model) Insert(extra ...url.Values) error {
	ARGS := self.ARGS
	campaign_id := ARGS.Get("campaign_id")

	data := ``
	err := self.DoSQL(
		`DELETE FROM adv_targetname WHERE campaign_id=?`, campaign_id)
	if err != nil {
		return err
	}

	hash := make(map[string]string)
	for attrname, attrname_id := range pzutil.AttrValue {
		if _, ok := ARGS[attrname]; ok {
			hash[attrname] = strconv.FormatUint(uint64(attrname_id), 10)
		}
	}
	for k, _ := range ARGS {
		parts := strings.Split(k, "_")
		if len(parts) < 2 {
			continue
		}
		id := parts[len(parts)-1]
		if pzutil.IsDigit(id) {
			hash[k] = id
		}
	}

	for attrname, attrname_id := range hash {
		err = self.DoSQL(
			`INSERT INTO adv_targetname (campaign_id, attrname_id) VALUES (?, ?)`,
			campaign_id, attrname_id)
		if err != nil {
			return err
		}
		targetname_id := strconv.FormatInt(self.LastID, 10)
		total := 0
		for _, id := range ARGS[attrname] {
			if pzutil.IsDigit(id) {
				data += `(` + targetname_id + `, ` + id + `),`
				total++
				*self.LISTS = append(*self.LISTS, map[string]interface{}{"campaign_id": campaign_id, "attrname_id": attrname_id, "value_id": id})
			}
		}
		if total == 0 {
			err = self.DoSQL(
				`DELETE FROM adv_targetname WHERE targetname_id=?`, targetname_id)
			if err != nil {
				return err
			}
		}
	}

	length := len(data)
	if length == 0 {
		return nil
	}

	return self.DoSQL(
		`INSERT INTO adv_targetvalue (targetname_id, value_id) VALUES ` + data[:length-1])
}

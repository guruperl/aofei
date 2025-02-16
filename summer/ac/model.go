//NEW
// pub.3, accepts adv.  4
// pub.3, accepts camp  41
// adv.4, accepts pub.  3
// adv.4, accepts site  31

//             1 | admin        | admin_id    |
//             3 | pub          | pub_id      |
//             4 | adv          | adv_id      |
//             5 | anon         | anon_id     |
//            31 | pub_site     | site_id     |
//            32 | pub_slot     | slot_id     |
//            41 | adv_campaign | campaign_id |
//            42 | adv_item     | item_id     |

//OLD
//publisher 3, can block advertiser 4
//publisher's site 31, can block advertiser 4
//advertiser 4, can block site 31
//advertiser's campaign 41, can block site 31

package ac

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Delete(extra ...url.Values) error {
	ARGS := self.ARGS

	return self.Do_sql(
		`DELETE FROM ac
WHERE ac_id=? AND entitytype_id=? AND entity_id=?`,
		ARGS.Get("ac_id"), ARGS.Get("entitytype_id"), ARGS.Get("entity_id"))
}

func (self *Model) Inserts(extra ...url.Values) error {
	ARGS := self.ARGS

	ads := make([]string, 0)
	if ARGS.Get("adv_ids") != "" {
		found := make(map[string]bool)
		for _, id := range ARGS["adv_ids"] {
			if found[id] {
				continue
			}
			found[id] = true
			if pzutil.IsDigit(id) {
				ads = append(ads, id)
			}
		}
	}
	campaigns := make([]string, 0)
	if ARGS.Get("campaign_ids") != "" {
		found := make(map[string]bool)
		for _, id := range ARGS["campaign_ids"] {
			if found[id] {
				continue
			}
			found[id] = true
			if pzutil.IsDigit(id) {
				ads = append(ads, id)
			}
		}
	}

	ref := make(map[string]bool)
	if len(ads) > 0 && len(campaigns) > 0 {
		lists := make([]map[string]interface{}, 0)
		err := self.Select_sql(&lists,
			`SELECT campaign_id
FROM adv_campaign
WHERE campaign_id IN (`+strings.Join(ads, ",")+`) AND adv_id IN (`+strings.Join(campaigns, ",")+`))`)
		if err != nil {
			return err
		}
		for _, item := range lists {
			ref[item["campaign_id"].(string)] = true
		}
	}

	str := `INSERT INTO ac (entitytype_id, entity_id, othertype_id, other_id) VALUES`
	n := 0
	if ARGS.Get("adv_ids") != "" {
		found_adv := make(map[string]bool)
		for _, adv_id := range ARGS["adv_ids"] {
			if found_adv[adv_id] {
				continue
			}
			found_adv[adv_id] = true
			if pzutil.IsDigit(adv_id) {
				n++
				str += fmt.Sprintf(" (%s, %s, 4, %s),", ARGS.Get("entitytype_id"), ARGS.Get("entity_id"), adv_id)
			}
		}
	}
	if ARGS.Get("campaign_ids") != "" {
		found_campaign := make(map[string]bool)
		for _, campaign_id := range ARGS["campaign_ids"] {
			if found_campaign[campaign_id] {
				continue
			}
			found_campaign[campaign_id] = true
			if ref[campaign_id] {
				continue
			}
			if pzutil.IsDigit(campaign_id) {
				n++
				str += fmt.Sprintf(" (%s, %s, 41, %s),", ARGS.Get("entitytype_id"), ARGS.Get("entity_id"), campaign_id)
			}
		}
	}
	if n == 0 {
		return nil
	}
	err := self.Do_sql(
		`DELETE FROM ac WHERE entitytype_id=? AND entity_id=?`, ARGS.Get("entitytype_id"), ARGS.Get("entity_id"))
	if err != nil {
		return err
	}
	return self.Do_sql(str[:len(str)-1])
}

func (self *Model) UpdateOrder(extra ...url.Values) error {
	ARGS := self.ARGS

	err := self.Do_sql(
		`DELETE FROM ac
WHERE entitytype_id=? AND entity_id=?`,
		ARGS.Get("entitytype_id"), ARGS.Get("entity_id"))
	if err != nil {
		return err
	}

	return self.Do_sql(
		`UPDATE `+ARGS.Get("table")+`
SET access_order=?
WHERE `+ARGS.Get("idname")+`=?`,
		ARGS.Get("access_order"), ARGS.Get("entity_id"))
}

func (self *Model) get_access_order() error {
	ARGS := self.ARGS
	return self.Get_args(ARGS,
		`SELECT access_order FROM `+ARGS.Get("table")+`
WHERE `+ARGS.Get("idname")+`=?`, ARGS.Get("entity_id"))
}

func (self *Model) Topics(extra ...url.Values) error {
	ARGS := self.ARGS

	if err := self.get_access_order(); err != nil {
		return err
	}
	if ARGS.Get("access_order") == "Inherit" {
		return nil
	}

	if ARGS.Get("entitytype_id") == "3" || ARGS.Get("entitytype_id") == "31" {
		return self.Select_sql(self.LISTS,
			`SELECT ac_id, adv.adv_id, a.company, a.url, '*' AS campaign_id, '*' AS campaign_name, a.url
FROM ac
INNER JOIN adv ON (ac.othertype_id=4 AND ac.other_id=adv.adv_id)
INNER JOIN add_address a USING (address_id)
WHERE entitytype_id=? AND entity_id=?
UNION
SELECT ac_id, adv.adv_id, a.company, a.url, c.campaign_id, c.campaign_name, a.url
FROM ac
INNER JOIN adv_campaign c ON (ac.othertype_id=41 AND ac.other_id=c.campaign_id)
INNER JOIN adv USING (adv_id)
INNER JOIN add_address a USING (address_id)
WHERE entitytype_id=? AND entity_id=?`,
			ARGS.Get("entitytype_id"), ARGS.Get("entity_id"),
			ARGS.Get("entitytype_id"), ARGS.Get("entity_id"))
	}

	return self.Select_sql(self.LISTS,
		`SELECT ac_id, pub.pub_id, a.company, a.url, '*' AS site_id, '*' AS site_name, '*' AS site_url
FROM ac
INNER JOIN pub ON (ac.othertype_id=3 AND ac.other_id=pub_id)
INNER JOIN add_address a USING (address_id)
WHERE entitytype_id=? AND entity_id=?
UNION
SELECT ac_id, pub.pub_id, a.company, a.url, s.site_id, s.site_name, s.site_url
FROM ac
INNER JOIN pub_site s ON (ac.othertype_id=31 AND ac.other_id=s.site_id)
INNER JOIN pub USING (pub_id)
INNER JOIN add_address a USING (address_id)
WHERE entitytype_id=? AND entity_id=?`,
		ARGS.Get("entitytype_id"), ARGS.Get("entity_id"),
		ARGS.Get("entitytype_id"), ARGS.Get("entity_id"))
}

func (self *Model) Startnew(extra ...url.Values) error {
	ARGS := self.ARGS

	var err error
	if err = self.get_access_order(); err != nil {
		return err
	}
	if ARGS.Get("access_order") == "Inherit" {
		return nil
	}

	err = self.Select_sql(self.LISTS,
		`SELECT campaign_id, ANY_VALUE(campaign_name) AS campaign_name,
	ANY_VALUE(adv_id) AS adv_id, ANY_VALUE(adv_name) AS adv_name,
	ANY_VALUE(othertype_id) AS othertype_id, ANY_VALUE(other_id) AS other_id,
	ANY_VALUE(ac_id) AS ac_id
FROM ViewSlotOpen WHERE `+ARGS.Get("idname")+`=?
GROUP BY campaign_id`, ARGS.Get("entity_id"))
	if err != nil {
		return err
	}

	return self.Process_after("startnew", extra...)
}

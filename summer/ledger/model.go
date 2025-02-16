package ledger

import (
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) TopicsPub24Hours(extra ...url.Values) error {
	ARGS := self.ARGS
	//`SELECT DATE_FORMAT(MIN(l.timely), '%Y-%m-%d %H:%i:00') AS hours,
	if err := self.Select_sql(self.LISTS,
		`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i:00') AS hours,
SUM(p.imps) AS imps, SUM(p.clis) AS clis, SUM(p.spend) AS spend
FROM ledger_pub p
INNER JOIN ledger_log l USING (log_id)
WHERE p.pub_id=? AND (timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
		ARGS.Get("pub_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
		return err
	}

	return self.Process_after("topicsPub24Hours", extra...)
}

func (self *Model) TopicsPubTopSlots(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.Select_sql(self.LISTS,
		`SELECT p.slot_id, s.slot_name, p.site_id, p.pub_id, p.imps, p.clis, p.spend, (p.spend*1000/p.imps) AS cpm, (p.spend/p.clis) AS cpc, (p.clis/p.imps) AS ctr
FROM daily_pub p
INNER JOIN daily_log l USING (log_id)
INNER JOIN pub_slot s USING (slot_id)
WHERE pub_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
ORDER BY p.spend DESC LIMIT ?`,
		ARGS.Get("pub_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) TopicsPubTopCampaigns(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.Select_sql(self.LISTS,
		`SELECT a.campaign_id, c.campaign_name, ANY_VALUE(a.adv_id) AS adv_id, SUM(pa.spend) AS spend, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, (SUM(pa.spend)*1000/SUM(pa.imps)) AS cpm, (SUM(pa.spend)/SUM(pa.clis)) AS cpc, (SUM(pa.clis)/SUM(pa.imps)) AS ctr
FROM daily_pub_adv pa
INNER JOIN daily_pub p USING (lp_id)
INNER JOIN daily_log l ON (p.log_id=l.log_id)
INNER JOIN daily_adv a USING (la_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE pub_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY a.campaign_id ORDER BY spend DESC LIMIT ?`,
		ARGS.Get("pub_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) TopicsAdv24Hours(extra ...url.Values) error {
	ARGS := self.ARGS
	if err := self.Select_sql(self.LISTS,
		`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i') AS hours,
SUM(a.imps) AS imps, SUM(a.clis) AS clis, SUM(a.spend) AS spend
FROM ledger_adv a
INNER JOIN ledger_log l USING (log_id)
WHERE a.adv_id=? AND (timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
		ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
		return err
	}
	return self.Process_after("topicsAdv24Hours", extra...)
}

func (self *Model) TopicsAdvTopItems(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.Select_sql(self.LISTS,
		`SELECT a.item_id, i.item_name, a.campaign_id, a.adv_id, a.imps, a.clis, a.spend, (a.spend*1000/a.imps) AS cpm, (a.spend/a.clis) AS cpc, (a.clis/a.imps) AS ctr
FROM daily_adv a
INNER JOIN daily_log l USING (log_id)
INNER JOIN adv_item i USING (item_id)
WHERE adv_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
ORDER BY a.spend DESC LIMIT ?`,
		ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) TopicsAdvTopSlots(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.Select_sql(self.LISTS,
		`SELECT p.slot_id, s.slot_name, ANY_VALUE(p.site_id) AS site_id, ANY_VALUE(p.pub_id) AS pub_id, SUM(pa.spend) AS spend, SUM(pa.imps) AS imps, SUM(pa.clis) AS clis, (SUM(pa.spend)*1000/SUM(pa.imps)) AS cpm, (SUM(pa.spend)/SUM(pa.clis)) AS cpc, (SUM(pa.clis)/SUM(pa.imps)) AS ctr
FROM daily_pub_adv pa
INNER JOIN daily_adv a USING (la_id)
INNER JOIN daily_log l ON (a.log_id=l.log_id)
INNER JOIN daily_pub p USING (lp_id)
INNER JOIN pub_slot s USING (slot_id)
WHERE adv_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY p.slot_id ORDER BY spend DESC LIMIT ?`,
		ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

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
	if err := self.SelectSQL(self.LISTS,
		`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i:00') AS hours,
SUM(p.imps) AS imps, SUM(p.clis) AS clis, SUM(p.spend) AS spend
FROM ledger_pub p
INNER JOIN ledger_log l USING (log_id)
WHERE p.pub_id=? AND (timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
		ARGS.Get("pub_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
		return err
	}

	return self.ProcessAfter("topicsPub24Hours", extra...)
}

func (self *Model) TopicsPubTopSlots(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.SelectSQL(self.LISTS,
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
	return self.SelectSQL(self.LISTS,
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
	if err := self.SelectSQL(self.LISTS,
		`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i') AS hours,
SUM(a.imps) AS imps, SUM(a.clis) AS clis, SUM(a.spend) AS spend
FROM ledger_adv a
INNER JOIN ledger_log l USING (log_id)
WHERE a.adv_id=? AND (timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
		ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
		return err
	}
	return self.ProcessAfter("topicsAdv24Hours", extra...)
}

func (self *Model) TopicsAdvTopItems(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.SelectSQL(self.LISTS,
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
	return self.SelectSQL(self.LISTS,
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

func (self *Model) TopicsMid24Hours(extra ...url.Values) error {
	ARGS := self.ARGS
	if self.isAdvertiserMiddlemanReport() {
		if err := self.SelectSQL(self.LISTS,
			`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i') AS hours,
SUM(m.imps) AS imps, SUM(m.clis) AS clis, SUM(m.pay_spend) AS spend
FROM ledger_mid m
INNER JOIN ledger_log l USING (log_id)
WHERE m.adv_id=? AND (timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
			ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
			return err
		}
		return self.processMiddlemanAfter()
	}
	if err := self.SelectSQL(self.LISTS,
		`SELECT DATE_FORMAT(MIN(l.timely), '%H:%i') AS hours,
SUM(m.wins) AS wins, SUM(m.losses) AS losses, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
SUM(m.charge_spend) AS charge_spend, SUM(m.pay_spend) AS pay_spend, SUM(m.margin_spend) AS margin_spend,
COALESCE(SUM(m.margin_spend)/NULLIF(SUM(m.charge_spend),0), 0) AS margin_rate,
SUM(m.forward_ok) AS forward_ok, SUM(m.forward_duplicate) AS forward_duplicate,
SUM(m.forward_missing + m.forward_error + m.forward_http_error + m.forward_invalid) AS forward_errors
FROM ledger_mid m
INNER JOIN ledger_log l USING (log_id)
WHERE timely BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?
GROUP BY FLOOR(UNIX_TIMESTAMP(timely) / 3600)`,
		ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day")+" 23:59:59"); err != nil {
		return err
	}
	return self.processMiddlemanAfter()
}

func (self *Model) processMiddlemanAfter() error {
	pages := []map[string]interface{}{
		{"model": "ledger", "action": "topicsMidTopBidders"},
	}
	if self.isAdvertiserMiddlemanReport() {
		pages = append(pages, map[string]interface{}{"model": "ledger", "action": "topicsMidTopSlots"})
	} else {
		pages = append(pages,
			map[string]interface{}{"model": "ledger", "action": "topicsMidTopRoutes"},
			map[string]interface{}{"model": "ledger", "action": "topicsMidTopPublishers"},
		)
	}
	for _, page := range pages {
		if err := self.CallOnce(page, make(url.Values)); err != nil {
			return err
		}
	}
	return nil
}

func (self *Model) TopicsMidTopBidders(extra ...url.Values) error {
	ARGS := self.ARGS
	if self.isAdvertiserMiddlemanReport() {
		return self.SelectSQL(self.LISTS,
			`SELECT m.bidder_id, b.bidder_name, m.adv_id,
SUM(m.pay_spend) AS spend, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
COALESCE(SUM(m.pay_spend)*1000/NULLIF(SUM(m.imps),0), 0) AS cpm,
COALESCE(SUM(m.pay_spend)/NULLIF(SUM(m.clis),0), 0) AS cpc,
COALESCE(SUM(m.clis)/NULLIF(SUM(m.imps),0), 0) AS ctr
FROM daily_mid m
INNER JOIN daily_log l USING (log_id)
LEFT JOIN adv_bidder b USING (bidder_id)
WHERE m.adv_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY m.bidder_id, b.bidder_name, m.adv_id
ORDER BY spend DESC LIMIT ?`,
			ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
	}
	return self.SelectSQL(self.LISTS,
		`SELECT m.bidder_id, b.bidder_name, ANY_VALUE(m.adv_id) AS adv_id,
SUM(m.wins) AS wins, SUM(m.losses) AS losses, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
SUM(m.charge_spend) AS charge_spend, SUM(m.pay_spend) AS pay_spend, SUM(m.margin_spend) AS margin_spend,
COALESCE(SUM(m.margin_spend)/NULLIF(SUM(m.charge_spend),0), 0) AS margin_rate,
SUM(m.forward_missing + m.forward_error + m.forward_http_error + m.forward_invalid) AS forward_errors
FROM daily_mid m
INNER JOIN daily_log l USING (log_id)
LEFT JOIN adv_bidder b USING (bidder_id)
WHERE l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?
GROUP BY m.bidder_id, b.bidder_name
ORDER BY margin_spend DESC LIMIT ?`,
		ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) isAdvertiserMiddlemanReport() bool {
	return self.ARGS.Get("adv_id") != "" && self.ARGS.Get("admin_id") == ""
}

func (self *Model) TopicsMidTopSlots(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.SelectSQL(self.LISTS,
		`SELECT m.slot_id, s.slot_name, ANY_VALUE(m.site_id) AS site_id, ANY_VALUE(m.pub_id) AS pub_id,
SUM(m.pay_spend) AS spend, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
COALESCE(SUM(m.pay_spend)*1000/NULLIF(SUM(m.imps),0), 0) AS cpm,
COALESCE(SUM(m.pay_spend)/NULLIF(SUM(m.clis),0), 0) AS cpc,
COALESCE(SUM(m.clis)/NULLIF(SUM(m.imps),0), 0) AS ctr
FROM daily_mid m
INNER JOIN daily_log l USING (log_id)
LEFT JOIN pub_slot s USING (slot_id)
WHERE m.adv_id=? AND (l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?)
GROUP BY m.slot_id, s.slot_name
ORDER BY spend DESC LIMIT ?`,
		ARGS.Get("adv_id"), ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) TopicsMidTopRoutes(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.SelectSQL(self.LISTS,
		`SELECT m.group_id, g.group_name, m.route_bidder_id, m.target_id,
SUM(m.wins) AS wins, SUM(m.losses) AS losses, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
SUM(m.charge_spend) AS charge_spend, SUM(m.pay_spend) AS pay_spend, SUM(m.margin_spend) AS margin_spend,
COALESCE(SUM(m.margin_spend)/NULLIF(SUM(m.charge_spend),0), 0) AS margin_rate,
SUM(m.forward_missing + m.forward_error + m.forward_http_error + m.forward_invalid) AS forward_errors
FROM daily_mid m
INNER JOIN daily_log l USING (log_id)
LEFT JOIN mid_route_group g USING (group_id)
WHERE l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?
GROUP BY m.group_id, g.group_name, m.route_bidder_id, m.target_id
ORDER BY margin_spend DESC LIMIT ?`,
		ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

func (self *Model) TopicsMidTopPublishers(extra ...url.Values) error {
	ARGS := self.ARGS
	return self.SelectSQL(self.LISTS,
		`SELECT m.pub_id, p.email AS pub_email, ANY_VALUE(m.site_id) AS site_id, ANY_VALUE(m.slot_id) AS slot_id,
SUM(m.wins) AS wins, SUM(m.losses) AS losses, SUM(m.imps) AS imps, SUM(m.clis) AS clis,
SUM(m.charge_spend) AS charge_spend, SUM(m.pay_spend) AS pay_spend, SUM(m.margin_spend) AS margin_spend,
COALESCE(SUM(m.margin_spend)/NULLIF(SUM(m.charge_spend),0), 0) AS margin_rate,
SUM(m.forward_missing + m.forward_error + m.forward_http_error + m.forward_invalid) AS forward_errors
FROM daily_mid m
INNER JOIN daily_log l USING (log_id)
LEFT JOIN pub p USING (pub_id)
WHERE l.daily BETWEEN DATE_SUB(?, INTERVAL ? DAY) AND ?
GROUP BY m.pub_id, p.email
ORDER BY charge_spend DESC LIMIT ?`,
		ARGS.Get("day"), ARGS.Get("idays"), ARGS.Get("day"), ARGS.Get("top"))
}

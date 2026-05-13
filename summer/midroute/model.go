package midroute

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/genelet/winter/summer"
)

type Model struct {
	summer.Model
}

func (self *Model) Topics(extra ...url.Values) error {
	return self.SelectSQL(self.LISTS, `
SELECT g.group_id, g.group_name, g.trigger_mode, g.total_timeout_ms,
	g.margin_pct, g.min_margin_cpm, g.active, g.created, g.updated,
	(SELECT COUNT(*) FROM mid_route_bidder rb WHERE rb.group_id=g.group_id) AS bidder_count,
	(SELECT COUNT(*) FROM mid_route_target rt WHERE rt.group_id=g.group_id) AS target_count
FROM mid_route_group g
ORDER BY g.group_id DESC`)
}

func (self *Model) Startnew(extra ...url.Values) error {
	*self.LISTS = []map[string]interface{}{{
		"trigger_mode":     "Fallback",
		"total_timeout_ms": "100",
		"margin_pct":       "0",
		"min_margin_cpm":   "0",
		"active":           "Yes",
	}}
	return nil
}

func (self *Model) Insert(extra ...url.Values) error {
	args := self.ARGS
	if err := self.DoSQL(`
INSERT INTO mid_route_group
	(group_name, trigger_mode, total_timeout_ms, margin_pct, min_margin_cpm, active, created)
VALUES (?, ?, ?, ?, ?, ?, NOW())`,
		args.Get("group_name"), args.Get("trigger_mode"), args.Get("total_timeout_ms"),
		args.Get("margin_pct"), args.Get("min_margin_cpm"), args.Get("active")); err != nil {
		return err
	}
	*self.LISTS = []map[string]interface{}{{"group_id": self.LastID}}
	return nil
}

func (self *Model) Edit(extra ...url.Values) error {
	if err := self.loadGroupList(self.ARGS.Get("group_id")); err != nil {
		return err
	}
	if len(*self.LISTS) == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (self *Model) Update(extra ...url.Values) error {
	args := self.ARGS
	current, err := self.currentGroup(args.Get("group_id"))
	if err != nil {
		return err
	}
	if err := self.DoSQL(`
UPDATE mid_route_group
SET group_name=?, trigger_mode=?, total_timeout_ms=?, margin_pct=?, min_margin_cpm=?, active=?
WHERE group_id=?`,
		mergedValue(args, current, "group_name"), mergedValue(args, current, "trigger_mode"),
		mergedValue(args, current, "total_timeout_ms"), mergedValue(args, current, "margin_pct"),
		mergedValue(args, current, "min_margin_cpm"), mergedValue(args, current, "active"),
		args.Get("group_id")); err != nil {
		return err
	}
	if err := self.loadGroupList(args.Get("group_id")); err != nil {
		return err
	}
	if len(*self.LISTS) == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (self *Model) Delete(extra ...url.Values) error {
	groupID := self.ARGS.Get("group_id")
	if err := self.DoSQL(`DELETE FROM mid_route_group WHERE group_id=?`, groupID); err != nil {
		return err
	}
	if self.Affected == 0 {
		return sql.ErrNoRows
	}
	*self.LISTS = []map[string]interface{}{{"group_id": groupID}}
	return nil
}

func (self *Model) Bidders(extra ...url.Values) error {
	if err := self.loadGroupOther(self.ARGS.Get("group_id")); err != nil {
		return err
	}
	return self.loadRouteBidders(self.ARGS.Get("group_id"))
}

func (self *Model) StartnewBidder(extra ...url.Values) error {
	groupID := self.ARGS.Get("group_id")
	if err := self.loadGroupOther(groupID); err != nil {
		return err
	}
	if err := self.loadBidderOptions(); err != nil {
		return err
	}
	*self.LISTS = []map[string]interface{}{{
		"group_id":       groupID,
		"bidder_id":      int64(0),
		"priority":       "100",
		"active":         "Yes",
		"timeout_ms":     "",
		"margin_pct":     "",
		"min_margin_cpm": "",
	}}
	return nil
}

func (self *Model) InsertBidder(extra ...url.Values) error {
	args := self.ARGS
	if err := self.DoSQL(`
INSERT INTO mid_route_bidder
	(group_id, bidder_id, priority, timeout_ms, margin_pct, min_margin_cpm, active, created)
VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		args.Get("group_id"), args.Get("bidder_id"), args.Get("priority"),
		nullableString(args.Get("timeout_ms")), nullableString(args.Get("margin_pct")),
		nullableString(args.Get("min_margin_cpm")), args.Get("active")); err != nil {
		return err
	}
	return self.loadRouteBidderByID(self.LastID)
}

func (self *Model) EditBidder(extra ...url.Values) error {
	if err := self.loadRouteBidderByIDString(self.ARGS.Get("route_bidder_id")); err != nil {
		return err
	}
	if err := self.loadBidderOptions(); err != nil {
		return err
	}
	return self.loadGroupOther(fmt.Sprint((*self.LISTS)[0]["group_id"]))
}

func (self *Model) UpdateBidder(extra ...url.Values) error {
	args := self.ARGS
	current, err := self.currentRouteBidder(args.Get("route_bidder_id"))
	if err != nil {
		return err
	}
	if err := self.DoSQL(`
UPDATE mid_route_bidder
SET bidder_id=?, priority=?, timeout_ms=?, margin_pct=?, min_margin_cpm=?, active=?
WHERE route_bidder_id=?`,
		mergedValue(args, current, "bidder_id"), mergedValue(args, current, "priority"),
		mergedNullableValue(args, current, "timeout_ms"), mergedNullableValue(args, current, "margin_pct"),
		mergedNullableValue(args, current, "min_margin_cpm"), mergedValue(args, current, "active"),
		args.Get("route_bidder_id")); err != nil {
		return err
	}
	return self.loadRouteBidderByIDString(args.Get("route_bidder_id"))
}

func (self *Model) DeleteBidder(extra ...url.Values) error {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `SELECT route_bidder_id, group_id FROM mid_route_bidder WHERE route_bidder_id=?`, self.ARGS.Get("route_bidder_id")); err != nil {
		return err
	}
	if len(row) == 0 {
		return sql.ErrNoRows
	}
	if err := self.DoSQL(`DELETE FROM mid_route_bidder WHERE route_bidder_id=?`, self.ARGS.Get("route_bidder_id")); err != nil {
		return err
	}
	*self.LISTS = []map[string]interface{}{row}
	return nil
}

func (self *Model) Targets(extra ...url.Values) error {
	if err := self.loadGroupOther(self.ARGS.Get("group_id")); err != nil {
		return err
	}
	return self.loadRouteTargets(self.ARGS.Get("group_id"))
}

func (self *Model) StartnewTarget(extra ...url.Values) error {
	groupID := self.ARGS.Get("group_id")
	if err := self.loadGroupOther(groupID); err != nil {
		return err
	}
	if err := self.loadSizeOptions(); err != nil {
		return err
	}
	*self.LISTS = []map[string]interface{}{{
		"group_id":          groupID,
		"priority":          "100",
		"active":            "Yes",
		"entitytype_id":     nil,
		"entitytype_global": true,
		"entity_id":         "",
		"size_id":           int64(0),
	}}
	return nil
}

func (self *Model) InsertTarget(extra ...url.Values) error {
	args := self.ARGS
	if err := self.DoSQL(`
INSERT INTO mid_route_target
	(group_id, entitytype_id, entity_id, size_id, priority, active, created)
VALUES (?, ?, ?, ?, ?, ?, NOW())`,
		args.Get("group_id"), nullableString(args.Get("entitytype_id")),
		nullableString(args.Get("entity_id")), nullableString(args.Get("size_id")),
		args.Get("priority"), args.Get("active")); err != nil {
		return err
	}
	return self.loadRouteTargetByID(self.LastID)
}

func (self *Model) EditTarget(extra ...url.Values) error {
	if err := self.loadRouteTargetByIDString(self.ARGS.Get("target_id")); err != nil {
		return err
	}
	if err := self.loadSizeOptions(); err != nil {
		return err
	}
	return self.loadGroupOther(fmt.Sprint((*self.LISTS)[0]["group_id"]))
}

func (self *Model) UpdateTarget(extra ...url.Values) error {
	args := self.ARGS
	current, err := self.currentRouteTarget(args.Get("target_id"))
	if err != nil {
		return err
	}
	if err := self.DoSQL(`
UPDATE mid_route_target
SET entitytype_id=?, entity_id=?, size_id=?, priority=?, active=?
WHERE target_id=?`,
		mergedNullableValue(args, current, "entitytype_id"), mergedNullableValue(args, current, "entity_id"),
		mergedNullableValue(args, current, "size_id"), mergedValue(args, current, "priority"), mergedValue(args, current, "active"),
		args.Get("target_id")); err != nil {
		return err
	}
	return self.loadRouteTargetByIDString(args.Get("target_id"))
}

func (self *Model) DeleteTarget(extra ...url.Values) error {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `SELECT target_id, group_id FROM mid_route_target WHERE target_id=?`, self.ARGS.Get("target_id")); err != nil {
		return err
	}
	if len(row) == 0 {
		return sql.ErrNoRows
	}
	if err := self.DoSQL(`DELETE FROM mid_route_target WHERE target_id=?`, self.ARGS.Get("target_id")); err != nil {
		return err
	}
	*self.LISTS = []map[string]interface{}{row}
	return nil
}

func (self *Model) loadGroupList(groupID interface{}) error {
	return self.SelectSQL(self.LISTS, `
SELECT group_id, group_name, trigger_mode, total_timeout_ms, margin_pct, min_margin_cpm, active, created, updated
FROM mid_route_group
WHERE group_id=?`, groupID)
}

func (self *Model) currentGroup(groupID interface{}) (map[string]interface{}, error) {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `
SELECT group_id, group_name, trigger_mode, total_timeout_ms, margin_pct, min_margin_cpm, active
FROM mid_route_group
WHERE group_id=?`, groupID); err != nil {
		return nil, err
	}
	if len(row) == 0 {
		return nil, sql.ErrNoRows
	}
	return row, nil
}

func (self *Model) loadGroupOther(groupID interface{}) error {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `
SELECT group_id, group_name, trigger_mode, total_timeout_ms, margin_pct, min_margin_cpm, active
FROM mid_route_group
WHERE group_id=?`, groupID); err != nil {
		return err
	}
	if len(row) == 0 {
		return sql.ErrNoRows
	}
	(*self.OTHER)["midroute_group"] = row
	return nil
}

func (self *Model) loadRouteBidders(groupID interface{}) error {
	return self.SelectSQL(self.LISTS, `
SELECT rb.route_bidder_id, rb.group_id, rb.bidder_id, rb.priority, rb.timeout_ms,
	rb.margin_pct, rb.min_margin_cpm, rb.active, rb.created, rb.updated,
	b.bidder_name, b.adv_id, a.email AS adv_email, b.credential_status AS bidder_credential_status,
	b.active AS bidder_active
FROM mid_route_bidder rb
INNER JOIN adv_bidder b USING (bidder_id)
INNER JOIN adv a USING (adv_id)
WHERE rb.group_id=?
ORDER BY rb.priority ASC, rb.route_bidder_id ASC`, groupID)
}

func (self *Model) loadRouteBidderByID(routeBidderID interface{}) error {
	return self.SelectSQL(self.LISTS, `
SELECT rb.route_bidder_id, rb.group_id, rb.bidder_id, rb.priority, rb.timeout_ms,
	rb.margin_pct, rb.min_margin_cpm, rb.active, rb.created, rb.updated,
	b.bidder_name, b.adv_id, a.email AS adv_email, b.credential_status AS bidder_credential_status,
	b.active AS bidder_active
FROM mid_route_bidder rb
INNER JOIN adv_bidder b USING (bidder_id)
INNER JOIN adv a USING (adv_id)
WHERE rb.route_bidder_id=?`, routeBidderID)
}

func (self *Model) loadRouteBidderByIDString(routeBidderID string) error {
	if err := self.loadRouteBidderByID(routeBidderID); err != nil {
		return err
	}
	if len(*self.LISTS) == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (self *Model) currentRouteBidder(routeBidderID interface{}) (map[string]interface{}, error) {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `
SELECT route_bidder_id, group_id, bidder_id, priority, timeout_ms, margin_pct, min_margin_cpm, active
FROM mid_route_bidder
WHERE route_bidder_id=?`, routeBidderID); err != nil {
		return nil, err
	}
	if len(row) == 0 {
		return nil, sql.ErrNoRows
	}
	return row, nil
}

func (self *Model) loadBidderOptions() error {
	lists := make([]map[string]interface{}, 0)
	if err := self.SelectSQL(&lists, `
SELECT b.bidder_id, b.bidder_name, b.adv_id, a.email AS adv_email, b.credential_status, b.active
FROM adv_bidder b
INNER JOIN adv a USING (adv_id)
ORDER BY b.active DESC, b.credential_status ASC, b.bidder_name ASC`); err != nil {
		return err
	}
	(*self.OTHER)["midroute_bidders"] = lists
	return nil
}

func (self *Model) loadRouteTargets(groupID interface{}) error {
	if err := self.SelectSQL(self.LISTS, targetSelectSQL()+`
WHERE t.group_id=?
ORDER BY t.priority ASC, t.target_id ASC`, groupID); err != nil {
		return err
	}
	decorateTargetRows(*self.LISTS)
	return nil
}

func (self *Model) loadRouteTargetByID(targetID interface{}) error {
	if err := self.SelectSQL(self.LISTS, targetSelectSQL()+`
WHERE t.target_id=?`, targetID); err != nil {
		return err
	}
	decorateTargetRows(*self.LISTS)
	return nil
}

func (self *Model) loadRouteTargetByIDString(targetID string) error {
	if err := self.loadRouteTargetByID(targetID); err != nil {
		return err
	}
	if len(*self.LISTS) == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (self *Model) currentRouteTarget(targetID interface{}) (map[string]interface{}, error) {
	row := make(map[string]interface{})
	if err := self.GetSQL(row, `
SELECT target_id, group_id, entitytype_id, entity_id, size_id, priority, active
FROM mid_route_target
WHERE target_id=?`, targetID); err != nil {
		return nil, err
	}
	if len(row) == 0 {
		return nil, sql.ErrNoRows
	}
	return row, nil
}

func (self *Model) loadSizeOptions() error {
	lists := make([]map[string]interface{}, 0)
	if err := self.SelectSQL(&lists, `
SELECT size_id, size_name, width, height
FROM def_size
ORDER BY width ASC, height ASC`); err != nil {
		return err
	}
	(*self.OTHER)["midroute_sizes"] = lists
	return nil
}

func targetSelectSQL() string {
	return `
SELECT t.target_id, t.group_id, t.entitytype_id, t.entity_id, t.size_id, t.priority,
	t.active, t.created, t.updated,
	CASE
		WHEN t.entitytype_id IS NULL THEN 'Global'
		WHEN t.entitytype_id=3 THEN (SELECT p.email FROM pub p WHERE p.pub_id=t.entity_id)
		WHEN t.entitytype_id=31 THEN (SELECT s.site_name FROM pub_site s WHERE s.site_id=t.entity_id)
		WHEN t.entitytype_id=32 THEN (SELECT sl.slot_name FROM pub_slot sl WHERE sl.slot_id=t.entity_id)
		ELSE CAST(t.entity_id AS CHAR)
	END AS entity_name,
	CONCAT(ds.size_name, ' ', ds.width, 'x', ds.height) AS size_name
FROM mid_route_target t
LEFT JOIN def_size ds USING (size_id)
`
}

func decorateTargetRows(rows []map[string]interface{}) {
	for _, row := range rows {
		switch row["entitytype_id"] {
		case int64(3), uint8(3), "3":
			row["entitytype_pub"] = true
		case int64(31), uint8(31), "31":
			row["entitytype_site"] = true
		case int64(32), uint8(32), "32":
			row["entitytype_slot"] = true
		default:
			row["entitytype_global"] = true
		}
	}
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func mergedValue(args url.Values, current map[string]interface{}, name string) interface{} {
	if args.Has(name) {
		return args.Get(name)
	}
	return current[name]
}

func mergedNullableValue(args url.Values, current map[string]interface{}, name string) interface{} {
	if args.Has(name) {
		return nullableString(args.Get(name))
	}
	return current[name]
}

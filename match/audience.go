// Package match provides functionality for handling audience matching and database operations.
package match

import (
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/genelet/winter/demo"
	"github.com/genelet/winter/dmp"
	"github.com/genelet/winter/ipsearch"
	"github.com/genelet/winter/pzutil"
	"github.com/genelet/winter/uadevice"
)

type Audience struct {
	dmp.DmpAudience
	ipsearch.GeoAudience
	demo.DemoAudience
	uadevice.UaAudience

	WeekDays  uint32
	WeekHours uint32
}

func (self *Audience) Pack() ([]byte, error) {
	return pzutil.PackObject(self)
}

func UnpackAudience(data []byte) (*Audience, error) {
	audience := new(Audience)
	err := pzutil.UnpackObject(data, audience)
	return audience, err
}

func (self *Audience) MatchWeekTime(current time.Time) bool {
	if self.WeekDays != 0 {
		wd := uint32(current.Weekday())
		if (uint32(self.WeekDays)>>wd)&1 != 1 {
			return false
		}
	}
	if self.WeekHours != 0 {
		hour := uint32(current.Hour())
		if (uint32(self.WeekHours)>>hour)&1 != 1 {
			return false
		}
	}
	return true
}

func AudienceFromArgs(ARGS url.Values) *Audience {
	dmpA := dmp.DmpAudienceFromArgs(ARGS)
	geoA := ipsearch.GeoAudienceFromArgs(ARGS)
	demoA := demo.DemoAudienceFromArgs(ARGS)
	uaA := uadevice.UaAudienceFromArgs(ARGS)
	aud := &Audience{DmpAudience: *dmpA, GeoAudience: *geoA, DemoAudience: *demoA, UaAudience: *uaA}

	f := func(_ url.Values, name string, which *uint32) {
		value := ARGS.Get(name)
		if value != "" {
			v, err := strconv.ParseUint(value, 10, 32)
			if err == nil {
				*which = uint32(v)
			}
		}
	}
	f(ARGS, "weekday", &aud.WeekDays)
	f(ARGS, "weekhour", &aud.WeekHours)

	return aud
}

func (self *Audience) ToArgs(ARGS url.Values) {
	dmpA := &self.DmpAudience
	dmpA.ToArgs(ARGS)
	geoA := &self.GeoAudience
	geoA.ToArgs(ARGS)
	demoA := &self.DemoAudience
	demoA.ToArgs(ARGS)
	uaA := &self.UaAudience
	uaA.ToArgs(ARGS)

	f := func(args url.Values, name string, value uint32) {
		if value > 0 {
			args.Add(name, strconv.FormatUint(uint64(value), 10))
		}
	}
	f(ARGS, "weekday", self.WeekDays)
	f(ARGS, "weekhour", self.WeekHours)
}

func DBGetAudience(db *sql.DB, campaignID uint32) (*Audience, error) {
	rows, err := db.Query(
		`SELECT tn.targetname_id, tv.targetvalue_id, tv.value_id,
	an.attrname_id, an.attrname, av.attrvalue_id
FROM adv_targetname tn
INNER JOIN adv_targetvalue tv USING (targetname_id)
INNER JOIN adv_attrname an USING (attrname_id)
LEFT JOIN adv_attrvalue av
	ON (an.attrname_id=av.attrname_id AND tv.value_id=av.attrvalue_id)
WHERE tn.campaign_id=?`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dmpA := new(dmp.DmpAudience)
	geoA := new(ipsearch.GeoAudience)
	demoA := new(demo.DemoAudience)
	uaA := new(uadevice.UaAudience)
	var weekdays, weekhours uint32

	i := 0
	for rows.Next() {
		var targetnameID, targetvalueID, valueID, attrnameID uint32
		var attrvalueID sql.NullInt64
		var attrname string
		err = rows.Scan(&targetnameID, &targetvalueID, &valueID, &attrnameID, &attrname, &attrvalueID)
		if err != nil {
			return nil, err
		}

		i++

		dmpA.DBFillDmpAudience(attrname, valueID)
		geoA.DBFillGeoAudience(attrname, valueID)
		demoA.DBFillDemoAudience(attrname, valueID)
		uaA.DBFillUaAudience(attrname, valueID)

		switch attrname {
		case "weekday":
			weekdays += 1 << valueID
		case "weekhour":
			weekhours += 1 << valueID
		default:
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if i == 0 {
		return nil, nil
	}

	return &Audience{DmpAudience: *dmpA, GeoAudience: *geoA, DemoAudience: *demoA, UaAudience: *uaA, WeekDays: weekdays, WeekHours: weekhours}, nil
}

func DBInsertAudience(db *sql.DB, ARGS url.Values) error {
	campaignID := ARGS.Get("campaign_id")

	data := ``
	_, err := db.Exec(
		`DELETE FROM adv_targetname WHERE campaign_id=?`, campaignID)
	if err != nil {
		return err
	}

	hash := make(map[string]string)
	for attrname, attrnameID := range pzutil.AttrValue {
		if _, ok := ARGS[attrname]; ok {
			hash[attrname] = strconv.FormatUint(uint64(attrnameID), 10)
		}
	}
	for k := range ARGS {
		parts := strings.Split(k, "_")
		id := parts[len(parts)-1]
		if pzutil.IsDigit(id) {
			hash[k] = id
		}
	}

	for attrname, attrnameID := range hash {
		result, err := db.Exec(
			`INSERT INTO adv_targetname (campaign_id, attrname_id) VALUES (?, ?)`,
			campaignID, attrnameID)
		if err != nil {
			return err
		}
		lastID, err := result.LastInsertId()
		if err != nil {
			continue
		}
		targetnameID := strconv.FormatInt(lastID, 10)
		total := 0
		for _, id := range ARGS[attrname] {
			if pzutil.IsDigit(id) {
				data += `(` + targetnameID + `, ` + id + `),`
				total++
			}
		}
		if total == 0 {
			_, err = db.Exec(
				`DELETE FROM adv_targetname WHERE targetname_id=?`, targetnameID)
			if err != nil {
				return err
			}
		}
	}

	length := len(data)
	if length == 0 {
		return nil
	}

	_, err = db.Exec(
		`INSERT INTO adv_targetvalue (targetname_id, value_id) VALUES ` + data[:length-1])
	return err
}

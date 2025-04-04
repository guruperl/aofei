package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/mediocregopher/radix/v4"
	"github.com/nats-io/nats.go"
)

type Creative struct {
	// creative name
	CreativeName string
	// creative content
	CreativeContent string
	SizeID          uint32
	// click landing, here for retrieve from Redis only
	Landing string
	// foreign_id of campaign
	Failback string
	// campaign quality check
	IURL string
	// this should be the domain name of the advertiser
}

// Pack serializes the audience into a byte slice.
func (self *Creative) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(self)
	return buf.Bytes(), err
}

// PackIO serializes the audience into a byte slice for IO.
func (self *Creative) PackIO(w io.Writer) error {
	return gob.NewEncoder(w).Encode(self)
}

// UnpackCreativeIO deserializes the audience from an IO reader.
func UnpackCreativeIO(r io.Reader) (*Creative, error) {
	audience := new(Creative)
	err := gob.NewDecoder(r).Decode(audience)
	return audience, err
}

// UnpackCreative deserializes the audience from a byte slice.
func UnpackCreative(data []byte) (*Creative, error) {
	audience := new(Creative)
	buf := bytes.NewReader(data)
	err := gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

// DBGetCreativesToRedisSpread retrieves all creatives from the database and inserts them into Redis.
func DBGetCreativesToRedisSpread(ctx context.Context, conn interface{}, db *sql.DB, extra ...string) error {
	var pars []interface{}
	str := `
SELECT r.creative_id, r.size_id, c.iurl, i.item_click, c.foreign_id, r.creative_name, r.content
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE r.active="Yes"`
	if len(extra) > 0 {
		switch extra[0] {
		case "item_id":
			str += " AND i.item_id=?"
		case "campaign_id":
			str += " AND c.campaign_id=?"
		case "creative_id":
			str += " AND r.creative_id=?"
		default:
		}
		pars = append(pars, extra[1])
	}
	rows, err := db.Query(str, pars...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var creativeID uint32
		var iurl, landing, failback, content sql.NullString
		cre := new(Creative)
		err = rows.Scan(&creativeID, &cre.SizeID, &iurl, &landing, &failback, &cre.CreativeName, &content)
		if err != nil {
			return err
		}
		if iurl.Valid {
			cre.IURL = iurl.String
		}
		if landing.Valid {
			cre.Landing = landing.String
		}
		if failback.Valid {
			cre.Failback = failback.String
		}
		if content.Valid {
			cre.CreativeContent = content.String
		}
		switch t := conn.(type) {
		case radix.Client:
			err = cre.ToRedis(ctx, t, creativeID)
		case *nats.Conn:
			err = cre.ToSpread(t, creativeID)
		default:
			err = fmt.Errorf("unknown connection type %T", t)
		}
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

// DBGetCreative retrieves category audience from the database.
func DBGetCreative(db *sql.DB, creativeID uint32) (*Creative, error) {
	cre := new(Creative)
	err := db.QueryRow(`
SELECT c.iurl, i.item_click, c.foreign_id, r.size_id, r.creative_name, r.content
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE r.creative_id=?`, creativeID).Scan(&cre.IURL, &cre.Landing, &cre.Failback, &cre.SizeID, &cre.CreativeName, &cre.CreativeContent)

	return cre, err
}

const (
	HashNameCreative = "creative"
)

// ToRedis inserts creative into Redis.
func (self *Creative) ToRedis(ctx context.Context, conn radix.Client, creativeID uint32) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	return conn.Do(ctx, radix.Cmd(nil, "HSET", HashNameCreative, fmt.Sprintf("%d", creativeID), string(data)))
}

// ToSpread publishes creative to nats.
func (self *Creative) ToSpread(conn *nats.Conn, creativeID uint32) error {
	data, err := self.Pack()
	if err != nil {
		return err
	}
	return conn.Publish(fmt.Sprintf("%s:%d", HashNameCreative, creativeID), data)
}

// CreativeFromRedis retrieves audience data from Redis.
func CreativeFromRedis(ctx context.Context, conn radix.Client, creativeID uint32) (*Creative, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameCreative, fmt.Sprintf("%d", creativeID)))
	if err != nil {
		return nil, err
	}
	return UnpackCreative(bs)
}

// CreativeFromIO retrieves audience data from IO.
func CreativeFromIO(top string, creativeID uint32) (*Creative, error) {
	fh, err := os.Open(fmt.Sprintf("%s/%s/%d", top, HashNameCreative, creativeID))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	return UnpackCreativeIO(fh)
}

// CreativeMapFromRedis retrieves all creatives from Redis.
func CreativeMapFromRedis(ctx context.Context, conn radix.Client) (map[uint32]*Creative, error) {
	var arr []string
	err := conn.Do(ctx, radix.Cmd(&arr, "HGETALL", HashNameCreative))
	if err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, nil // No creatives found in Redis
	}
	if len(arr)%2 != 0 {
		return nil, sql.ErrNoRows // Invalid format
	}
	creatives := make(map[uint32]*Creative)
	for i := 0; i < len(arr); i += 2 {
		creativeID := arr[i]
		data := []byte(arr[i+1])
		cre, err := UnpackCreative(data)
		if err != nil {
			return nil, err
		}
		id, err := strconv.ParseUint(creativeID, 10, 32)
		if err != nil {
			return nil, err
		}
		creatives[uint32(id)] = cre
	}
	return creatives, nil
}

// CreativeMapFromIO retrieves all creatives from IO.
func CreativeMapFromIO(top string) (map[uint32]*Creative, error) {
	var creatives = make(map[uint32]*Creative)
	files, err := os.ReadDir(fmt.Sprintf("%s/%s", top, HashNameCreative))
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		creativeID, err := strconv.ParseUint(file.Name(), 10, 32)
		if err != nil {
			return nil, err
		}
		creative, err := CreativeFromIO(top, uint32(creativeID))
		if err != nil {
			return nil, err
		}
		creatives[creative.SizeID] = creative
	}
	return creatives, nil
}

func (self *Creative) AdM(attr *Attribute, trackers ...string) (string, error) {
	w, h := SizeID1To2(self.SizeID)
	if attr.NativeFormat != nil || attr.IsApp {
		return DefaultImgNative(self.CreativeContent, self.CreativeName, w, h).AdM(self.Landing, self.Failback, trackers...)
	} else if attr.IsVideo {
		return DefaultVideoNative(self.CreativeContent).AdM(self.Landing, self.Failback, trackers...)
	}

	return fmt.Sprintf(`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" marginheight="0" marginwidth="0" topmargin="0" leftmargin="0"></iframe>`, self.CreativeContent, w, h), nil
}

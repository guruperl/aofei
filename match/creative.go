package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"fmt"
	"strconv"

	"github.com/mediocregopher/radix/v4"
)

type Creative struct {
	// creative name
	CreativeName string
	// creative content
	CreativeContent string
	// click landing, here for retrieve from Redis only
	SizeID  uint32
	Landing string
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

// UnpackCreative deserializes the audience from a byte slice.
func UnpackCreative(data []byte) (*Creative, error) {
	audience := new(Creative)
	buf := bytes.NewReader(data)
	err := gob.NewDecoder(buf).Decode(audience)
	return audience, err
}

// DBGetCreativesToRedis retrieves all creatives from the database and inserts them into Redis.
func DBGetCreativesToRedis(ctx context.Context, conn radix.Client, db *sql.DB) error {
	rows, err := db.Query(`
SELECT r.creative_id, r.size_id, c.iurl, i.item_click, r.creative_name, r.content
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE r.active="Yes"`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var creativeID uint32
		var iurl, landing, content sql.NullString
		cre := new(Creative)
		err = rows.Scan(&creativeID, &cre.SizeID, &iurl, &landing, &cre.CreativeName, &content)
		if err != nil {
			return err
		}
		if iurl.Valid {
			cre.IURL = iurl.String
		}
		if landing.Valid {
			cre.Landing = landing.String
		}
		if content.Valid {
			cre.CreativeContent = content.String
		}
		if err = cre.ToRedis(ctx, conn, creativeID); err != nil {
			return err
		}
	}
	return rows.Err()
}

// DBGetCreative retrieves category audience from the database.
func DBGetCreative(db *sql.DB, creativeID uint32) (*Creative, error) {
	cre := new(Creative)
	err := db.QueryRow(`
SELECT c.iurl, i.item_click, r.size_id, r.creative_name, r.content
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
INNER JOIN adv_campaign c USING (campaign_id)
WHERE r.creative_id=?`, creativeID).Scan(&cre.IURL, &cre.Landing, &cre.SizeID, &cre.CreativeName, &cre.CreativeContent)

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

// CreativeFromRedis retrieves audience data from Redis.
func CreativeFromRedis(ctx context.Context, conn radix.Client, creativeID uint32) (*Creative, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameCreative, fmt.Sprintf("%d", creativeID)))
	if err != nil {
		return nil, err
	}
	return UnpackCreative(bs)
}

// CreativesFromRedis retrieves all creatives from Redis.
func CreativesFromRedis(ctx context.Context, conn radix.Client) (map[uint32]*Creative, error) {
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

func (self *Creative) AdM(attr *Attribute, trackers ...string) (string, error) {
	w, h := SizeID1To2(self.SizeID)
	if attr.NativeFormat != nil || attr.IsApp {
		return DefaultImgNative(self.CreativeContent, self.CreativeName, w, h).AdM(trackers...)
	} else if attr.IsVideo {
		return DefaultVideoNative(self.CreativeContent).AdM(trackers...)
	}

	return fmt.Sprintf(`<iframe src="%s" width="%d" height="%d" frameborder="0" scrolling="no" marginheight="0" marginwidth="0" topmargin="0" leftmargin="0"></iframe>`, self.CreativeContent, w, h), nil
}

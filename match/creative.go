package match

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"strconv"

	"github.com/mediocregopher/radix/v4"
)

type Creative struct {
	// creative name
	CreativeName string
	// creative content
	CreativeContent string
	// click landing, here for retrieve from Redis only
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

// DBGetCreative retrieves category audience from the database.
func DBGetCreative(db *sql.DB, creativeID uint32) (*Creative, error) {
	cre := new(Creative)
	err := db.QueryRow(`
SELECT c.iurl, i.item.click, r.creative_name, r.creative_content
FROM adv_creative r
INNER JOIN adv_item i USING (item_id)
WHERE r.creative_id=?`, creativeID).Scan(&cre.IURL, &cre.Landing, &cre.CreativeName, &cre.CreativeContent)

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
	return conn.Do(ctx, radix.FlatCmd(nil, "HSET", HashNameCreative, creativeID, string(data)))
}

// CreativeFromRedis retrieves audience data from Redis.
func CreativeFromRedis(ctx context.Context, conn radix.Client, creativeID uint32) (*Creative, error) {
	var bs []byte
	err := conn.Do(ctx, radix.Cmd(&bs, "HGET", HashNameCreative, strconv.FormatUint(uint64(creativeID), 10)))
	if err != nil {
		return nil, err
	}
	return UnpackCreative(bs)
}

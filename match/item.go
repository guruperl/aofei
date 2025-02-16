package match

import (
	"database/sql"
	"errors"

	"github.com/genelet/winter/pzutil"
)

type Creative struct {
	CreativeID uint32
	Weight     float32
	Content    string
}

type Item struct {
	ItemID     uint32
	AdvID      uint32
	SizeID     uint32
	Click      string
	ClickTotal uint8
	Creatives  []*Creative
	IsImage    bool
	IsHTML     bool
	IsJs       bool
	IsVideo    bool
}

func (self *Item) SelectCreative() *Creative {
	var weights []float32
	for _, creative := range self.Creatives {
		weights = append(weights, creative.Weight)
	}
	index := SelectOne(weights)
	return self.Creatives[index]
}

func (self *Item) Pack() ([]byte, error) {
	return pzutil.PackObject(self)
}

func UnpackItem(data []byte) (*Item, error) {
	item := new(Item)
	err := pzutil.UnpackObject(data, item)
	return item, err
}

func DBGetItem(db *sql.DB, itemID uint32) (*Item, error) {
	rows, err := db.Query(`
SELECT m.adv_id, i.size_id, i.item_click, i.qa_mime,
		c.creative_id, c.weight, c.content, cpc_fc
FROM adv_item i 
INNER JOIN adv_creative c USING (item_id)
INNER JOIN adv_campaign m USING (campaign_id)
WHERE i.item_id=? AND c.active='Yes'`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	item := new(Item)
	found := false
	for rows.Next() {
		c := new(Creative)
		var Click sql.NullString
		var Content sql.NullString
		var Cpc sql.NullInt64
		var mime string
		err = rows.Scan(&item.AdvID, &item.SizeID, &Click, &mime, &c.CreativeID, &c.Weight, &Content, &Cpc)
		if err != nil {
			return nil, err
		}
		if Click.Valid {
			item.Click = Click.String
		}
		if Cpc.Valid {
			item.ClickTotal = uint8(Cpc.Int64)
		}
		if Content.Valid {
			c.Content = Content.String
		}
		found = true
		switch mime {
		case "html":
			item.IsHTML = true
		case "image":
			item.IsImage = true
		case "js":
			item.IsJs = true
		case "video":
			item.IsVideo = true
		}
		item.ItemID = itemID
		item.Creatives = append(item.Creatives, c)
	}
	if !found {
		return nil, errors.New("item not found in DB")
	}
	return item, nil
}

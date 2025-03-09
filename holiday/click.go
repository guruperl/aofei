package holiday

import (
	"time"
	"strings"
	"errors"
    "net/http"
    "net/url"
    "encoding/base64"
	"github.com/golang/glog"
)

type rawImp struct {
	imp_id	int64
	raw_id	int64
	user_id	int64
	RAdv
	RPub
}

func newRawImp(imp_id, raw_id, user_id int64, radv RAdv, rpub RPub) *rawImp {
	return &rawImp{imp_id, raw_id, user_id, radv, rpub}
}

func (self rawImp) ToArgs() map[string]interface{} {
	return map[string]interface{}{
	"imp_id"     : self.imp_id,
	"raw_id"     : self.raw_id,
	"user_id"    : self.user_id,
	"adv_id"     : self.RAdv.AdvId,
	"campaign_id": self.RAdv.CampaignId,
	"item_id"    : self.RAdv.ItemId,
	"creative_id": self.RAdv.CreativeId,
	"cost_type"  : self.RAdv.CostType,
	"price"      : self.RAdv.Price,
	"slot_id"    : self.RPub.SlotId,
	"site_id"    : self.RPub.SiteId,
	"pub_id"     : self.RPub.PubId,
	}
}

func (self *Controller) serveClick(w http.ResponseWriter, r *http.Request) error {
	current := time.Now()
	path := r.URL.Path
	length := len(self.C.Handlers["click"])
	if len(path) < length || path[0:length]!=self.C.Handlers["click"] {
		return errors.New("Wrong path")
	}
    two := strings.SplitN(path[length:], ".", -1)
    rawimp := new(rawImp)
	user_id := rawimp.user_id
    err := UnpackFixedURL(rawimp, two[0])
	if err != nil { return err }
    self.ModelRawclick.ARGS = rawimp.ToArgs()
	err = self.ModelRawclick.Insert()
	if err != nil { return err }

	// making click cap refresh. If error, ignore
	item_id := rawimp.RAdv.ItemId
    self.ModelItem.ARGS = map[string]interface{}{"item_id":item_id}
    if err = self.ModelItem.Topics(map[string]interface{}{"creative_id":rawimp.RAdv.CreativeId}); err != nil {
		glog.Errorf("Get creatives of item %d: %v", item_id, err)
    } else if len(self.ModelItem.LISTS)>0 {
		creative := creativeFromTao(self.ModelItem.LISTS[0])
		item := &Item{rawimp.RAdv, creative.Cap}
        if item.Cap != nil && item.Cap.ClickNumber > 0 {
			if monitors, err := self.getMonitors(user_id); err == nil {
				if err = self.itemRefreshFcap(current, user_id, item, 2, monitors); err != nil {
					glog.Errorf("Refresh click: %v", err)
				}
			} else {
				glog.Errorf("Get monitors for %d: %v", user_id, err)
			}
        }
	}

    if len(two)==2 {
        click, err := url.QueryUnescape(two[1])
        if err != nil { return err }
		w.Header().Set("Location", click)
        w.WriteHeader(303)
    } else {
		w.Header().Set("Content-Type", "image/png")
        w.WriteHeader(200)
        data, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABAQMAAAAl21bKAAAAA1BMVEUAAACnej3aAAAAAXRSTlMAQObYZgAAAApJREFUCNdjYAAAAAIAAeIhvDMAAAAASUVORK5CYII=")
        w.Write(data)
	}
	return nil
}

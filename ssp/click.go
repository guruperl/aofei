package ssp

// "github.com/golang/glog"

/*
func (self *Controller) setCcookie(ctx context.Context, w http.ResponseWriter, user *User, campaignid, itemid uint32) {
	if item, err := self.RedisGetItem(ctx, itemid); err == nil {
		if item.ClickTotal > 0 {
			if user.CCaps == nil {
				user.CCaps = make(map[uint32]match.Fcap)
			}
			match.UpdateFcaps(&(user.CCaps), campaignid, user.FullTime)
			if str, err := match.PackFcaps(user.CCaps); err == nil {
				http.SetCookie(w, &http.Cookie{Name: self.C.Ccookie, Value: str, Path: "/", MaxAge: self.C.CcookieMaxAge})
			}
		}
	}
}

func (self *Controller) serveClick(ctx context.Context, w http.ResponseWriter, status pzutil.Status, user *User, adImp *match.AdImp, clk *match.Clk) {
	c := self.C
	header := w.Header()

	if status.Mime == pzutil.UnknownMime && clk.Click == "" {
		sizeid := adImp.GetSizeID()
		header.Set("Location", (c.Sizes)[sizeid].Click)
		w.WriteHeader(303)
		return
	}

	ipub := adImp.RPub
	record := match.Record{
		Imp: user.ToImp(status, ipub),
		Wins: []match.Win{{
			SlotID: ipub.SlotID,
			RAdv:   clk.RAdv,
		}},
	}
	if bs, err := record.Pack(); err == nil {
		self.Nc.Publish("user", bs)
		self.Nc.Flush()
	}

	switch status.Source {
	case pzutil.BROWSER, pzutil.MOBILE, pzutil.SDK:
		self.setCcookie(ctx, w, user, clk.RAdv.CampaignID, clk.RAdv.ItemID)
	default:
	}

	switch status.Mime {
	case pzutil.UnknownMime:
		header.Set("Location", clk.Click)
		w.WriteHeader(303)
		return
	case pzutil.GIF:
		header.Set("Content-Type", "image/gif")
		w.WriteHeader(200)
		data, _ := base64.StdEncoding.DecodeString(pzutil.GifPixel)
		w.Write(data)
		return
	case pzutil.PNG:
		header.Set("Content-Type", "image/png")
		w.WriteHeader(200)
		data, _ := base64.StdEncoding.DecodeString(pzutil.PngPixel)
		w.Write(data)
		return
	default:
		header.Set("Content-Type", "application/x-javascript")
		w.WriteHeader(200)
		w.Write([]byte(""))
		return
	}
}
*/

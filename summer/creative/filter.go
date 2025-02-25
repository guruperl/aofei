package creative

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/genelet/winter/summer"
)

type Filter struct {
	summer.Filter
}

func (self *Filter) Preset() error {
	if err := self.Filter.Preset(); err != nil {
		return err
	}

	return nil
}

func (self *Filter) Before(model *Model, extra url.Values, nextextra url.Values) error {
	if err := self.Filter.Before(&model.Model, extra, nextextra); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action

	if action == "topics" {
		extra.Set("item_id", self.R.Form.Get("item_id"))
	} else if action == "insert" {
		medias := []string{"media_1", "media_2", "media_3", "media_4", "media_5"}
		found := false
		for _, fn := range medias {
			if ARGS.Get(fn) != "" {
				found = true
				break
			}
		}
		if found {
			item_id := ARGS.Get("item_id")
			dir := self.C.UploadDir + "/" + item_id
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err = os.Mkdir(dir, 0755); err != nil {
					return err
				}
			}
			for i, fn := range medias {
				file := ARGS.Get(fn)
				if file == "" {
					continue
				}
				err := self.Uploading(dir, item_id, file, strconv.Itoa(i+1))
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (self *Filter) After(model *Model) error {
	if err := self.Filter.After(&model.Model); err != nil {
		return err
	}

	ARGS := self.R.Form
	action := self.Action
	//who := self.RoleValue
	lists := *model.LISTS
	//other := *model.OTHER

	if action == "insert" && ARGS.Get("media") != "" {
		for i, m := range ARGS["media"] {
			if err := model.DoSQL(
				`INSERT INTO adv_media (creative_id, series, media, disk, mime, created)
VALUES (?,?,?,?,?,NOW())`, lists[0]["creative_id"], ARGS["series"][i], m,
				ARGS["disk"][i], ARGS["mime"][i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func (self *Filter) Uploading(dir, item_id, file, series string) error {
	ARGS := self.R.Form

	fh, err := os.Open(self.C.UploadDir + "/" + file)
	if err != nil {
		return err
	}
	defer fh.Close()

	buffer := make([]byte, 512)
	_, err = fh.Read(buffer)
	if err != nil {
		return err
	}
	mime := http.DetectContentType(buffer)
	if ARGS.Get("qa_mime") == "video" && mime == "application/octet-stream" {
		arrs := strings.Split(file, ".")
		popular := map[string]string{
			"m3u": "application/x-mpegURL", "m3u8": "application/x-mpegURL",
			"flv": "video/x-flv", "mp4": "video/mp4", "ogg": "video/ogg",
			"webm": "video/webm", "m4v": "video/x-m4v", "ts": "video/MP2T",
			"3gp": "video/3gpp", "mov": "video/quicktime", "avi": "video/x-msvideo",
			"asf": "video/ms-asf", "wma": "video/ms-asf", "wmv": "video/x-ms-wmv"}
		if m, ok := popular[strings.ToLower(arrs[len(arrs)-1])]; ok {
			mime = m
		}
	}

	err = os.Rename(self.C.UploadDir+"/"+file, dir+"/"+file)
	if err != nil {
		return err
	}

	media := self.C.UploadURL + "/" + item_id + "/" + file
	ARGS.Add("mime", mime)
	ARGS.Add("series", series)
	ARGS.Add("media", media)
	ARGS.Add("disk", dir+"/"+file)
	ARGS.Set("content", strings.Replace(ARGS.Get("content"), "MEDIA_"+series, media, -1))
	ARGS.Set("content", strings.Replace(ARGS.Get("content"), `"`, `\"`, -1))

	return nil
}

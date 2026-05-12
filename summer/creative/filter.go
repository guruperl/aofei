// Package creative implements creative modules, including uploading
package creative

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	ARGS := self.R.Form
	action := self.Action
	if action == "insert" || action == "update" {
		err := summer.SetSizeID(ARGS)
		if err != nil {
			return err
		}
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
		switch ARGS.Get("randomChoice") {
		case "2", "3":
			itemID := ARGS.Get("item_id")
			dir := filepath.Join(self.C.UploadDir, itemID)
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				if err = os.MkdirAll(dir, 0755); err != nil {
					return err
				}
			}
			for _, fn := range []string{"media_1", "media_2"} {
				if ARGS.Get("randomChoice") == "3" && fn == "media_1" {
					continue
				}
				if ARGS.Get("randomChoice") == "2" && fn == "media_2" {
					continue
				}
				file := ARGS.Get(fn)
				if file == "" {
					continue
				}
				file, err := summer.CleanUploadName(file)
				if err != nil {
					return err
				}
				if err := self.uploading(dir, itemID, file, "1"); err != nil {
					return err
				}
			}
		default:
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
			if err := model.DoSQL(`
INSERT INTO adv_media (creative_id, series, media, disk, mime, created)
VALUES (?,?,?,?,?,NOW())`,
				lists[0]["creative_id"], ARGS["series"][i], m, ARGS["disk"][i], ARGS["mime"][i]); err != nil {
				return err
			}
		}
	} else if action == "topics" {
		for _, item := range lists {
			summer.SetWH(item)
		}
	}

	return nil
}

func (self *Filter) uploading(dir, itemID, file, series string) error {
	ARGS := self.R.Form

	file, err := summer.CleanUploadName(file)
	if err != nil {
		return err
	}
	fh, err := os.Open(filepath.Join(self.C.UploadDir, file))
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
	if ARGS.Get("randomChoice") == "3" && mime == "application/octet-stream" {
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

	dest := filepath.Join(dir, file)
	err = os.Rename(filepath.Join(self.C.UploadDir, file), dest)
	if err != nil {
		return err
	}

	media := self.C.UploadURL + "/" + itemID + "/" + file
	ARGS.Add("mime", mime)
	ARGS.Add("series", series)
	ARGS.Add("media", media)
	ARGS.Add("disk", dest)
	switch ARGS.Get("randomChoice") {
	case "2":
		ARGS.Set("content", media)
	case "3":
		ARGS.Set("media_type", "Video")
		ARGS.Set("content", `<video controls><source src="`+media+`" type="`+mime+`">not supported</video>`)
	case "4":
	// TODO build native adm
	default:
	}

	return nil
}

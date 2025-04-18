package uploaded

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type UploadAudience struct {
	Uploads uint32
}

func (self *UploadAudience) Has(ctx context.Context, conn radix.Client, bid *openrtb2.BidRequest, advID uint32) (bool, error) {
	if self.Uploads == 0 {
		return false, nil
	}

	var did, dpid, mac string
	if bid.Device != nil {
		did = bid.Device.DIDMD5
		if did == "" {
			did = bid.Device.DIDSHA1
		}
		dpid = bid.Device.DPIDMD5
		if dpid == "" {
			dpid = bid.Device.DPIDSHA1
		}
		mac = bid.Device.MACMD5
		if mac == "" {
			mac = bid.Device.MACSHA1
		}
	}

	args := map[UploadType]string{
		UploadBuyerUID: bid.User.BuyerUID,
		UploadUserID:   bid.User.ID,
		UploadIP:       bid.Device.IP,
		UploadIFA:      bid.Device.IFA,
		UploadDID:      did,
		UploadDPID:     dpid,
		UploadMAC:      mac,
	}

	for typ, v := range args {
		if v == "" {
			continue
		}
		if !self.hasUpload(typ) {
			continue
		}
		ok, err := self.findUploaded(ctx, conn, advID, UploadType2String[typ], v)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}

// uploadName is the name of the audience data in Redis.
func uploadName(advID uint32, marker string) string {
	return fmt.Sprintf("upload:%d:%s", advID, marker)
}

// findUploaded checks if the given value is present in the audience data.
func (self *UploadAudience) findUploaded(ctx context.Context, conn radix.Client, advID uint32, marker string, target string) (bool, error) {
	if marker == "" || target == "" {
		return false, nil
	}

	var ok int
	err := conn.Do(ctx, radix.Cmd(&ok, "SISMEMBER", uploadName(advID, marker), target))
	if err != nil {
		return false, err
	}
	if ok == 1 {
		return true, nil
	}

	return false, nil
}

func UploadSingle(ctx context.Context, conn radix.Client, advID uint32, marker string, single string) error {
	return conn.Do(ctx, radix.Cmd(nil, "SADD", uploadName(advID, marker), single))
}

func UploadMany(ctx context.Context, conn radix.Client, advID uint32, marker string, data []string) error {
	if len(data) == 0 {
		return nil
	}

	arr := make([]string, len(data)+1)
	arr[0] = uploadName(advID, marker)
	for i, d := range data {
		arr[i+1] = d
	}
	return conn.Do(ctx, radix.Cmd(nil, "SADD", arr...))
}

// UploadedResetArgs resets the ARGS to the values in the DemoAudience, ready to be inserted or updated in the database.
func UploadedResetArgs(ARGS url.Values) error {
	var par []uint32
	if values, ok := ARGS["uploads"]; ok {
		for _, value := range values {
			if value == "" || value == "0" {
				par = nil
				break
			}
			v, err := strconv.ParseInt(value, 10, 32)
			if err != nil {
				return err
			}
			par = append(par, uint32(v))
		}
	}

	if par == nil {
		ARGS.Del("uploads")
		return nil
	}

	uploads := NewUploads(par)
	ARGS.Set("uploads", strconv.FormatInt(int64(uploads), 10))

	return nil
}

func (self *UploadAudience) hasUpload(uploadType UploadType) bool {
	if uploadType == UploadUnknown {
		return false
	}
	if self.Uploads&(1<<uploadType) != 0 {
		return true
	}

	return false
}

// Tmpls returns the map of attribute name and valueID ready to use on web page.
func (self *UploadAudience) Tmpls() map[int][]interface{} {
	uploads := make(map[int][]interface{})
	for str, typ := range String2UploadType {
		if self.Uploads == 0 {
			if typ == UploadUnknown {
				uploads[int(typ)] = []interface{}{str, true}
			} else {
				uploads[int(typ)] = []interface{}{str, false}
			}
		} else {
			uploads[int(typ)] = []interface{}{str, self.hasUpload(typ)}
		}
	}
	return uploads
}

// DBFillUploadAudience fills UploadAudience with attribute name and valueID, derived from the database.
func (self *UploadAudience) DBFillUploadAudience(attrname string, valueID uint32) int {
	switch attrname {
	case "uploads":
		self.Uploads = valueID
		return 1
	default:
	}

	return 0
}

package uploaded

import (
	"context"
	"fmt"

	"github.com/mediocregopher/radix/v4"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type UploadAudience struct {
	ItemID  uint32
	Uploads uint32
}

func (self *UploadAudience) Has(ctx context.Context, conn radix.Client, bid *openrtb2.BidRequest) (bool, error) {
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

	args := map[string]string{
		"ip":       bid.Device.IP,
		"userid":   bid.User.ID,
		"buyeruid": bid.User.BuyerUID,
		"ifa":      bid.User.BuyerUID,
		"did":      did,
		"dpid":     dpid,
		"mac":      mac,
	}

	for _, k := range WUploads(self.Uploads).ToStrings() {
		v, ok := args[k]
		if !ok {
			continue
		}
		ok, err := self.findUploaded(ctx, conn, k, v)
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
func (self *UploadAudience) uploadName(marker string) string {
	return fmt.Sprintf("upload:%d:%s", self.ItemID, marker)
}

// findUploaded checks if the given value is present in the audience data.
func (self *UploadAudience) findUploaded(ctx context.Context, conn radix.Client, marker string, target string) (bool, error) {
	if marker == "" || target == "" {
		return false, nil
	}

	var ok int
	err := conn.Do(ctx, radix.Cmd(&ok, "SISMEMBER", self.uploadName(marker), target))
	if err != nil {
		return false, err
	}
	if ok == 1 {
		return true, nil
	}

	return false, nil
}

func (self *UploadAudience) UnpackAudience(ctx context.Context, conn radix.Client, marker string, data []string) error {
	if len(data) == 0 {
		return nil
	}

	arr := make([]string, len(data)+1)
	arr[0] = self.uploadName(marker)
	for i, d := range data {
		arr[i+1] = d
	}
	return conn.Do(ctx, radix.Cmd(nil, "SADD", arr...))
}

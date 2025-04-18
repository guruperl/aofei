// Package uploaded handles uploaded audience data.
package uploaded

type UploadType int8

const (
	UploadUnknown UploadType = iota
	UploadBuyerUID
	UploadUserID
	UploadIP
	UploadIFA
	UploadDID
	UploadDPID
	UploadMAC
)

var String2UploadType = map[string]UploadType{
	"All":      UploadUnknown,
	"buyeruid": UploadBuyerUID,
	"userid":   UploadUserID,
	"ip":       UploadIP,
	"ifa":      UploadIFA,
	"did":      UploadDID,
	"dpid":     UploadDPID,
	"mac":      UploadMAC,
}
var UploadType2String = map[UploadType]string{
	UploadUnknown:  "All",
	UploadBuyerUID: "buyeruid",
	UploadUserID:   "userid",
	UploadIP:       "ip",
	UploadIFA:      "ifa",
	UploadDID:      "did",
	UploadDPID:     "dpid",
	UploadMAC:      "mac",
}

type WUploads uint32

// NewUploads creates WUploads from a list of upload types.
func NewUploads(uploads []uint32) WUploads {
	var wuploads WUploads
	for _, upload := range uploads {
		if upload > 7 {
			continue
		}
		wuploads |= (1 << upload)
	}
	return wuploads
}

// ToStrings converts WUploads to array of strings.
func (self WUploads) ToStrings() []string {
	var arr []string
	for i := 0; i < 8; i++ {
		if self&(1<<i) != 0 {
			arr = append(arr, UploadType2String[UploadType(i)])
		}
	}
	return arr
}

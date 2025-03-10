package holiday

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type NativeTitle struct {
	Len int                    `json:"len"`
	Ext map[string]interface{} `json:"ext"`
}

type NativeImg struct {
	Type  int                    `json:"type"`
	W     uint16                 `json:"w"`
	H     uint16                 `json:"h"`
	WMin  uint16                 `json:"wmin"`
	HMin  uint16                 `json:"hmin"`
	Mimes []string               `json:"mimes"`
	Ext   map[string]interface{} `json:"ext"`
}

func (self *NativeImg) Sizes() (uint16, uint16) {
	if self.W > 0 && self.H > 0 {
		return self.W, self.H
	}
	return self.WMin, self.HMin
}

type NativeVideo struct {
	Mimes       []string               `json:"mimes"`
	Minduration int                    `json:"minduration"`
	Maxduration int                    `json:"maxduration"`
	Protocols   []int                  `json:"protocols"`
	Ext         map[string]interface{} `json:"ext"`
}

type NativeData struct {
	Type int                    `json:"type"`
	Len  int                    `json:"len"`
	Ext  map[string]interface{} `json:"ext"`
}

type AssetType struct {
	ID       int                    `json:"id"`
	Required int                    `json:"required"`
	Title    *NativeTitle           `json:"title"`
	Image    *NativeImg             `json:"img"`
	Video    *NativeVideo           `json:"video"`
	Data     *NativeData            `json:"data"`
	Ext      map[string]interface{} `json:"ext"`
}

type NativeType struct {
	Ver      string                 `json:"ver"`
	Layout   int                    `json:"layout"`
	Adunit   int                    `json:"adunit"`
	Plcmtcnt int                    `json:"plcmtnt"`
	Seq      int                    `json:"seq"`
	Assets   []*AssetType           `json:"assets"`
	Ext      map[string]interface{} `json:"ext"`
}

func NewNativeType(native *openrtb2.Native) (*NativeType, error) {
	if native != nil && native.Request != "" {
		x := map[string]*NativeType{}
		if err := json.Unmarshal([]byte(native.Request), &x); err != nil {
			return nil, err
		}
		return x["native"], nil
	}

	return nil, nil
}

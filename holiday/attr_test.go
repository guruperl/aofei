package holiday

import (
	"testing"
)

func TestAttr(t *testing.T) {
	attrs := NewAttrsFromNames([]string{
"continent", "country", "state", "city", "dma", "zip", "isp", "bandwidth", "areacode"})
	ids := attrs.GetAttrIDs()
	for i, id := range ids {
		if IndexUint32([]uint32{1101,1102,1103,1104,1105,1106,1107,1108,1111}, id) < 0 {
			t.Errorf("%#v", attrs.AttrArray[i])
		}
	}
}

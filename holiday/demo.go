package holiday

import (
	"demo"
)

type Demo struct {
    demo.Demo
	YOB int64
	GENDER string
}

func (self *Demo) GetTags() *Tags {
	if self.YOB == 0 && self.GENDER == "" { return nil }

	ref := make(map[uint32][]uint32)
    if self.GENDER != "" {
		self.Gender = demo.GenderType_value[self.GENDER]
		ref[1301] = []uint32{uint32(self.Gender)}
	}
    if self.YOB > 0 {
		self.Yob = uint32(self.YOB-1937)
		ref[1302] = []uint32{uint32(self.Yob)}
	}

	return &Tags{TagHashArray:ref}
}

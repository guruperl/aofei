package slot

import (
	"time"
	"fmt"
	"github.com/genelet/taodbi"
)

type Model struct {
	taodbi.Smodel
}

func (self *Model)Inserts(lists []map[string]interface{}) error {
	self.CurrentTable = fmt.Sprintf("slot_%d", self.ARGS[self.Tags[0]])
	self.Tags = nil
	self.InsertPars = []string{"slot_id", "adv_id", "campaign_id", "item_id", "cost_type", "price", "endx", "cpm_fc", "cpm_length", "cpm_throttle", "cpc_fc", "cpc_length"}
	for _, item := range lists {
		if item["release"] != nil {
			delete(item, "release")
		}
		if err := self.InsertHash(item); err != nil {
			return err
		}
		time.Sleep(1 * time.Millisecond)
	}
	self.CurrentTable = "slot"
	self.Tags = []string{"release"}
	self.InsertPars = []string{"slot_id", "adv_id", "campaign_id", "item_id", "cost_type", "price", "endx", "release", "cpm_fc", "cpm_length", "cpm_throttle", "cpc_fc", "cpc_length"}
	return nil
}

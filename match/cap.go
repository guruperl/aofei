package match

import (
	"time"
)

// Cap is a struct that contains the cap information. It is of 8 bytes
type Cap struct {
	CapNumber   uint8
	CapPeriod   uint16
	CapThrottle uint16
	ClickNumber uint8
	ClickPeriod uint16
}

func capFromTao(hash map[string]interface{}) Cap {
	c := Cap{}
	if i, ok := hash["cpm_fc"]; ok {
		c.CapNumber = uint8(i.(int32))
	}
	if i, ok := hash["cpm_length"]; ok {
		c.CapPeriod = uint16(i.(int32))
	}
	if i, ok := hash["cpm_throttle"]; ok {
		c.CapThrottle = uint16(i.(int32))
	}
	if i, ok := hash["cpc_fc"]; ok {
		c.ClickNumber = uint8(i.(int32))
	}
	return c
}

func (self Cap) ValidPeriodImp(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.CapPeriod
}

func (self Cap) ValidPeriodCli(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.ClickPeriod
}

func (self Cap) CanServeImp(when time.Time, fcap Fcap) bool {
	if self.CapThrottle > 0 && fcap.SinceLast(when) < self.CapThrottle {
		return false
	}
	if self.CapNumber > 0 && self.ValidPeriodImp(when, fcap) &&
		fcap.Total >= self.CapNumber {
		return false
	}
	return true
}

func (self Cap) CanServeCli(when time.Time, fcap Fcap) bool {
	if self.ClickNumber > 0 && self.ValidPeriodCli(when, fcap) &&
		fcap.Total >= self.ClickNumber {
		return false
	}
	return true
}

func (self Cap) CanServe(when time.Time, object BothCap) bool {
	if self.CanServeImp(when, object.Imp) == false {
		return false
	}
	if self.CanServeCli(when, object.Cli) == false {
		return false
	}
	return true
}

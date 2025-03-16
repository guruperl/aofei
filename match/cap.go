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

func (self Cap) ValidPeriodImp(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.CapPeriod
}

func (self Cap) ValidPeriodCli(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.ClickPeriod
}

func (self Cap) canServeImp(when time.Time, fcap Fcap) bool {
	if self.CapThrottle > 0 && fcap.SinceLast(when) < self.CapThrottle {
		return false
	}
	if self.CapNumber > 0 && self.ValidPeriodImp(when, fcap) && fcap.Total >= self.CapNumber {
		return false
	}
	return true
}
func (self Cap) canServeCli(when time.Time, fcap Fcap) bool {
	if self.ClickNumber > 0 && self.ValidPeriodCli(when, fcap) && fcap.Total >= self.ClickNumber {
		return false
	}
	return true
}
func (self Cap) CanServe(when time.Time, object BothCap) bool {
	if !self.canServeImp(when, object.Imp) {
		return false
	}
	if !self.canServeCli(when, object.Cli) {
		return false
	}
	return true
}

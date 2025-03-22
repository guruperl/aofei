package match

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"time"
)

// Cap is a struct that contains the cap information. It is of 8 bytes
type Cap struct {
	CapNumber   uint8  `json:"cap_number,omitempty"`
	CapPeriod   uint16 `json:"cap_period,omitempty"`
	CapThrottle uint16 `json:"cap_throttle,omitempty"`
	ClickNumber uint8  `json:"click_number,omitempty"`
	ClickPeriod uint16 `json:"click_period,omitempty"`
}

// Pack packs the Cap to bytes
func (self Cap) Pack() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, self)
	return buf.Bytes(), err
}

// PackString packs the Cap to RawURL string
func (self Cap) PackString() (string, error) {
	ba, err := self.Pack()
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(ba), nil
}

// UnpackCap unpacks the Cap from bytes
func UnpackCap(data []byte) (Cap, error) {
	buf := bytes.NewReader(data)
	cap := Cap{}
	err := binary.Read(buf, binary.LittleEndian, &cap)
	return cap, err
}

// UnpackCapString unpacks the Cap from RawURL string
func UnpackCapString(text string) (Cap, error) {
	data, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil {
		return Cap{}, err
	}
	return UnpackCap(data)
}

// ValidPeriodImp checks if the impression is valid in the period.
// If not, the cap is expired and the impression is allowed to serve.
func (self Cap) ValidPeriodImp(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.CapPeriod
}

// ValidPeriodCli checks if the click is valid in the period.
// If not, the cap is expired and the impression is allowed to serve.
func (self Cap) ValidPeriodCli(when time.Time, fcap Fcap) bool {
	return fcap.SinceStart(when) < self.ClickPeriod
}

// canServeImp checks if the cap can serve the impression, althought it is a capped ad
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

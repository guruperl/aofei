package tencent

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	Dsp_id string
	Token string
	Timeout int
}

func NewConfig(filename string) *Config {
	parsed := new(Config)
	content, err := ioutil.ReadFile(filename)
	if err != nil { panic(err) }
	err = json.Unmarshal(content, parsed)
	if err != nil { panic(err) }

	if parsed.Dsp_id=="" { panic("dsp_id") }
	if parsed.Token=="" { panic("token") }
	if parsed.Timeout==0 {
		parsed.Timeout = 30
	}

	return parsed
}

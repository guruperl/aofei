package holiday

import (
	"encoding/json"
	"io/ioutil"
)

type PSA struct {
    W    uint16
    H    uint16
    Display  string
    Click  string
	Price float32
}

type Config struct {
	DocumentRoot string `json:"document_root"`
	ServerURL	 string `json:"server_url"`
	Handlers     map[string]string `json:"handlers,omitempty"`
	HhLock		 string `json:"hhlock,omitempty"`
	Ips          string `json:"ipsearch,omitempty"`
	Aofei	     string `json:"aofei,omitempty"`
	Telecom	     string `json:"telecom,omitempty"`
	DefEncrypt   string `json:"def_encrypt,omitempty"`
	AndroidEncrypt string `json:"android_encrypt,omitempty"`
	IphoneEncrypt  string `json:"iphone_encrypt,omitempty"`
	Db		   []string `json:"db,omitempty"`
	Ucookie		 string `json:"ucookie,omitempty"`
	UcookieMaxAge   int `json:"ucookie_maxage,omitempty"`
	PSAs	map[uint32]*PSA `json:"sizes"`
}

func NewConfig(filename string) (*Config, error) {
	parsed := new(Config)
	content, err := ioutil.ReadFile(filename)
	if err != nil { return nil, err }
	err = json.Unmarshal(content, parsed)
	if err != nil { return nil, err }

	if parsed.Ips == ""   {
		parsed.Ips = "qq-pz.dat"
	}
	if parsed.HhLock  == ""   {
		parsed.HhLock = "/var/tmp/hh.lock"
	}
	if parsed.Ucookie == "" {
		parsed.Ucookie = "uid"
	}
    if parsed.UcookieMaxAge == 0 {
		parsed.UcookieMaxAge = 15553000
	}

	if parsed.Handlers == nil {
		parsed.Handlers = map[string]string{
			"api":"/api", "ssp":"/ssp", "click":"/click", "static":"/static"}
	}

	return parsed, nil
}

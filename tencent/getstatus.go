package tencent

import (
	"encoding/json"
)

type FileUrlInfo struct {
	File_url_info []string
}

type GetstatusRecord struct {
    Status string
    Reason string
}

type GetstatusMsg struct {
	Total int
	Count int
	Records map[string]map[string]GetstatusRecord
}

type Getstatus struct {
    Ret_code int
	Error_code int
	Ret_msg *GetstatusMsg
}

func ParseGetstatus(body []byte) (*Getstatus, error) {
	parsed := new(Getstatus)
	err := json.Unmarshal(body, parsed)
	return parsed, err
}

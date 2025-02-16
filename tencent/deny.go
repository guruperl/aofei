package tencent

import (
	"encoding/json"
)

type DenyRecord struct {
    File_url string
    Targeting_url string
    Reason string
}

type DenyMsg struct {
	Total int
	Count int
	Records []DenyRecord
}

type Deny struct {
    Ret_code int
	Error_code int
	Ret_msg *DenyMsg
}

func Parse(body []byte) (*Deny, error) {
	parsed := new(Deny)
	err := json.Unmarshal(body, parsed)
	return parsed, err
}

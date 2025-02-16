package tencent

import (
	"encoding/json"
)

type NamesInfo struct {
	Names []string
}

type InfoClientRecord struct {
    Verify_status string
    Audit_info string
	Is_black string
}

type InfoClient struct {
    Ret_code int
	Error_code int
	Ret_msg map[string]InfoClientRecord
}

func ParseInfoClient(body []byte) (*InfoClient, error) {
	parsed := new(InfoClient)
	err := json.Unmarshal(body, parsed)
	return parsed, err
}

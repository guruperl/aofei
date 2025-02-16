package tencent

import (
	"encoding/json"
)

type QualiClientRecord struct {
    Client_id string
    Name string
    Url string
    Is_black string
    Verify_status string
    Add_time string
    Memo string
    Type string
}

type QualiClientMsg struct {
	Total int
	Count int
	Records []QualiClientRecord
}

type QualiClient struct {
    Ret_code int
	Error_code int
	Ret_msg *QualiClientMsg
}

func ParseQualiClient(body []byte) (*QualiClient, error) {
	parsed := new(QualiClient)
	err := json.Unmarshal(body, parsed)
	return parsed, err
}

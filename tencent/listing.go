package tencent

import (
	"encoding/json"
)

type ListingRecord struct {
    Location_id string
    Location_name string
    Style string
    Cpm_start_price float32
    Block_vocation []string
    Allow_file []string
    Screen string
    Review_pic string
}

type ListingMsg struct {
	Total int
	Count int
	Records []ListingRecord
}

type Listing struct {
    Ret_code int
	Error_code int
	Ret_msg *ListingMsg
}

func ParseListing(body []byte) (*Listing, error) {
	parsed := new(Listing)
	err := json.Unmarshal(body, parsed)
	return parsed, err
}

package dsp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Red struct {
	Network string
	Addr    string
	User    string
	Pass    string
	Size    int
}

type Config struct {
	DocumentRoot string   `json:"document_root"`
	ServerURL    string   `json:"server_url"`
	ServerPort   string   `json:"server_port"`
	HhLock       string   `json:"hhlock,omitempty"`
	Ips          string   `json:"ips,omitempty"`
	Redis        *Red     `json:"redis,omitempty"`
	NatsURL      string   `json:"nats_url,omitempty"`
	ConnectArray []string `json:"connect_array,omitempty"`
	Spread       string   `json:"spread,omitempty"`
	LogRequest   string   `json:"log_request,omitempty"`
	LogResponse  string   `json:"log_response,omitempty"`
	LogAttribute string   `json:"log_attribute,omitempty"`
	LogWinLoss   string   `json:"log_winloss,omitempty"`
}

func NewConfig(filename string) (*Config, error) {
	parsed := new(Config)
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, parsed)
	if err != nil {
		return nil, err
	}

	if parsed.Ips == "" {
		parsed.Ips = "qq-pz.dat"
	}
	if parsed.HhLock == "" {
		parsed.HhLock = "/var/tmp/hh.lock"
	}
	if parsed.ConnectArray == nil {
		if os.Getenv("DBUSER") != "" && os.Getenv("DBPASS") != "" && os.Getenv("DBNAME") != "" {
			host := "localhost:3306"
			if x := os.Getenv("DBHOST"); x != "" {
				host = x
				if !strings.Contains(host, ":") {
					host += ":3306"
				}
			}
			parsed.ConnectArray = []string{"mysql", os.Getenv("DBUSER") + ":" + os.Getenv("DBPASS") + "@tcp(" + host + ")/" + os.Getenv("DBNAME")}
		} else {
			return nil, fmt.Errorf("ConnectArray is not set")
		}

	}

	if parsed.Redis.Network == "" {
		parsed.Redis.Network = "tcp"
	}
	if parsed.Redis.User == "" && os.Getenv("REDISUSER") != "" {
		parsed.Redis.User = os.Getenv("REDISUSER")
	}
	if parsed.Redis.Pass == "" && os.Getenv("REDISPASS") != "" {
		parsed.Redis.Pass = os.Getenv("REDISPASS")
	}
	if parsed.Redis.Addr == "" && os.Getenv("REDISHOST") != "" {
		parsed.Redis.Addr = os.Getenv("REDISHOST")
	}
	if parsed.Redis.Addr == "" {
		parsed.Redis.Addr = "localhost"
	}
	if !strings.Contains(parsed.Redis.Addr, ":") {
		parsed.Redis.Addr += ":6379"
	}

	if parsed.ServerPort == "" {
		parsed.ServerPort = "80"
	}

	if parsed.NatsURL == "" {
		parsed.NatsURL = "nats://localhost:4222"
	}

	return parsed, nil
}

type Stamp struct {
	CurrentMinute int64
	LastMinute    int64
	CurrentTimely time.Time
	LastTimely    time.Time
	Interval      int
	CurrentDay    string
	LastDay       string
}

// NewStamp creates a new Stamp with the given interval in minutes.
func NewStamp(interval int, stamp ...int) *Stamp {
	when := time.Now()
	current := when.Unix() / int64(interval*60)
	lastMinute := current - 1
	if len(stamp) > 0 {
		lastMinute = int64(stamp[0])
	}
	currentTimely := time.Unix(current*int64(interval*60), 0)
	lastTimely := time.Unix(lastMinute*int64(interval*60), 0)
	return &Stamp{current, lastMinute, currentTimely, lastTimely, interval, when.Format("2006-01-02"), when.AddDate(0, 0, -1).Format("2006-01-02")}
}

// NewLogfileName creates a new file name with the given logname and timestamp.
func (self *Config) NewLogfileName(name string, stamp *Stamp, uptonow ...bool) string {
	if len(uptonow) > 0 && uptonow[0] {
		switch name {
		case SUBJECTWinLoss:
			return fmt.Sprintf(self.LogWinLoss+"/winloss.%d", stamp.CurrentMinute)
		case SUBJECTAttribute:
			return fmt.Sprintf(self.LogAttribute+"/attribute.%d", stamp.CurrentMinute)
		case SUBJECTRequest:
			return fmt.Sprintf(self.LogRequest+"/request.%d", stamp.CurrentMinute)
		case SUBJECTResponse:
			return fmt.Sprintf(self.LogResponse+"/response.%d", stamp.CurrentMinute)
		}
	} else {
		switch name {
		case SUBJECTWinLoss:
			return fmt.Sprintf(self.LogWinLoss+"/winloss.%d", stamp.LastMinute)
		case SUBJECTAttribute:
			return fmt.Sprintf(self.LogAttribute+"/attribute.%d", stamp.LastMinute)
		case SUBJECTRequest:
			return fmt.Sprintf(self.LogRequest+"/request.%d", stamp.LastMinute)
		case SUBJECTResponse:
			return fmt.Sprintf(self.LogResponse+"/response.%d", stamp.LastMinute)
		}
	}
	return ""
}

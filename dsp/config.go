package dsp

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	RPubMap      string   `json:"rpub_map,omitempty"`
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

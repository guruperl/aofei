package pzutil

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
	DocumentRoot string
	Handle       map[string]string
	ServerURL    string
	Logfile      string
	ServerPort   string
	HhLock       string
	Ips          string
	Redis        Red
	NatsURL      string
	ConnectArray []string

	SITE     string
	SLOT     string
	ITEM     string
	WEIGHT   string
	AUDIENCE string

	Ucookie       string
	UcookieMaxAge int
	Icookie       string
	IcookieMaxAge int
	Ccookie       string
	CcookieMaxAge int

	Sizes map[uint32]Size
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
	if parsed.HhLock == "" {
		parsed.HhLock = "/var/tmp/hh.lock"
	}
	if parsed.NatsURL == "" {
		parsed.NatsURL = "nats://localhost:4222"
	}
	if parsed.SITE == "" {
		parsed.SITE = "SITE"
	}
	if parsed.SLOT == "" {
		parsed.SLOT = "SLOT"
	}
	if parsed.ITEM == "" {
		parsed.SLOT = "ITEM"
	}
	if parsed.AUDIENCE == "" {
		parsed.AUDIENCE = "AUDIENCE"
	}

	if parsed.Ucookie == "" {
		parsed.Ucookie = "uid"
	}
	if parsed.UcookieMaxAge == 0 {
		parsed.UcookieMaxAge = 15553000
	}
	if parsed.Icookie == "" {
		parsed.Icookie = "i"
	}
	if parsed.IcookieMaxAge == 0 {
		parsed.IcookieMaxAge = 3888000
	}
	if parsed.Ccookie == "" {
		parsed.Ccookie = "c"
	}
	if parsed.CcookieMaxAge == 0 {
		parsed.CcookieMaxAge = 3888000
	}

	if parsed.Handle == nil {
		parsed.Handle = map[string]string{"ssp": "/ssp"}
	}

	return parsed, nil
}

package dsp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mediocregopher/radix/v4"
)

type Red struct {
	Network string
	Addr    string
	User    string
	Pass    string
	Size    int
}

type Config struct {
	DocumentRoot                string   `json:"document_root"`
	ServerURL                   string   `json:"server_url"`
	ServerPort                  string   `json:"server_port"`
	HhLock                      string   `json:"hhlock,omitempty"`
	Ips                         string   `json:"ips,omitempty"`
	Redis                       *Red     `json:"redis,omitempty"`
	NatsURL                     string   `json:"nats_url,omitempty"`
	TrackingSecret              string   `json:"tracking_secret,omitempty"`
	TrackingSignatureTTLSeconds int      `json:"tracking_signature_ttl_seconds,omitempty"`
	ConnectArray                []string `json:"connect_array,omitempty"`
	Spread                      string   `json:"spread,omitempty"`
	IsLocal                     bool     `json:"is_local,omitempty"`
	MiddlemanEnabled            bool     `json:"middleman_enabled,omitempty"`
	MiddlemanAlwaysEnabled      bool     `json:"middleman_always_enabled,omitempty"`
	MiddlemanTimeoutMS          int      `json:"middleman_timeout_ms,omitempty"`
	MiddlemanMaxBiddersPerImp   int      `json:"middleman_max_bidders_per_imp,omitempty"`
	MiddlemanExchangeDomain     string   `json:"middleman_exchange_domain,omitempty"`
	MiddlemanCallbackTTLSeconds int      `json:"middleman_callback_ttl_seconds,omitempty"`
	MiddlemanCallbackTimeoutMS  int      `json:"middleman_callback_timeout_ms,omitempty"`
	MiddlemanCallbackBaseURL    string   `json:"middleman_callback_base_url,omitempty"`
	LogRequest                  string   `json:"log_request,omitempty"`
	LogResponse                 string   `json:"log_response,omitempty"`
	LogAttribute                string   `json:"log_attribute,omitempty"`
	LogWinLoss                  string   `json:"log_winloss,omitempty"`
	DBMaxOpenConns              int      `json:"db_max_open_conns,omitempty"`
	DBMaxIdleConns              int      `json:"db_max_idle_conns,omitempty"`
	DBConnMaxLifetimeSeconds    int      `json:"db_conn_max_lifetime_seconds,omitempty"`
	DBConnMaxIdleTimeSeconds    int      `json:"db_conn_max_idle_time_seconds,omitempty"`
}

type ConfigMode string

const (
	ConfigModeBid      ConfigMode = "bid"
	ConfigModeCache    ConfigMode = "cache"
	ConfigModeLedger   ConfigMode = "ledger"
	ConfigModeRetry    ConfigMode = "retry"
	ConfigModeSpread   ConfigMode = "spread"
	ConfigModeMaxMind  ConfigMode = "maxmind"
	ConfigModeNATS     ConfigMode = "nats"
	ConfigModeDatabase ConfigMode = "database"
	ConfigModeRedis    ConfigMode = "redis"
)

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

	if parsed.Redis == nil {
		parsed.Redis = &Red{}
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
	if parsed.TrackingSecret == "" {
		parsed.TrackingSecret = os.Getenv("TRACKING_SECRET")
	}
	if parsed.TrackingSignatureTTLSeconds <= 0 {
		parsed.TrackingSignatureTTLSeconds = int(defaultTrackingSignatureTTL.Seconds())
	}
	if parsed.MiddlemanTimeoutMS <= 0 {
		parsed.MiddlemanTimeoutMS = 100
	}
	if parsed.MiddlemanMaxBiddersPerImp <= 0 {
		parsed.MiddlemanMaxBiddersPerImp = 5
	}
	if parsed.MiddlemanExchangeDomain == "" {
		parsed.MiddlemanExchangeDomain = serverURLHost(parsed.ServerURL)
	}
	if parsed.MiddlemanCallbackTTLSeconds <= 0 {
		parsed.MiddlemanCallbackTTLSeconds = 86400
	}
	if parsed.MiddlemanCallbackTimeoutMS <= 0 {
		parsed.MiddlemanCallbackTimeoutMS = 1000
	}
	if parsed.MiddlemanCallbackBaseURL == "" {
		parsed.MiddlemanCallbackBaseURL = parsed.ServerURL
	}

	if err := parsed.Validate(); err != nil {
		return nil, err
	}
	return parsed, nil
}

func (self *Config) Validate(modes ...ConfigMode) error {
	if self == nil {
		return fmt.Errorf("config is nil")
	}
	if len(self.ConnectArray) < 2 {
		return fmt.Errorf("connect_array must contain driver and DSN")
	}
	if strings.TrimSpace(self.ConnectArray[0]) == "" {
		return fmt.Errorf("connect_array driver is empty")
	}
	if strings.TrimSpace(self.ConnectArray[1]) == "" {
		return fmt.Errorf("connect_array DSN is empty")
	}
	if self.Redis == nil {
		return fmt.Errorf("redis config is missing")
	}
	if self.Redis.Network == "" {
		return fmt.Errorf("redis network is empty")
	}
	if self.Redis.Network != "tcp" && self.Redis.Network != "unix" {
		return fmt.Errorf("redis network %q is invalid", self.Redis.Network)
	}
	if self.Redis.Addr == "" {
		return fmt.Errorf("redis addr is empty")
	}
	if self.Redis.Network == "tcp" {
		if _, _, err := net.SplitHostPort(self.Redis.Addr); err != nil {
			return fmt.Errorf("redis addr %q must be host:port: %w", self.Redis.Addr, err)
		}
	}
	if self.TrackingSignatureTTLSeconds <= 0 {
		return fmt.Errorf("tracking_signature_ttl_seconds must be positive")
	}
	if self.MiddlemanCallbackTTLSeconds <= 0 {
		return fmt.Errorf("middleman_callback_ttl_seconds must be positive")
	}
	if self.MiddlemanCallbackTimeoutMS <= 0 {
		return fmt.Errorf("middleman_callback_timeout_ms must be positive")
	}

	needNATS := false
	needTracking := false
	needSpread := false
	for _, mode := range modes {
		switch mode {
		case ConfigModeBid:
			needNATS = true
			needTracking = true
		case ConfigModeSpread, ConfigModeNATS:
			needNATS = true
		case ConfigModeCache, ConfigModeLedger, ConfigModeRetry, ConfigModeMaxMind, ConfigModeDatabase, ConfigModeRedis:
		default:
			return fmt.Errorf("unknown config validation mode %q", mode)
		}
		if mode == ConfigModeSpread {
			needSpread = true
		}
	}
	if needTracking && strings.TrimSpace(self.TrackingSecret) == "" {
		return fmt.Errorf("tracking_secret is required")
	}
	if needNATS {
		u, err := url.Parse(self.NatsURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("nats_url is invalid")
		}
	}
	if needSpread && strings.TrimSpace(self.Spread) == "" {
		return fmt.Errorf("spread path is required")
	}
	if self.MiddlemanEnabled {
		base := self.MiddlemanCallbackBaseURL
		if base == "" {
			base = self.ServerURL
		}
		u, err := url.Parse(base)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("middleman_callback_base_url is invalid")
		}
		if strings.TrimSpace(self.TrackingSecret) == "" {
			return fmt.Errorf("tracking_secret is required when middleman is enabled")
		}
	}
	return nil
}

func serverURLHost(raw string) string {
	if raw == "" {
		return ""
	}
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		parts := strings.SplitN(raw, "://", 2)
		raw = parts[1]
	}
	if i := strings.IndexByte(raw, '/'); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.IndexByte(raw, ':'); i >= 0 {
		raw = raw[:i]
	}
	return raw
}

// GetRedisDB returns the Redis conn and database handler
func (self *Config) GetRedisDB(ctx context.Context, modes ...ConfigMode) (radix.Client, *sql.DB, error) {
	if err := self.Validate(ConfigModeDatabase, ConfigModeRedis); err != nil {
		return nil, nil, err
	}
	db, err := self.OpenDB(ctx, modes...)
	if err != nil {
		return nil, nil, err
	}
	red := self.Redis
	cfg := radix.PoolConfig{
		Dialer: radix.Dialer{
			AuthUser: red.User,
			AuthPass: red.Pass,
		},
	}
	if red.Size != 0 {
		cfg.Size = red.Size
	}
	redis, err := cfg.New(ctx, red.Network, red.Addr)
	if err != nil {
		return nil, nil, err
	}
	return redis, db, nil
}

func (self *Config) OpenDB(ctx context.Context, modes ...ConfigMode) (*sql.DB, error) {
	if err := self.Validate(ConfigModeDatabase); err != nil {
		return nil, err
	}
	db, err := sql.Open(self.ConnectArray[0], self.ConnectArray[1])
	if err != nil {
		return nil, err
	}
	self.ApplyDBPool(db, modes...)
	if ctx != nil {
		if err := db.PingContext(ctx); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (self *Config) ApplyDBPool(db *sql.DB, modes ...ConfigMode) {
	if self == nil || db == nil {
		return
	}
	defaults := dbPoolDefaults(modes...)
	maxOpen := choosePositive(self.DBMaxOpenConns, defaults.maxOpen)
	maxIdle := choosePositive(self.DBMaxIdleConns, defaults.maxIdle)
	if maxIdle > maxOpen && maxOpen > 0 {
		maxIdle = maxOpen
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Duration(choosePositive(self.DBConnMaxLifetimeSeconds, defaults.maxLifetimeSeconds)) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(choosePositive(self.DBConnMaxIdleTimeSeconds, defaults.maxIdleTimeSeconds)) * time.Second)
}

type dbPoolProfile struct {
	maxOpen            int
	maxIdle            int
	maxLifetimeSeconds int
	maxIdleTimeSeconds int
}

func dbPoolDefaults(modes ...ConfigMode) dbPoolProfile {
	for _, mode := range modes {
		switch mode {
		case ConfigModeLedger, ConfigModeRetry, ConfigModeCache, ConfigModeMaxMind:
			return dbPoolProfile{maxOpen: 4, maxIdle: 2, maxLifetimeSeconds: 600, maxIdleTimeSeconds: 120}
		}
	}
	return dbPoolProfile{maxOpen: 32, maxIdle: 8, maxLifetimeSeconds: 1800, maxIdleTimeSeconds: 300}
}

func choosePositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
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

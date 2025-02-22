// Package genelet is a genelet package for genelet framework.
package genelet

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type PatternCase int

const (
	REROUTE PatternCase = iota
	CACHE
	STATIC
)

type Pattern struct {
	Reg      string
	Regs     *regexp.Regexp
	Keys     []string
	Initials string
	Expire   int
	Case     PatternCase
}

type Issuer struct {
	Default       bool
	Screen        int8
	Sql           string
	Sql_as        string
	Provider_pars map[string]string
	Credential    []string
	In_pars       []string
	Out_pars      []string
	Condition_uri [][]string
}

type Role struct {
	Id_name    string
	Id_cipher  bool
	Type_id    int
	Is_admin   bool
	Attributes []string

	Coding    string
	Secret    string
	Surface   string
	Length    int8
	Duration  int
	Userlist  []string
	Grouplist []string
	Logout    string
	Domain    string
	Path      string
	Max_age   int

	Issuers map[string]Issuer
}

type Config struct {
	Upload_dir      string
	Template        string
	Pubrole         string
	Secret          string
	ServerURL       string
	Upload_url      string
	ServerPort      string
	DocumentRoot    string
	DocumentRoots   map[string]string
	ProjectRoot     string
	Script          string
	Component_name  string
	Action_name     string
	Default_actions map[string]string
	Role_name       string
	Oauth2s         []string
	Oauth1s         []string
	Login_name      string
	Logout_name     string
	Tag_name        string
	Provider_name   string
	Callback_name   string
	Go_stamp_name   string
	Go_md5_name     string
	Go_uri_name     string
	Go_probe_name   string
	Go_err_name     string

	ConnectArray []string
	Blks         map[string]map[string]string
	Chartags     map[string]Chartag
	Roles        map[string]Role
	Errors       map[string]string
	Custom       map[string]string
	Patterns     []Pattern
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

	if parsed.ServerURL == "" {
		parsed.ServerURL = "http://localhost"
	}
	if parsed.ServerPort == "" {
		parsed.ServerPort = "80"
	}
	if parsed.Upload_dir == "" {
		parsed.Upload_dir = "/tmp"
	}
	if parsed.Upload_url == "" {
		parsed.Upload_url = parsed.ServerURL + "/uploads"
	}
	if parsed.Component_name == "" {
		parsed.Component_name = "component"
	}
	if parsed.Action_name == "" {
		parsed.Action_name = "action"
	}
	if parsed.Go_stamp_name == "" {
		parsed.Go_stamp_name = "go_stamp"
	}
	if parsed.Go_md5_name == "" {
		parsed.Go_md5_name = "go_md5"
	}
	if parsed.Go_uri_name == "" {
		parsed.Go_uri_name = "go_uri"
	}
	if parsed.Role_name == "" {
		parsed.Role_name = "role"
	}
	if parsed.Oauth2s == nil {
		parsed.Oauth2s = []string{"google", "facebook", "microsoft", "qq", "sina"}
	}
	if parsed.Oauth1s == nil {
		parsed.Oauth1s = []string{"twitter", "linkedin"}
	}
	if parsed.Login_name == "" {
		parsed.Login_name = "login"
	}
	if parsed.Logout_name == "" {
		parsed.Logout_name = "logout"
	}
	if parsed.Tag_name == "" {
		parsed.Tag_name = "tag"
	}
	if parsed.Provider_name == "" {
		parsed.Provider_name = "provider"
	}
	if parsed.Callback_name == "" {
		parsed.Callback_name = "callback"
	}
	if parsed.Go_probe_name == "" {
		parsed.Go_probe_name = "go_probe"
	}
	if parsed.Go_err_name == "" {
		parsed.Go_err_name = "go_err"
	}
	if parsed.Errors == nil {
		parsed.Errors = make(map[string]string)
	}

	if parsed.Default_actions == nil {
		parsed.Default_actions = map[string]string{"GET": "dashboard", "GET_item": "edit", "PUT": "update", "POST": "insert", "DELETE": "delete"}
	}

	//for _, pattern := range parsed.Patterns {
	//pattern.Regs = regexp.MustCompile(pattern.Reg)
	//}

	return parsed, nil
}

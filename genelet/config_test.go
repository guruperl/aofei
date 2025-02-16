package genelet

import (
	"testing"
)

const (
	filename = "test.conf"
)

func TestConfig(t *testing.T) {
	c := NewConfig(filename)
	if c.DocumentRoot != "aa" {
		t.Errorf("%s wanted", "aa")
	}
	c.DocumentRoot = "root"
	if c.DocumentRoot != "root" {
		t.Errorf("%s wanted", "root")
	}
	if c.Script != "bb" {
		t.Errorf("%s wanted", "bb")
	}
	if c.Pubrole != "cc" {
		t.Errorf("%s wanted", "cc")
	}
	if c.Secret != "" {
		t.Errorf("%s is empty", "secret")
	}
	c.Secret = "dd"
	if c.Secret != "dd" {
		t.Errorf("%s wanted", "dd")
	}
	if c.Template != "ee" {
		t.Errorf("%s wanted", "ee")
	}
	if c.Action_name != "action" {
		t.Errorf("%s wanted", "action")
	}
	if c.Go_uri_name != "go_uri" {
		t.Errorf("%s wanted", "go_uri")
	}
	if c.Db[0] != "mysql" {
		t.Errorf("%s wanted", c.Db[0])
	}
	if c.Db[1] != "eightran:12pass34@tcp(vm0:3306)/gotest" {
		t.Errorf("%s wanted", c.Db[1])
	}
	if c.Db[2] != "ccc" {
		t.Errorf("%s wanted", "ccc")
	}
	/*
	   	if c.Blck["_gmail"]["Address"] != "email-smtp.us-west-2.amazonaws.com:465" {
	           t.Errorf("%s wanted", "email-smtp.us-west-2.amazonaws.com:465")
	       }
	   	if c.Blck["_gmail"]["From"] != "peter@greetingland.com" {
	   		t.Errorf("%s wanted", "peter@greetingland.com")
	   	}
	*/

	char := c.Chartags["json"]
	if char.Content_type != "application/json; charset=\"UTF-8\"" {
		t.Errorf("%s wanted", "application/json; charset=\"UTF-8\"")
	}
	if char.Challenge != "challenge" {
		t.Errorf("%s wanted", "challenge")
	}
}

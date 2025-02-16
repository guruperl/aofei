package genelet;

import (
    "testing"
)

const (
	full = "full"
	short = "short"
	c = 1
	challenge = "challenge"
	logged = "logged"
	logout = "logout"
	failed = "failed"
)

func TestChartag(t *testing.T) {
	char := Chartag{Content_type: full, Short: short, Case: c, Challenge: challenge, Logged: logged, Logout: logout, Failed: failed}
	if char.Content_type != full {
		t.Errorf("%s wanted", full)
	}
	if char.Case != c {
		t.Errorf("%d wanted", c)
	}
	if char.Challenge != challenge {
		t.Errorf("%s wanted", challenge)
	}
	if char.Call_challenge() != "{\"data\":\"challenge\"}" {
		t.Errorf("%s wanted", char.Call_challenge())
	}
	char.Case=2
	if char.Call_challenge() != "<?xml version=\"1.0\" encoding=\"UTF-8\"?><data>challenge</data>" {
		t.Errorf("%s wanted", char.Call_challenge())
	}
}

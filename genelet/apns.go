package genelet

import (
)

type Apns struct {
	Badge	int8
	Sound	string
	Device_token	string
	Cert	string
	Key	string
	Passphrase	string
}

func (self *Apns) Send(body string) error {
	return nil
}

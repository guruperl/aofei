package genelet

type Chartag struct {
	Content_type	string
	Short	string
	Case	int8
	Challenge	string
	Logged  string
	Logout  string
	Failed  string
}

func (self Chartag)Call_challenge() string { return self.charcase_string(self.Challenge); }
func (self Chartag)Call_logged() string { return self.charcase_string(self.Logged); }
func (self Chartag)Call_logout() string { return self.charcase_string(self.Logout); }
func (self Chartag)Call_failed() string { return self.charcase_string(self.Failed); }

func (self Chartag)charcase_string(in string) string {
	if self.Case==2 {
		return `<?xml version="1.0" encoding="UTF-8"?><data>`+in+`</data>`
	} else if self.Case==1 {
		return `{"data":"`+in+`"}`
	}
	return ""
}

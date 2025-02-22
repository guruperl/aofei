package genelet

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

type TProcedure struct {
	Procedure
}

func New_TProcedure(base Base, db *sql.DB, uri string, provider string) *TProcedure {
	a := new(TProcedure)
	a.CGI = a
	a.Base = base
	a.Db = db
	a.Uri = uri
	a.Provider = provider
	return a
}
func (self *TProcedure) Set_ip() string {
	return "123.123.123.123"
}
func (self *TProcedure) Set_when() int {
	return 1
}

func TestDbiProcedure(t *testing.T) {
	configure, err := NewConfig(filename)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("mysql", "eightran:12pass34@tcp(vm0:3306)/gotest")
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", "http://example.com/foo?email=hello&passwd=world", nil)
	if err != nil {
		log.Fatal(err)
	}
	w := httptest.NewRecorder()
	b := new_Base(configure, "m", "json", w, req)

	tticket := New_TProcedure(*b, db, "http://xxx.yyy.zzz/foo/bar", "db")
	//issuer := tticket.C.Roles["m"].Issuers["db"]
	db.Exec("DROP TABLE IF EXISTS user")
	db.Exec("CREATE TABLE `user` (   `m_id` int(11) NOT NULL AUTO_INCREMENT,   `email` varchar(255) DEFAULT NULL,   `passwd` varchar(255) DEFAULT NULL,   `first_name` varchar(255) DEFAULT NULL,   `last_name` varchar(255) DEFAULT NULL,   `address` varchar(255) DEFAULT NULL,   `company` varchar(255) DEFAULT NULL,   PRIMARY KEY (`m_id`) )")
	db.Exec("INSERT INTO user (email, passwd, first_name, last_name, address, company) VALUES ('a','b','c','d','e','f')")
	tticket.Uri = "asw"
	ret := tticket.Authenticate("a", "b")
	if ret != nil {
		t.Errorf("%s corrent login expected", ret.Error())
	}
	if tticket.Out_hash["m_id"].(int64) != 1 {
		t.Errorf("%d wanted", tticket.Out_hash["m_id"].(int64))
	}
	if string(tticket.Out_hash["first_name"].(string)) != "c" {
		t.Errorf("%s wanted", tticket.Out_hash["first_name"].(string))
	}
	ret = tticket.Handler_fields()
	if ret != nil {
		t.Errorf("%s returned", ret.Error())
	}

	// test Handler with direct login
	req, _ = http.NewRequest("GET", "http://example.com/foo?email=a&passwd=b&direct=1", nil)
	w = httptest.NewRecorder()
	b = new_Base(configure, "m", "e", w, req)
	tticket = New_TProcedure(*b, db, "http://xxx.yyy.zzz/foo/bar", "db")
	_ = req.ParseForm()
	ret = tticket.Handler_login()
	if ret.(Gerror).Code != 303 {
		t.Errorf("%s wanted", ret.Error())
	}
	h := w.Header()
	matched, err := regexp.MatchString("^mc=Ec9rwEEzh1\\/0UTuoE7dvi\\/ByyTC1F2qsXRbY4yYNbq3RoCpNDcLkiEYmJdaLrz8uBYUQZe\\/dglVvVtXQsRI\\/", h.Get("Set-Cookie"))
	if !matched {
		// t.Errorf("%s wanted", h.Get("Set-Cookie"))
	}
	db.Close()
}

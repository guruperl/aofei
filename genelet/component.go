package genelet

import (
    "encoding/json"
	"io/ioutil"
)

type Component struct {
    Actions     map[string]map[string][]string
    Fks         map[string][]string

	Nextpages   map[string][]map[string]interface{}

	Current_table	string
	Current_tables  []Table
	Current_key		string
	Current_keys	[]string
	Current_id_auto	string
	Key_in			map[string]string

    Insert_pars     []string
    Edit_pars       []string
    Update_pars     []string
    Insupd_pars     []string
    Topics_pars     []string
    Topics_hash		map[string]string

    Total_force     int
	Empties			string
	Fields			string
	Maxpageno		string
	Totalno			string
	Rowcount		string
	Pageno			string
	Sortreverse		string
	Sortby			string
}

func NewComponent(filename string) *Component {
    var parsed *Component
    content, err := ioutil.ReadFile(filename)
    if err != nil {
        panic(err)
    }
    err = json.Unmarshal(content, &parsed)
    if err != nil {
        panic(err)
    }

	if parsed.Sortby == "" {
		parsed.Sortby = "sortby"
	}
	if parsed.Sortreverse == "" {
		parsed.Sortreverse = "sortreverse"
	}
	if parsed.Pageno == "" {
		parsed.Pageno = "pageno"
	}
	if parsed.Totalno == "" {
		parsed.Totalno = "totalno"
	}
	if parsed.Rowcount == "" {
		parsed.Rowcount = "rowcount"
	}
	if parsed.Maxpageno == "" {
		parsed.Maxpageno = "maxpage"
	}
	if parsed.Fields == "" {
		parsed.Fields = "fields"
	}
	if parsed.Empties == "" {
		parsed.Empties = "empties"
	}
	if parsed.Total_force == 0 {
		parsed.Total_force = 1
	}

	return parsed
}

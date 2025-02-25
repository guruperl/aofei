package genelet

import (
	"database/sql"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type Table struct {
	Name  string
	Alias string
	Type  string
	Using string
	On    string
}

type Crud struct {
	DBI
	CurrentTable  string  `json:"current_table"`
	CurrentTables []Table `json:"current_tables"`
}

func NewCrud(db *sql.DB, table string, tables []Table) *Crud {
	crud := new(Crud)
	crud.DB = db
	crud.CurrentTable = table
	if tables != nil {
		crud.CurrentTables = tables
	}
	return crud
}

func TableString(currentTables []Table) string {
	sql := ""
	for i, table := range currentTables {
		name := table.Name
		if table.Alias != "" {
			name += " " + table.Alias
		}
		if i == 0 {
			sql = name
		} else if table.Using != "" {
			sql += "\n" + table.Type + " JOIN " + name + " USING (" + table.Using + ")"
		} else {
			sql += "\n" + table.Type + " JOIN " + name + " ON (" + table.On + ")"
		}
	}

	return sql
}

func SelectLabelString(selectPars interface{}) (string, []string) {
	select_labels := make([]string, 0)
	sql := ""
	switch selectPars.(type) {
	case []string:
		for _, v := range selectPars.([]string) {
			select_labels = append(select_labels, v)
		}
		sql = strings.Join(select_labels, ", ")
	case map[string]string:
		i := 0
		for key, val := range selectPars.(map[string]string) {
			if i == 0 {
				sql = key
			} else {
				sql += ", " + key
			}
			i++
			select_labels = append(select_labels, val)
		}
	default:
		sql = selectPars.(string)
		select_labels = append(select_labels, sql)
	}
	return sql, select_labels
}

func SelectConditionString(extra url.Values, table ...string) (string, []interface{}) {
	sql := ""
	values := make([]interface{}, 0)
	i := 0
	for field, value := range extra {
		if i > 0 {
			sql += " AND "
		}
		sql += "("

		if table != nil && table[0] != "" {
			match, err := regexp.MatchString("\\.", field)
			if err == nil && !match {
				field = table[0] + "." + field
			}
		}
		n := len(value)
		if n > 1 {
			sql += field + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + ")"
			for _, v := range value {
				values = append(values, v)
			}
		} else if n == 1 {
			if field[(len(field)-5):] == "_gsql" {
				sql += value[0]
			} else {
				sql += field + " =?"
				values = append(values, value[0])
			}
		}
		sql += ")"
		i++
	}

	return sql, values
}

func SingleConditionString(keyname interface{}, ids []interface{}, extra ...url.Values) (string, []interface{}) {
	sql := ""
	extraValues := make([]interface{}, 0)

	switch keyname.(type) {
	case []string:
		for i, item := range keyname.([]string) {
			val := ids[i]
			if i == 0 {
				sql = "("
			} else {
				sql += " AND "
			}
			switch val.(type) {
			case []interface{}:
				n := len(val.([]interface{}))
				sql += item + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + ")"
				for _, v := range val.([]interface{}) {
					extraValues = append(extraValues, v)
				}
			default:
				sql += item + " =?"
				extraValues = append(extraValues, val)
			}
		}
		sql += ")"
	case string:
		n := len(ids)
		if n > 1 {
			sql = "(" + keyname.(string) + " IN (" + strings.Join(strings.Split(strings.Repeat("?", n), ""), ",") + "))"
		} else {
			sql = "(" + keyname.(string) + "=?)"
		}
		for _, v := range ids {
			extraValues = append(extraValues, v)
		}
	}

	if extra != nil && len(extra) > 0 {
		s, arr := SelectConditionString(extra[0])
		if s != "" {
			sql += " AND " + s
			for _, v := range arr {
				extraValues = append(extraValues, v)
			}
		}
	}

	return sql, extraValues
}

func (self *Crud) InsertHash(fieldValues url.Values) error {
	return self.insertHash_("INSERT", fieldValues)
}

func (self *Crud) ReplaceHash(fieldValues url.Values) error {
	return self.insertHash_("REPLACE", fieldValues)
}

func (self *Crud) insertHash_(how string, fieldValues url.Values) error {
	fields := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range fieldValues {
		fields = append(fields, k)
		values = append(values, v[0])
	}
	sql := how + " INTO " + self.CurrentTable + " (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(strings.Split(strings.Repeat("?", len(fields)), ""), ",") + ")"
	return self.DoSQL(sql, values...)
}

func (self *Crud) UpdateHash(fieldValues url.Values, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	empties := make([]string, 0)
	return self.UpdateHashNulls(fieldValues, keyname, ids, empties, extra...)
}

func (self *Crud) UpdateHashNulls(fieldValues url.Values, keyname interface{}, ids []interface{}, empties []string, extra ...url.Values) error {
	fields := make([]string, 0)
	field0 := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range fieldValues {
		fields = append(fields, k)
		field0 = append(field0, k+"=?")
		values = append(values, v[0])
	}

	sql := "UPDATE " + self.CurrentTable + " SET " + strings.Join(field0, ", ")
	for _, v := range empties {
		if fieldValues.Get(v) != "" {
			continue
		}
		switch keyname.(type) {
		case []string:
			if Grep(keyname.([]string), v) {
				continue
			}
		case string:
			if v == keyname.(string) {
				continue
			}
		}
		sql += ", " + v + "=NULL"
	}

	where, extraValues := SingleConditionString(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}
	for _, v := range extraValues {
		values = append(values, v)
	}

	return self.DoSQL(sql, values...)
}

func (self *Crud) InsupdTable(fieldValues url.Values, keyname string, uniques []string, s_hash *string) error {
	s := "SELECT " + keyname + " FROM " + self.CurrentTable + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, fieldValues.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQL(&lists, s, v...)
	if err != nil {
		return err
	}
	if len(lists) > 1 {
		return Err(1070)
	}

	if len(lists) == 1 {
		id := lists[0][keyname].(int64)
		err = self.UpdateHash(fieldValues, keyname, []interface{}{id}, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
		fieldValues.Set(keyname, strconv.FormatInt(id, 10))
	} else {
		err = self.InsertHash(fieldValues)
		if err != nil {
			return err
		}
		*s_hash = "insert"
		fieldValues.Set(keyname, strconv.FormatInt(self.LastID, 10))
	}

	return nil
}

func (self *Crud) InsupdHash(fieldValues url.Values, upd_fieldValues url.Values, keyname interface{}, uniques []string, s_hash *string) error {
	var ks []string
	switch keyname.(type) {
	case []string:
		ks = keyname.([]string)
	default:
		ks = []string{keyname.(string)}
	}
	s := "SELECT " + strings.Join(ks, ",") + " FROM " + self.CurrentTable + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, fieldValues.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.SelectSQL(&lists, s, v...)
	if err != nil {
		return err
	}
	if len(lists) > 1 {
		return Err(1070)
	}

	if len(lists) == 1 {
		ids := make([]interface{}, len(ks))
		for i, k := range ks {
			ids[i] = lists[0][k]
			fieldValues.Set(k, Interface2String(ids[i]))
		}
		err = self.UpdateHash(upd_fieldValues, keyname, ids, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
	} else {
		err = self.InsertHash(fieldValues)
		if err != nil {
			return err
		}
		*s_hash = "insert"
	}

	return nil
}

func (self *Crud) DeleteHash(keyname interface{}, ids []interface{}, extra ...url.Values) error {
	sql := "DELETE FROM " + self.CurrentTable
	where, extraValues := SingleConditionString(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.DoSQL(sql, extraValues...)
}

func (self *Crud) EditHash(lists *[]map[string]interface{}, selectPars interface{}, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	sql, select_labels := SelectLabelString(selectPars)
	sql = "SELECT " + sql + "\nFROM " + self.CurrentTable
	where, extraValues := SingleConditionString(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.SelectSQLLabel(lists, sql, select_labels, extraValues...)
}

func (self *Crud) TopicsHash(lists *[]map[string]interface{}, selectPars interface{}, extra ...url.Values) error {
	return self.TopicsHashOrder(lists, selectPars, "", extra...)
}

func (self *Crud) TopicsHashOrder(lists *[]map[string]interface{}, selectPars interface{}, order string, extra ...url.Values) error {
	sql, select_labels := SelectLabelString(selectPars)
	table := ""
	if len(self.CurrentTables) > 0 {
		sql = "SELECT " + sql + "\nFROM " + TableString(self.CurrentTables)
		table = self.CurrentTables[0].Alias
		if table == "" {
			table = self.CurrentTables[0].Name
		}
	} else {
		sql = "SELECT " + sql + "\nFROM " + self.CurrentTable
	}

	if len(extra) > 0 {
		where, values := SelectConditionString(extra[0], table)
		if where != "" {
			sql += "\nWHERE " + where
		}
		if order != "" {
			sql += "\n" + order
		}
		return self.SelectSQLLabel(lists, sql, select_labels, values...)
	}

	if order != "" {
		sql += "\n" + order
	}
	return self.SelectSQLLabel(lists, sql, select_labels)
}

func (self *Crud) TotalHash(hash map[string]interface{}, label string, extra ...url.Values) error {
	table := ""
	sql := "SELECT COUNT(*) FROM "
	if self.CurrentTables != nil {
		sql += TableString(self.CurrentTables)
		table = self.CurrentTables[0].Alias
		if table == "" {
			table = self.CurrentTables[0].Name
		}
	} else {
		sql += self.CurrentTable
	}

	if len(extra) > 0 {
		where, values := SelectConditionString(extra[0], table)
		if where != "" {
			sql += "\nWHERE " + where
		}
		return self.GetSQLLabel(hash, sql, []string{label}, values...)
	}

	return self.GetSQLLabel(hash, sql, []string{label})
}

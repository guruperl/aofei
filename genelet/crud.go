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
	Current_table  string
	Current_tables []Table
}

func New_Crud(db *sql.DB, table string, tables []Table) *Crud {
	crud := new(Crud)
	crud.Db = db
	crud.Current_table = table
	if tables != nil {
		crud.Current_tables = tables
	}
	return crud
}

func Table_string(current_tables []Table) string {
	sql := ""
	for i, table := range current_tables {
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

func Select_label_string(select_pars interface{}) (string, []string) {
	select_labels := make([]string, 0)
	sql := ""
	switch select_pars.(type) {
	case []string:
		for _, v := range select_pars.([]string) {
			select_labels = append(select_labels, v)
		}
		sql = strings.Join(select_labels, ", ")
	case map[string]string:
		i := 0
		for key, val := range select_pars.(map[string]string) {
			if i == 0 {
				sql = key
			} else {
				sql += ", " + key
			}
			i++
			select_labels = append(select_labels, val)
		}
	default:
		sql = select_pars.(string)
		select_labels = append(select_labels, sql)
	}
	return sql, select_labels
}

func Select_condition_string(extra url.Values, table ...string) (string, []interface{}) {
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

func Single_condition_string(keyname interface{}, ids []interface{}, extra ...url.Values) (string, []interface{}) {
	sql := ""
	extra_values := make([]interface{}, 0)

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
					extra_values = append(extra_values, v)
				}
			default:
				sql += item + " =?"
				extra_values = append(extra_values, val)
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
			extra_values = append(extra_values, v)
		}
	}

	if extra != nil && len(extra) > 0 {
		s, arr := Select_condition_string(extra[0])
		if s != "" {
			sql += " AND " + s
			for _, v := range arr {
				extra_values = append(extra_values, v)
			}
		}
	}

	return sql, extra_values
}

func (self *Crud) Insert_hash(field_values url.Values) error {
	return self.insert_hash_("INSERT", field_values)
}

func (self *Crud) Replace_hash(field_values url.Values) error {
	return self.insert_hash_("REPLACE", field_values)
}

func (self *Crud) insert_hash_(how string, field_values url.Values) error {
	fields := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range field_values {
		fields = append(fields, k)
		values = append(values, v[0])
	}
	sql := how + " INTO " + self.Current_table + " (" + strings.Join(fields, ", ") + ") VALUES (" + strings.Join(strings.Split(strings.Repeat("?", len(fields)), ""), ",") + ")"
	return self.Do_sql(sql, values...)
}

func (self *Crud) Update_hash(field_values url.Values, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	empties := make([]string, 0)
	return self.Update_hash_nulls(field_values, keyname, ids, empties, extra...)
}

func (self *Crud) Update_hash_nulls(field_values url.Values, keyname interface{}, ids []interface{}, empties []string, extra ...url.Values) error {
	fields := make([]string, 0)
	field0 := make([]string, 0)
	values := make([]interface{}, 0)
	for k, v := range field_values {
		fields = append(fields, k)
		field0 = append(field0, k+"=?")
		values = append(values, v[0])
	}

	sql := "UPDATE " + self.Current_table + " SET " + strings.Join(field0, ", ")
	for _, v := range empties {
		if field_values.Get(v) != "" {
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

	where, extra_values := Single_condition_string(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}
	for _, v := range extra_values {
		values = append(values, v)
	}

	return self.Do_sql(sql, values...)
}

func (self *Crud) Insupd_table(field_values url.Values, keyname string, uniques []string, s_hash *string) error {
	s := "SELECT " + keyname + " FROM " + self.Current_table + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, field_values.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.Select_sql(&lists, s, v...)
	if err != nil {
		return err
	}
	if len(lists) > 1 {
		return Err(1070)
	}

	if len(lists) == 1 {
		id := lists[0][keyname].(int64)
		err = self.Update_hash(field_values, keyname, []interface{}{id}, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
		field_values.Set(keyname, strconv.FormatInt(id, 10))
	} else {
		err = self.Insert_hash(field_values)
		if err != nil {
			return err
		}
		*s_hash = "insert"
		field_values.Set(keyname, strconv.FormatInt(self.Last_id, 10))
	}

	return nil
}

func (self *Crud) Insupd_hash(field_values url.Values, upd_field_values url.Values, keyname interface{}, uniques []string, s_hash *string) error {
	var ks []string
	switch keyname.(type) {
	case []string:
		ks = keyname.([]string)
	default:
		ks = []string{keyname.(string)}
	}
	s := "SELECT " + strings.Join(ks, ",") + " FROM " + self.Current_table + "\nWHERE "
	v := make([]interface{}, 0)
	for i, val := range uniques {
		if i > 0 {
			s += " AND "
		}
		s += val + "=?"
		v = append(v, field_values.Get(val))
	}

	lists := make([]map[string]interface{}, 0)
	err := self.Select_sql(&lists, s, v...)
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
			field_values.Set(k, Interface2String(ids[i]))
		}
		err = self.Update_hash(upd_field_values, keyname, ids, nil)
		if err != nil {
			return err
		}
		*s_hash = "update"
	} else {
		err = self.Insert_hash(field_values)
		if err != nil {
			return err
		}
		*s_hash = "insert"
	}

	return nil
}

func (self *Crud) Delete_hash(keyname interface{}, ids []interface{}, extra ...url.Values) error {
	sql := "DELETE FROM " + self.Current_table
	where, extra_values := Single_condition_string(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.Do_sql(sql, extra_values...)
}

func (self *Crud) Edit_hash(lists *[]map[string]interface{}, select_pars interface{}, keyname interface{}, ids []interface{}, extra ...url.Values) error {
	sql, select_labels := Select_label_string(select_pars)
	sql = "SELECT " + sql + "\nFROM " + self.Current_table
	where, extra_values := Single_condition_string(keyname, ids, extra...)
	if where != "" {
		sql += "\nWHERE " + where
	}

	return self.Select_sql_label(lists, sql, select_labels, extra_values...)
}

func (self *Crud) Topics_hash(lists *[]map[string]interface{}, select_pars interface{}, extra ...url.Values) error {
	return self.Topics_hash_order(lists, select_pars, "", extra...)
}

func (self *Crud) Topics_hash_order(lists *[]map[string]interface{}, select_pars interface{}, order string, extra ...url.Values) error {
	sql, select_labels := Select_label_string(select_pars)
	table := ""
	if len(self.Current_tables) > 0 {
		sql = "SELECT " + sql + "\nFROM " + Table_string(self.Current_tables)
		table = self.Current_tables[0].Alias
		if table == "" {
			table = self.Current_tables[0].Name
		}
	} else {
		sql = "SELECT " + sql + "\nFROM " + self.Current_table
	}

	if extra != nil && len(extra) > 0 {
		where, values := Select_condition_string(extra[0], table)
		if where != "" {
			sql += "\nWHERE " + where
		}
		if order != "" {
			sql += "\n" + order
		}
		return self.Select_sql_label(lists, sql, select_labels, values...)
	}

	if order != "" {
		sql += "\n" + order
	}
	return self.Select_sql_label(lists, sql, select_labels)
}

func (self *Crud) Total_hash(hash map[string]interface{}, label string, extra ...url.Values) error {
	table := ""
	sql := "SELECT COUNT(*) FROM "
	if self.Current_tables != nil {
		sql += Table_string(self.Current_tables)
		table = self.Current_tables[0].Alias
		if table == "" {
			table = self.Current_tables[0].Name
		}
	} else {
		sql += self.Current_table
	}

	if extra != nil && len(extra) > 0 {
		where, values := Select_condition_string(extra[0], table)
		if where != "" {
			sql += "\nWHERE " + where
		}
		return self.Get_sql_label(hash, sql, []string{label}, values...)
	}

	return self.Get_sql_label(hash, sql, []string{label})
}

package genelet

import (
	"database/sql"
	"net/url"
	"strings"

	"github.com/golang/glog"
)

type DBI struct {
	Db       *sql.DB
	Last_id  int64
	Affected int64
}

func (self *DBI) Exec_sql(sql string) error {
	_, err := self.Db.Exec(sql)
	if err != nil {
		return err
	}
	return nil
}

func (self *DBI) Do_sql(sql string, args ...interface{}) error {
	glog.Infof("%s\n", sql)
	glog.Infof("%#v\n", args)
	sth, err := self.Db.Prepare(sql)
	//defer sth.Close()
	if err != nil {
		return err
	}
	result, err := sth.Exec(args...)
	if err != nil {
		return err
	}
	last_id, err := result.LastInsertId()
	if err == nil {
		self.Last_id = last_id
	}
	affected, err := result.RowsAffected()
	if err == nil {
		self.Affected = affected
	}
	sth.Close()
	return nil
}

func (self *DBI) Do_sqls(sql string, args [][]interface{}) error {
	glog.Infof("%s\n", sql)
	glog.Infof("%#v\n", args)
	sth, err := self.Db.Prepare(sql)
	defer sth.Close()
	if err != nil {
		return err
	}
	for _, v := range args {
		result, err := sth.Exec(v...)
		if err != nil {
			return err
		}
		last_id, err := result.LastInsertId()
		if err == nil {
			self.Last_id = last_id
		}
		affected, err := result.RowsAffected()
		if err == nil {
			self.Affected = affected
		}
	}
	return nil
}

func (self *DBI) Get_sql(res map[string]interface{}, sql string, args ...interface{}) error {
	return self.Get_sql_label(res, sql, nil, args...)
}
func (self *DBI) Get_args(ARGS url.Values, sql string, args ...interface{}) error {
	res := make(map[string]interface{})
	if err := self.Get_sql(res, sql, args...); err != nil {
		return err
	}
	for k, v := range res {
		if v == nil {
			continue
		}
		ARGS.Set(k, Interface2String(v))
	}
	return nil
}

func (self *DBI) Select_sql(lists *[]map[string]interface{}, sql string, args ...interface{}) error {
	return self.Select_sql_label(lists, sql, nil, args...)
}

func (self *DBI) Get_sql_label(res map[string]interface{}, sql string, select_labels []string, args ...interface{}) error {
	lists := make([]map[string]interface{}, 0)
	err := self.Select_sql_label(&lists, sql, select_labels, args...)
	if err != nil {
		return err
	}
	if len(lists) == 1 {
		for k, v := range lists[0] {
			res[k] = v
		}
	}
	return nil
}

func (self *DBI) Select_sql_label(lists *[]map[string]interface{}, sql string, select_labels []string, args ...interface{}) error {
	glog.Infof("%s\n", sql)
	glog.Infof("%v\n", args)
	sth, err := self.Db.Prepare(sql)
	if err != nil {
		return err
	}
	defer sth.Close()
	rows, err := sth.Query(args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if select_labels == nil {
		select_labels, err = rows.Columns()
		if err != nil {
			return err
		}
	}
	names := make([]interface{}, len(select_labels))
	x := make([]interface{}, len(select_labels))
	for j := range select_labels {
		x[j] = &names[j]
	}
	for rows.Next() {
		err = rows.Scan(x...)
		if err != nil {
			return err
		}
		res := make(map[string]interface{})
		for j, v := range select_labels {
			name := names[j]
			if name != nil {
				switch name.(type) {
				case []uint8:
					res[v] = string(name.([]uint8))
					//				case int64:
					//					res[v] = strconv.FormatInt(name.(int64), 10)
				default:
					res[v] = name
				}
				//			} else {
				//				res[v] = nil
			}
		}
		*lists = append(*lists, res)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return nil
}

func (self *DBI) Do_proc(hash map[string]interface{}, names []string, proc_name string, args ...interface{}) error {
	n := len(args)
	str_q := strings.Join(strings.Split(strings.Repeat("?", n), ""), ",")
	str := "CALL " + proc_name + "(" + str_q
	str_n := "@" + strings.Join(names, ",@")
	if names != nil {
		str += ", " + str_n
	}
	str += ")"

	err := self.Do_sql(str, args...)
	if err != nil {
		return err
	}
	return self.Get_sql_label(hash, "SELECT "+str_n, names)
}

func (self *DBI) Select_proc(lists *[]map[string]interface{}, proc_name string, args ...interface{}) error {
	return self.Select_do_proc_label(lists, nil, nil, proc_name, nil, args...)
}

func (self *DBI) Select_proc_label(lists *[]map[string]interface{}, proc_name string, select_labels []string, args ...interface{}) error {
	return self.Select_do_proc_label(lists, nil, nil, proc_name, select_labels, args...)
}

func (self *DBI) Select_do_proc(lists *[]map[string]interface{}, hash map[string]interface{}, names []string, proc_name string, args ...interface{}) error {
	return self.Select_do_proc_label(lists, hash, names, proc_name, nil, args...)
}

func (self *DBI) Select_do_proc_label(lists *[]map[string]interface{}, hash map[string]interface{}, names []string, proc_name string, select_labels []string, args ...interface{}) error {
	n := len(args)
	str_q := strings.Join(strings.Split(strings.Repeat("?", n), ""), ",")
	str := "CALL " + proc_name + "(" + str_q
	str_n := "@" + strings.Join(names, ",@")
	if names != nil {
		str += ", " + str_n
	}
	str += ")"

	err := self.Select_sql_label(lists, str, select_labels, args...)
	if err != nil {
		return err
	}
	if hash == nil {
		return nil
	}
	return self.Get_sql_label(hash, "SELECT "+str_n, names)
}

package genelet

import (
//"github.com/golang/glog"
	"net/url"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"strconv"
	"database/sql"
)

type Model struct {
	Crud

	ARGS	url.Values
	LISTS   *[]map[string]interface{}
	OTHER   *map[string]interface{}

	SORTBY		string
	SORTREVERSE	string
	PAGENO		string
	ROWCOUNT	string
	TOTALNO		string
	MAX_PAGENO	string
	FIELD		string
	EMPTIES		string

	Nextpages	map[string][]map[string]interface{}
	Storage		map[string]interface{}

	Current_key		string
	Current_keys	[]string
	Current_id_auto	string
	Key_in			map[string]string //delete , stop if key in other tables

	Insert_pars		[]string
	Edit_pars		[]string
	Update_pars		[]string
	Insupd_pars		[]string
	Topics_pars		[]string
	Topics_hashpars	map[string]string

	Total_force		int
}

func (self *Model)Initialize(comp *Component) {
	self.SORTBY		 = comp.Sortby
	self.SORTREVERSE = comp.Sortreverse
	self.PAGENO		 = comp.Pageno
	self.ROWCOUNT	 = comp.Rowcount
	self.TOTALNO	 = comp.Totalno
	self.MAX_PAGENO	 = comp.Maxpageno
	self.FIELD		 = comp.Fields
	self.EMPTIES	 = comp.Empties

	self.Nextpages	 = comp.Nextpages

	self.Current_table = comp.Current_table
	self.Current_tables = comp.Current_tables
	self.Current_key = comp.Current_key
	self.Current_keys= comp.Current_keys
	self.Current_id_auto = comp.Current_id_auto
	self.Key_in			 = comp.Key_in

	self.Insert_pars	 = comp.Insert_pars
	self.Edit_pars		 = comp.Edit_pars
	self.Update_pars	 = comp.Update_pars
	self.Insupd_pars	 = comp.Insupd_pars
	self.Topics_pars	 = comp.Topics_pars
	self.Topics_hashpars = comp.Topics_hash

	self.Total_force	 = comp.Total_force
}

func (self *Model)Set_db(db *sql.DB) {
	self.Db = db
}

func (self *Model)Set_defaults(args url.Values, lists *[]map[string]interface{}, other *map[string]interface{}, storage map[string]interface{}) {
	self.ARGS  = args
	self.LISTS = lists
	self.OTHER = other
	self.Storage = storage
}

func (self *Model)filtered_fields(pars []string) []string {
	ARGS := self.ARGS
	field := ARGS[self.FIELD]
	if (field == nil) {
		return pars
	}

	out := make([]string,0)
	for _, val := range field {
		for _, v := range pars {
			if val==v {
				out = append(out, v)
				break
			}
		}
	}
	return out
}

func (self *Model)get_fv(pars []string) url.Values {
  ARGS := self.ARGS
  field_values := make(url.Values)
  for _, f := range self.filtered_fields(pars) {
    if (ARGS.Get(f) != "") {
      field_values[f] = ARGS[f]
    }
  }
  return field_values
}

func (self *Model)get_id_val(extra url.Values) (interface{}, []interface{}) {
  ARGS := self.ARGS
  if (self.Current_keys != nil && len(self.Current_keys)>0) {
    val := make([]interface{},0)
    for _, v := range self.Current_keys {
      if (ARGS.Get(v) != "") {
        val = append(val, ARGS.Get(v))
      } else if (extra[v] != nil) {
        val = append(val, extra[v])
      } else {
        return self.Current_keys, nil
      }
    }
    return self.Current_keys, val
  }

  if ARGS.Get(self.Current_key) != "" {
    return self.Current_key, []interface{}{ARGS.Get(self.Current_key)}
  } else if extra.Get(self.Current_key) != "" {
    return self.Current_key, []interface{}{extra.Get(self.Current_key)}
  } else if ARGS.Get("_gid_url") != "" {
    return self.Current_key, []interface{}{ARGS.Get("_gid_url")}
  }
  return self.Current_key, nil
}

func (self *Model) Topics(extra ...url.Values) error {
  ARGS  := self.ARGS
  totalno := self.TOTALNO
  if (self.Total_force != 0 && ARGS.Get(self.ROWCOUNT) != "" && (ARGS.Get(self.PAGENO)=="" || ARGS.Get(self.PAGENO)=="1")) {
    var nt int
    if (self.Total_force < -1) {
      nt = int(math.Abs(float64(self.Total_force)))
    } else if (self.Total_force == -1 || ARGS.Get(totalno)=="") {
      hash := make(map[string]interface{})
      err := self.Total_hash(hash, totalno, extra ...)
      if (err != nil) { return err }
      nt = int(hash[totalno].(int64))
    } else {
      nt, _ = strconv.Atoi(ARGS.Get(totalno))
    }
    ARGS.Set(totalno, strconv.Itoa(nt))
    nr, _ := strconv.Atoi(ARGS.Get(self.ROWCOUNT))
    max_pageno := int( (nt-1)/nr )+1
    ARGS.Set(self.MAX_PAGENO, strconv.Itoa(max_pageno))
  }

  var fields interface{}
  if self.Topics_hashpars==nil {
    fields = self.filtered_fields(self.Topics_pars)
  } else {
    fields = self.Topics_hashpars
  }
  err := self.Topics_hash_order(self.LISTS, fields, self.Get_order_string(), extra...)
  if err != nil {return err}

  return self.Process_after("topics", extra...);
}

func (self *Model) Edit(extra ...url.Values) error {
  var id interface{}
  var val []interface{}
  if extra == nil || len(extra)==0 {
    id, val = self.get_id_val(nil)
  } else {
    id, val = self.get_id_val(extra[0])
  }
  if val == nil {return Err(1040)}

  fields := self.filtered_fields(self.Edit_pars)
  if fields == nil {return Err(1077)}

  var err error
  if extra != nil && len(extra)>0 && extra[0] != nil {
    err = self.Edit_hash(self.LISTS, fields, id, val, extra[0])
  } else {
    err = self.Edit_hash(self.LISTS, fields, id, val)
  }
  if err != nil { return err }

  return self.Process_after("edit", extra...)
}

// use 'extra' to override field_values for selected fields
func (self *Model) Insert(extra ...url.Values) error {
  field_values := self.get_fv(self.Insert_pars)

  if (extra != nil && len(extra) > 0) {
    for key, value := range extra[0] {
      if (Grep(self.Insert_pars, key)) {
        field_values[key] = value
      }
    }
  }
  if field_values == nil { return Err(1078) }

  err := self.Insert_hash(field_values)
  if err != nil { return err }

  if (self.Current_id_auto != "") {
	auto_id := strconv.FormatInt(self.Last_id,10)
    field_values.Set(self.Current_id_auto, auto_id)
	self.ARGS.Set(self.Current_id_auto, auto_id)
  }
  *self.LISTS = from_fv(field_values);

  return self.Process_after("insert", extra...)
}

func (self *Model) Insupd(extra ...url.Values) error {
  uniques := self.Insupd_pars
  if (uniques==nil) { return Err(1078) }

  field_values := self.get_fv(self.Insert_pars)
  if (extra != nil && len(extra)>0) {
    for key, value := range extra[0] {
      if (Grep(self.Insert_pars, key)) {
        field_values[key] = value
      }
    }
  }
  if field_values == nil { return Err(1076) }

  for _, v := range uniques {
    if (field_values[v] == nil) { return Err(1075) }
  }

  upd_field_values := self.get_fv(self.Update_pars)

  s_hash := ""
  err := self.Insupd_hash(field_values, upd_field_values, self.Current_key, uniques, &s_hash)
  if err != nil { return err }

  if (self.Current_id_auto != "" && s_hash=="insert") {
    field_values[self.Current_id_auto] = []string{strconv.FormatInt(self.Last_id,10)}
  }
  hash := make(map[string]interface{})
  for k, v := range field_values {
    hash[k] = v[0]
  }
  *self.LISTS = append(*self.LISTS, hash)

  return self.Process_after("insupd", extra...)
}

func (self *Model) Update(extra ...url.Values) error {
  ARGS := self.ARGS
  var id interface{}
  var val []interface{}
  if extra != nil && len(extra)>0 {
    id, val = self.get_id_val(extra[0])
  } else {
    id, val = self.get_id_val(nil)
  }
  if (val == nil) { return Err(1040) }

  field_values := self.get_fv(self.Update_pars)
  if field_values == nil { return Err(1074) }

  if (len(field_values)<=1 && field_values[id.(string)] != nil) {
    *self.LISTS = from_fv(field_values)
    return self.Process_after("update", extra...)
  }

  var err error
  if (ARGS.Get(self.EMPTIES) != "") {
    err = self.Update_hash_nulls(field_values, id, val, ARGS[self.EMPTIES], extra...)
  } else {
    err = self.Update_hash(field_values, id, val, extra...)
  }
  if (err != nil) { return err }

  switch id.(type) {
  case []string:
    for i, v := range id.([]string) {
      field_values.Set(v, val[i].(string))
    }
  case string:
    field_values.Set(id.(string), val[0].(string))
  }
  *self.LISTS = from_fv(field_values)

  return self.Process_after("Update", extra...)
}

func from_fv(field_values url.Values) []map[string]interface{} {
	hash := make(map[string]interface{})
	for k, v := range field_values {
		hash[k] = v[0]
	}
	return []map[string]interface{}{hash};
}

func (self *Model) Delete(extra ...url.Values) error {
  var id interface{}
  var val []interface{}
  if (extra == nil) {
    id, val = self.get_id_val(nil)
  } else {
    id, val = self.get_id_val(extra[0])
  }
  if (val == nil) { return Err(1040) }

  if (self.Key_in != nil) {
    for table, keyname := range self.Key_in {
      for _, v := range val {
        err := self.Existing(table, keyname, v)
        if (err != nil) { return err }
      }
    }
  }

  err := self.Delete_hash(id, val, extra ...)
  if (err != nil) { return err }

  hash := make(map[string]interface{})
  switch id.(type) {
  case []string:
    for i, v := range id.([]string) {
      hash[v] = val[i]
    }
  case string:
    hash[id.(string)] = val[0]
  }
  *self.LISTS = make([]map[string]interface{},0)
  *self.LISTS = append(*self.LISTS, hash)

  return self.Process_after("delete", extra...)
}

func (self *Model) Existing(table string, field string, val interface{}) error {
  hash := make(map[string]interface{})
  err := self.Get_sql(hash, "SELECT " + field + " FROM " + table + " WHERE " + field + "=?", val)
  if (err != nil) { return err }
  if (hash[field] != nil) { return Err(1075) }

  return nil
}

func (self *Model) Randomid(table string, field string, m ...interface{}) error {
  ARGS := self.ARGS
  var min, max, trials int
  if (m == nil) {
    min = 0
    max = 4294967295
    trials = 10
  } else {
	min = m[0].(int)
	max = m[1].(int)
    if (m[2] == nil) {
      trials = 10
    } else {
      trials = m[2].(int)
    }
  }

  for i:=0; i<trials; i++ {
    val := min + int(rand.Float32()*float32(max-min))
    err := self.Existing(table, field, val)
    if (err!=nil) { continue }
    ARGS.Set(field, strconv.Itoa(val))
    return nil
  }

  return Err(1076)
}

func (self *Model) Get_order_string() string {
  ARGS := self.ARGS
  var column string
  if (ARGS.Get(self.SORTBY)=="") {
    if (self.Current_keys != nil && len(self.Current_keys)>0) {
      column = strings.Join(self.Current_keys, ",")
    } else {
      column = self.Current_key
    }
  } else {
    column = ARGS.Get(self.SORTBY)
  }

  matched, err := regexp.MatchString("\\.", column)
  if (self.Current_tables != nil && err == nil && matched) {
	if !strings.Contains(column, `.`) {
      if (self.Current_tables[0].Alias != "") {
        column = self.Current_tables[0].Alias + "." + column
      } else {
        column = self.Current_tables[0].Name + "." + column
      }
    }
  }
  order := "ORDER BY " + column
  if (ARGS.Get(self.SORTREVERSE) != "") {order += " DESC"}

  if (ARGS.Get(self.ROWCOUNT) != "") {
    rowcount, err := strconv.Atoi(ARGS.Get(self.ROWCOUNT))
    if (err != nil) { return "" }
    pageno := 1
    if (ARGS.Get(self.PAGENO) == "") {
      ARGS.Set(self.PAGENO, "1")
    } else {
      pageno, err = strconv.Atoi(ARGS.Get(self.PAGENO))
      if (err != nil) { return "" }
    }
    order += " LIMIT " + strconv.Itoa(rowcount) + " OFFSET " + strconv.Itoa((pageno-1) * rowcount)
  }

  matched, err = regexp.MatchString("[;'\"]", order)
  if (err != nil || matched) {
    return ""
  }
  return order
}

func (self *Model)another_object(page map[string]interface{}, args url.Values, lists *[]map[string]interface{}, other *map[string]interface{}) (interface{}, string, string, error) {
	model := page["model"].(string)
	if self.Storage == nil {
		return nil, "", "", Err(2013, "No storage")
	}
	p := self.Storage[model]
	if p== nil {
		return nil, "", "", Err(2013, "No stored " + model)
	}

    Invoke0(p, "Set_defaults", args, lists, other, self.Storage)
	Invoke0(p, "Set_db", self.Db)

	action := page["action"].(string)
	return p, strings.Title(action), model+"_"+action, nil
}

func (self *Model)Call_once(page map[string]interface{}, extra ...url.Values) error {
	args := self.ARGS
    ARGS, err := url.ParseQuery(args.Encode())
    if err != nil { return err }
	if ARGS.Get("sortby")      !="" { ARGS.Del("sortby") }
	if ARGS.Get("sortreverse") !="" { ARGS.Del("sortreverse") }
    lists := make([]map[string]interface{},0)
    other := make(map[string]interface{})

	var next_extra url.Values
	if extra != nil {
		next_extra = extra[0]
	}
	if page["manual"] != nil {
		if next_extra==nil { next_extra = url.Values{} }
		for k, v := range page["manual"].(map[string]interface{}) {
			next_extra.Set(k, Interface2String(v))
		}
	}

	p, action, marker, err := self.another_object(page, ARGS, &lists, &other)
	if err != nil { return err }

	if page["alias"] != nil {
		marker = page["alias"].(string)
	}

	OTHER := *self.OTHER
	if OTHER[marker] != nil { return nil }

	if extra != nil {
		ret := Invoke(p, action, next_extra)[0].Interface()
		if ret != nil { return ret.(error) }
	} else {
		ret := Invoke(p, action)[0].Interface()
		if ret != nil { return ret.(error) }
	}

	if len(lists)>0 {
		OTHER[marker] = lists
	}
	if len(other)>0 {
		for k, v := range other {
			OTHER[k] = v
		}
	}
	return nil
}

func (self *Model)Call_nextpage(page map[string]interface{}, extra ...url.Values) error {
	LISTS := *self.LISTS
	if (len(LISTS) < 1) { return nil }

	next_extra := make(url.Values)
	var err error
	if (extra != nil) {
		next_extra, err = url.ParseQuery(extra[0].Encode())
		if (err != nil) { return nil }
	}
	if (page["manual"] != nil) {
		for k, v := range page["manual"].(map[string]interface{}) {
			next_extra.Set(k, Interface2String(v))
		}
	}

	args := self.ARGS
    ARGS, err := url.ParseQuery(args.Encode())
    if (err != nil) { return err }
	if ARGS.Get("sortby")      !="" { ARGS.Del("sortby") }
	if ARGS.Get("sortreverse") !="" { ARGS.Del("sortreverse") }
    lists := make([]map[string]interface{},0)
    other := make(map[string]interface{})
	p, action, marker, err := self.another_object(page, ARGS, &lists, &other)
	if (err != nil) { return err }

	if page["alias"] != nil {
		marker = page["alias"].(string)
	}

	OTHER := *self.OTHER
	for _, item := range LISTS {
		found := false
		for k, v := range page["relate_item"].(map[string]interface{}) {
			if (item[k] != nil) {
				found = true
				next_extra.Set(v.(string), Interface2String(item[k]))
			} else {
				next_extra.Del(v.(string))
			}
		}
		if found==false { continue }
		nextextra := make(url.Values)
		x:=strings.ToUpper(action[:1])+action[1:]
		ret := Invoke(p, x, next_extra, nextextra)[0].Interface()
		if (ret != nil) { return ret.(error) }

		if (len(lists)>0) {
			item[marker] = lists
		}
		if (len(other)>0) {
			for k, v := range other {
				OTHER[k] = v
			}
		}
		lists = make([]map[string]interface{},0)
		other = make(map[string]interface{})
    }

	return nil
}

func (self *Model)Process_after(action string, extra ...url.Values) error {
	if self.Nextpages == nil || self.Nextpages[action] == nil { return nil }

	for i, page := range self.Nextpages[action] {
		var err error
		if page["relate_item"]==nil {
			if extra != nil && len(extra)>=(i+2) && extra[i+1] != nil {
				err = self.Call_once(page, extra[i+1])
			} else {
				err = self.Call_once(page, make(url.Values))
			}
		} else if len(*self.LISTS)==0 {
			return nil
		} else {
			if extra != nil && len(extra)>=(i+2) && extra[i+1] != nil {
				err = self.Call_nextpage(page, extra[i+1])
			} else {
				err = self.Call_nextpage(page, make(url.Values))
			}
		}
		if err != nil { return err }
	}
	return nil
}

func (self *Model)ProperValue(v string, extra url.Values) string {
  ARGS := self.ARGS
  var out string
  if extra==nil {
    out = ARGS.Get(v)
  } else {
    out = extra.Get(v)
    if out=="" { out = ARGS.Get(v) }
  }
  return out
}

func (self *Model)ProperValues(vs []string, extra url.Values) []string {
  ARGS := self.ARGS
  outs := make([]string, len(vs))
  for i, v := range vs {
    if extra==nil {
      outs[i] = ARGS.Get(v)
    } else {
      outs[i] = extra.Get(v)
      if outs[i]=="" { outs[i] = ARGS.Get(v) }
    }
  }

  return outs
}

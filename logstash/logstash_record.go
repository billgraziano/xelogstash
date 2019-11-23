package logstash

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Record holds the parent struct of what we will send to logstash
type Record map[string]interface{}

// NewRecord initializes a new record
func NewRecord() Record {
	r := make(map[string]interface{})
	return r
}

// ToLower sets most fields to lower case.  Fields like message
// and various SQL statements are unchanged
// func (e *Record) ToLower() {
// 	for k, v := range *e {
// 		if k != "message" && k != "timestamp" && k != "sql_text" && k != "statement" && k != "batch_text" {
// 			s, ok := v.(string)
// 			if ok {
// 				(*e)[k] = strings.ToLower(s)
// 			}
// 		}
// 	}
// }

// ToJSON marshalls to a string
func (r *Record) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return "", errors.Wrap(err, "marshal")
	}

	jsonString := string(jsonBytes)
	return jsonString, nil
}

// ToJSONBytes marshalls to a byte array
func (r *Record) ToJSONBytes() ([]byte, error) {
	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return []byte{}, errors.Wrap(err, "marshal")
	}
	return jsonBytes, nil
}

// ProcessMods applies adds, renames, and moves to a JSON string
func ProcessMods(json string, adds, copies, moves map[string]string) (string, error) {
	var err error

	// Adds
	for k, v := range adds {
		i := getValue(v)
		if gjson.Get(json, k).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", k)
		}
		json, err = sjson.Set(json, k, i)
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %s", k, v)
		}
	}

	// Copies
	for src, dst := range copies {

		if gjson.Get(json, dst).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", dst)
		}
		r := gjson.Get(json, src)
		if !r.Exists() {
			continue
		}
		json, err = sjson.Set(json, dst, doubleSlashes(r.Value()))
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %v", dst, r.Value())
		}
		//fmt.Println(r.Value(), doubleSlashes(r.Value()))
	}

	// Moves
	for src, dst := range moves {

		if gjson.Get(json, dst).Exists() {
			return json, errors.Wrapf(err, "can't overwrite key: %s", dst)
		}
		r := gjson.Get(json, src)
		if !r.Exists() {
			continue
		}
		json, err = sjson.Set(json, dst, doubleSlashes(r.Value()))
		if err != nil {
			return json, errors.Wrapf(err, "sjson.set: %s %v", dst, r.Value())
		}
		json, err = sjson.Delete(json, src)
		if err != nil {
			return json, errors.Wrapf(err, "can't delete: %s", src)
		}
	}

	return json, err
}

func doubleSlashes(v interface{}) interface{} {
	x, ok := v.(string)
	if !ok {
		return v
	}
	return strings.Replace(x, "\\", "\\\\", -1)
}

func getValue(s string) (v interface{}) {
	var err error
	v, err = strconv.ParseBool(s)
	if err == nil {
		return v
	}

	v, err = strconv.ParseInt(s, 0, 64)
	if err == nil {
		return v
	}

	v, err = strconv.ParseFloat(s, 64)
	if err == nil {
		return v
	}

	// check for '0.7' => (string) 0.7
	if len(s) >= 2 && strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'") {
		s = s[1 : len(s)-1]
	}

	return doubleSlashes(s)
}

// Set assigns a string value to a key in the event
func (r *Record) Set(key string, value interface{}) {
	(*r)[key] = value
}

// Copy value from srckey to newkey
func (r *Record) Copy(srckey, newkey string) {
	v, ok := (*r)[srckey]
	if !ok {
		r.Set(newkey, "")
		return
	}
	(*r)[newkey] = v
}

// Move old key to new key
func (r *Record) Move(oldkey, newkey string) {
	(*r).Copy(oldkey, newkey)
	delete((*r), oldkey)
}

// SetIfEmpty sets a value if one doesn't already exist
func (r *Record) SetIfEmpty(key string, value interface{}) {
	_, exists := (*r)[key]
	if !exists {
		r.Set(key, value)
	}
}

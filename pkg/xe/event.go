package xe

import (
	"fmt"
	"strconv"
	"time"
)

// Event is a key value of entries for the XE event
type Event map[string]any

// Name returns the "name" attribute from the event
// It returns an empty string if not found or not a string
func (e *Event) Name() string {
	i, ok := (*e)["name"]
	if !ok {
		return ""
	}
	s, ok := i.(string)
	if !ok {
		return ""
	}
	return s
}

// Timestamp returns the "timestamp" attribute from the event
// It returns the zero value if it doesn't exist
func (e Event) Timestamp() time.Time {
	zero := time.Time{}
	i, ok := e["timestamp"]
	if !ok {
		return zero
	}
	ts, ok := i.(time.Time)
	if !ok {
		return zero
	}
	return ts
}

// ErrorNumber returns the error number if it exists in the event
func (e Event) ErrorNumber() (int64, bool) {
	raw, ok := e["error_number"]
	if !ok {
		return 0, false
	}
	println("got an error number")
	fmt.Printf("error_number: %+v (%T)\n", raw, raw)
	val, ok := raw.(int64)
	println("is int:", ok)
	if !ok {
		return 0, false
	}
	println("error_number:", val)
	return val, true
}

// GetInt64 returns an integer value.  The raw map value must be an int64.
func (e *Event) GetInt64(key string) (int64, bool) {
	raw, ok := (*e)[key]
	if !ok {
		return 0, false
	}

	i64, ok := raw.(int64)
	if !ok {
		return 0, false
	}

	return i64, true
}

// GetIntFromString returns an int64 from a string'd interface
func (e *Event) GetIntFromString(key string) (int64, bool) {
	v, exists := (*e)[key]
	if !exists {
		return 0, false
	}
	str := fmt.Sprintf("%v", v)
	i64, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, false
	}
	return i64, true
}

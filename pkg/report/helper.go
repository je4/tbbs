package report

import (
	"fmt"
	"reflect"
	"strconv"
)

// FormatInt formats an integer with grouping decimals, in decimal radix.
// Grouping signs are inserted after every groupSize digits, starting from the right.
// A groupingSize less than 1 will default to 3.
// Only ASCII grouping decimal signs are supported which may be provided with
// grouping.
//
// For details, see https://github.com/icza/gox
// For details, see https://stackoverflow.com/a/31046325/1705598
func FormatInt(n int64, groupSize int, grouping byte) string {
	if groupSize < 1 {
		groupSize = 3
	}

	in := strconv.FormatInt(n, 10)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits-- // First character is the - sign (not a digit)
	}
	numOfCommas := (numOfDigits - 1) / groupSize

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == groupSize {
			j, k = j-1, 0
			out[j] = grouping
		}
	}
}

func StructToMap(value interface{}) map[string][]interface{} {
	m := make(map[string][]interface{})
	relType := reflect.TypeOf(value)
	item := reflect.ValueOf(value)
	//fmt.Printf("Type: %v", relType)
	switch item.Kind() {
	case reflect.Slice:
		for i := 0; i < item.Len(); i++ {
			vs := StructToMap(item.Index(i).Interface())
			for name, vals := range vs {
				m[fmt.Sprintf("%d.%s", i, name)] = vals
			}
		}
	case reflect.Map:
		for _, k := range item.MapKeys() {
			ret := StructToMap(item.MapIndex(k).Interface())
			name := k.String()
			for key, vals := range ret {
				if key != "" {
					name += "." + key
				}
				m[name] = []interface{}{}
				for _, val := range vals {
					m[name] = append(m[name], val)
				}
			}
		}
	case reflect.Struct:
		for i := 0; i < relType.NumField(); i++ {
			ret := StructToMap(item.Field(i).Interface())
			for key, vals := range ret {
				name := relType.Field(i).Name
				if key != "" {
					name += "." + key
				}
				m[name] = []interface{}{}
				for _, val := range vals {
					m[name] = append(m[name], val)
				}
			}
		}
	default:
		switch v := value.(type) {
		case string:
			if v != "" && v != "0/0" {
				m[""] = []interface{}{v}
			}
		case int:
			if v != 0 {
				m[""] = []interface{}{v}
			}
		}
	}
	return m
}

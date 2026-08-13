package utils

import (
	"reflect"
	"strings"

	"github.com/sagernet/sing/common/json"
)

// MarshalArrayF serializes v (which must be a slice/array) to a JSON array
// string where elements are separated by ", " and the brackets are tight,
// e.g. [1, 2, 3] instead of [1,2,3].
func MarshalArrayF(v any) string {
	rv := reflect.ValueOf(v)
	parts := make([]string, rv.Len())
	for i := range parts {
		b, _ := json.Marshal(rv.Index(i).Interface())
		parts[i] = string(b)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

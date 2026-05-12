// Package query implements the query-string encoding rules of the
// WebScraping.AI API.
//
// The API mixes three OpenAPI query-string styles, and no off-the-shelf
// encoder handles the combination correctly:
//
//   - deepObject + explode for the "headers" and "fields" dicts
//     → headers[Cookie]=foo&fields[title]=bar
//   - form + explode for the "selectors" array
//     → selectors=h1&selectors=.price (no [] brackets)
//   - flat key=value for everything else, with booleans serialised as
//     the strings "true" / "false".
//
// nil values are dropped at every level. Spaces are encoded as %20
// (not +) so the output matches what hits the wire.
package query

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Param is the value side of a query parameter. Allowed concrete types:
//
//   - string, int, int64, float64, bool — scalars
//   - []string — repeated key (used for "selectors")
//   - map[string]string — deepObject (used for "headers", "fields")
//   - nil — dropped entirely
//
// Other types are rejected by Encode at runtime; this keeps the encoder
// small without pulling in reflect for every call.
type Param any

// Params is an ordered list of key/value pairs. We keep insertion order
// so URLs are stable across runs (helps tests, caching, and humans).
type Params []Pair

// Pair is one (key, value) entry in a Params list.
type Pair struct {
	Key   string
	Value Param
}

// Set appends or replaces a key. If the key is already present its value
// is replaced in-place (preserves order).
func (p *Params) Set(key string, value Param) {
	for i := range *p {
		if (*p)[i].Key == key {
			(*p)[i].Value = value
			return
		}
	}
	*p = append(*p, Pair{Key: key, Value: value})
}

// Encode flattens p into an ordered slice of [2]string{key, value} pairs.
// Keys and values are *not* percent-encoded; use EncodeToString for the
// wire form.
func Encode(p Params) [][2]string {
	out := make([][2]string, 0, len(p))
	for _, pair := range p {
		if pair.Value == nil {
			continue
		}
		switch v := pair.Value.(type) {
		case string:
			out = append(out, [2]string{pair.Key, v})
		case int:
			out = append(out, [2]string{pair.Key, strconv.Itoa(v)})
		case int64:
			out = append(out, [2]string{pair.Key, strconv.FormatInt(v, 10)})
		case float64:
			out = append(out, [2]string{pair.Key, strconv.FormatFloat(v, 'f', -1, 64)})
		case bool:
			if v {
				out = append(out, [2]string{pair.Key, "true"})
			} else {
				out = append(out, [2]string{pair.Key, "false"})
			}
		case *bool:
			if v == nil {
				continue
			}
			if *v {
				out = append(out, [2]string{pair.Key, "true"})
			} else {
				out = append(out, [2]string{pair.Key, "false"})
			}
		case *int:
			if v == nil {
				continue
			}
			out = append(out, [2]string{pair.Key, strconv.Itoa(*v)})
		case *string:
			if v == nil {
				continue
			}
			out = append(out, [2]string{pair.Key, *v})
		case []string:
			for _, item := range v {
				out = append(out, [2]string{pair.Key, item})
			}
		case map[string]string:
			// Sort subkeys so the wire output is deterministic; humans
			// reading test failures benefit, and HTTP semantics don't
			// care about subkey order.
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				out = append(out, [2]string{
					pair.Key + "[" + k + "]",
					v[k],
				})
			}
		default:
			panic(fmt.Sprintf("query: unsupported param type %T for key %q", pair.Value, pair.Key))
		}
	}
	return out
}

// EncodeToString renders p as an already-escaped query string (without
// the leading "?"). Spaces become %20; brackets become %5B/%5D.
func EncodeToString(p Params) string {
	pairs := Encode(p)
	var b strings.Builder
	for i, kv := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(percent(kv[0]))
		b.WriteByte('=')
		b.WriteString(percent(kv[1]))
	}
	return b.String()
}

// percent encodes s for the query component. We do this by hand instead
// of using url.QueryEscape so that spaces become %20 (not +). The set of
// "safe" characters mirrors RFC 3986 unreserved + sub-delims minus '&',
// '=', '+', '#'.
func percent(s string) string {
	// Fast path: already safe.
	safe := true
	for i := 0; i < len(s); i++ {
		if !isUnreserved(s[i]) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			b.WriteByte(c)
		} else {
			b.WriteByte('%')
			b.WriteByte(hex(c >> 4))
			b.WriteByte(hex(c & 0xf))
		}
	}
	return b.String()
}

func isUnreserved(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z':
		return true
	case c >= 'a' && c <= 'z':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	switch c {
	case '-', '_', '.', '~':
		return true
	}
	return false
}

func hex(n byte) byte {
	if n < 10 {
		return '0' + n
	}
	return 'A' + (n - 10)
}

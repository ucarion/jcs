package jcs

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"unicode/utf16"
)

// func Canonicalize(v interface{}) []byte {
// 	return nil
// }

func Canonicalize(v interface{}) []byte {
	buf := []byte{}
	canonicalize(&buf, v)
	return buf
}

var nullBytes = []byte("null")
var trueBytes = []byte("true")
var falseBytes = []byte("false")
var startArrayBytes = []byte("[")
var endArrayBytes = []byte("]")
var startObjectBytes = []byte("{")
var endObjectBytes = []byte("}")
var commaBytes = []byte(",")
var colonBytes = []byte(":")

var matchLeading0Exponent = regexp.MustCompile(`([eE][\+\-])0+([1-9])`) // 1e-07 => 1e-7

func canonicalize(buf *[]byte, v interface{}) {
	if v == nil {
		*buf = append(*buf, nullBytes...)
		return
	}

	switch v := v.(type) {
	case bool:
		if v {
			*buf = append(*buf, trueBytes...)
		} else {
			*buf = append(*buf, falseBytes...)
		}
	case float64:
		// Special-case zero because of the possibility of negative zero.
		if v == 0 {
			*buf = append(*buf, '0')
			return
		}

		// Silently ignore NaN and Inf.
		//
		// TODO return an error instead.
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return
		}

		var s string

		abs := math.Abs(v)
		if abs < 1e+21 && abs >= 1e-6 {
			s = strconv.FormatFloat(v, 'f', -1, 64)
		} else {
			s = matchLeading0Exponent.ReplaceAllString(strconv.FormatFloat(v, 'g', -1, 64), "$1$2")
		}

		*buf = append(*buf, []byte(s)...)
	case string:
		canonicalizeString(buf, v)
	case []interface{}:
		*buf = append(*buf, startArrayBytes...)

		for i, elem := range v {
			canonicalize(buf, elem)

			if i != len(v)-1 {
				*buf = append(*buf, commaBytes...)
			}
		}

		*buf = append(*buf, endArrayBytes...)
	case map[string]interface{}:
		pairs := make([]keyVal, 0, len(v))
		for k, v := range v {
			pairs = append(pairs, keyVal{key: k, val: v})
		}

		sort.Slice(pairs, func(i, j int) bool {
			a := utf16.Encode([]rune(pairs[i].key))
			b := utf16.Encode([]rune(pairs[j].key))

			for i := 0; i < len(a) && i < len(b); i++ {
				if a[i] < b[i] {
					return true
				}

				if a[i] > b[i] {
					return false
				}
			}

			return len(a) < len(b)
		})

		*buf = append(*buf, startObjectBytes...)
		for i, kv := range pairs {
			canonicalizeString(buf, kv.key)
			*buf = append(*buf, colonBytes...)
			canonicalize(buf, kv.val)

			if i != len(pairs)-1 {
				*buf = append(*buf, commaBytes...)
			}
		}

		*buf = append(*buf, endObjectBytes...)
	}
}

func canonicalizeString(buf *[]byte, s string) {
	*buf = append(*buf, '"')

	for _, c := range s {
		if c == '\\' {
			*buf = append(*buf, '\\', '\\')
		} else if c == '"' {
			*buf = append(*buf, '\\', '"')
		} else if c == '\b' {
			*buf = append(*buf, '\\', 'b')
		} else if c == '\f' {
			*buf = append(*buf, '\\', 'f')
		} else if c == '\n' {
			*buf = append(*buf, '\\', 'n')
		} else if c == '\r' {
			*buf = append(*buf, '\\', 'r')
		} else if c == '\t' {
			*buf = append(*buf, '\\', 't')
		} else if c < 0x20 {
			*buf = append(*buf, []byte(fmt.Sprintf("\\u%04x", c))...)
		} else {
			*buf = append(*buf, []byte(string(c))...)
		}
	}

	*buf = append(*buf, '"')
}

type keyVal struct {
	key string
	val interface{}
}

package regex

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-ttlcache"
	"github.com/GRbit/go-pcre"
)

type Regexp = pcre.Regexp

type str interface{
	[]byte | string
}

var regReReplaceQuote pcre.Regexp = pcre.MustCompileJIT(`\\[\\']`, pcre.UTF8, pcre.CONFIG_JIT)
var regReReplaceComment pcre.Regexp = pcre.MustCompileJIT(`\(\?\#.*?\)`, pcre.UTF8, pcre.CONFIG_JIT)

var varType map[string]reflect.Type

var cache *ttlcache.Cache[string, pcre.Regexp] = ttlcache.New[string, pcre.Regexp](2 * time.Hour, 1 * time.Hour)

func init() {
	varType = map[string]reflect.Type{}

	varType["array"] = reflect.TypeOf([]interface{}{})
	varType["map"] = reflect.TypeOf(map[string]interface{}{})

	varType["int"] = reflect.TypeOf(int(0))
	varType["int64"] = reflect.TypeOf(int64(0))
	varType["float64"] = reflect.TypeOf(float64(0))
	varType["float32"] = reflect.TypeOf(float32(0))

	varType["string"] = reflect.TypeOf("")
	varType["byteArray"] = reflect.TypeOf([]byte{})
	varType["byte"] = reflect.TypeOf([]byte{0}[0])
	varType["byteArrayArray"] = reflect.TypeOf([][]byte{})

	// int32 returned instead of byte
	varType["int32"] = reflect.TypeOf(' ')

	Compile(`(\\|)\$([0-9]|\{[0-9]+\})`)
}

// An easy way to join multiple values into a single []byte
// accepts: []byte, byte, int32, string, [][]byte, int, int64, float64, float32
func JoinBytes(bytes ...interface{}) []byte {
	res := []byte{}
	for _, b := range bytes {
		switch reflect.TypeOf(b) {
		case varType["byteArray"]:
			res = append(res, b.([]byte)...)
		case varType["byte"]:
			res = append(res, b.(byte))
		case varType["int32"]:
			res = append(res, byte(b.(int32)))
		case varType["string"]:
			res = append(res, []byte(b.(string))...)
		case varType["byteArrayArray"]:
			for _, v := range b.([][]byte) {
				res = append(res, v...)
			}
		case varType["int"]:
			res = append(res, []byte(strconv.Itoa(b.(int)))...)
		case varType["int64"]:
			res = append(res, []byte(strconv.Itoa(int(b.(int64))))...)
		case varType["float64"]:
			res = append(res, []byte(strconv.FormatFloat(b.(float64), 'f', -1, 64))...)
		case varType["float32"]:
			res = append(res, []byte(strconv.FormatFloat(float64(b.(float32)), 'f', -1, 32))...)
		}
	}
	return res
}

// An easy way to join multiple values with different types, into a single value with one type
// accepts: []byte, byte, int32, string, [][]byte, int, int64, float64, float32
func Join[T str](bytes ...interface{}) T {
	res := []byte{}
	for _, b := range bytes {
		switch reflect.TypeOf(b) {
		case varType["byteArray"]:
			res = append(res, b.([]byte)...)
		case varType["byte"]:
			res = append(res, b.(byte))
		case varType["int32"]:
			res = append(res, byte(b.(int32)))
		case varType["string"]:
			res = append(res, []byte(b.(string))...)
		case varType["byteArrayArray"]:
			for _, v := range b.([][]byte) {
				res = append(res, v...)
			}
		case varType["int"]:
			res = append(res, []byte(strconv.Itoa(b.(int)))...)
		case varType["int64"]:
			res = append(res, []byte(strconv.Itoa(int(b.(int64))))...)
		case varType["float64"]:
			res = append(res, []byte(strconv.FormatFloat(b.(float64), 'f', -1, 64))...)
		case varType["float32"]:
			res = append(res, []byte(strconv.FormatFloat(float64(b.(float32)), 'f', -1, 32))...)
		}
	}
	return T(res)
}

func setCache(re string, reg pcre.Regexp) {
	cache.Set(re, reg)
}

func getCache(re string) (pcre.Regexp, bool) {
	if val, ok := cache.Get(re); ok {
		return val, true
	}

	return pcre.Regexp{}, false
}

// Compile a regular expression and store it in the cache
func Compile(re string) Regexp {
	if strings.Contains(re, `\'`) {
		r := []byte(re)
		ind := regReReplaceQuote.FindAllIndex(r, 0)

		for i := len(ind) - 1; i >= 0; i-- {
			if r[ind[i][1]-1] == '\'' {
				r[ind[i][0]] = 0
				r[ind[i][1]-1] = '`'
			}
		}

		r = bytes.ReplaceAll(r, []byte{0}, []byte(""))
		re = string(r)
	}

	if strings.Contains(re, `(?#`) {
		re = regReReplaceComment.ReplaceAllString(re, ``, 0)
	}

	if val, ok := getCache(re); ok {
		return val
	} else {
		reg := pcre.MustCompile(re, pcre.UTF8)

		// commented below methods compiled 10000 times in 0.1s (above method being used finished in half of that time)
		// reg := pcre.MustCompileParse(re)
		// reg := pcre.MustCompileJIT(re, pcre.UTF8, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.EXTRA, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.JAVASCRIPT_COMPAT, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileParseJIT(re, pcre.STUDY_JIT_COMPILE)

		go setCache(re, reg)
		return reg
	}
}

// Replace a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
func RepFunc[T str, J str](str T, re string, rep func(data func(int) []byte) J, blank ...bool) J {
	b := []byte(str)

	reg := Compile(re)

	// ind := reg.FindAllIndex(b, pcre.UTF8)
	ind := reg.FindAllIndex(b, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := b[pos[0]:pos[1]]
		m := reg.Matcher(v, 0)

		if len(blank) != 0 {
			gCache := map[int][]byte{}
			r := rep(func(g int) []byte {
				if v, ok := gCache[g]; ok {
					return v
				}
				v := m.Group(g)
				gCache[g] = v
				return v
			})

			if []byte(r) == nil {
				return J([]byte{})
			}
		} else {
			if trim == 0 {
				res = append(res, b[:pos[0]]...)
			} else {
				res = append(res, b[trim:pos[0]]...)
			}
			trim = pos[1]

			gCache := map[int][]byte{}
			r := rep(func(g int) []byte {
				if v, ok := gCache[g]; ok {
					return v
				}
				v := m.Group(g)
				gCache[g] = v
				return v
			})

			if []byte(r) == nil {
				res = append(res, b[trim:]...)
				return J(res)
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return J([]byte{})
	}

	res = append(res, b[trim:]...)

	return J(res)
}

// a copy of the RepFunc method modified to only run once
func RepFuncFirst[T str, J str](str T, re string, rep func(func(int) []byte) J, blank ...bool) J {
	b := []byte(str)
	
	reg := Compile(re)

	// ind := reg.FindAllIndex(b, pcre.UTF8)
	// ind := reg.FindAllIndex(b, 0)
	pos := reg.FindIndex(b, 0)

	res := []byte{}
	trim := 0
	// for _, pos := range ind {
	v := b[pos[0]:pos[1]]
	m := reg.Matcher(v, 0)

	if len(blank) != 0 {
		gCache := map[int][]byte{}
		r := rep(func(g int) []byte {
			if v, ok := gCache[g]; ok {
				return v
			}
			v := m.Group(g)
			gCache[g] = v
			return v
		})

		if []byte(r) == nil {
			return J([]byte{})
		}
	} else {
		if trim == 0 {
			res = append(res, b[:pos[0]]...)
		} else {
			res = append(res, b[trim:pos[0]]...)
		}
		trim = pos[1]

		gCache := map[int][]byte{}
		r := rep(func(g int) []byte {
			if v, ok := gCache[g]; ok {
				return v
			}
			v := m.Group(g)
			gCache[g] = v
			return v
		})

		if []byte(r) == nil {
			res = append(res, b[trim:]...)
			return J(res)
		}

		res = append(res, r...)
	}
	// }

	if len(blank) != 0 {
		return J([]byte{})
	}

	res = append(res, b[trim:]...)

	return J(res)
}

// Replace with a string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
func RepStr[T str, J str](str T, re string, rep J) J {
	reg := Compile(re)

	// return reg.ReplaceAll([]byte(str), []byte(rep), pcre.UTF8)
	return J(reg.ReplaceAll([]byte(str), []byte(rep), 0))
}

// A more complex version of the RepStr method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
func RepStrComplex[T str, J str](str T, re string, rep J, blank ...bool) J {
	b := []byte(str)

	reg := Compile(re)

	repB := []byte(rep)

	// ind := reg.FindAllIndex(b, pcre.UTF8)
	ind := reg.FindAllIndex(b, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := b[pos[0]:pos[1]]
		m := reg.Matcher(v, 0)

		if len(blank) != 0 {
			r := RepFunc(repB, `(\\|)$([0-9]|\{[0-9]+\})`, func(data func(int) []byte) []byte {
				if len(data(1)) != 0 {
					return data(0)
				}
				n := data(2)
				if len(n) > 1 {
					n = n[1:len(n)-1]
				}
				if i, err := strconv.Atoi(string(n)); err == nil {
					return m.Group(i)
				}
				return []byte{}
			})

			if r == nil {
				return J([]byte{})
			}
		} else {
			if trim == 0 {
				res = append(res, b[:pos[0]]...)
			} else {
				res = append(res, b[trim:pos[0]]...)
			}
			trim = pos[1]

			r := RepFunc(repB, `(\\|)\$([0-9]|\{[0-9]+\})`, func(data func(int) []byte) []byte {
				if len(data(1)) != 0 {
					return data(0)
				}
				n := data(2)
				if len(n) > 1 {
					n = n[1:len(n)-1]
				}
				if i, err := strconv.Atoi(string(n)); err == nil {
					return m.Group(i)
				}
				return []byte{}
			})

			if r == nil {
				res = append(res, b[trim:]...)
				return J(res)
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return J([]byte{})
	}

	res = append(res, b[trim:]...)

	return J(res)
}

// Returns true if a string matches a regex
func Match[T str](str T, re string) bool {
	reg := Compile(re)

	// return reg.Match([]byte(str), pcre.UTF8)
	return reg.Match([]byte(str), 0)
}

// Splits a string, and keeps capture groups
// Similar to JavaScript .split(/re/)
func Split[T str](str T, re string) []T {
	b := []byte(str)

	reg := Compile(re)

	ind := reg.FindAllIndex(b, 0)

	res := []T{}
	trim := 0
	for _, pos := range ind {
		v := b[pos[0]:pos[1]]
		m := reg.Matcher(v, 0)

		if trim == 0 {
			res = append(res, T(b[:pos[0]]))
		} else {
			res = append(res, T(b[trim:pos[0]]))
		}
		trim = pos[1]

		for i := 1; i <= m.Groups; i++ {
			g := m.Group(i)
			if len(g) != 0 {
				res = append(res, T(m.Group(i)))
			}
		}
	}

	e := b[trim:]
	if len(e) != 0 {
		res = append(res, T(b[trim:]))
	}

	return res
}

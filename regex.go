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

var regReReplaceQuote pcre.Regexp = pcre.MustCompileJIT(`\\[\\']`, pcre.UTF8, pcre.CONFIG_JIT)
var regReReplaceComment pcre.Regexp = pcre.MustCompileJIT(`\(\?\#.*?\)`, pcre.UTF8, pcre.CONFIG_JIT)

var varType map[string]reflect.Type

var cache *ttlcache.Cache[string, pcre.Regexp] = ttlcache.New[string, pcre.Regexp](2 * time.Hour, 1 * time.Hour)

func init() {
	varType = map[string]reflect.Type{}

	varType["array"] = reflect.TypeOf([]interface{}{})
	varType["map"] = reflect.TypeOf(map[string]interface{}{})

	varType["int"] = reflect.TypeOf(int(0))
	varType["float64"] = reflect.TypeOf(float64(0))
	varType["float32"] = reflect.TypeOf(float32(0))

	varType["string"] = reflect.TypeOf("")
	varType["byteArray"] = reflect.TypeOf([]byte{})
	varType["byte"] = reflect.TypeOf([]byte{0}[0])
	varType["byteArrayArray"] = reflect.TypeOf([][]byte{})

	// int32 returned instead of byte
	varType["int32"] = reflect.TypeOf(' ')
}

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
		case varType["float64"]:
			res = append(res, []byte(strconv.FormatFloat(b.(float64), 'f', -1, 64))...)
		case varType["float32"]:
			res = append(res, []byte(strconv.FormatFloat(float64(b.(float32)), 'f', -1, 32))...)
		}
	}
	return res
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
		// reg := pcre.MustCompileJIT(re, pcre.UTF8, pcre.CONFIG_JIT)
		reg := pcre.MustCompile(re, pcre.UTF8)
		go setCache(re, reg)
		return reg
	}
}

func RepFunc[T interface{string|[]byte}](str T, re string, rep func(func(int) []byte) []byte, blank ...bool) T {
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

			if r == nil {
				return T([]byte{})
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

			if r == nil {
				res = append(res, b[trim:]...)
				return T(res)
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return T([]byte{})
	}

	res = append(res, b[trim:]...)

	return T(res)
}

func RepFuncFirst[T interface{string|[]byte}](str T, re string, rep func(func(int) []byte) []byte, blank ...bool) T {
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

		if r == nil {
			return T([]byte{})
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

		if r == nil {
			res = append(res, b[trim:]...)
			return T(res)
		}

		res = append(res, r...)
	}
	// }

	if len(blank) != 0 {
		return T([]byte{})
	}

	res = append(res, b[trim:]...)

	return T(res)
}

func RepStr[T interface{string|[]byte}](str T, re string, rep T) T {
	reg := Compile(re)

	// return reg.ReplaceAll([]byte(str), []byte(rep), pcre.UTF8)
	return T(reg.ReplaceAll([]byte(str), []byte(rep), 0))
}

func Match[T interface{string|[]byte}](str T, re string) bool {
	reg := Compile(re)

	// return reg.Match([]byte(str), pcre.UTF8)
	return reg.Match([]byte(str), 0)
}

func Split[T interface{string|[]byte}](str T, re string) []T {
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

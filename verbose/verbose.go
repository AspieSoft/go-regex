package regex

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-regex/v5/common"
	"github.com/AspieSoft/go-syncterval"
	"github.com/AspieSoft/go-ttlcache"
	"github.com/GRbit/go-pcre"
	"github.com/pbnjay/memory"
)

type Regexp struct {
	RE *pcre.Regexp
}

var regReReplaceQuote *pcre.Regexp
var regReReplaceComment *pcre.Regexp
var regReReplaceParam *pcre.Regexp
var regValid *pcre.Regexp

var regReReplaceQuoteRE2 *regexp.Regexp
var regReReplaceCommentRE2 *regexp.Regexp
var regReReplaceParamRE2 *regexp.Regexp

var regComplexSel interface{}
var regEscape interface{}

var regParamIndexCache *ttlcache.Cache[string, *pcre.Regexp] = ttlcache.New[string, *pcre.Regexp](2 * time.Hour, 1 * time.Hour)
var cache *ttlcache.Cache[string, *Regexp] = ttlcache.New[string, *Regexp](2 * time.Hour, 1 * time.Hour)

func init() {
	if re, err := pcre.CompileJIT(`\\[\\']`,  pcre.UTF8, pcre.CONFIG_JIT); err == nil {
		regReReplaceQuote = &re
	}else{
		regReReplaceQuoteRE2 = regexp.MustCompile(`\\[\\']`)
	}

	if re, err := pcre.CompileJIT(`\(\?\#.*?\)`,  pcre.UTF8, pcre.CONFIG_JIT); err == nil {
		regReReplaceComment = &re
	}else{
		regReReplaceCommentRE2 = regexp.MustCompile(`\(\?\#.*?\)`)
	}

	if re, err := pcre.CompileJIT(`(?<!\\)(%\{[0-9]+\}|%[0-9])`,  pcre.UTF8, pcre.CONFIG_JIT); err == nil {
		regReReplaceParam = &re
	}else{
		regReReplaceParamRE2 = regexp.MustCompile(`(\\|)(%\{[0-9]+\}|%[0-9])`)
	}

	if re, err := pcre.CompileJIT(`^((?:\(\?[\w]+\)|)(?:(?:[^?+*{}()[\]\\|]+|\\.|\[(?:\^?\\.|\^[^\\]|[^\\^])(?:[^\]\\]+|\\.)*\]|\((?:\?[:=!]|\?<[=!]|\?>)?(?1)??\)|\(\?(?:R|[+-]?\d+)\))(?:(?:[?+*]|\{\d+(?:,\d*)?\})[?+]?)?|\|)*)$`,  pcre.UTF8, pcre.CONFIG_JIT); err == nil {
		regValid = &re
	}

	if re, err := CompileTry(`(\\|)\$([0-9]|\{[0-9]+\})`); err == nil {
		regComplexSel = re
	}else{
		regComplexSel = re2Comp(`(\\|)\$([0-9]|\{[0-9]+\})`)
	}

	if re, err := CompileTry(`[\\\^\$\.\|\?\*\+\(\)\[\]\{\}\%]`); err == nil {
		regEscape = re
	}else{
		regEscape = re2Comp(`[\\\^\$\.\|\?\*\+\(\)\[\]\{\}\%]`)
	}

	go func(){
		// clear cache items older than 10 minutes if there are only 200MB of free memory
		syncterval.New(10 * time.Second, func() {
			if common.FormatMemoryUsage(memory.FreeMemory()) < 200 {
				cache.ClearEarly(10 * time.Minute)
				regParamIndexCache.ClearEarly(30 * time.Minute)
			}
		})
	}()
}

func setCache(re string, reg *Regexp) {
	cache.Set(re, reg)
}

func getCache(re string) (*Regexp, bool) {
	if val, ok := cache.Get(re); ok {
		return val, true
	}

	return &Regexp{}, false
}

// Compile compiles a regular expression and store it in the cache
func Compile(re string, params ...string) *Regexp {
	if strings.Contains(re, `\'`) {
		r := []byte(re)

		var ind [][]int
		if regReReplaceQuote != nil {
			ind = regReReplaceQuote.FindAllIndex(r, 0)
		}else{
			ind = regReReplaceQuoteRE2.FindAllIndex(r, -1)
		}

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
		if regReReplaceComment != nil {
			re = regReReplaceComment.ReplaceAllString(re, ``, 0)
		}else{
			re = regReReplaceCommentRE2.ReplaceAllString(re, ``)
		}
	}

	for i, v := range params {
		var pRe string
		ind := strconv.Itoa(i+1)
		if len(ind) == 1 {
			pRe = `(?<!\\)(%\{`+ind+`\}|%`+ind+`)`
		} else {
			pRe = `(?<!\\)(%\{`+ind+`\})`
		}

		var pReC pcre.Regexp
		if cache, ok := regParamIndexCache.Get(pRe); ok {
			pReC = *cache
		}else{
			pReC = pcre.MustCompileJIT(pRe, pcre.UTF8, pcre.CONFIG_JIT)
		}

		re = pReC.ReplaceAllString(re, Escape(v), 0)
	}

	if regReReplaceParam != nil {
		re = regReReplaceParam.ReplaceAllString(re, ``, 0)
	}else{
		re = string(regReReplaceParamRE2.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
			if len(b) != 0 && b[0] == '\\' {
				return b
			}
			return []byte{}
		}))
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

		compRe := Regexp{RE: &reg}

		go setCache(re, &compRe)
		return &compRe
	}
}

// CompileTry tries to compile or returns an error
func CompileTry(re string, params ...string) (*Regexp, error) {
	if strings.Contains(re, `\'`) {
		r := []byte(re)

		var ind [][]int
		if regReReplaceQuote != nil {
			ind = regReReplaceQuote.FindAllIndex(r, 0)
		}else{
			ind = regReReplaceQuoteRE2.FindAllIndex(r, -1)
		}

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
		if regReReplaceComment != nil {
			re = regReReplaceComment.ReplaceAllString(re, ``, 0)
		}else{
			re = regReReplaceCommentRE2.ReplaceAllString(re, ``)
		}
	}

	for i, v := range params {
		var pRe string
		ind := strconv.Itoa(i+1)
		if len(ind) == 1 {
			pRe = `(?<!\\)(%\{`+ind+`\}|%`+ind+`)`
		} else {
			pRe = `(?<!\\)(%\{`+ind+`\})`
		}

		var pReC pcre.Regexp
		if cache, ok := regParamIndexCache.Get(pRe); ok {
			pReC = *cache
		}else{
			var err error
			pReC, err = pcre.CompileJIT(pRe, pcre.UTF8, pcre.CONFIG_JIT)
			if err != nil {
				return &Regexp{}, err
			}
		}

		re = pReC.ReplaceAllString(re, Escape(v), 0)
	}

	if regReReplaceParam != nil {
		re = regReReplaceParam.ReplaceAllString(re, ``, 0)
	}else{
		re = string(regReReplaceParamRE2.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
			if len(b) != 0 && b[0] == '\\' {
				return b
			}
			return []byte{}
		}))
	}

	if val, ok := getCache(re); ok {
		return val, nil
	} else {
		reg, err := pcre.Compile(re, pcre.UTF8)
		if err != nil {
			return &Regexp{}, err
		}

		compRe := Regexp{RE: &reg}

		go setCache(re, &compRe)
		return &compRe, nil
	}
}

// ReplaceFunc replaces a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
func (reg *Regexp) ReplaceFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

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
				return []byte{}
			}
		} else {
			if trim == 0 {
				res = append(res, str[:pos[0]]...)
			} else {
				res = append(res, str[trim:pos[0]]...)
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
				res = append(res, str[trim:]...)
				return res
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return []byte{}
	}

	res = append(res, str[trim:]...)

	return res
}

// ReplaceString replaces a string with another string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
func (reg *Regexp) ReplaceString(str []byte, rep []byte) []byte {
	return reg.RE.ReplaceAll(str, rep, 0)
}

// ReplaceStringComplex is a more complex version of the RepStr method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
func (reg *Regexp) ReplaceStringComplex(str []byte, rep []byte) []byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

		if trim == 0 {
			res = append(res, str[:pos[0]]...)
		} else {
			res = append(res, str[trim:pos[0]]...)
		}
		trim = pos[1]

		r := []byte{}
		if reg, ok := regComplexSel.(*Regexp); ok {
			r = reg.ReplaceFunc(rep, func(data func(int) []byte) []byte {
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
		}else if reg, ok := regComplexSel.(*RegexpRE2); ok {
			r = reg.RepFunc(rep, func(data func(int) []byte) []byte {
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
		}


		if r == nil {
			res = append(res, str[trim:]...)
			return res
		}

		res = append(res, r...)
		
	}

	res = append(res, str[trim:]...)

	return res
}

// Match returns true if a []byte matches a regex
func (reg *Regexp) Match(str []byte) bool {
	return reg.RE.Match(str, 0)
}

// Split splits a string, and keeps capture groups
// Similar to JavaScript .split(/re/)
func (reg *Regexp) Split(str []byte) [][]byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := [][]byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

		if trim == 0 {
			res = append(res, str[:pos[0]])
		} else {
			res = append(res, str[trim:pos[0]])
		}
		trim = pos[1]

		for i := 1; i <= m.Groups; i++ {
			g := m.Group(i)
			if len(g) != 0 {
				res = append(res, m.Group(i))
			}
		}
	}

	e := str[trim:]
	if len(e) != 0 {
		res = append(res, str[trim:])
	}

	return res
}


// IsValid will return true if a regex is valid and can compile
func IsValid(str []byte) bool {
	if regValid == nil || regValid.Match(str, 0) {
		if _, err := pcre.CompileJIT(string(str), pcre.UTF8, pcre.CONFIG_JIT); err == nil {
			return true
		}
	}
	return false
}

// Escape will escape regex special chars
func Escape(re string) string {
	if reg, ok := regEscape.(*Regexp); ok {
		return string(reg.ReplaceStringComplex([]byte(re), []byte(`\$1`)))
	}else if reg, ok := regEscape.(*RegexpRE2); ok {
		return string(reg.RepStrComp([]byte(re), []byte(`\$1`)))
	}

	return ""
}


// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	return common.JoinBytes(bytes...)
}

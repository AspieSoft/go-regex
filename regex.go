package regex

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/AspieSoft/go-regex/v8/common"
	"github.com/AspieSoft/go-syncterval"
	"github.com/AspieSoft/go-ttlcache"
	"github.com/GRbit/go-pcre"
	"github.com/pbnjay/memory"
)

type PCRE pcre.Regexp
type RE2 *regexp.Regexp

type Regexp struct {
	RE pcre.Regexp
	len int64
}

type bgPart struct {
	ref []byte
	b []byte
}

var regCompCommentAndChars *regexp.Regexp = regexp.MustCompile(`(\\|)\(\?#.*?\)|%!|!%|\\[\\']`)
var regCompParam *regexp.Regexp = regexp.MustCompile(`(\\|)%(\{[0-9]+\}|[0-9])`)
var regCompBG *regexp.Regexp = regexp.MustCompile(`\[^?(\\[\\\]]|[^\]])+\]`)
var regCompBGRefChar *regexp.Regexp = regexp.MustCompile(`%!|!%`)
var regCompBGRef *regexp.Regexp = regexp.MustCompile(`%!([0-9]+|o|c)!%`)

var regComplexSel *Regexp
var regEscape *Regexp

var cache *ttlcache.Cache[string, *Regexp] = ttlcache.New[string, *Regexp](2 * time.Hour, 1 * time.Hour)
var compCache *ttlcache.Cache[string, []byte] = ttlcache.New[string, []byte](2 * time.Hour, 1 * time.Hour)

func init() {
	regComplexSel = Comp(`(\\|)\$([0-9]|\{[0-9]+\})`)
	regEscape = Comp(`[\\\^\$\.\|\?\*\+\(\)\[\]\{\}\%]`)

	go func(){
		// clear cache items older than 10 minutes if there are only 200MB of free memory
		syncterval.New(10 * time.Second, func() {
			if common.FormatMemoryUsage(memory.FreeMemory()) < 200 {
				cache.ClearEarly(10 * time.Minute)
				compCache.ClearEarly(5 * time.Minute)
			}
		})
	}()
}

// this method compiles the RE string to add more functionality to it
func compRE(re string, params []string) string {
	if val, ok := compCache.Get(re); ok {
		return string(regCompParam.ReplaceAllFunc(val, func(b []byte) []byte {
			if b[1] == '{' && b[len(b)-1] == '}' {
				b = b[2:len(b)-1]
			}else{
				b = b[1:]
			}
	
			if n, e := strconv.Atoi(string(b)); e == nil && n > 0 && n <= len(params) {
				return []byte(Escape(params[n-1]))
			}
			return []byte{}
		}))
	}
	
	reB := []byte(re)

	reB = regCompCommentAndChars.ReplaceAllFunc(reB, func(b []byte) []byte {
		if bytes.Equal(b, []byte("%!")) {
			return []byte("%!o!%")
		}else if bytes.Equal(b, []byte("!%")) {
			return []byte("%!c!%")
		}else if b[0] == '\\' {
			if b[1] == '\'' {
				return []byte{'`'}
			}
			return b
		}
		return []byte{}
	})

	bgList := [][]byte{}
	reB = regCompBG.ReplaceAllFunc(reB, func(b []byte) []byte {
		bgList = append(bgList, b)
		return common.JoinBytes('%', '!', len(bgList)-1, '!', '%')
	})

	for ind, bgItem := range bgList {
		charS := []byte{'['}
		if bgItem[1] == '^' {
			bgItem = bgItem[2:len(bgItem)-1]
			charS = append(charS, '^')
		}else{
			bgItem = bgItem[1:len(bgItem)-1]
		}

		newBG := []bgPart{}
		for i := 0; i < len(bgItem); i++ {
			if i+1 < len(bgItem) {
				if bgItem[i] == '\\' {
					newBG = append(newBG, bgPart{ref: []byte{bgItem[i+1]}, b: []byte{bgItem[i], bgItem[i+1]}})
					i++
					continue
				}else if bgItem[i+1] == '-' && i+2 < len(bgItem) {
					newBG = append(newBG, bgPart{ref: []byte{bgItem[i], bgItem[i+2]}, b: []byte{bgItem[i], bgItem[i+1], bgItem[i+2]}})
					i += 2
					continue
				}
			}
			newBG = append(newBG, bgPart{ref: []byte{bgItem[i]}, b: []byte{bgItem[i]}})
		}

		sort.Slice(newBG, func(i, j int) bool {
			if len(newBG[i].ref) > len(newBG[j].ref) {
				return true
			}else if len(newBG[i].ref) < len(newBG[j].ref) {
				return false
			}

			for k := 0; k < len(newBG[i].ref); k++ {
				if newBG[i].ref[k] < newBG[j].ref[k] {
					return true
				}else if newBG[i].ref[k] > newBG[j].ref[k] {
					return false
				}
			}

			return false
		})

		bgItem = charS
		for i := 0; i < len(newBG); i++ {
			bgItem = append(bgItem, newBG[i].b...)
		}
		bgItem = append(bgItem, ']')

		bgList[ind] = bgItem
	}

	reB = regCompBGRef.ReplaceAllFunc(reB, func(b []byte) []byte {
		b = b[2:len(b)-2]

		if b[0] == 'o' {
			return []byte(`%!`)
		}else if b[0] == 'c' {
			return []byte(`!%`)
		}

		if n, e := strconv.Atoi(string(b)); e == nil && n < len(bgList) {
			return bgList[n]
		}
		return []byte{}
	})

	compCache.Set(re, reB)

	return string(regCompParam.ReplaceAllFunc(reB, func(b []byte) []byte {
		if b[1] == '{' && b[len(b)-1] == '}' {
			b = b[2:len(b)-1]
		}else{
			b = b[1:]
		}

		if n, e := strconv.Atoi(string(b)); e == nil && n > 0 && n <= len(params) {
			return []byte(Escape(params[n-1]))
		}
		return []byte{}
	}))
}


//* regex compile methods

// Comp compiles a regular expression and store it in the cache
func Comp(re string, params ...string) *Regexp {
	re = compRE(re, params)

	if val, ok := cache.Get(re); ok {
		return val
	} else {
		reg := pcre.MustCompile(re, pcre.UTF8)

		// commented below methods compiled 10000 times in 0.1s (above method being used finished in half of that time)
		// reg := pcre.MustCompileParse(re)
		// reg := pcre.MustCompileJIT(re, pcre.UTF8, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.EXTRA, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.JAVASCRIPT_COMPAT, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileParseJIT(re, pcre.STUDY_JIT_COMPILE)

		compRe := Regexp{RE: reg, len: int64(len(re))}

		cache.Set(re, &compRe)
		return &compRe
	}
}

// CompTry tries to compile or returns an error
func CompTry(re string, params ...string) (*Regexp, error) {
	re = compRE(re, params)

	if val, ok := cache.Get(re); ok {
		return val, nil
	} else {
		reg, err := pcre.Compile(re, pcre.UTF8)
		if err != nil {
			return &Regexp{}, err
		}

		// commented below methods compiled 10000 times in 0.1s (above method being used finished in half of that time)
		// reg := pcre.MustCompileParse(re)
		// reg := pcre.MustCompileJIT(re, pcre.UTF8, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.EXTRA, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileJIT(re, pcre.JAVASCRIPT_COMPAT, pcre.STUDY_JIT_COMPILE)
		// reg := pcre.MustCompileParseJIT(re, pcre.STUDY_JIT_COMPILE)

		compRe := Regexp{RE: reg, len: int64(len(re))}

		cache.Set(re, &compRe)
		return &compRe, nil
	}
}


//* regex methods

// RepFunc replaces a string with the result of a function
//
// similar to JavaScript .replace(/re/, function(data){})
func (reg *Regexp) RepFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.NewMatcher(v, 0)

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

// RepStrLit replaces a string with another string
//
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
func (reg *Regexp) RepStrLit(str []byte, rep []byte) []byte {
	return reg.RE.ReplaceAll(str, rep, 0)
}

// RepStr is a more complex version of the RepStrLit method
//
// this function will replace things in the result like $1 with your capture groups
//
// use $0 to use the full regex capture group
//
// use ${123} to use numbers with more than one digit
func (reg *Regexp) RepStr(str []byte, rep []byte) []byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.NewMatcher(v, 0)

		if trim == 0 {
			res = append(res, str[:pos[0]]...)
		} else {
			res = append(res, str[trim:pos[0]]...)
		}
		trim = pos[1]

		r := regComplexSel.RepFunc(rep, func(data func(int) []byte) []byte {
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
	return reg.RE.MatchWFlags(str, 0)
}

// Split splits a string, and keeps capture groups
//
// Similar to JavaScript .split(/re/)
func (reg *Regexp) Split(str []byte) [][]byte {
	ind := reg.RE.FindAllIndex(str, 0)

	res := [][]byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.NewMatcher(v, 0)

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


//* other regex methods

// Escape will escape regex special chars
func Escape(re string) string {
	return string(regEscape.RepStr([]byte(re), []byte(`\$1`)))
}

// IsValid will return true if a regex is valid and can be compiled by this module
func IsValid(re string) bool {
	re = compRE(re, []string{})
	if _, err := pcre.Compile(re, pcre.UTF8); err == nil {
		return true
	}
	return false
}

// IsValidPCRE will return true if a regex is valid and can be compiled by the PCRE module
func IsValidPCRE(re string) bool {
	if _, err := pcre.Compile(re, pcre.UTF8); err == nil {
		return true
	}
	return false
}

// IsValidRE2 will return true if a regex is valid and can be compiled by the builtin RE2 module
func IsValidRE2(re string) bool {
	if _, err := regexp.Compile(re); err == nil {
		return true
	}
	return false
}


// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	return common.JoinBytes(bytes...)
}


// RepFileStr replaces a regex match with a new []byte in a file
//
// @all: if true, will replace all text matching @re,
// if false, will only replace the first occurrence
func (reg *Regexp) RepFileStr(name string, rep []byte, all bool, maxReSize ...int64) error {
	stat, err := os.Stat(name)
	if err != nil || stat.IsDir() {
		return err
	}

	file, err := os.OpenFile(name, os.O_RDWR, stat.Mode().Perm())
	if err != nil {
		return err
	}
	defer file.Close()

	var found bool

	l := int64(reg.len * 10)
	if l < 1024 {
		l = 1024
	}
	for _, maxRe := range maxReSize {
		if l < maxRe {
			l = maxRe
		}
	}

	i := int64(0)

	buf := make([]byte, l)
	size, err := file.ReadAt(buf, i)
	buf = buf[:size]
	for err == nil {
		if reg.Match(buf) {
			found = true

			repRes := reg.RepStr(buf, rep)

			rl := int64(len(repRes))
			if rl == l {
				file.WriteAt(repRes, i)
				file.Sync()
			}else if rl < l {
				file.WriteAt(repRes, i)
				rl = l - rl

				j := i+l

				b := make([]byte, 1024)
				s, e := file.ReadAt(b, j)
				b = b[:s]

				for e == nil {
					file.WriteAt(b, j-rl)
					j += 1024
					b = make([]byte, 1024)
					s, e = file.ReadAt(b, j)
					b = b[:s]
				}

				if s != 0 {
					file.WriteAt(b, j-rl)
					j += int64(s)
				}

				file.Truncate(j-rl)
				file.Sync()
			}else if rl > l {
				rl -= l

				dif := int64(1024)
				if rl > dif {
					dif = rl
				}

				j := i+l

				b := make([]byte, dif)
				s, e := file.ReadAt(b, j)
				bw := b[:s]

				file.WriteAt(repRes, i)
				j += rl

				for e == nil {
					b = make([]byte, dif)
					s, e = file.ReadAt(b, j+dif-rl)
				
					file.WriteAt(bw, j)
					bw = b[:s]

					j += dif
				}

				file.WriteAt(bw, j)
				file.Sync()
			}

			if !all {
				file.Sync()
				file.Close()
				return nil
			}

			i += int64(len(repRes))
		}

		i++
		buf = make([]byte, l)
		size, err = file.ReadAt(buf, i)
		buf = buf[:size]
	}

	if reg.Match(buf) {
		found = true

		repRes := reg.RepStr(buf, rep)

		rl := int64(len(repRes))
		if rl == l {
			file.WriteAt(repRes, i)
			file.Sync()
		}else if rl < l {
			file.WriteAt(repRes, i)
			rl = l - rl

			j := i+l

			b := make([]byte, 1024)
			s, e := file.ReadAt(b, j)
			b = b[:s]

			for e == nil {
				file.WriteAt(b, j-rl)
				j += 1024
				b = make([]byte, 1024)
				s, e = file.ReadAt(b, j)
				b = b[:s]
			}

			if s != 0 {
				file.WriteAt(b, j-rl)
				j += int64(s)
			}

			file.Truncate(j-rl)
			file.Sync()
		}else if rl > l {
			rl -= l

			dif := int64(1024)
			if rl > dif {
				dif = rl
			}

			j := i+l

			b := make([]byte, dif)
			s, e := file.ReadAt(b, j)
			bw := b[:s]

			file.WriteAt(repRes, i)
			j += rl

			for e == nil {
				b = make([]byte, dif)
				s, e = file.ReadAt(b, j+dif-rl)
			
				file.WriteAt(bw, j)
				bw = b[:s]

				j += dif
			}

			file.WriteAt(bw, j)
			file.Sync()
		}
	}

	file.Sync()
	file.Close()

	if !found {
		return io.EOF
	}
	return nil
}

// RepFileFunc replaces a regex match with the result of a callback function in a file
//
// @all: if true, will replace all text matching @re,
// if false, will only replace the first occurrence
func (reg *Regexp) RepFileFunc(name string, rep func(data func(int) []byte) []byte, all bool, maxReSize ...int64) error {
	stat, err := os.Stat(name)
	if err != nil || stat.IsDir() {
		return err
	}

	file, err := os.OpenFile(name, os.O_RDWR, stat.Mode().Perm())
	if err != nil {
		return err
	}
	defer file.Close()

	var found bool

	l := int64(reg.len * 10)
	if l < 1024 {
		l = 1024
	}
	for _, maxRe := range maxReSize {
		if l < maxRe {
			l = maxRe
		}
	}

	i := int64(0)

	buf := make([]byte, l)
	size, err := file.ReadAt(buf, i)
	buf = buf[:size]
	for err == nil {
		if reg.Match(buf) {
			found = true

			repRes := reg.RepFunc(buf, rep)

			rl := int64(len(repRes))
			if rl == l {
				file.WriteAt(repRes, i)
				file.Sync()
			}else if rl < l {
				file.WriteAt(repRes, i)
				rl = l - rl

				j := i+l

				b := make([]byte, 1024)
				s, e := file.ReadAt(b, j)
				b = b[:s]

				for e == nil {
					file.WriteAt(b, j-rl)
					j += 1024
					b = make([]byte, 1024)
					s, e = file.ReadAt(b, j)
					b = b[:s]
				}

				if s != 0 {
					file.WriteAt(b, j-rl)
					j += int64(s)
				}

				file.Truncate(j-rl)
				file.Sync()
			}else if rl > l {
				rl -= l

				dif := int64(1024)
				if rl > dif {
					dif = rl
				}

				j := i+l

				b := make([]byte, dif)
				s, e := file.ReadAt(b, j)
				bw := b[:s]

				file.WriteAt(repRes, i)
				j += rl

				for e == nil {
					b = make([]byte, dif)
					s, e = file.ReadAt(b, j+dif-rl)
				
					file.WriteAt(bw, j)
					bw = b[:s]

					j += dif
				}

				file.WriteAt(bw, j)
				file.Sync()
			}

			if !all {
				file.Sync()
				file.Close()
				return nil
			}

			i += int64(len(repRes))
		}

		i++
		buf = make([]byte, l)
		size, err = file.ReadAt(buf, i)
		buf = buf[:size]
	}

	if reg.Match(buf) {
		found = true

		repRes := reg.RepFunc(buf, rep)

		rl := int64(len(repRes))
		if rl == l {
			file.WriteAt(repRes, i)
			file.Sync()
		}else if rl < l {
			file.WriteAt(repRes, i)
			rl = l - rl

			j := i+l

			b := make([]byte, 1024)
			s, e := file.ReadAt(b, j)
			b = b[:s]

			for e == nil {
				file.WriteAt(b, j-rl)
				j += 1024
				b = make([]byte, 1024)
				s, e = file.ReadAt(b, j)
				b = b[:s]
			}

			if s != 0 {
				file.WriteAt(b, j-rl)
				j += int64(s)
			}

			file.Truncate(j-rl)
			file.Sync()
		}else if rl > l {
			rl -= l

			dif := int64(1024)
			if rl > dif {
				dif = rl
			}

			j := i+l

			b := make([]byte, dif)
			s, e := file.ReadAt(b, j)
			bw := b[:s]

			file.WriteAt(repRes, i)
			j += rl

			for e == nil {
				b = make([]byte, dif)
				s, e = file.ReadAt(b, j+dif-rl)
			
				file.WriteAt(bw, j)
				bw = b[:s]

				j += dif
			}

			file.WriteAt(bw, j)
			file.Sync()
		}
	}

	file.Sync()
	file.Close()

	if !found {
		return io.EOF
	}
	return nil
}

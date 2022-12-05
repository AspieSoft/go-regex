package regex

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/AspieSoft/go-syncterval"
	"github.com/AspieSoft/go-ttlcache"
	"github.com/GRbit/go-pcre"
	"github.com/pbnjay/memory"
)

type PCRE = pcre.Regexp

type Regexp struct {
	RE *pcre.Regexp
}

var regReReplaceQuote pcre.Regexp = pcre.MustCompileJIT(`\\[\\']`, pcre.UTF8, pcre.CONFIG_JIT)
var regReReplaceComment pcre.Regexp = pcre.MustCompileJIT(`\(\?\#.*?\)`, pcre.UTF8, pcre.CONFIG_JIT)
var regReReplaceParam pcre.Regexp = pcre.MustCompileJIT(`(?<!\\)(%\{[0-9]+\}|%[0-9])`, pcre.UTF8, pcre.CONFIG_JIT)

var regComplexSel *Regexp

var regParamIndexCache *ttlcache.Cache[string, pcre.Regexp] = ttlcache.New[string, pcre.Regexp](2 * time.Hour, 1 * time.Hour)

var varType map[string]reflect.Type

var cache *ttlcache.Cache[string, *Regexp] = ttlcache.New[string, *Regexp](2 * time.Hour, 1 * time.Hour)

func init() {
	/* man := getLinuxInstaller([]string{`apt-get`, `apt`, `yum`})
	if man == "apt-get" || man == "apt" {
		if !hasLinuxPkg([]string{`libpcre3-dev`}) {
			fmt.Println("Nodice: for pcre regex to work, you may need to install libpcre3-dev as a dependency\nsudo "+man+" install libpcre3-dev")
		}
	}else if man == "yum" {
		if !hasLinuxPkg([]string{`pcre-dev`}) {
			fmt.Println("Nodice: for pcre regex to work, you may need to install pcre-dev as a dependency\nsudo "+man+" install pcre-dev")
		}
	} */

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

	regComplexSel = Compile(`(\\|)\$([0-9]|\{[0-9]+\})`)

	go func(){
		// clear cache items older than 10 minutes if there are only 200MB of free memory
		syncterval.New(10 * time.Second, func() {
			if formatMemoryUsage(memory.FreeMemory()) < 200 {
				cache.ClearEarly(10 * time.Minute)
				regParamIndexCache.ClearEarly(30 * time.Minute)
			}
		})
	}()
}

// AutoInstallLinuxDependencies will automatically detect and install dependencies if missing from debian or arch linux
// debian: libpcre3-dev
// arch: pcre-dev
func AutoInstallLinuxDependencies(){
	man := getLinuxInstaller([]string{`apt-get`, `apt`, `yum`})
	if man == "apt-get" || man == "apt" {
		installLinuxPkg([]string{`libpcre3-dev`}, man)
	}else if man == "yum" {
		installLinuxPkg([]string{`pcre-dev`}, man)
	}
}

// JoinBytes is an easy way to join multiple values into a single []byte
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
/* func Join[T str](bytes ...interface{}) T {
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
} */

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
			pReC = cache
		}else{
			pReC = pcre.MustCompileJIT(pRe, pcre.UTF8, pcre.CONFIG_JIT)
		}

		re = pReC.ReplaceAllString(re, Escape(v), 0)
	}

	re = regReReplaceParam.ReplaceAllString(re, ``, 0)

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

// RepFunc replaces a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
func (reg *Regexp) RepFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
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

// RepFuncRef replace a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
// Uses Pointers For Improved Performance
func (reg *Regexp) RepFuncRef(str *[]byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	ind := reg.RE.FindAllIndex(*str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := (*str)[pos[0]:pos[1]]
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
				res = append(res, (*str)[:pos[0]]...)
			} else {
				res = append(res, (*str)[trim:pos[0]]...)
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
				res = append(res, (*str)[trim:]...)
				return res
			}

			res = append(res, r...)
		}
	}

	if len(blank) != 0 {
		return []byte{}
	}

	res = append(res, (*str)[trim:]...)

	return res
}

// RepFuncFirst is a copy of the RepFunc method modified to only run once
func (reg *Regexp) RepFuncFirst(str []byte, rep func(func(int) []byte) []byte, blank ...bool) []byte {
	pos := reg.RE.FindIndex(str, 0)

	res := []byte{}
	trim := 0
	// for _, pos := range ind {
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
	// }

	if len(blank) != 0 {
		return []byte{}
	}

	res = append(res, str[trim:]...)

	return res
}

// RepStr replaces a string with another string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
func (reg *Regexp) RepStr(str []byte, rep []byte) []byte {
	return reg.RE.ReplaceAll(str, rep, 0)
}

// RepStrRef replaces a string with another string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
// Uses Pointers For Improved Performance
func (reg *Regexp) RepStrRef(str *[]byte, rep []byte) []byte {
	return reg.RE.ReplaceAll(*str, rep, 0)
}

// RepStrRefRes replaces a string with another string
// note: this function is optimized for performance, and the replacement string does not accept replacements like $1
// Uses Pointers For Improved Performance (also on result)
func (reg *Regexp) RepStrRefRes(str *[]byte, rep *[]byte) []byte {
	return reg.RE.ReplaceAll(*str, *rep, 0)
}

// RepStrComplex is a more complex version of the RepStr method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
func (reg *Regexp) RepStrComplex(str []byte, rep []byte) []byte {
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

// RepStrComplexRef is a more complex version of the RepStrRef method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
// Uses Pointers For Improved Performance
func (reg *Regexp) RepStrComplexRef(str *[]byte, rep []byte) []byte {
	ind := reg.RE.FindAllIndex(*str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := (*str)[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

		if trim == 0 {
			res = append(res, (*str)[:pos[0]]...)
		} else {
			res = append(res, (*str)[trim:pos[0]]...)
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
			res = append(res, (*str)[trim:]...)
			return res
		}

		res = append(res, r...)
		
	}

	res = append(res, (*str)[trim:]...)

	return res
}

// RepStrComplexRefRes is a more complex version of the RepStrRefRes method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
// Uses Pointers For Improved Performance (also on result)
func (reg *Regexp) RepStrComplexRefRes(str *[]byte, rep *[]byte) []byte {
	ind := reg.RE.FindAllIndex(*str, 0)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := (*str)[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

		if trim == 0 {
			res = append(res, (*str)[:pos[0]]...)
		} else {
			res = append(res, (*str)[trim:pos[0]]...)
		}
		trim = pos[1]

		r := regComplexSel.RepFunc(*rep, func(data func(int) []byte) []byte {
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
			res = append(res, (*str)[trim:]...)
			return res
		}

		res = append(res, r...)
		
	}

	res = append(res, (*str)[trim:]...)

	return res
}

// Match returns true if a []byte matches a regex
func (reg *Regexp) Match(str []byte) bool {
	return reg.RE.Match(str, 0)
}

// MatchRef returns true if a string matches a regex
// Uses Pointers For Improved Performance
func (reg *Regexp) MatchRef(str *[]byte) bool {
	return reg.RE.Match(*str, 0)
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

// SplitRef splits a string, and keeps capture groups
// Similar to JavaScript .split(/re/)
// Uses Pointers For Improved Performance
func (reg *Regexp) SplitRef(str *[]byte) [][]byte {
	ind := reg.RE.FindAllIndex(*str, 0)

	res := [][]byte{}
	trim := 0
	for _, pos := range ind {
		v := (*str)[pos[0]:pos[1]]
		m := reg.RE.Matcher(v, 0)

		if trim == 0 {
			res = append(res, (*str)[:pos[0]])
		} else {
			res = append(res, (*str)[trim:pos[0]])
		}
		trim = pos[1]

		for i := 1; i <= m.Groups; i++ {
			g := m.Group(i)
			if len(g) != 0 {
				res = append(res, m.Group(i))
			}
		}
	}

	e := (*str)[trim:]
	if len(e) != 0 {
		res = append(res, (*str)[trim:])
	}

	return res
}

var regEscape1 pcre.Regexp = pcre.MustCompileJIT(`\\`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape2 pcre.Regexp = pcre.MustCompileJIT(`\^`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape3 pcre.Regexp = pcre.MustCompileJIT(`\$`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape4 pcre.Regexp = pcre.MustCompileJIT(`\.`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape5 pcre.Regexp = pcre.MustCompileJIT(`\|`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape6 pcre.Regexp = pcre.MustCompileJIT(`\?`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape7 pcre.Regexp = pcre.MustCompileJIT(`\*`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape8 pcre.Regexp = pcre.MustCompileJIT(`\+`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape9 pcre.Regexp = pcre.MustCompileJIT(`\(`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape10 pcre.Regexp = pcre.MustCompileJIT(`\)`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape11 pcre.Regexp = pcre.MustCompileJIT(`\[`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape12 pcre.Regexp = pcre.MustCompileJIT(`\]`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape13 pcre.Regexp = pcre.MustCompileJIT(`\{`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape14 pcre.Regexp = pcre.MustCompileJIT(`\}`, pcre.UTF8, pcre.CONFIG_JIT)
var regEscape15 pcre.Regexp = pcre.MustCompileJIT(`\%`, pcre.UTF8, pcre.CONFIG_JIT)

// Escape will escape regex special chars
func Escape(re string) string {
	re = regEscape1.ReplaceAllString(re, `\\`, 0)
	re = regEscape2.ReplaceAllString(re, `\^`, 0)
	re = regEscape3.ReplaceAllString(re, `\$`, 0)
	re = regEscape4.ReplaceAllString(re, `\.`, 0)
	re = regEscape5.ReplaceAllString(re, `\|`, 0)
	re = regEscape6.ReplaceAllString(re, `\?`, 0)
	re = regEscape7.ReplaceAllString(re, `\*`, 0)
	re = regEscape8.ReplaceAllString(re, `\+`, 0)
	re = regEscape9.ReplaceAllString(re, `\(`, 0)
	re = regEscape10.ReplaceAllString(re, `\)`, 0)
	re = regEscape11.ReplaceAllString(re, `\[`, 0)
	re = regEscape12.ReplaceAllString(re, `\]`, 0)
	re = regEscape13.ReplaceAllString(re, `\{`, 0)
	re = regEscape14.ReplaceAllString(re, `\}`, 0)
	re = regEscape15.ReplaceAllString(re, `\%`, 0)

	return re
}


// formatMemoryUsage converts bytes to megabytes
func formatMemoryUsage(b uint64) float64 {
	return math.Round(float64(b) / 1024 / 1024 * 100) / 100
}


func installLinuxPkg(pkg []string, man ...string){
	if !hasLinuxPkg(pkg) {
		var pkgMan string
		if len(man) != 0 {
			pkgMan = man[0]
		}else{
			pkgMan = getLinuxInstaller([]string{`apt-get`, `apt`, `yum`})
		}

		cmd := exec.Command(`sudo`, append([]string{pkgMan, `install`, `-y`}, pkg...)...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return
		}

		go (func() {
			out := bufio.NewReader(stdout)
			for {
				s, err := out.ReadString('\n')
				if err == nil {
					fmt.Println(s)
				}
			}
		})()

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return
		}

		go (func() {
			out := bufio.NewReader(stderr)
			for {
				s, err := out.ReadString('\n')
				if err == nil {
					fmt.Println(s)
				}
			}
		})()

		cmd.Run()
	}
}

func hasLinuxPkg(pkg []string) bool {
	for _, name := range pkg {
		hasPackage := false
		cmd := exec.Command(`dpkg`, `-s`, name)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return true
		}
		go (func() {
			out := bufio.NewReader(stdout)
			for {
				_, err := out.ReadString('\n')
				if err == nil {
					hasPackage = true
				}
			}
		})()
		for i := 0; i < 3; i++ {
			cmd.Run()
			if hasPackage {
				break
			}
		}
		if !hasPackage {
			return false
		}
	}

	return true
}

func getLinuxInstaller(man []string) string {
	hasInstaller := ""

	for _, m := range man {
		cmd := exec.Command(`dpkg`, `-s`, m)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			continue
		}
		go (func() {
			out := bufio.NewReader(stdout)
			for {
				_, err := out.Peek(1)
				if err == nil {
					hasInstaller = m
				}
			}
		})()

		for i := 0; i < 3; i++ {
			cmd.Run()
			if hasInstaller != "" {
				break
			}
		}

		if hasInstaller != "" {
			break
		}
	}

	return hasInstaller
}

package regex

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

type RegexpRE2 struct {
	RE *regexp.Regexp
}

// Comp compiles a regular expression and store it in the cache
func re2Comp(re string, params ...string) *RegexpRE2 {
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
			pRe = `(\\|)(%\{`+ind+`\}|%`+ind+`)`
		} else {
			pRe = `(\\|)(%\{`+ind+`\})`
		}

		pReC := regexp.MustCompile(pRe)

		re = string(pReC.ReplaceAllFunc([]byte(re), func(b []byte) []byte {
			if len(b) != 0 && b[0] == '\\' {
				return b
			}
			return []byte(Escape(v))
		}))
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

	reg := regexp.MustCompile(re)
	compRe := RegexpRE2{RE: reg}

	return &compRe
}

// RepFunc replaces a string with the result of a function
// similar to JavaScript .replace(/re/, function(data){})
func (reg *RegexpRE2) RepFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	ind := reg.RE.FindAllIndex(str, -1)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.FindAllSubmatch(v, -1)

		if len(blank) != 0 {
			gCache := map[int][]byte{}
			r := rep(func(g int) []byte {
				if v, ok := gCache[g]; ok {
					return v
				}
				v := []byte{}
				if len(m[0]) > g {
					v = m[0][g]
				}
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
				v := []byte{}
				if len(m[0]) > g {
					v = m[0][g]
				}
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

// RepStrComp is a more complex version of the RepStr method
// this function will replace things in the result like $1 with your capture groups
// use $0 to use the full regex capture group
// use ${123} to use numbers with more than one digit
func (reg *RegexpRE2) RepStrComp(str []byte, rep []byte) []byte {
	ind := reg.RE.FindAllIndex(str, -1)

	res := []byte{}
	trim := 0
	for _, pos := range ind {
		v := str[pos[0]:pos[1]]
		m := reg.RE.FindAllSubmatch(v, -1)

		if trim == 0 {
			res = append(res, str[:pos[0]]...)
		} else {
			res = append(res, str[trim:pos[0]]...)
		}
		trim = pos[1]

		r := []byte{}
		if reg, ok := regComplexSel.(*Regexp); ok {
			r = reg.RepFunc(rep, func(data func(int) []byte) []byte {
				if len(data(1)) != 0 {
					return data(0)
				}
				n := data(2)
				if len(n) > 1 {
					n = n[1:len(n)-1]
				}
				if i, err := strconv.Atoi(string(n)); err == nil {
					if len(m[0]) > i {
						return m[0][i]
					}
					return []byte{}
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
					if len(m[0]) > i {
						return m[0][i]
					}
					return []byte{}
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

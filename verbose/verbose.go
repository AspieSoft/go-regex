package regex

import (
	"regexp"

	"github.com/AspieSoft/go-regex/v8"
	"github.com/AspieSoft/go-regex/v8/common"
	"github.com/GRbit/go-pcre"
)

type PCRE pcre.Regexp
type RE2 *regexp.Regexp

type Regexp struct {
	RE pcre.Regexp
	reg *regex.Regexp
}


//* regex compile methods

// Comp compiles a regular expression and store it in the cache
func Compile(re string, params ...string) *Regexp {
	reg := regex.Comp(re, params...)
	return &Regexp{RE: reg.RE, reg: reg}
}

// CompTry tries to compile or returns an error
func CompileTry(re string, params ...string) (*Regexp, error) {
	reg, err := regex.CompTry(re, params...)
	if err != nil {
		return &Regexp{}, err
	}
	return &Regexp{RE: reg.RE}, nil
}


//* regex methods

// RepFunc replaces a string with the result of a function
//
// similar to JavaScript .replace(/re/, function(data){})
func (reg *Regexp) ReplaceFunc(str []byte, rep func(data func(int) []byte) []byte, blank ...bool) []byte {
	return reg.reg.RepFunc(str, rep, blank...)
}

// ReplaceStringLiteral replaces a string with another string
//
// @rep uses the literal string, and does Not use args like $1
func (reg *Regexp) ReplaceStringLiteral(str []byte, rep []byte) []byte {
	return reg.reg.RepStrLit(str, rep)
}

// ReplaceString is a more complex version of the RepStr method
//
// this function will replace things in the result like $1 with your capture groups
//
// use $0 to use the full regex capture group
//
// use ${123} to use numbers with more than one digit
func (reg *Regexp) ReplaceString(str []byte, rep []byte) []byte {
	return reg.reg.RepStr(str, rep)
}

// Match returns true if a []byte matches a regex
func (reg *Regexp) Match(str []byte) bool {
	return reg.reg.Match(str)
}

// Split splits a string, and keeps capture groups
//
// Similar to JavaScript .split(/re/)
func (reg *Regexp) Split(str []byte) [][]byte {
	return reg.reg.Split(str)
}


//* other regex methods

// Escape will escape regex special chars
func Escape(re string) string {
	return regex.Escape(re)
}

// IsValid will return true if a regex is valid and can compile
func IsValid(str []byte) bool {
	if _, err := regexp.Compile(string(str)); err == nil {
		return true
	}
	return false
}

// JoinBytes is an easy way to join multiple values into a single []byte
func JoinBytes(bytes ...interface{}) []byte {
	return common.JoinBytes(bytes...)
}

package regex

import (
	"bytes"
	"errors"
	"testing"
)

func TestCompile(t *testing.T) {
	var check = func(s string){
		re1 := Compile(s)
		re2 := Compile(s)
		if re1.Groups() != re2.Groups() {
			t.Error(s, errors.New("first result does not match cache result"))
		}
	}
	check("")
	check("a(b)")
}

func TestReplace(t *testing.T) {
	var check = func(s string, re, r string, e string){
		res := RepStr([]byte(s), re, []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error(res, errors.New("result does not match expected result"))
		}
	}
	check("this is a test", `(?#a\s+)test`, "", "this is a ")
	check("string with `block` quotes", `\'.*?\'`, "'single'", "string with 'single' quotes")
}


package regex

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestCompile(t *testing.T) {
	var check = func(s string) {
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
	var check = func(s string, re, r string, e string) {
		res := RepStr([]byte(s), re, []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error(res, errors.New("result does not match expected result"))
		}
	}

	check("this is a test", `(?#a\s+)test`, "", "this is a ")
	check("string with `block` quotes", `\'.*?\'`, "'single'", "string with 'single' quotes")
}

func TestReplaceFirst(t *testing.T) {
	var check = func(s string, re string, r func(func(int) []byte) []byte, e string) {
		res := RepFuncFirst([]byte(s), re, r)
		if !bytes.Equal(res, []byte(e)) {
			t.Error(res, errors.New("result does not match expected result"))
		}
	}

	check("test 1 and test 2", `test`, func(data func(int) []byte) []byte {
		return []byte{}
	}, " 1 and test 2")
}

func TestConcurent(t *testing.T) {
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			go (func() {
				res := RepFunc([]byte("test"), `(t)`, func(data func(int) []byte) []byte {
					return data(1)
				})
				_ = res
				time.Sleep(10)
			})()
		}

		time.Sleep(1000000 * 1000) // 1 second
	}
}

func TestCache(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := RepStr([]byte(s), re, []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error(res, errors.New("result does not match expected result"))
		}
	}

	check("this is a test", `\sis\s`, " was ", "this was a test")
	check("this is a test", `\sis\s`, " was ", "this was a test")
}

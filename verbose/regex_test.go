package regex

import (
	"bytes"
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestCompile(t *testing.T) {
	var check = func(s string) {
		re1 := Compile(s)
		re2 := Compile(s)
		if re1.RE.Groups() != re2.RE.Groups() {
			t.Error("[", s, "]\n", errors.New("first result does not match cache result"))
		}
	}

	check("")
	check("a(b)")

	reC := Compile("this is test %1", "a")
	if reC.RE.ReplaceAllString(`this is test a`, `this is test b`, 0) != `this is test b` {
		t.Error(`[this is test %1] [a]`, "\n", errors.New("failed to compile params"))
	}

	re := `test .*`
	reEscaped := Escape(re)
	if re == reEscaped || Compile(reEscaped).Match([]byte(`test 1`)) {
		t.Error("[", reEscaped, "]\n", errors.New("escape function failed"))
	}

	r := Compile(`test %1`, "%2", "a")
	if r.Match([]byte(`test a`)) {
		t.Error(`[test %1] [%2, a]`, "\n", errors.New("escape function failed to escape '%' char"))
	}
}

func TestReplaceStr(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := Compile(re).ReplaceString([]byte(s), []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error("[", string(res), "]\n", errors.New("result does not match expected result"))
		}
	}

	check("this is a test", `(?#a\s+)test`, "", "this is a ")
	check("string with `block` quotes", `\'.*?\'`, "'single'", "string with 'single' quotes")
}

func TestReplaceStrComplex(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := Compile(re).ReplaceStringComplex([]byte(s), []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error("[", string(res), "]\n", errors.New("result does not match expected result"))
		}
	}

	check("this is a Test", `(?i)a (test)`, "some $1", "this is some Test")
	check("I Need Coffee!!!", `Coffee(!*)`, "More Coffee$1", "I Need More Coffee!!!")
}

func TestReplaceFunc(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := Compile(re).ReplaceFunc([]byte(s), func(data func(int) []byte) []byte {
			return JoinBytes(data(1), ' ', r)
		})
		if !bytes.Equal(res, []byte(e)) {
			t.Error("[", string(res), "]\n", errors.New("result does not match expected result"))
		}
	}

	check("this is a new test", `(new) test`, "pizza", "this is a new pizza")
	check("a random string", `(a) random`, "not so random", "a not so random string")
}

func TestConcurrent(t *testing.T) {
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			go (func() {
				res := Compile(`(t)`).ReplaceFunc([]byte("test"), func(data func(int) []byte) []byte {
					return data(1)
				})
				_ = res
				time.Sleep(10 * time.Nanosecond)
			})()
		}

		// time.Sleep(1000000 * 1000) // 1 second
		time.Sleep(1000000 * 100) // 0.1 second
	}
}

func TestCache(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := Compile(re).ReplaceString([]byte(s), []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error("[", string(res), "]\n", errors.New("result does not match expected result"))
		}
	}

	check("this is a test", `\sis\s`, " was ", "this was a test")
	check("this is a test", `\sis\s`, " was ", "this was a test")
}

func TestFlags(t *testing.T) {
	var check = func(s string, re, r string, e string) {
		res := Compile(re).ReplaceString([]byte(s), []byte(r))
		if !bytes.Equal(res, []byte(e)) {
			t.Error("[", string(res), "]\n", errors.New("result does not match expected result"))
		}
	}

	check("this is a\nmultiline text", `(?s)a\s*multiline`, "", "this is  text")
	check("list line 1\nlist line 2\n list line 3", `(?m)^list`, "a list", "a list line 1\na list line 2\n list line 3")
	check("a MultiCase text", `(?i)multicase`, "", "a  text")
	check("a MultiCase text no flag", `multicase`, "", "a MultiCase text no flag")

	// check("a multi\nline text", `multi\s*line`, "", "a multi\nline text")
}

func TestPerformance(t *testing.T) {
	for i := 0; i < 10000; i++ {
		Compile(strconv.Itoa(rand.Int()))
	}
}

func TestValid(t *testing.T) {
	var check = func(re string, e bool) {
		res := IsValid([]byte(re))
		if res != e {
			t.Error("[", string(re), "]\n", errors.New("result does not match expected result"))
		}
	}

	check(`[\w_\-]+`, true)
	check(`[\w_-.]+`, false)
	check(`(?<test>)`, false)
	check(`(?i)test`, true)
}

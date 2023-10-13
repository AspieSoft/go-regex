package regex

import (
	"io"
	"os"
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
	len int64
}


//* regex compile methods

// Comp compiles a regular expression and store it in the cache
func Compile(re string, params ...string) *Regexp {
	reg := regex.Comp(re, params...)
	return &Regexp{RE: reg.RE, reg: reg, len: int64(len(re))}
}

// CompTry tries to compile or returns an error
func CompileTry(re string, params ...string) (*Regexp, error) {
	reg, err := regex.CompTry(re, params...)
	if err != nil {
		return &Regexp{}, err
	}
	return &Regexp{RE: reg.RE, len: int64(len(re))}, nil
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


// ReplaceFileString replaces a regex match with a new []byte in a file
//
// @all: if true, will replace all text matching @re,
// if false, will only replace the first occurrence
func (reg *Regexp) ReplaceFileString(name string, rep []byte, all bool, maxReSize ...int64) error {
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

			repRes := reg.ReplaceString(buf, rep)

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

		repRes := reg.ReplaceString(buf, rep)

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

// ReplaceFileFunc replaces a regex match with the result of a callback function in a file
//
// @all: if true, will replace all text matching @re,
// if false, will only replace the first occurrence
func (reg *Regexp) ReplaceFileFunc(name string, rep func(data func(int) []byte) []byte, all bool, maxReSize ...int64) error {
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

			repRes := reg.ReplaceFunc(buf, rep)

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

		repRes := reg.ReplaceFunc(buf, rep)

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

# Go Regex

[![donation link](https://img.shields.io/badge/buy%20me%20a%20coffee-paypal-blue)](https://paypal.me/shaynejrtaylor?country.x=US&locale.x=en_US)

A High Performance PCRE Regex Package That Uses A Cache.

Simplifies the the go-pcre regex package.
After calling a regex, the compiled output gets cached to improve performance.

This package uses the [go-pcre](https://github.com/GRbit/go-pcre) package for better performance.

## Installation

```shell script

  sudo apt-get install libpcre3-dev
  go get github.com/AspieSoft/go-regex/v3

```

## Usage

```go

import (
  "github.com/AspieSoft/go-regex/v3"
)

// pre compile a regex into the cache
// this method also returns the compiled pcre.Regexp struct
regex.Compile(`re`)

// compile a regex and safely escape user input
regex.Compile(`re %1`, `this will be escaped .*`); // output: this will be escaped \.\*
regex.Compile(`re %1`, `hello \n world`); // output: hello \\n world (note: the \ was escaped, and the n is literal)

// use %n to reference a param
// use %{n} for param indexes with more than 1 digit
regex.Compile(`re %1 and %2 ... %{12}`, `param 1`, `param 2` ..., `param 12`);

// manually escape a string
regex.Escape(`(.*)? \$ \\$ \\\$ regex hack failed`)

// run a replace function (most advanced feature)
regex.RepFunc(myByteArray, regex.Compile(`(?flags)re(capture group)`), func(data func(int) []byte) []byte {
  data(0) // get the string
  data(1) // get the first capture group

  return []byte("")

  // if the last option is true, returning nil will stop the loop early
  return nil
}, true /* optional: if true, will not process a return output */)

// run a simple light replace function
regex.RepStr(myByteArray, regex.Compile(`re`), myReplacementByteArray)

// return a bool if a regex matches a byte array
regex.Match(myByteArray, regex.Compile(`re`))

// split a byte array in a similar way to JavaScript
regex.Split(myByteArray, regex.Compile(`re|(keep this and split like in JavaScript)`))

// a regex string is modified before compiling, to add a few other features
`use \' in place of ` + "`" + ` to make things easier`
`(?#This is a comment in regex)`

// an alias of pcre.Regexp
regex.Regexp


// another helpful function
// this method makes it easier to return results to a regex function
regex.JoinBytes("string", []byte("byte array"), 10, 'c', data(2))

// the above method can be used in place of
append(append(append(append([]byte("string"), []byte("byte array")...), []byte(strconv.Itoa(10))...), 'c'), data(2)...)

```

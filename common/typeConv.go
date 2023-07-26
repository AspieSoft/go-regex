package common

import (
	"reflect"
	"strconv"
)

var varType map[string]reflect.Type

func init(){
	varType = map[string]reflect.Type{}

	varType["[]interface{}"] = reflect.TypeOf([]interface{}{})
	varType["array"] = varType["[]interface{}"]
	varType["[][]byte"] = reflect.TypeOf([][]byte{})
	varType["map[string]interface{}"] = reflect.TypeOf(map[string]interface{}{})
	varType["map"] = varType["map[string]interface{}"]

	varType["int"] = reflect.TypeOf(int(0))
	varType["int64"] = reflect.TypeOf(int64(0))
	varType["float64"] = reflect.TypeOf(float64(0))
	varType["float32"] = reflect.TypeOf(float32(0))

	varType["string"] = reflect.TypeOf("")
	varType["[]byte"] = reflect.TypeOf([]byte{})
	varType["byteArray"] = varType["[]byte"]
	varType["byte"] = reflect.TypeOf([]byte{0}[0])

	// ' ' returned int32 instead of byte
	varType["int32"] = reflect.TypeOf(int32(0))
	varType["rune"] = reflect.TypeOf(rune(0))

	varType["func"] = reflect.TypeOf(func(){})

	varType["bool"] = reflect.TypeOf(false)

	varType["int8"] = reflect.TypeOf(int8(0))
	varType["int16"] = reflect.TypeOf(int16(0))
	
	varType["uint"] = reflect.TypeOf(uint(0))
	varType["uint8"] = reflect.TypeOf(uint8(0))
	varType["uint16"] = reflect.TypeOf(uint16(0))
	varType["uint32"] = reflect.TypeOf(uint32(0))
	varType["uint64"] = reflect.TypeOf(uint64(0))
	varType["uintptr"] = reflect.TypeOf(uintptr(0))

	varType["complex128"] = reflect.TypeOf(complex128(0))
	varType["complex64"] = reflect.TypeOf(complex64(0))

	varType["map[byte]interface{}"] = reflect.TypeOf(map[byte]interface{}{})
	varType["map[rune]interface{}"] = reflect.TypeOf(map[byte]interface{}{})
	varType["map[int]interface{}"] = reflect.TypeOf(map[int]interface{}{})
	varType["map[int64]interface{}"] = reflect.TypeOf(map[int64]interface{}{})
	varType["map[int32]interface{}"] = reflect.TypeOf(map[int32]interface{}{})
	varType["map[float64]interface{}"] = reflect.TypeOf(map[float64]interface{}{})
	varType["map[float32]interface{}"] = reflect.TypeOf(map[float32]interface{}{})

	varType["map[int8]interface{}"] = reflect.TypeOf(map[int8]interface{}{})
	varType["map[int16]interface{}"] = reflect.TypeOf(map[int16]interface{}{})

	varType["map[uint]interface{}"] = reflect.TypeOf(map[uint]interface{}{})
	varType["map[uint8]interface{}"] = reflect.TypeOf(map[uint8]interface{}{})
	varType["map[uint16]interface{}"] = reflect.TypeOf(map[uint16]interface{}{})
	varType["map[uint32]interface{}"] = reflect.TypeOf(map[uint32]interface{}{})
	varType["map[uint64]interface{}"] = reflect.TypeOf(map[uint64]interface{}{})
	varType["map[uintptr]interface{}"] = reflect.TypeOf(map[uintptr]interface{}{})

	varType["map[complex128]interface{}"] = reflect.TypeOf(map[complex128]interface{}{})
	varType["map[complex64]interface{}"] = reflect.TypeOf(map[complex64]interface{}{})

	varType["[]string"] = reflect.TypeOf([]string{})
	varType["[]bool"] = reflect.TypeOf([]bool{})
	varType["[]rune"] = reflect.TypeOf([]bool{})
	varType["[]int"] = reflect.TypeOf([]int{})
	varType["[]int64"] = reflect.TypeOf([]int64{})
	varType["[]int32"] = reflect.TypeOf([]int32{})
	varType["[]float64"] = reflect.TypeOf([]float64{})
	varType["[]float32"] = reflect.TypeOf([]float32{})

	varType["[]int8"] = reflect.TypeOf([]int8{})
	varType["[]int16"] = reflect.TypeOf([]int16{})

	varType["[]uint"] = reflect.TypeOf([]uint{})
	varType["[]uint8"] = reflect.TypeOf([]uint8{})
	varType["[]uint16"] = reflect.TypeOf([]uint16{})
	varType["[]uint32"] = reflect.TypeOf([]uint32{})
	varType["[]uint64"] = reflect.TypeOf([]uint64{})
	varType["[]uintptr"] = reflect.TypeOf([]uintptr{})

	varType["[]complex128"] = reflect.TypeOf([]complex128{})
	varType["[]complex64"] = reflect.TypeOf([]complex64{})
}

// toString converts multiple types to a string|[]byte
//
// accepts: string, []byte, byte, int (and variants), [][]byte, []interface{}
func ToString[T interface{string | []byte}](val interface{}) T {
	switch reflect.TypeOf(val) {
		case varType["string"]:
			return T(val.(string))
		case varType["[]byte"]:
			return T(val.([]byte))
		case varType["byte"]:
			return T([]byte{val.(byte)})
		case varType["int"]:
			return T(strconv.Itoa(val.(int)))
		case varType["int64"]:
			return T(strconv.Itoa(int(val.(int64))))
		case varType["int32"]:
			return T([]byte{byte(val.(int32))})
		case varType["int16"]:
			return T([]byte{byte(val.(int16))})
		case varType["int8"]:
			return T([]byte{byte(val.(int8))})
		case varType["uintptr"]:
			return T(strconv.FormatUint(uint64(val.(uintptr)), 10))
		case varType["uint"]:
			return T(strconv.FormatUint(uint64(val.(uint)), 10))
		case varType["uint64"]:
			return T(strconv.FormatUint(val.(uint64), 10))
		case varType["uint32"]:
			return T(strconv.FormatUint(uint64(val.(uint32)), 10))
		case varType["uint16"]:
			return T(strconv.FormatUint(uint64(val.(uint16)), 10))
		case varType["uint8"]:
			return T(strconv.FormatUint(uint64(val.(uint8)), 10))
		case varType["float64"]:
			return T(strconv.FormatFloat(val.(float64), 'f', -1, 64))
		case varType["float32"]:
			return T(strconv.FormatFloat(float64(val.(float32)), 'f', -1, 32))
		case varType["rune"]:
			return T([]byte{byte(val.(rune))})
		case varType["[]interface{}"]:
			b := make([]byte, len(val.([]interface{})))
			for i, v := range val.([]interface{}) {
				b[i] = byte(ToNumber[int32](v))
			}
			return T(b)
		case varType["[]int"]:
			b := make([]byte, len(val.([]int)))
			for i, v := range val.([]int) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]int64"]:
			b := make([]byte, len(val.([]int64)))
			for i, v := range val.([]int64) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]int32"]:
			b := make([]byte, len(val.([]int32)))
			for i, v := range val.([]int32) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]int16"]:
			b := make([]byte, len(val.([]int16)))
			for i, v := range val.([]int16) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]int8"]:
			b := make([]byte, len(val.([]int8)))
			for i, v := range val.([]int8) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uint"]:
			b := make([]byte, len(val.([]uint)))
			for i, v := range val.([]uint) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uint8"]:
			b := make([]byte, len(val.([]uint8)))
			for i, v := range val.([]uint8) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uint16"]:
			b := make([]byte, len(val.([]uint16)))
			for i, v := range val.([]uint16) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uint32"]:
			b := make([]byte, len(val.([]uint32)))
			for i, v := range val.([]uint32) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uint64"]:
			b := make([]byte, len(val.([]uint64)))
			for i, v := range val.([]uint64) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]uintptr"]:
			b := make([]byte, len(val.([]uintptr)))
			for i, v := range val.([]uintptr) {
				b[i] = byte(v)
			}
			return T(b)
		case varType["[]string"]:
			b := []byte{}
			for _, v := range val.([]string) {
				b = append(b, []byte(v)...)
			}
			return T(b)
		case varType["[][]byte"]:
			b := []byte{}
			for _, v := range val.([][]byte) {
				b = append(b, v...)
			}
			return T(b)
		case varType["[]rune"]:
			b := []byte{}
			for _, v := range val.([]rune) {
				b = append(b, byte(v))
			}
			return T(b)
		default:
			return T("")
	}
}

// toNumber converts multiple types to a number
//
// accepts: int (and variants), string, []byte, byte, bool
func ToNumber[T interface{int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | uintptr | float64 | float32}](val interface{}) T {
	switch reflect.TypeOf(val) {
		case varType["int"]:
			return T(val.(int))
		case varType["int32"]:
			return T(val.(int32))
		case varType["int64"]:
			return T(val.(int64))
		case varType["float64"]:
			return T(val.(float64))
		case varType["float32"]:
			return T(val.(float32))
		case varType["string"]:
			var varT interface{} = T(0)
			if _, ok := varT.(float64); ok {
				if f, err := strconv.ParseFloat(val.(string), 64); err == nil {
					return T(f)
				}
			}else if _, ok := varT.(float32); ok {
				if f, err := strconv.ParseFloat(val.(string), 32); err == nil {
					return T(f)
				}
			}else if i, err := strconv.Atoi(val.(string)); err == nil {
				return T(i)
			}
			return 0
		case varType["[]byte"]:
			if i, err := strconv.Atoi(string(val.([]byte))); err == nil {
				return T(i)
			}
			return 0
		case varType["byte"]:
			if i, err := strconv.Atoi(string(val.(byte))); err == nil {
				return T(i)
			}
			return 0
		case varType["bool"]:
			if val.(bool) == true {
				return 1
			}
			return 0
		case varType["int8"]:
			return T(val.(int8))
		case varType["int16"]:
			return T(val.(int16))
		case varType["uint"]:
			return T(val.(uint))
		case varType["uint8"]:
			return T(val.(uint8))
		case varType["uint16"]:
			return T(val.(uint16))
		case varType["uint32"]:
			return T(val.(uint32))
		case varType["uint64"]:
			return T(val.(uint64))
		case varType["uintptr"]:
			return T(val.(uintptr))
		case varType["rune"]:
			if i, err := strconv.Atoi(string(val.(rune))); err == nil {
				return T(i)
			}
			return 0
		default:
			return 0
	}
}

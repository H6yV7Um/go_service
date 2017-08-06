/*

本模块提供encoding/json之外对json进行编解码的方法
by xuesong3（xuesong3@staff.weibo.com）2016-11-10

本模块只是为了最大程度优化特定部分的代码，只支持特定格式的json对象，在对源代码没有
充分理解之前*不建议*使用本模块。

编码：通过提供layout描述字段和字段类型，直接序列化成字符串
解码：通过有限自动机的方式处理输入流，只提供非常有限的格式解析，只支持最多两层的格式。

 */

package very_simple_json

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"reflect"
	"strings"
)

type State int

type SimpleJsonDecoder struct {
	r     *bufio.Reader
	state State
	err   error
	buf   []byte
}

const (
	stateStart = 1
	stateFinish = 2
	stateObjectStart = 3
	stateObjectKeyStart = 4
	stateObjectKeyEnd = 5
	stateObjectValueStart = 6
	stateObjectValueEnd = 7
	stateArrayStart = 8
)

func NewSimpleJsonDecoder(b []byte) *SimpleJsonDecoder {
	return &SimpleJsonDecoder{
		r:     bufio.NewReader(bytes.NewReader(b)),
		state: stateStart,
		buf:   make([]byte, 10240),  // TODO: exceed max
	}
}

func (d *SimpleJsonDecoder)readByte() byte {
	c, err := d.r.ReadByte()
	if err != nil {
		d.state = stateFinish
		d.err = err
		return 0
	}
	//fmt.Printf("%c", c)
	return c
}

func isSpace(c byte) bool {
	if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
		return true
	} else {
		return false
	}
}

// a simple implement
func isNumber(c byte) bool {
	if (c >= '0' && c <= '9') || c == '+' || c == '.' || c == '-' {
		return true
	} else {
		return false
	}
}

func (d *SimpleJsonDecoder)readByteSkipBlank() byte {
	var c byte = ' '
	for isSpace(c) && d.err == nil {
		c = d.readByte()
	}
	return c
}

func (d *SimpleJsonDecoder)readToken() string {
	i := 0
	var c byte
	for d.err == nil && i < len(d.buf) {
		c = d.readByte()
		if c == '"' {
			break
		}
		d.buf[i] = c
		i++
	}
	if i == 0 {
		//d.fail("invalid key")
		return ""
	} else if d.err == nil {
		return string(d.buf[:i])
	}
	return ""
}

func (d *SimpleJsonDecoder)readNumber() interface{} {
	i := 0
	var c byte
	isFloat := false
	for d.err == nil && i < len(d.buf) {
		c = d.readByte()
		if isSpace(c) {
			continue
		}
		if !isNumber(c) {
			break
		}
		if c == '.' {
			isFloat = true
		}
		d.buf[i] = c
		i++
	}
	//fmt.Printf("i=%v\n", i)
	if i == 0 {
		d.fail("invalid number")
		return 0
	} else if d.err == nil {
		d.r.UnreadByte()  // unread the byte after the number
		s := string(d.buf[:i])
		//fmt.Printf("s=%v\n", s)
		if isFloat {
			n, err := strconv.ParseFloat(s, 0)
			if err != nil {
				d.fail("invalid number")
				return 0
			}
			return n
		} else {
			n, err := strconv.ParseInt(s, 10, 0)
			if err != nil {
				d.fail("invalid number")
				return 0
			}
			return n
		}
		return string(d.buf[:i])
	}
	return 0
}

func (d *SimpleJsonDecoder)expect(expectedByte byte) bool {
	var c byte
	for c != expectedByte && d.err == nil {
		c = d.readByte()
	}
	if d.err != nil {
		d.fail(fmt.Sprintf("expect byte %c", expectedByte))
		return false
	}
	return true
}

func (d *SimpleJsonDecoder)fail(reason string) {
	d.err = errors.New(reason)
	d.state = stateFinish
}

func (d *SimpleJsonDecoder)next(state State) {
	if d.err == nil {
		d.state = state
	}
}

func (d *SimpleJsonDecoder)Decode() (interface{}, error) {
	var c byte
	var result interface{}
	var resultArray []interface{}
	var p *interface{}
	var o map[string]interface{}
	jsonType := "object"
	key := ""
	value := ""
	for d.state != stateFinish {
		switch d.state {
		case stateStart:
			c = d.readByteSkipBlank()
			if c == '{' {
				d.next(stateObjectStart)
				p = &result
			} else if c == '[' {
				d.next(stateArrayStart)
				result = []interface{}{}
				jsonType = "array"
				p = &result
			} else {
				//d.fail(fmt.Sprintf("invalid byte %v, expect '{' or '['", c))
				d.err = nil
				d.next(stateFinish)
			}
		case stateObjectStart:
			o = make(map[string]interface{})
			c = d.readByteSkipBlank()
			if c == '}' {
				
			} else if c == '"' {
				d.next(stateObjectKeyStart)
			} else {
				d.fail(fmt.Sprintf("invalid byte %v, expect '}' or '\"'", c))
			}
		case stateObjectKeyStart:
			key = d.readToken()
			if key == "" {
				d.fail(fmt.Sprintf("empty key"))
			} else {
				if(d.expect(':')) {
					d.next(stateObjectValueStart)
				}
			}
		case stateObjectValueStart:
			c = d.readByteSkipBlank()
			if c == '"' {
				value = d.readToken()
				o[key] = value
			} else {
				d.r.UnreadByte()
				number := d.readNumber()
				o[key] = number
			}
			d.next(stateObjectValueEnd)
		case stateObjectValueEnd:
			*p = o
			c = d.readByteSkipBlank()
			if c == ',' {
				if(d.expect('"')) {
					d.next(stateObjectKeyStart)
				}
			} else if c == '}' {
				if jsonType == "object" {
					d.next(stateFinish)
				} else {
					resultArray = append(resultArray, o)
					if(d.expect(',') && d.expect('{')) {
						d.next(stateObjectStart)
					} else {
						d.err = nil
						d.next(stateFinish)
					}
				}
			} else {
				d.fail(fmt.Sprintf("invalid byte %v, expect ',' or '}'", c))
			}
		case stateObjectKeyEnd:
		case stateArrayStart:
			c = d.readByteSkipBlank()
			if c == '{' {
				d.next(stateObjectStart)
				p = &result
			} else if c == ']' {
				d.next(stateFinish)
			} else {
				d.fail(fmt.Sprintf("invalid byte %v, expect '{' or '['", c))
			}
		}
	}
	if jsonType == "object" {
		return result, d.err
	} else {
		return resultArray, d.err
	}
}

type Layout struct {
	field	string
	typ		reflect.Kind
}

func ToJsonObject(o map[string]interface{}, layout map[string]interface{}, buf *bytes.Buffer) error {
	buf.WriteByte('{')
	fields := []string{}
	for k, v := range(layout) {
		fieldValue, exists := o[k]
		if !exists {
			continue
		}
		switch typ := v.(type) {
		case reflect.Kind:
			switch typ {
			case reflect.Int64:
				fields = append(fields, fmt.Sprintf("\"%s\": %v", k, fieldValue))
			case reflect.Float64:
				fields = append(fields, fmt.Sprintf("\"%s\": %v", k, fieldValue))
			case reflect.String:
				fields = append(fields, fmt.Sprintf("\"%s\": \"%v\"", k, fieldValue))
			default:
				return errors.New(fmt.Sprintf("invalid type:%#v", typ))
			}
		case map[string]interface{}:
		default:
			return errors.New(fmt.Sprintf("invalid type:%#v", typ))
		}
	}
	buf.WriteString(strings.Join(fields, ", "))
	buf.WriteByte('}')
	return nil
}

func ToJson(i map[string]interface{}, layout map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	err := ToJsonObject(i, layout, &buf)
	return buf.String(), err
}

func ToJsonArray(v []map[string]interface{}, layout map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	var err error
	buf.WriteByte('[')
	for i, o := range(v) {
		err = ToJsonObject(o, layout, &buf)
		if err != nil {
			return "", err
		}
		if i != len(v)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(']')
	return buf.String(), err
}
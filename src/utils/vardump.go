package utils

import (
	"reflect"
	"strings"
	"fmt"
)

const(
	indent = "  "
)

type VarDumpHelper struct {
	buf 	string
	last 	interface{}
}

func (vd *VarDumpHelper)isNewline(i interface{}) bool {
	s, ok := i.(string)
	return ok && s == "\n"
}

func (vd *VarDumpHelper)Print(str ...interface{}) {
	for _, s := range str {
		if vd.isNewline(s) && vd.isNewline(vd.last) {
			continue
		}
		fmt.Print(s)
		vd.last = s
	}
}

func (vd *VarDumpHelper)PrintStruct(t reflect.Type, v reflect.Value, pc int) {
	vd.Print("\n")
	for i := 0; i < t.NumField(); i++ {
		vd.Print(strings.Repeat(indent, pc), t.Field(i).Name, ": ")
		value := v.Field(i)
		vd.PrintVar(value.Interface(), pc+2, true, reflect.Struct)
		if i != t.NumField() - 1 {
			vd.Print(",\n")
		}
	}
}

func (vd *VarDumpHelper)PrintArraySlice(v reflect.Value, pc int) {
	vd.Print("[")
	for j := 0; j < v.Len(); j++ {
		vd.PrintVar(v.Index(j).Interface(), 0, false, reflect.Array)
		if j != v.Len() - 1 {
			vd.Print(",")
		}
	}
	vd.Print("]")
}

func (vd *VarDumpHelper)PrintMap(v reflect.Value, pc int) {
	vd.Print("\n", strings.Repeat(indent, pc), "{\n")
	for _, k := range v.MapKeys() {
		vd.PrintVar(k.Interface(), pc+2, false, reflect.Map)
		fmt.Print(": ")
		vd.PrintVar(v.MapIndex(k).Interface(), pc+2, true, reflect.Map)
		vd.Print(",\n")
	}
	vd.Print(strings.Repeat(indent, pc), "}")
}

func (vd *VarDumpHelper)PrintVar(i interface{}, ident int, inline bool, parentType reflect.Kind) {
	t := reflect.TypeOf(i)
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = reflect.ValueOf(i).Elem()
		t = v.Type()
	}
	switch v.Kind() {
	case reflect.Array:
		vd.PrintArraySlice(v, ident)
	case reflect.Chan:
		vd.Print("Chan\n")
	case reflect.Func:
		vd.Print("Func\n")
	case reflect.Interface:
		vd.Print("Interface\n")
	case reflect.Map:
		if parentType == reflect.Struct {
			ident -= 2
		}
		vd.PrintMap(v, ident)
	case reflect.Slice:
		vd.PrintArraySlice(v, ident)
	case reflect.Struct:
		vd.PrintStruct(t, v, ident)
	case reflect.UnsafePointer:
		vd.Print("UnsafePointer\n")
	case reflect.String:
		if inline {
			vd.Print("\"", v.Interface(), "\"")
		} else {
			vd.Print(strings.Repeat(indent, ident), "\"", v.Interface(), "\"")
		}
	default:
		if inline {
			vd.Print(v.Interface())
		} else {
			vd.Print(strings.Repeat(indent, ident), v.Interface())
		}
	}
}

func VarDump(i interface{}) {
	helper := &VarDumpHelper{}
	helper.PrintVar(i, 0, false, reflect.String)
	helper.Print("\n")
}

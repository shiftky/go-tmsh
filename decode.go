package tmsh

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Unmarshal parses the 'field-fmt' formatted data and stores the result in the value pointed to by out.
func Unmarshal(data string, out interface{}) error {
	data = strings.Trim(data, "\n")

	l := Lexer{s: newScanner(data)}
	if yyParse(&l) != 0 {
		return fmt.Errorf("Parse error")
	}

	v := reflect.ValueOf(out)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}

	unmarshal(l.result, v)

	return nil
}

func unmarshal(n *node, out reflect.Value) {
	switch n.kind {
	case ltmNode:
		// Store LTM components info into embedded struct
		for i := 0; i < out.NumField(); i++ {
			fieldValue := out.Field(i)
			if fieldValue.Kind() == reflect.Struct {
				unmarshal(n.children[0], fieldValue)
			}
		}

		// Store LTM components info into each struct fields
		unmarshal(n.children[0], out)

		// Set name field
		if f, ok := lookupField("name", out); ok {
			if f.Kind() == reflect.String && f.String() == "" {
				f.SetString(n.value)
			}
		}

		// Set component field
		if f, ok := lookupField("component", out); ok {
			if f.Kind() == reflect.String && f.String() == "" {
				f.SetString(n.component)
			}
		}
	case structNode:
		decodeStructNode(n, out)
	case keyNode:
		decodeKeyNode(n, out)
	case scalarNode:
		decodeScalarNode(n, out)
	default:
		panic("Unknown node kind")
	}
}

func decodeStructNode(n *node, out reflect.Value) {
	l := len(n.children)

	switch out.Kind() {
	case reflect.Struct:
		for _, c := range n.children {
			unmarshal(c, out)
		}
	case reflect.Slice:
		out.Set(reflect.MakeSlice(out.Type(), l, l))
		et := out.Type().Elem()
		for i := 0; i < l; i++ {
			e := reflect.New(et).Elem()
			for _, c := range n.children[i].children {
				unmarshal(c, e)
			}
			out.Index(i).Set(e)
		}
	case reflect.Map:
		out.Set(reflect.MakeMap(out.Type()))
		et := out.Type().Elem()
		for i := 0; i < l; i++ {
			k := reflect.ValueOf(n.children[i].value)
			v := reflect.New(et).Elem()
			for _, c := range n.children[i].children {
				unmarshal(c, v)
			}
			out.SetMapIndex(k, v)
		}
	}
}

func decodeKeyNode(n *node, out reflect.Value) {
	switch out.Kind() {
	case reflect.Struct:
		if f, ok := lookupField(n.value, out); ok {
			if len(n.children) > 0 {
				unmarshal(n.children[0], f)
			}
		}
	}
}

func decodeScalarNode(n *node, out reflect.Value) {
	switch out.Kind() {
	case reflect.Int:
		i, _ := strconv.ParseInt(n.value, 10, 64)
		out.SetInt(i)
	case reflect.String:
		out.SetString(n.value)
	}
}

func lookupField(tag string, v reflect.Value) (reflect.Value, bool) {
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fi := typ.Field(i)
		if fi.Tag.Get("ltm") == tag {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

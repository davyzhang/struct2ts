package struct2ts

import (
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"
)

type Options struct {
	Indent string

	NoAssignDefaults bool
	InterfaceOnly    bool

	MarkOptional  bool
	NoConstructor bool
	NoToObject    bool
	NoDate        bool

	indents [3]string
}

func New(opts *Options) *StructToTS {
	if opts == nil {
		opts = &Options{}
	}

	if opts.Indent == "" {
		opts.Indent = "\t"
	}

	for i := range opts.indents {
		opts.indents[i] = strings.Repeat(opts.Indent, i)
	}

	return &StructToTS{
		structs: []*Struct{},
		seen:    map[reflect.Type]*Struct{},
		opts:    opts,
	}
}

type StructToTS struct {
	structs []*Struct
	seen    map[reflect.Type]*Struct
	opts    *Options
}

func (s *StructToTS) Add(v interface{}) *Struct {
	var t reflect.Type
	switch v := v.(type) {
	case reflect.Type:
		t = v
	case reflect.Value:
		t = v.Type()
	default:
		t = reflect.TypeOf(v)
	}

	return s.addType(t, "")
}

func (s *StructToTS) addType(t reflect.Type, prefix string) (out *Struct) {
	t = indirect(t)

	if out = s.seen[t]; out != nil {
		return out
	}

	out = &Struct{
		Name:   prefix + t.Name(),
		Fields: make([]Field, 0, t.NumField()),

		t: t,
	}

	for i := 0; i < t.NumField(); i++ {
		var (
			sf  = t.Field(i)
			sft = sf.Type
			tf  Field
			k   = sft.Kind()
		)

		if k == reflect.Ptr {
			tf.CanBeNull = true
			sft = indirect(sft)
			k = sft.Kind()
		}

		if tf.setProps(sf) {
			continue
		}

		switch {
		case isNumber(k):
			tf.TsType = "number"

		case k == reflect.String:
			tf.TsType = "string"

		case k == reflect.Bool:
			tf.TsType = "boolean"

		case k == reflect.Map:
			tf.TsType, tf.KeyType, tf.ValType = "map", stripType(sft.Key()), stripType(sft.Elem())

			if isStruct(sft.Elem()) {
				tf.ValType = s.addType(sft.Elem(), out.Name).Name
			}

		case k == reflect.Slice, k == reflect.Array:
			tf.TsType, tf.ValType = "array", stripType(sft.Elem())

			if isStruct(sft.Elem()) {
				tf.ValType = s.addType(sft.Elem(), out.Name).Name
			}

		case k == reflect.Struct:
			tf.TsType = "object"
			tf.ValType = s.addType(sft, out.Name).Name

		case k == reflect.Interface:
			tf.TsType, tf.ValType = "object", ""

		default:
			log.Println("unhandled", k, sft)
		}

		out.Fields = append(out.Fields, tf)
	}

	s.structs = append(s.structs, out)
	s.seen[t] = out
	return
}

func (s *StructToTS) RenderTo(w io.Writer) (err error) {
	for _, st := range s.structs {
		if err = st.RenderTo(s.opts, w); err != nil {
			return
		}
		fmt.Fprintln(w)
	}
	return
}
func indirect(t reflect.Type) reflect.Type {
	k := t.Kind()
	for k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	return t
}

func isNumber(k reflect.Kind) bool {
	switch k {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Float32, reflect.Float64:

		return true

	default:
		return false
	}
}

func isStruct(t reflect.Type) bool {
	return indirect(t).Kind() == reflect.Struct
}

func stripType(t reflect.Type) string {
	n := t.String()
	if i := strings.IndexByte(n, '.'); i > -1 {
		n = n[i+1:]
	}
	return n
}

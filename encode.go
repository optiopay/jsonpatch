// Package jsonpatch implements applying and creation of JSON patch as defined in RFC 6902.
package jsonpatch

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/optiopay/jsonpatch/deep"
)

var (
	hyphens = regexp.MustCompile(`[\-_]`)
)

var (
	ErrNonPointer     = fmt.Errorf("jsonpatch: interface non-pointer")
	ErrCouldNotCopy   = fmt.Errorf("jsonpatch: could not make a copy")
	ErrUnmarshal      = fmt.Errorf("jsonpatch: error while unmarshalling the patch")
	ErrNodeNil        = fmt.Errorf("jsonpatch: node was empty")
	ErrIncorrectIndex = fmt.Errorf("jsonpatch: incorrect index")
	ErrGarbage        = fmt.Errorf("jsonpatch: garbage value")
	ErrNotImplemented = fmt.Errorf("jsonpatch: not implemented")
)

type ErrUnsupported struct {
	Err string
}

func (e *ErrUnsupported) Error() string {
	return fmt.Sprintf("jsonpatch: unsupported type for key %s", e.Err)
}

type patch struct {
	Op    string
	Path  string
	From  string
	Value json.RawMessage
}

// Apply applies a patch as defined in RFC 6902 to the passed interface.
//
// Apply makes a deep copy of the entire structure. Thus patches on large
// data structures will not be efficient.
func Apply(data []byte, x interface{}) error {
	rx := reflect.ValueOf(x)
	if rx.Kind() != reflect.Ptr || rx.IsNil() {
		return ErrNonPointer
	}

	var patches []patch
	err := json.Unmarshal(data, &patches)
	if err != nil {
		return err
	}

	ry := reflect.New(rx.Elem().Type())
	// I am making a copy of the interface so that when an
	// error arises while performing one of the patches the
	// original data structure does not get altered.
	err = deep.Copy(x, ry.Interface())
	if err != nil {
		return ErrCouldNotCopy
	}

	for _, p := range patches {
		path := strings.Trim(p.Path, "/")
		err := rapply(path, &p, ry)
		if err != nil {
			return err
		}
	}

	rx.Elem().Set(ry.Elem())
	return nil
}

func rapply(path string, p *patch, x reflect.Value) error {
	args := strings.SplitN(path, "/", 2)
	if len(args) == 2 {
		return findNode(args[0], args[1], p, x)
	}
	return applyNode(args[0], p, x)
}

func findNode(root, node string, p *patch, x reflect.Value) error {
	var child reflect.Value
	if x.Kind() == reflect.Ptr {
		if x.IsNil() {
			t := x.Type().Elem()
			x.Set(reflect.New(t))
		}
		x = x.Elem()
	}
	switch x.Kind() {
	case reflect.Slice, reflect.Array:
		pos, err := strconv.Atoi(root)
		if err != nil {
			return ErrIncorrectIndex
		}
		if pos >= x.Len() {
			return ErrIncorrectIndex
		}
		child = x.Index(pos)
	case reflect.Map:
		child = x.MapIndex(reflect.ValueOf(root))
	case reflect.Struct:
		t := x.Type()
		name := bestMatch(root, t)
		if name == "" {
			return ErrIncorrectIndex
		}
		child = x.FieldByName(name)
	case reflect.Ptr:
		child = x.Elem()
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		// TODO:
		return &ErrUnsupported{root}
	default:
		// these are primitive types thus should not have fields
		return ErrGarbage
	}
	// Case when the child is a pointer and is nil
	if child.Kind() == reflect.Ptr {
		if !child.IsNil() {
			return rapply(node, p, child)
		}
		newval := reflect.New(child.Type().Elem())
		child.Set(newval)
		return rapply(node, p, child)
	}

	// Case when the value is a zero value
	if !child.IsValid() {
		newval := reflect.New(child.Type().Elem())
		//newval returns a pointer to the element.
		child.Set(newval.Elem())
	}

	if child.CanAddr() {
		return rapply(node, p, child.Addr())
	}

	return &ErrUnsupported{root}
}

// bestMatch returns the field name of the struct field which is the
// closest to the name passed.
func bestMatch(name string, t reflect.Type) string {
	key := strings.ToLower(hyphens.ReplaceAllString(name, ""))
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Name == name {
			return field.Name
		}
		j := field.Tag.Get("json")
		if j != "" {
			flags := strings.Split(j, ",")
			for _, flag := range flags {
				if name == flag {
					return field.Name
				}
			}
		}
		lname := strings.ToLower(hyphens.ReplaceAllString(field.Name, ""))
		if key == lname {
			return field.Name
		}
	}
	return ""
}

func applyNode(node string, p *patch, x reflect.Value) error {
	switch p.Op {
	case "add":
		add(node, p, x)
	case "replace":
		replace(node, p, x)
	case "remove":
		remove(node, p, x)
	case "test":
		test(node, p, x)
	case "copy":
	case "move":
	}
	return nil
}

func add(node string, p *patch, v reflect.Value) error {
	var child reflect.Value
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			t := v.Type().Elem()
			v.Set(reflect.New(t))
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice:
		l := v.Len()
		if node == "-" {
			sl := reflect.MakeSlice(v.Type(), l+1, l+1)
			reflect.Copy(sl, v)
			child = sl.Index(l)
			n := child.Interface()
			err := json.Unmarshal(p.Value, &n)
			if err != nil {
				return err
			}
			child.Set(reflect.ValueOf(n))
			v.Set(sl)
			return nil
		}
		pos, err := strconv.Atoi(node)
		if err != nil {
			return ErrIncorrectIndex
		}
		child = v.Index(pos)
		n := child.Interface()
		err = json.Unmarshal(p.Value, &n)
		if err != nil {
			return err
		}
		sl := reflect.MakeSlice(v.Type(), 0, l+1)
		sl = reflect.AppendSlice(sl, v.Slice(0, pos))
		sl = reflect.Append(sl, reflect.ValueOf(n))
		sl = reflect.AppendSlice(sl, v.Slice(pos, l))
		v.Set(sl)

	case reflect.Map:
		if v.IsNil() {
			v.Set(reflect.MakeMap(v.Type()))
		}
		n := reflect.Zero(v.Type().Elem()).Interface()
		err := json.Unmarshal(p.Value, &n)
		if err != nil {
			return err
		}
		v.SetMapIndex(reflect.ValueOf(node), reflect.ValueOf(n))

	case reflect.Struct:
		child := v.FieldByName(strings.Title(node))
		if child.Kind() == reflect.Ptr && child.IsNil() {
			n := reflect.New(child.Type().Elem())
			err := json.Unmarshal(p.Value, n.Interface())
			if err != nil {
				return err
			}
			child.Set(n)
			return nil
		}
		n := reflect.New(child.Type())
		err := json.Unmarshal(p.Value, n.Interface())
		if err != nil {
			return err
		}
		child.Set(n.Elem())

	case reflect.Ptr:
		if v.IsNil() {
			child := reflect.New(v.Type().Elem())
			v.Set(child)
		}
		el := v.Elem().Interface()
		err := json.Unmarshal(p.Value, &el)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(el).Addr())
	}
	return nil
}

func replace(node string, p *patch, v reflect.Value) error {
	var child reflect.Value
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		pos, err := strconv.Atoi(node)
		if err != nil {
			return ErrIncorrectIndex
		}
		if pos > v.Len() {
			return ErrIncorrectIndex
		}
		child = v.Index(pos)
		n := child.Interface()
		err = json.Unmarshal(p.Value, &n)
		if err != nil {
			return err
		}
		child.Set(reflect.ValueOf(n))
		return nil

	case reflect.Map:
		child := v.MapIndex(reflect.ValueOf(node))
		if !child.IsValid() {
			return errors.New("map element not found")
		}
		n := child.Interface()
		err := json.Unmarshal(p.Value, &n)
		if err != nil {
			return err
		}
		v.SetMapIndex(reflect.ValueOf(node), reflect.ValueOf(n))
		return nil

	case reflect.Struct:
		child := v.FieldByName(strings.Title(node))
		n := reflect.New(child.Type())
		err := json.Unmarshal(p.Value, n.Interface())
		if err != nil {
			return err
		}
		child.Set(n.Elem())
		return nil

	case reflect.Ptr:
		//TODO
		return ErrNotImplemented

	}
	return nil
}

func remove(node string, p *patch, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		pos, err := strconv.Atoi(node)
		if err != nil {
			return ErrIncorrectIndex
		}
		sl := reflect.MakeSlice(v.Type(), 0, v.Len()-1)
		sl = reflect.AppendSlice(sl, v.Slice(0, pos))
		sl = reflect.AppendSlice(sl, v.Slice(pos+1, v.Len()))
		v.Set(sl)
		return nil

	case reflect.Map:
		child := v.MapIndex(reflect.ValueOf(node))
		v.SetMapIndex(reflect.ValueOf(node), reflect.Zero(child.Type()))
		return nil

	case reflect.Struct:
		child := v.FieldByName(strings.Title(node))
		child.Set(reflect.Zero(child.Type()))
		return nil

	case reflect.Ptr:
		//TODO
		return ErrNotImplemented

	}
	return nil
}

func test(node string, p *patch, v reflect.Value) error {
	if v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	var child reflect.Value
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		pos, err := strconv.Atoi(node)
		if err != nil {
			return ErrIncorrectIndex
		}
		child = v.Index(pos)
		if child.Kind() == reflect.Ptr {
			if child.IsNil() {
				//TODO: what to do with nil
				return nil
			}
			child = reflect.Indirect(child)
		}

	case reflect.Map:
		child = v.MapIndex(reflect.ValueOf(node))

	case reflect.Struct:
		child = v.FieldByName(strings.Title(node))

	case reflect.Ptr:
		//TODO
		return ErrNotImplemented
	}
	n := child.Interface()
	m := child.Interface()
	err := json.Unmarshal(p.Value, &n)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(n, m) {
		return errors.New("elements are not equal")
	}
	return nil
}

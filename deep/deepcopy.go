package deep

import (
	"errors"
	"reflect"
	"strings"
)

var (
	ErrNonPointer     = errors.New("deep: non pointer interfaces")
	ErrDifferentKinds = errors.New("deep: interfaces not the same kind")
	ErrUnsupported    = errors.New("deep: unsupported kind")
)

// Copy makes a recursive deep copy of obj x to y
//
// It does not recurse into map keys. Due to restrictions in the reflect package,
// only types with all public members may be copied.
func Copy(x, y interface{}) error {
	rx := reflect.ValueOf(x)
	if rx.Kind() != reflect.Ptr {
		return ErrNonPointer
	}
	// No point doing anything when the original value is nil
	if rx.IsNil() {
		return nil
	}

	ry := reflect.ValueOf(y)
	if ry.Kind() != reflect.Ptr {
		return ErrNonPointer
	}
	if rx.Kind() != ry.Kind() {
		return ErrDifferentKinds
	}
	return rcopy(rx, ry)
}

func rcopy(x, y reflect.Value) error {
	if x.Kind() == reflect.Ptr {
		x = x.Elem()
	}
	if y.Kind() == reflect.Ptr {
		if y.IsNil() {
			y.Set(reflect.New(y.Type().Elem()))
		}
		y = y.Elem()
	}
	var err error
	switch x.Kind() {
	case reflect.Slice, reflect.Array:
		err = copyArray(x.Addr(), y.Addr())
	case reflect.Map:
		err = copyMap(x.Addr(), y.Addr())
	case reflect.Struct:
		err = copyStruct(x.Addr(), y.Addr())
	case reflect.Ptr:
		vx := x.Elem()
		y.Set(reflect.New(vx.Type()))
		vy := y.Elem()
		if !vx.CanAddr() || !vy.CanAddr() {
			vy.Set(vx)
			return nil
		}
		err = rcopy(vx.Addr(), vy.Addr())

	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.Interface, reflect.UnsafePointer:
		// TODO:
		err = ErrUnsupported

	default:
		if !x.CanAddr() || !y.CanAddr() {
			y.Set(x)
			return nil
		}
		err = copyPrimitives(x.Addr(), y.Addr())
	}
	return err
}

func copyArray(x, y reflect.Value) error {
	if x.Kind() == reflect.Ptr {
		x = x.Elem()
	}
	if y.Kind() == reflect.Ptr {
		y = y.Elem()
	}

	l := x.Len()
	if x.Kind() == reflect.Slice {
		sl := reflect.MakeSlice(x.Type(), l, l)
		y.Set(sl)
	}
	for i := 0; i < l; i++ {
		vx := x.Index(i)
		vy := y.Index(i)
		if vx.Kind() == reflect.Ptr {
			err := rcopy(vx, vy)
			if err != nil {
				return err
			}
			continue
		}
		if !vx.CanAddr() {
			vy.Set(vx)
			continue
		}
		if !vy.CanAddr() {
			vy.Set(reflect.ValueOf(vx.Interface()))
		}
		err := rcopy(vx.Addr(), vy.Addr())
		if err != nil {
			return err
		}
	}
	return nil
}

func copyMap(x, y reflect.Value) error {
	if x.Kind() == reflect.Ptr {
		x = reflect.Indirect(x)
	}
	if y.Kind() == reflect.Ptr {
		y = reflect.Indirect(y)
	}
	if x.IsNil() {
		return nil
	}

	keys := x.MapKeys()
	y.Set(reflect.MakeMap(x.Type()))
	for _, key := range keys {
		vx := x.MapIndex(key)
		vy := y.MapIndex(key)
		if vx.Kind() == reflect.Ptr {
			el := vx.Elem()
			if !el.IsValid() {
				continue
			}
			y.SetMapIndex(key, reflect.New(el.Type()))
			vy = y.MapIndex(key)
			err := rcopy(vx, vy)
			if err != nil {
				return err
			}
			continue
		}
		if !vx.CanAddr() {
			y.SetMapIndex(key, vx)
			continue
		}
		if !vy.CanAddr() {
			vy = vx
		}
		err := rcopy(vx.Addr(), vy.Addr())
		if err != nil {
			return err
		}
		y.SetMapIndex(key, vy)
	}
	return nil
}

func copyStruct(x, y reflect.Value) error {
	if x.Kind() == reflect.Ptr {
		x = reflect.Indirect(x)
	}
	if y.Kind() == reflect.Ptr {
		y = reflect.Indirect(y)
	}

	if x.Kind() == reflect.Ptr {
		return rcopy(x, y)
	}

	n := x.Type().NumField()
	if y.IsValid() {
		y.Set(x)

	}
	for i := 0; i < n; i++ {
		st := x.Type().Field(i)
		if st.Anonymous {
			continue
		}
		// Ignore private fields
		if string(st.Name[0]) != strings.ToUpper(string(st.Name[0])) {
			continue
		}
		vx := x.Field(i)
		vy := y.Field(i)
		if vx.Kind() == reflect.Ptr {
			el := vx.Elem()
			if !el.IsValid() {
				continue
			}
			vy.Set(reflect.New(el.Type()))
			err := rcopy(vx, vy)
			if err != nil {
				return err
			}
			continue
		}
		if !vx.CanAddr() {
			vy.Set(vx)
			continue
		}
		if !vy.CanAddr() {
			vy = vx
		}
		err := rcopy(vx.Addr(), vy.Addr())
		if err != nil {
			return err
		}
	}
	return nil
}

func copyPrimitives(x, y reflect.Value) error {
	if x.Kind() == reflect.Ptr {
		x = reflect.Indirect(x)
	}
	if y.Kind() == reflect.Ptr {
		y = reflect.Indirect(y)
	}
	y.Set(x)
	return nil
}

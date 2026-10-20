package history

import "reflect"

// deepClone copies v and everything it reaches through pointers, slices and maps. Records hold
// nested args, seeds and rule maps, so a shallow copy would let a caller write into the cache.
func deepClone[T any](v T) T {
	var out T
	c := cloner{flat: map[reflect.Type]bool{}}
	c.copy(reflect.ValueOf(&out).Elem(), reflect.ValueOf(&v).Elem())
	return out
}

// cloner remembers, for one clone, which types a plain assignment already copies deeply.
type cloner struct{ flat map[reflect.Type]bool }

// copy sets dst to a copy of src. A struct is assigned whole first, so unexported fields
// (time.Time's) are copied by value; its exported fields are then copied deeply.
func (c cloner) copy(dst, src reflect.Value) {
	switch src.Kind() { //nolint:exhaustive // the default assigns every scalar kind
	case reflect.Pointer, reflect.Interface:
		if !src.IsNil() {
			v := reflect.New(src.Elem().Type())
			c.copy(v.Elem(), src.Elem())
			if src.Kind() == reflect.Interface {
				v = v.Elem()
			}
			dst.Set(v)
		}
	case reflect.Slice:
		c.copySlice(dst, src)
	case reflect.Map:
		c.copyMap(dst, src)
	case reflect.Array:
		for i := range src.Len() {
			c.copy(dst.Index(i), src.Index(i))
		}
	case reflect.Struct:
		dst.Set(src)
		if !c.flatType(src.Type()) {
			for i := range src.NumField() {
				if src.Type().Field(i).IsExported() {
					c.copy(dst.Field(i), src.Field(i))
				}
			}
		}
	default:
		dst.Set(src)
	}
}

func (c cloner) copySlice(dst, src reflect.Value) {
	if src.IsNil() {
		return
	}
	out := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
	if c.flatType(src.Type().Elem()) {
		reflect.Copy(out, src)
	} else {
		for i := range src.Len() {
			c.copy(out.Index(i), src.Index(i))
		}
	}
	dst.Set(out)
}

func (c cloner) copyMap(dst, src reflect.Value) {
	if src.IsNil() {
		return
	}
	out := reflect.MakeMapWithSize(src.Type(), src.Len())
	for it := src.MapRange(); it.Next(); {
		v := reflect.New(src.Type().Elem()).Elem()
		c.copy(v, it.Value())
		out.SetMapIndex(it.Key(), v)
	}
	dst.Set(out)
}

// flatType reports a type whose plain assignment is already a deep copy: no pointer, slice,
// map or interface among its exported parts. Unexported fields are copied by value either way.
func (c cloner) flatType(t reflect.Type) bool {
	if flat, ok := c.flat[t]; ok {
		return flat
	}
	flat := true
	switch t.Kind() { //nolint:exhaustive // every other kind is a scalar, which is flat
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		flat = false
	case reflect.Array:
		flat = c.flatType(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if f := t.Field(i); f.IsExported() && !c.flatType(f.Type) {
				flat = false
				break
			}
		}
	}
	c.flat[t] = flat
	return flat
}

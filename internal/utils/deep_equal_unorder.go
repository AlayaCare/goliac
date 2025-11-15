package utils

import "reflect"

/*
 * DeepEqualUnordered compares two interfaces by their values, ignoring the order of the keys in the map
 */
func DeepEqualUnordered(a, b interface{}) bool {
	return deepEqual(reflect.ValueOf(a), reflect.ValueOf(b))
}

func deepEqual(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}

	if a.Type() != b.Type() {
		return false
	}

	switch a.Kind() {
	case reflect.Map:
		if a.Len() != b.Len() {
			return false
		}
		for _, key := range a.MapKeys() {
			av := a.MapIndex(key)
			bv := b.MapIndex(key)
			if !bv.IsValid() || !deepEqual(av, bv) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		// Try all combinations for unordered comparison
		used := make([]bool, b.Len())
		for i := 0; i < a.Len(); i++ {
			matchFound := false
			for j := 0; j < b.Len(); j++ {
				if !used[j] && deepEqual(a.Index(i), b.Index(j)) {
					used[j] = true
					matchFound = true
					break
				}
			}
			if !matchFound {
				return false
			}
		}
		return true
	case reflect.Interface:
		return deepEqual(a.Elem(), b.Elem())
	default:
		return reflect.DeepEqual(a.Interface(), b.Interface())
	}
}

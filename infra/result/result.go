// Package result provides shared transformations for mo.Result values.
package result

import "github.com/samber/mo"

// FlatMap applies mapper to a successful result and propagates an error
// without invoking mapper. Unlike mo.Result.FlatMap, it supports changing the
// contained type.
func FlatMap[I, O any](input mo.Result[I], mapper func(I) mo.Result[O]) mo.Result[O] {
	value, err := input.Get()
	if err != nil {
		return mo.Err[O](err)
	}
	return mapper(value)
}

package cmp

import "reflect"

type Normalizer[T any] interface {
	Normalize() T
}

func SemanticEquals[T Normalizer[T]](this, that T) bool {
	return reflect.DeepEqual(this.Normalize(), that.Normalize())
}

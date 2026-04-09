// Package lib provides general-purpose types shared across the taffy-go codebase.
package lib

// Option represents an optional value, similar to Rust's Option<T>.
type Option[T any] struct {
	Value  T
	IsSome bool
}

// Some returns an Option containing the given value.
func Some[T any](v T) Option[T] {
	return Option[T]{Value: v, IsSome: true}
}

// None returns an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// UnwrapOr returns the contained value if Some, otherwise returns the alt value.
func (o Option[T]) UnwrapOr(alt T) T {
	if o.IsSome {
		return o.Value
	}
	return alt
}

// Or returns this Option if Some, otherwise returns alt.
func (o Option[T]) Or(alt Option[T]) Option[T] {
	if o.IsSome {
		return o
	}
	return alt
}

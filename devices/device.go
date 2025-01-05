package devices

// BaseTypes is the set of types we can create effects for
type BaseTypes interface {
	int64 | float64 | string | bool
}

// Effect[T] defines a callback function that operates on an object of type T and returns an error.
//
// Effects are registered by a device.
type Effect[T BaseTypes] func(T) error

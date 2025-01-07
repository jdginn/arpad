package devices

// BaseTypes is the set of types we can create callbacks for in arpad
type BaseTypes interface {
	int64 | float64 | string | bool
}

// Callback[T] defines a function bound to an event such that it runs each time its event occurs.
type Callback[T BaseTypes] func(T) error

package devices

type BaseTypes interface {
	int64 | float64 | string | bool
}

type (
	Effect[T BaseTypes] func(T) error
)

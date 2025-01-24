package defaults

func Get[T comparable](v T, value ...T) T {
	var zero T
	if len(value) > 0 && v == zero {
		return value[0]
	}
	return v
}

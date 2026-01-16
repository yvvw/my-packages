package utils

func Map[T any, U any](list []T, f func(T) U) []U {
	var result []U
	for _, i := range list {
		result = append(result, f(i))
	}
	return result
}

func Filter[T any](list []T, f func(T) bool) []T {
	var result []T
	for _, i := range list {
		if f(i) {
			result = append(result, i)
		}
	}
	return result
}

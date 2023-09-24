package utils

func Uniq[T comparable](values []T) []T {
	var s = make(map[T]struct{}, len(values))
	var out = make([]T, 0, len(values))
	for _, v := range values {
		if _, ok := s[v]; !ok {
			s[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

type Set[T comparable] map[T]struct{}

func NewSet[T comparable](values ...T) Set[T] {
	var s = make(Set[T], len(values))
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

func (s Set[T]) Has(values ...T) bool {
	for _, v := range values {
		if _, ok := s[v]; !ok {
			return false
		}
	}
	return true
}

func (s Set[T]) Add(values ...T) {
	for _, v := range values {
		s[v] = struct{}{}
	}
}

func Keys[T comparable, V any](store map[T]V) []T {
	var ans = make([]T, 0, len(store))
	for k := range store {
		ans = append(ans, k)
	}
	return ans
}

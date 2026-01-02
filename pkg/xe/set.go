package xe

type void struct{}

// Set holds a set of values
type Set[T comparable] struct {
	m map[T]void
}

// New creates a new empty set
func NewSet[T comparable]() Set[T] {
	return Set[T]{m: make(map[T]void)}
}

// Add inserts an element into a Set
func (s *Set[T]) Add(v T) {
	s.m[v] = void{}
}

func (s *Set[T]) Contains(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.m)
}

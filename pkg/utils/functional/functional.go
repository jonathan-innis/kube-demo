package functional

type Copyable[T any] interface {
	DeepCopy() T
}

func DeepCopyMap[K comparable, V Copyable[V]](m map[K]V) map[K]V {
	newMap := make(map[K]V)
	for k, v := range m {
		newMap[k] = v.DeepCopy()
	}
	return newMap
}

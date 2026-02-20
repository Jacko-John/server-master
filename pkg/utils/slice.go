package utils

// CollectUnique reads slices from the provided channel and returns a deduplicated slice 
// containing all unique elements.
func CollectUnique[T comparable](ch <-chan []T) []T {
	set := NewSet[T]()
	for items := range ch {
		set.AddAll(items)
	}
	return set.ToSlice()
}

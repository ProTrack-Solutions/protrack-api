package assign

func SetIfNotNil[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}

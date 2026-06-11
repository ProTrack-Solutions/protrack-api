package assign


func SetIfNotEmpty(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

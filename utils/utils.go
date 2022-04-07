package utils

func EmptyStringIfNil(a *string) {
	if a == nil {
		b := ""
		a = &b
	}
}

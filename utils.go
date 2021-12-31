package main

func emptyStringIfNull(a *string) {
	if a == nil {
		b := ""
		a = &b
	}
}

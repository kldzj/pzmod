package util

func YesNo(b bool) string {
	if b {
		return "Yes"
	}

	return "No"
}

func Quote(s string) string {
	return `"` + s + `"`
}

func Paren(s string) string {
	return "(" + s + ")"
}

func ParseBool(s string) bool {
	return s == "true"
}

func BoolString(b bool) string {
	if b {
		return "true"
	}

	return "false"
}

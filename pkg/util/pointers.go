package util

func StrP(s string) *string {
	return &s
}

func BoolP(b bool) *bool {
	return &b
}

func MapP(m map[string]interface{}) *map[string]interface{} {
	return &m
}

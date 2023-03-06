package util

func StrP(s string) *string {
	return &s
}

func MapP(m map[string]interface{}) *map[string]interface{} {
	return &m
}

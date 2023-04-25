package util

import "time"

func StrP(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func BoolP(b bool) *bool {
	return &b
}

func MapP(m map[string]interface{}) *map[string]interface{} {
	return &m
}

func TimeP(t time.Time) *time.Time {
	return &t
}

func Int32P(i int32) *int32 {
	return &i
}

package util

// contains is a helper function that iterates over a slice and returns true if the given value is found
func Contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

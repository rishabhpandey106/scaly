package utils

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func ToBase62(n int64) string {
	if n == 0 {
		return "a"
	}

	result := ""

	for n > 0 {
		result = string(charset[n%62]) + result
		n = n / 62
	}

	return result
}

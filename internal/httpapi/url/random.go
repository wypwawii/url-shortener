package url

import (
	"fmt"
	"math/rand/v2"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandomString(length int) (string, error) {
	if length < 0 {
		return "", fmt.Errorf("length must not be negative")
	}

	result := make([]byte, length)
	for i := range result {
		result[i] = alphabet[rand.IntN(len(alphabet))]
	}

	return string(result), nil
}

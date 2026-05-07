package protocol

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateRoomCode generates a 6-digit room code with checksum
// Format: XXX-XXY where X are random digits, Y is checksum
func GenerateRoomCode() (string, error) {
	// Generate 4 random digits
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}

	// Format as 4 digits with leading zeros
	code := fmt.Sprintf("%04d", n.Int64())

	// Calculate checksum (simple sum of digits mod 100)
	sum := 0
	for _, c := range code {
		sum += int(c - '0')
	}
	checksum := sum % 100

	// Final format: XXXX-YY
	return fmt.Sprintf("%s-%02d", code, checksum), nil
}

// ValidateRoomCode checks if the room code checksum is valid
func ValidateRoomCode(roomCode string) bool {
	// Remove dash if present
	code := roomCode
	if len(code) == 7 && code[4] == '-' {
		code = code[:4] + code[5:7]
	}

	if len(code) != 6 {
		return false
	}

	// Calculate checksum from first 4 digits
	sum := 0
	for i := 0; i < 4; i++ {
		sum += int(code[i] - '0')
	}
	expectedChecksum := sum % 100

	// Parse actual checksum
	actualChecksum := int(code[4]-'0')*10 + int(code[5]-'0')

	return expectedChecksum == actualChecksum
}

// NormalizeRoomCode removes dashes and converts to standard format
func NormalizeRoomCode(roomCode string) string {
	// Remove all non-digits
	result := ""
	for _, c := range roomCode {
		if c >= '0' && c <= '9' {
			result += string(c)
		}
	}

	// Add dash if we have 6 digits
	if len(result) == 6 {
		return result[:4] + "-" + result[4:6]
	}

	return result
}

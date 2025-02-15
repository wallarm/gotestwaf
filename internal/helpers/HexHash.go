package helpers

import (
	"crypto/sha256"
	"encoding/hex"
)

func HexOfHashOfTestIdentifier(testSet string, testCase string, placeholder string, encoder string, payload string) string {
	hash := sha256.New()
	hash.Reset()
	hash.Write([]byte(testSet))
	hash.Write([]byte(testCase))
	hash.Write([]byte(placeholder))
	hash.Write([]byte(encoder))
	hash.Write([]byte(payload))

	return hex.EncodeToString(hash.Sum(nil)) // Return the hexadecimal string representation
}

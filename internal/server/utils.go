package server

import (
    "crypto/rand"
    "encoding/hex"
)

// GenerateUniqueID generates a cryptographically secure random ID.
func GenerateUniqueID() string {
    b := make([]byte, 16)
    _, err := rand.Read(b)
    if err != nil {
        return ""
    }
    return hex.EncodeToString(b)
}
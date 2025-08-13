package main

import (
	"encoding/base64"
	"log"
)

func decodeBase64Param(encoded string) string {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		log.Printf("Failed to decode base64 param: %s", encoded)
		return ""
	}
	return string(decoded)
}

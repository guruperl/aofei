package main

import (
	"os"
	"strings"
	"testing"
)

func TestSummerExampleUsesBcryptPasswordHashField(t *testing.T) {
	data, err := os.ReadFile("summer.example.json")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(strings.ToUpper(text), "SHA1(") {
		t.Fatal("summer example must not use SHA1 password SQL")
	}
	if !strings.Contains(text, `"Password_hash": "passwd"`) {
		t.Fatal("summer example should verify stored password hashes through Password_hash")
	}
}

package main

import (
	"os"
	"testing"
)

func TestConnectDB_MissingEnvVars(t *testing.T) {
	vars := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"}
	originals := make(map[string]string)
	for _, v := range vars {
		originals[v] = os.Getenv(v)
		os.Unsetenv(v)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			if v != "" {
				os.Setenv(k, v)
			}
		}
	})

	_, err := connectDB()
	if err == nil {
		t.Error("expected error when DB env vars are missing, got nil")
	}
}

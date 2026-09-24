package config_test

import (
	"testing"

	"gotiktokdownloader/backend/config"
)

func TestDefaultValid(t *testing.T) {
	if err := config.Default().Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidConcurrency(t *testing.T) {
	c := config.Default()
	c.Download.MaxConcurrency = 101
	if err := c.Validate(); err == nil {
		t.Fatal("expected error")
	}
}

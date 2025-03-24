package analyzer

import (
	"testing"
)

func TestSetConfig(t *testing.T) {
	config := &Config{
		IgnoreFiles: []string{"file1", "file2"},
		Verbose:     true,
	}

	SetConfig(config)

	if globalConfig != config {
		t.Errorf("SetConfig() did not set the global configuration")
	}
}

func TestGetConfig(t *testing.T) {
	config := &Config{
		IgnoreFiles: []string{"file1", "file2"},
		Verbose:     true,
	}

	globalConfig = config

	result := GetConfig()

	if result != config {
		t.Errorf("GetConfig() did not return the global configuration")
	}
}

func TestGetConfigDefault(t *testing.T) {
	globalConfig = nil

	result := GetConfig()

	if result == nil {
		t.Errorf("GetConfig() returned nil")
	}

	if result.IgnoreFiles != nil {
		t.Errorf("GetConfig() returned non-nil IgnoreFiles")
	}

	if result.Verbose != false {
		t.Errorf("GetConfig() returned non-false Verbose")
	}
}

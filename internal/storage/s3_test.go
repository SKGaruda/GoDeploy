package storage

import (
	"context"
	"strings"
	"testing"
)

func TestNewS3StorageRequiresEndpoint(t *testing.T) {
	t.Setenv("FLOCI_ENDPOINT", "")

	_, err := NewS3Storage(context.Background())

	if err == nil {
		t.Fatal("expected an error when FLOCI_ENDPOINT is missing")
	}

	if !strings.Contains(err.Error(), "FLOCI_ENDPOINT") {
		t.Errorf("expected error to mention FLOCI_ENDPOINT, got: %v", err)
	}
}

func TestSaveDeploymentRequiresTimestamp(t *testing.T) {
	storage := &S3Storage{}

	record := DeploymentRecord{
		Application: "OpsPulse",
		Version:     "1.0.0",
		Image:       "godeploy-app:v1",
		Status:      "SUCCESS",
		Environment: "production",
		DeployedAt:  "",
	}

	err := storage.SaveDeployment(context.Background(), record)

	if err == nil {
		t.Fatal("expected an error when deployment timestamp is missing")
	}

	expectedError := "deployment timestamp is required"

	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

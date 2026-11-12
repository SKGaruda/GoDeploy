package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3DeploymentSaveAndRetrieve(t *testing.T) {
	if os.Getenv("FLOCI_INTEGRATION_TEST") != "1" {
		t.Skip("set FLOCI_INTEGRATION_TEST=1 to run Floci integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	store, err := NewS3Storage(ctx)
	if err != nil {
		t.Fatalf("failed to connect to Floci: %v", err)
	}

	if err := store.EnsureBucket(ctx, BucketName); err != nil {
		t.Fatalf("failed to ensure S3 bucket: %v", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	record := DeploymentRecord{
		Application: "GoDeploy-StorageIntegrationTest",
		Version:     "1.0.0",
		Image:       "godeploy-storage-test:integration",
		Status:      "TEST",
		Environment: "test",
		DeployedAt:  timestamp,
	}

	key := "deployments/" + timestamp + ".json"

	if err := store.SaveDeployment(ctx, record); err != nil {
		t.Fatalf("failed to save deployment record: %v", err)
	}

	// Use a separate context because the main test context
	// is canceled before cleanup callbacks execute.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cleanupCancel()

		_, err := store.client.DeleteObject(cleanupCtx, &s3.DeleteObjectInput{
			Bucket: aws.String(BucketName),
			Key:    aws.String(key),
		})

		if err != nil {
			t.Errorf("failed to clean up test record: %v", err)
		}
	})

	records, err := store.ListDeployments(ctx)
	if err != nil {
		t.Fatalf("failed to retrieve deployment records: %v", err)
	}

	found := false

	for _, savedRecord := range records {
		if savedRecord.DeployedAt == timestamp {
			found = true

			if savedRecord.Application != record.Application ||
				savedRecord.Version != record.Version ||
				savedRecord.Image != record.Image ||
				savedRecord.Status != record.Status ||
				savedRecord.Environment != record.Environment {
				t.Errorf(
					"retrieved record does not match saved record: got %+v",
					savedRecord,
				)
			}

			break
		}
	}

	if !found {
		t.Errorf("saved deployment record was not found: %s", key)
	}
}

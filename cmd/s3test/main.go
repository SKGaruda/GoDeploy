package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"godeploy/internal/storage"
)

func main() {
	ctx := context.Background()

	// Connect to Floci S3
	store, err := storage.NewS3Storage(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Floci: %v", err)
	}

	fmt.Println("[SUCCESS] Connected to Floci S3!")

	// Create the bucket
	err = store.CreateBucket(ctx, storage.BucketName)
	if err != nil {
		log.Fatalf("Failed to create S3 bucket: %v", err)
	}

	fmt.Println("[SUCCESS] S3 bucket is ready!")

	// Prepare deployment metadata
	record := storage.DeploymentRecord{
		Application: "OpsPulse",
		Version:     "1.0.0",
		Image:       "godeploy-app:v1",
		Status:      "SUCCESS",
		Environment: "development",
		DeployedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}

	// Upload metadata to S3
	err = store.SaveDeployment(ctx, record)
	if err != nil {
		log.Fatalf("Failed to save deployment metadata: %v", err)
	}

	fmt.Println("[SUCCESS] Deployment metadata uploaded to Floci S3!")
}

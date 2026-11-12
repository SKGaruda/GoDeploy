package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const BucketName = "godeploy-deployment-metadata"

type S3Storage struct {
	client *s3.Client
}

type DeploymentRecord struct {
	Application string `json:"application"`
	Version     string `json:"version"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	Environment string `json:"environment"`
	DeployedAt  string `json:"deployed_at"`
}

func NewS3Storage(ctx context.Context) (*S3Storage, error) {
	endpoint := os.Getenv("FLOCI_ENDPOINT")
	if endpoint == "" {
		return nil, fmt.Errorf("FLOCI_ENDPOINT environment variable is not set")
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})

	return &S3Storage{client: client}, nil
}

func (s *S3Storage) CreateBucket(ctx context.Context, bucketName string) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return fmt.Errorf("create S3 bucket %q: %w", bucketName, err)
	}

	return nil
}

func (s *S3Storage) EnsureBucket(ctx context.Context, bucketName string) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err == nil {
		fmt.Printf("[INFO] S3 bucket already exists: %s\n", bucketName)
		return nil
	}

	if err := s.CreateBucket(ctx, bucketName); err != nil {
		return fmt.Errorf("ensure S3 bucket %q: %w", bucketName, err)
	}

	fmt.Printf("[SUCCESS] S3 bucket created: %s\n", bucketName)
	return nil
}

func (s *S3Storage) SaveDeployment(ctx context.Context, record DeploymentRecord) error {
	if record.DeployedAt == "" {
		return fmt.Errorf("deployment timestamp is required")
	}

	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode deployment metadata: %w", err)
	}

	key := "deployments/" + record.DeployedAt + ".json"

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("save deployment metadata: %w", err)
	}

	fmt.Printf("[SUCCESS] Deployment metadata saved: s3://%s/%s\n", BucketName, key)
	return nil
}

// ListDeployments retrieves all deployment records from S3.
func (s *S3Storage) ListDeployments(ctx context.Context) ([]DeploymentRecord, error) {
	var records []DeploymentRecord

	paginator := s3.NewListObjectsV2Paginator(
		s.client,
		&s3.ListObjectsV2Input{
			Bucket: aws.String(BucketName),
			Prefix: aws.String("deployments/"),
		},
	)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list deployment records: %w", err)
		}

		for _, object := range page.Contents {
			if object.Key == nil {
				continue
			}

			result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
				Bucket: aws.String(BucketName),
				Key:    object.Key,
			})
			if err != nil {
				return nil, fmt.Errorf("get deployment record %q: %w", *object.Key, err)
			}

			data, readErr := io.ReadAll(result.Body)
			closeErr := result.Body.Close()

			if readErr != nil {
				return nil, fmt.Errorf("read deployment record %q: %w", *object.Key, readErr)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("close deployment record %q: %w", *object.Key, closeErr)
			}

			var record DeploymentRecord
			if err := json.Unmarshal(data, &record); err != nil {
				return nil, fmt.Errorf("decode deployment record %q: %w", *object.Key, err)
			}

			records = append(records, record)
		}
	}

	// Sort deployment records from newest to oldest.
	sort.Slice(records, func(i, j int) bool {
		return records[i].DeployedAt > records[j].DeployedAt
	})

	return records, nil
}

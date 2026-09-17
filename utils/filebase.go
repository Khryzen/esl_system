package utils

import (
	"context"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func UploadToFilebase(file multipart.File, filename string) (string, error) {
	ctx := context.TODO()

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           "https://s3.filebase.io",
			SigningRegion: "auto",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			os.Getenv("FILEBASE_ACCESS_KEY"),
			os.Getenv("FILEBASE_SECRET_KEY"),
			"",
		)),
		config.WithRegion("auto"),
	)
	if err != nil {
		return "", err
	}

	client := s3.NewFromConfig(cfg)
	bucketName := os.Getenv("FILEBASE_BUCKET")

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
		Body:   file,
		ACL:    types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", err
	}

	fileURL := "https://" + bucketName + ".s3.filebase.io/" + filename
	return fileURL, nil
}

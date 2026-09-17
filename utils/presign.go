package utils

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GetPresignedFileURL(filename string) (string, error) {
	ctx := context.TODO()
	client := getS3Client()
	presigner := s3.NewPresignClient(client)
	bucketName := os.Getenv("FILEBASE_BUCKET")

	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	}, s3.WithPresignExpires(time.Hour*2))

	if err != nil {
		return "", err
	}

	return request.URL, nil
}

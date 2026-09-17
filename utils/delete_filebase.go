package utils

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DeleteFromFilebase(filename string) error {
	ctx := context.TODO()
	client := getS3Client()
	bucketName := os.Getenv("FILEBASE_BUCKET")

	_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(filename),
	})
	return err
}

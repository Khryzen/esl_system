package utils

import (
	"context"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func UploadToFilebase(file multipart.File, filename string) (string, error) {
	ctx := context.TODO()
	client := getS3Client()
	bucketName := os.Getenv("FILEBASE_BUCKET")

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
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

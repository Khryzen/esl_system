package utils

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getS3Client() *s3.Client {
	creds := credentials.NewStaticCredentialsProvider(
		os.Getenv("FILEBASE_ACCESS_KEY"),
		os.Getenv("FILEBASE_SECRET_KEY"),
		"",
	)
	return s3.New(s3.Options{
		BaseEndpoint: aws.String("https://s3.filebase.io"),
		Region:       "auto",
		Credentials:  creds,
	})
}

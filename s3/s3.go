package s3

import (
	"context"
	"os"
	"path/filepath"

	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
)

var (
	sub zerolog.Logger
)

// UploadDirToS3 uploads the contents of a directory to S3-compatible storage
func UploadDirToS3(directory string, s3repo types.S3Repo) error {
	// Initialize minio client object.
	client, err := minio.New(s3repo.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3repo.AccessKey, s3repo.SecretKey, ""),
		Secure: s3repo.UseSSL,
		Region: s3repo.Region,
	})
	if err != nil {
		return err
	}
	err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || (info.Mode()&os.ModeSymlink != 0) {
			return nil // Skip directories and symbolic links
		}

		// Open the file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Get file info
		stat, err := file.Stat()
		if err != nil {
			return err
		}

		// Upload the file to S3-compatible storage
		objectName := filepath.ToSlash(path[len(directory)+1:]) // Object name in bucket

		_, err = client.PutObject(context.Background(), s3repo.Bucket, objectName, file, stat.Size(), minio.PutObjectOptions{})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteObjectsNotInRepo deletes objects from the bucket that are not present in the repository
func DeleteObjectsNotInRepo(directory, bucketdir string, s3repo types.S3Repo) error {
	sub = logger.CreateSubLogger("stage", "s3", "endpoint", s3repo.Endpoint, "bucket", s3repo.Bucket)
	// Initialize minio client object.
	client, err := minio.New(s3repo.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3repo.AccessKey, s3repo.SecretKey, ""),
		Secure: s3repo.UseSSL,
		Region: s3repo.Region,
	})
	if err != nil {
		return err
	}

	// List objects in the bucket within the specified directory (prefix)
	for object := range client.ListObjects(context.Background(), s3repo.Bucket, minio.ListObjectsOptions{
		Prefix:    bucketdir + "/", // Only list objects within the specific bucket directory
		Recursive: true,
	}) {
		if object.Err != nil {
			return object.Err
		}
		objectPath := filepath.Join(directory, object.Key)
		if _, err := os.Stat(objectPath); err != nil {
			if os.IsNotExist(err) {
				sub.Debug().Msgf("Removing %s from bucket %s", object.Key, s3repo.Bucket)
				// File does not exist in the repository, delete it from the bucket
				err := client.RemoveObject(context.Background(), s3repo.Bucket, object.Key, minio.RemoveObjectOptions{})
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}

package azureblob

import (
	"context"
	"os"
	"path/filepath"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/cooperspencer/gickup/logger"
	"github.com/cooperspencer/gickup/types"
)

func NewAzureBlobClient(azureblob types.AzureBlob) (*azblob.Client, error) {
	if azureblob.UseCliCredential {
		// Use Azure CLI Credential
		cred, err := azidentity.NewAzureCLICredential(nil)
		if err != nil {
			return nil, err
		}
		client, err := azblob.NewClient(azureblob.Url, cred, nil)
		return client, err
	}

	// Use anonymous credential with SAS URL
	client, err := azblob.NewClientWithNoCredential(azureblob.Url, nil)

	return client, err
}

// UploadDirToBlobStorage uploads the contents of a directory to Azure blob storage
func UploadDirToBlobStorage(directory string, blobstorage types.AzureBlob, azureblobclient *azblob.Client) error {
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
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

		// Upload the file to blob storage
		blobName := filepath.ToSlash(path[len(directory)+1:]) // Blob name in container

		_, err = azureblobclient.UploadFile(context.Background(), blobstorage.Container, blobName, file, &azblob.UploadFileOptions{})
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

// DeleteObjectsNotInRepo deletes objects from the container that are not present in the repository
func DeleteObjectsNotInRepo(directory string, blobdir string, blobstorage types.AzureBlob, azureblobclient *azblob.Client) error {
	sub := logger.CreateSubLogger("stage", "azureblob", "container", blobstorage.Container)
	blobprefix := blobdir + "/"

	pager := azureblobclient.NewListBlobsFlatPager(blobstorage.Container, &azblob.ListBlobsFlatOptions{
		Prefix:  &blobprefix,
		Include: azblob.ListBlobsInclude{Snapshots: false, Versions: false},
	})

	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			return err
		}

		for _, blobItem := range page.Segment.BlobItems {
			blobName := *blobItem.Name
			localPath := filepath.Join(directory, blobName)
			if _, err := os.Stat(localPath); err != nil {
				if os.IsNotExist(err) {
					// File does not exist locally, delete from blob storage
					_, err := azureblobclient.DeleteBlob(context.Background(), blobstorage.Container, blobName, &azblob.DeleteBlobOptions{
						DeleteSnapshots: to.Ptr(azblob.DeleteSnapshotsOptionTypeInclude),
					})
					if err != nil {
						sub.Error().Err(err).Msgf("Failed to delete blob %s", blobName)
						return err
					}
					sub.Info().Msgf("Deleted blob %s as it does not exist locally", blobName)
				}
			}
		}
	}

	return nil
}

package s3

import (
	"testing"

	"github.com/cooperspencer/gickup/types"
)

func TestGetCredentialsStaticCredsReturnsStaticValues(t *testing.T) {
	t.Parallel()

	s3repo := types.S3Repo{
		UseStaticCreds: true,
		AccessKey:      "test-access",
		SecretKey:      "test-secret",
		Token:          "test-token",
	}

	creds := getCredentials(s3repo)
	if creds == nil {
		t.Fatal("expected non-nil credentials")
	}

	val, err := creds.Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if val.AccessKeyID != "test-access" {
		t.Fatalf("AccessKeyID = %q, want %q", val.AccessKeyID, "test-access")
	}

	if val.SecretAccessKey != "test-secret" {
		t.Fatalf("SecretAccessKey = %q, want %q", val.SecretAccessKey, "test-secret")
	}

	if val.SessionToken != "test-token" {
		t.Fatalf("SessionToken = %q, want %q", val.SessionToken, "test-token")
	}
}

func TestGetCredentialsIAMReturnsNonNil(t *testing.T) {
	t.Parallel()

	creds := getCredentials(types.S3Repo{UseStaticCreds: false})
	if creds == nil {
		t.Fatal("expected non-nil IAM credentials")
	}
}

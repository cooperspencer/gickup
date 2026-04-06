package s3

import "testing"

func TestNormalizeEndpointHostOnly(t *testing.T) {
	t.Parallel()

	endpoint, secure, err := normalizeEndpoint("s3.eu-west-1.amazonaws.com", true)
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}

	if endpoint != "s3.eu-west-1.amazonaws.com" {
		t.Fatalf("endpoint = %q, want s3.eu-west-1.amazonaws.com", endpoint)
	}

	if !secure {
		t.Fatal("expected secure to remain true")
	}
}

func TestNormalizeEndpointHTTPSURL(t *testing.T) {
	t.Parallel()

	endpoint, secure, err := normalizeEndpoint("https://s3.eu-west-1.amazonaws.com", false)
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}

	if endpoint != "s3.eu-west-1.amazonaws.com" {
		t.Fatalf("endpoint = %q, want s3.eu-west-1.amazonaws.com", endpoint)
	}

	if !secure {
		t.Fatal("expected https scheme to force secure=true")
	}
}

func TestNormalizeEndpointHTTPURL(t *testing.T) {
	t.Parallel()

	endpoint, secure, err := normalizeEndpoint("http://minio.internal:9000", true)
	if err != nil {
		t.Fatalf("normalizeEndpoint() error = %v", err)
	}

	if endpoint != "minio.internal:9000" {
		t.Fatalf("endpoint = %q, want minio.internal:9000", endpoint)
	}

	if secure {
		t.Fatal("expected http scheme to force secure=false")
	}
}

func TestNormalizeEndpointRejectsPath(t *testing.T) {
	t.Parallel()

	_, _, err := normalizeEndpoint("https://s3.eu-west-1.amazonaws.com/mybucket", true)
	if err == nil {
		t.Fatal("expected endpoint path to be rejected")
	}
}

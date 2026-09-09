package core

import "testing"

func TestBase62Encode(t *testing.T) {
	got := Base62Encode(1)
	want := "6LAzd"

	if got == "" {
		t.Fatal("Base62Encode returned an empty string")
	}

	if got != want {
		t.Errorf("Base62Encode(1) = %q, want %q", got, want)
	}
}

func TestBase62EncodeDecodeRoundTrip(t *testing.T) {
	ids := []uint64{1, 2, 10, 100, 1000, 12345}

	for _, id := range ids {
		code := Base62Encode(id)
		got := Base62Decode(code)

		if got != id {
			t.Errorf("Round trip failed for ID %d: encoded %q, decoded %d", id, code, got)
		}
	}
}

func TestHashURL(t *testing.T) {
	url := "https://example.com"

	got := HashURL(url)

	if len(got) != 64 {
		t.Errorf("HashURL returned %d chars, want 64", len(got))
	}

	if got != HashURL(url) {
		t.Error("HashURL is not deterministic")
	}
}

func TestValidateLongURLRejectsInvalidScheme(t *testing.T) {
	_, err := ValidateLongURL("ftp://example.com")

	if err == nil {
		t.Fatal("Expected invalid scheme to be rejected")
	}
}

func TestValidateShortURL(t *testing.T) {
	shortURL := "https://example.com/abc123"

	code, err := ValidateShortURL(shortURL, "example.com")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if code != "abc123" {
		t.Errorf("code = %q, want %q", code, "abc123")
	}
}

func TestValidateShortURLRejectsWrongHost(t *testing.T) {
	shortURL := "https://evil.com/abc123"

	_, err := ValidateShortURL(shortURL, "example.com")

	if err == nil {
		t.Fatal("Expected wrong host to be rejected")
	}
}

func TestBloomFilter(t *testing.T) {
	InitBloom(1000, 0.01)
	
	BloomEnabled = true

	url := "https://example.com/bloom-test"

	if MightExistInBloom(url) {
		t.Fatal("URL should not exist before being added")
	}

	AddToBloom(url)

	if !MightExistInBloom(url) {
		t.Fatal("URL should exist after being added")
	}
}
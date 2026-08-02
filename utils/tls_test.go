package utils

import (
	"crypto/x509"
	"testing"
	"time"
)

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCert() error = %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("expected at least one certificate in chain")
	}
	if cert.PrivateKey == nil {
		t.Fatal("expected a non-nil private key")
	}

	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse generated certificate: %v", err)
	}

	now := time.Now()
	if now.Before(parsed.NotBefore) || now.After(parsed.NotAfter) {
		t.Errorf("certificate not valid at current time: NotBefore=%v NotAfter=%v now=%v", parsed.NotBefore, parsed.NotAfter, now)
	}

	found := false
	for _, name := range parsed.DNSNames {
		if name == "localhost" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected DNSNames to include localhost, got %v", parsed.DNSNames)
	}

	// Regenerating must not reuse the serial number, since colliding serials
	// would break clients that cache certs by (issuer, serial).
	cert2, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("second GenerateSelfSignedCert() error = %v", err)
	}
	parsed2, err := x509.ParseCertificate(cert2.Certificate[0])
	if err != nil {
		t.Fatalf("failed to parse second generated certificate: %v", err)
	}
	if parsed.SerialNumber.Cmp(parsed2.SerialNumber) == 0 {
		t.Error("expected distinct serial numbers across independent calls")
	}
}

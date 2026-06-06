package validation

import "testing"

func TestNormalizeSubdomainAcceptsLowercaseDNSLabel(t *testing.T) {
	got, err := NormalizeSubdomain(" My-App ")
	if err != nil {
		t.Fatalf("NormalizeSubdomain returned error: %v", err)
	}

	if got != "my-app" {
		t.Fatalf("expected my-app, got %q", got)
	}
}

func TestNormalizeSubdomainRejectsInvalidLabels(t *testing.T) {
	cases := []string{"", "-bad", "bad-", "bad_name", "bad.name", "a"}
	for _, input := range cases {
		if _, err := NormalizeSubdomain(input); err == nil {
			t.Fatalf("expected %q to be invalid", input)
		}
	}
}

func TestValidateLocalTargetAllowsLoopbackHTTPPort(t *testing.T) {
	target := LocalTarget{Host: "127.0.0.1", Port: 3000, Protocol: "http"}
	if err := ValidateLocalTarget(target, false); err != nil {
		t.Fatalf("expected target to be valid: %v", err)
	}
}

func TestValidateLocalTargetBlocksSensitivePorts(t *testing.T) {
	target := LocalTarget{Host: "127.0.0.1", Port: 5432, Protocol: "http"}
	if err := ValidateLocalTarget(target, false); err == nil {
		t.Fatal("expected sensitive port to be blocked")
	}
}

func TestValidateLocalTargetCanOverrideSensitivePortBlock(t *testing.T) {
	target := LocalTarget{Host: "127.0.0.1", Port: 5432, Protocol: "http"}
	if err := ValidateLocalTarget(target, true); err != nil {
		t.Fatalf("expected override to allow sensitive port: %v", err)
	}
}

func TestValidateLocalTargetRejectsNonLoopbackHosts(t *testing.T) {
	target := LocalTarget{Host: "192.168.1.20", Port: 3000, Protocol: "http"}
	if err := ValidateLocalTarget(target, false); err == nil {
		t.Fatal("expected non-loopback host to be rejected")
	}
}

package api

import "testing"

func TestValidateServerRoutes(t *testing.T) {
	cases := []struct {
		raw     string
		wantErr bool
	}{
		{"", false},
		{"192.168.30.0/24", false},
		{"192.168.30.0/24, 10.50.0.0/16", false},
		{"192.168.30.999/24", true},
		{"10.50.0.0", true},
		{"not-a-network", true},
		{"192.168.30.0/24,,10.50.0.0/16", false}, // empty elements are skipped, not an error
		{"0.0.0.0/0", true},
	}
	for _, c := range cases {
		problems := ValidateServerRoutes(c.raw)
		if (len(problems) > 0) != c.wantErr {
			t.Errorf("ValidateServerRoutes(%q) = %v", c.raw, problems)
		}
	}
}

func TestNormalizeServerRoutes(t *testing.T) {
	got := NormalizeServerRoutes(" 192.168.30.0/24 ,10.50.0.0/16, 192.168.30.0/24")
	if got != "192.168.30.0/24, 10.50.0.0/16" {
		t.Errorf("NormalizeServerRoutes = %q", got)
	}
}

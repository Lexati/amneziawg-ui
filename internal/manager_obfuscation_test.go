package internal

import (
	"encoding/base64"
	"testing"

	"amneziawg-web-ui/web-ui/api"
)

// The shape of a generated parameter set is the shared generator's business
// and is tested there. What belongs here are the two fields it deliberately
// leaves alone, because only the backend can fill them: the MTU it was asked
// for, and a key from a cryptographic source.
func TestGeneratedObfuscationParamsCarryTheMTUAndAKey(t *testing.T) {
	m := &Manager{Config: &AppConfig{}}

	for _, mtu := range []int{api.MinMTU, 1420, api.MaxMTU} {
		p := m.generateObfuscationParams(mtu)

		if p.MTU != mtu {
			t.Errorf("MTU = %d, want %d", p.MTU, mtu)
		}
		key, err := base64.StdEncoding.DecodeString(p.HeaderProtectionKey)
		if err != nil {
			t.Errorf("MTU %d: header protection key %q is not base64: %v", mtu, p.HeaderProtectionKey, err)
			continue
		}
		if len(key) != 32 {
			t.Errorf("MTU %d: header protection key is %d bytes, want 32", mtu, len(key))
		}
		if err := validateObfuscationParams(&p, mtu); err != nil {
			t.Errorf("MTU %d: generated params rejected: %v", mtu, err)
		}
	}
}

// Two servers must not come out with the same header protection key.
func TestGeneratedHeaderProtectionKeysDiffer(t *testing.T) {
	m := &Manager{Config: &AppConfig{}}

	first := m.generateObfuscationParams(api.MinMTU).HeaderProtectionKey
	second := m.generateObfuscationParams(api.MinMTU).HeaderProtectionKey
	if first == second {
		t.Fatalf("both servers got the same key %q", first)
	}
}

// validateObfuscationParams is a thin wrapper: the rules are the shared ones,
// and its own job is to turn them into one error the handlers can map to a
// 400 - including the MTU-dependent checks, which it used to skip entirely.
func TestValidateObfuscationParamsReportsSharedRules(t *testing.T) {
	p := ObfuscationParams{
		Jc: 8, Jmin: 8, Jmax: 80,
		S1: 50, S2: 50, S3: 50, S4: 50,
		H1: 1, H2: 2, H3: 3, H4: 4,
	}
	if err := validateObfuscationParams(&p, api.MinMTU); err != nil {
		t.Fatalf("valid parameters rejected: %v", err)
	}

	p.S1 = api.MinMTU - 147 // one past MTU-148
	if err := validateObfuscationParams(&p, api.MinMTU); err == nil {
		t.Fatal("want the MTU-dependent rule to be enforced, got no error")
	}

	if err := validateObfuscationParams(nil, api.MinMTU); err == nil {
		t.Fatal("want an error for a missing parameter set, got none")
	}
}

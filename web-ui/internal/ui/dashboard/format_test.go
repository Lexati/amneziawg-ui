package dashboard

import "testing"

func TestFormatBytes(t *testing.T) {
	cases := map[float64]string{
		0:                "0 B",
		512:              "512 B",
		1024:             "1.00 KB",
		14060:            "13.73 KB",
		1245540515:       "1.16 GB",
		-5:               "0 B",
		1 << 50 * 3000.0: "3000.00 PB",
	}
	for in, want := range cases {
		if got := formatBytes(in); got != want {
			t.Errorf("formatBytes(%v) = %q, want %q", in, got, want)
		}
	}
	if got := formatRate(14060); got != "13.73 KB/s" {
		t.Errorf("formatRate = %q", got)
	}
}

func TestFormatUsage(t *testing.T) {
	if got := formatUsage(3.1*1024*1024*1024, 7.6*1024*1024*1024); got != "3.10 / 7.60 GB" {
		t.Errorf("formatUsage = %q", got)
	}
	if got := formatPercent(0.1237); got != "12.4%" {
		t.Errorf("formatPercent = %q", got)
	}
}

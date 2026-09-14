package dashboard

import "fmt"

// units are binary - the same base ifconfig and awg use for the figures on
// the cards, so a total on a tile never disagrees with the sum of the
// interfaces below it.
var units = []string{"B", "KB", "MB", "GB", "TB", "PB"}

// formatBytes renders a byte count as "1.16 GB": two decimals from KB up,
// none for plain bytes.
func formatBytes(v float64) string {
	if v < 0 {
		v = 0
	}
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.2f %s", v, units[i])
}

// formatRate is formatBytes per second.
func formatRate(v float64) string {
	return formatBytes(v) + "/s"
}

// formatPercent renders a ratio in [0, 1] as "12.4%".
func formatPercent(ratio float64) string {
	return fmt.Sprintf("%.1f%%", 100*ratio)
}

// formatUsage renders "3.10 / 7.60 GB": both figures in the unit of the total,
// so the pair reads as one fraction.
func formatUsage(used, total float64) string {
	i := 0
	for total >= 1024 && i < len(units)-1 {
		total /= 1024
		used /= 1024
		i++
	}
	return fmt.Sprintf("%.2f / %.2f %s", used, total, units[i])
}

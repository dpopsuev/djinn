// badge.go — Numeric badge and compact number formatting.
package elements

import "fmt"

// Badge renders a labeled value: Badge("tokens", 8150) → "8.2k tokens".
func Badge(label string, value int) string {
	return CompactNumber(value) + " " + label
}

// CompactNumber formats large numbers: 1200→"1.2k", 3400000→"3.4M", 42→"42".
func CompactNumber(n int) string {
	if n < 0 {
		return "-" + CompactNumber(-n)
	}
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

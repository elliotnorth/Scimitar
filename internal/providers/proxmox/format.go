package proxmox

import "fmt"

func formatBytes(b int64) string {
	return fmt.Sprintf("%.1f GiB", float64(b)/(1<<30))
}

func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := seconds % 86400 / 3600
	minutes := seconds % 3600 / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	}
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

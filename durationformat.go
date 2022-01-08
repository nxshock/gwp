package gwp

import (
	"fmt"
	"time"
)

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)

	days := d / time.Hour / 24
	d -= days * 24 * time.Hour

	hours := d / time.Hour
	d -= hours * time.Hour

	minites := d / time.Minute
	d -= minites * time.Minute

	seconds := d / time.Second
	d -= seconds * time.Second

	var resultStr string

	if days > 0 {
		resultStr += fmt.Sprintf("%3dd", days)
	} else {
		resultStr += "    "
	}

	if hours > 0 {
		resultStr += fmt.Sprintf(" %2dh", hours)
	} else {
		resultStr += "    "
	}

	if minites > 0 {
		resultStr += fmt.Sprintf(" %2dm", minites)
	} else {
		resultStr += "    "
	}

	resultStr += fmt.Sprintf(" %2ds", seconds)

	return resultStr
}

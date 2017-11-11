package connectsvc

import "time"

func timeFormat(millis int64, format string) string {
	goTime := time.Unix(millis / 1e3, (millis % 1e3) * 1e6)
	return goTime.Format(format)
}
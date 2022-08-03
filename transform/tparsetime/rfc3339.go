package tparsetime

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// parseRFC3339Timestamp parse timestamp in RFC3339 format with fraction part of variable size
// ex: 2019-08-15T15:50:46.866915+03:00
// ex: 2019-08-15T15:50:46.866Z
func parseRFC3339Timestamp(timeStr string, timezoneCache map[string]*time.Location) (time.Time, error) {
	t := timeStr
	if t[4] != '-' || t[7] != '-' || t[10] != 'T' || t[13] != ':' || t[16] != ':' {
		return time.Now(), fmt.Errorf("invalid timestamp")
	}
	year := atoi4(t[0:4])
	month := atoi2(t[5:7])
	date := atoi2(t[8:10])
	hour := atoi2(t[11:13])
	min := atoi2(t[14:16])
	sec := atoi2(t[17:19])
	var frac float64
	fracStr, tzStr := splitFractionAndTimezone(t[19:])
	switch len(fracStr) - 1 {
	case -1:
		frac = 0.0
	case 3:
		frac = atof3(fracStr)
	case 6:
		frac = atof6(fracStr)
	case 9:
		frac = atof9(fracStr)
	default:
		f, err := strconv.ParseFloat(fracStr, 64)
		if err != nil {
			return time.Now(), fmt.Errorf("invalid fraction '%s': %w", fracStr, err)
		}
		frac = f
	}
	var location *time.Location
	if len(tzStr) > 0 {
		if loc, ok := timezoneCache[tzStr]; ok {
			location = loc
		} else {
			var layout string
			if strings.Contains(tzStr, ":") {
				layout = "Z07:00"
			} else {
				layout = "Z0700"
			}
			z, err := time.Parse(layout, tzStr)
			if err != nil {
				return time.Now(), fmt.Errorf("invalid timezone '%s': %w", tzStr, err)
			}
			tzName, tzOffset := z.Zone()
			location = time.FixedZone(tzName, tzOffset)
			timezoneCache[tzStr] = location
		}
	} else {
		location = time.Local
	}
	return time.Date(year, time.Month(month), date, hour, min, sec, int(frac*1000000000.0), location), nil
}

// splitFractionAndTimezone splits e.g. ".123+07:00" to .123 and +07:00
func splitFractionAndTimezone(s string) (string, string) {
	if len(s) > 1 && s[0] == '.' {
		i := 1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return s[:i], s[i:]
	}
	return "", s
}

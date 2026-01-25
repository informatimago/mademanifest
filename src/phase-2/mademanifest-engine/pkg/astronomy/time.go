package astronomy

import (
    "fmt"
    "github.com/bxparks/acetimego/acetime"
    "github.com/bxparks/acetimego/zonedb2025"
    "time"
)

func ConvertLocalTimeToUTC(localTime time.Time, timezone string) (time.Time, error) {
    // Initialize a zone manager using the TZDB data
    zm := acetime.NewZoneManager(zonedb2025.DataContext)

    // Lookup the time zone
    z, ok := zm.Lookup(timezone)
    if !ok {
        return time.Time{}, fmt.Errorf("timezone not found: %s", timezone)
    }

    // Create a "ZonedDateTime" from local date+time
    // (year,month,day,hour,min,sec,nsec)
    zdt := acetime.NewZonedDateTime(
        int(localTime.Year()), int(localTime.Month()), localTime.Day(),
        localTime.Hour(), localTime.Minute(), localTime.Second(), localTime.Nanosecond(),
        z)

    // Convert to UTC instant
    utcInstant := zdt.ToInstantUTC()

    // Convert that instant to standard time.Time
    return time.Unix(utcInstant.Unix(), utcInstant.Nanosecond()).UTC(), nil
}

func ConvertUTCToJulianDay(utcTime time.Time) float64 {
    utc := utcTime.UTC()
    seconds := float64(utc.Unix()) + float64(utc.Nanosecond())/1e9
    daysSinceUnixEpoch := seconds / 86400.0
    julianDay := daysSinceUnixEpoch + 2440587.5
    return julianDay
}

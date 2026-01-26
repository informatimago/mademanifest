package astronomy

import (
    "time"
    "github.com/bxparks/acetimego/acetime"
    "github.com/bxparks/acetimego/zonedb2025"
    "github.com/mshafiee/swephgo"
)

// ConvertLocalTimeToUTC converts local time to UTC using acetimego (tzdb-2025c imezone handling).
// This timezone algorithm corrects for historical timezone quirks.
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

// ConvertUTCToJulianDay converts UTC time to Julian Day using the Swiss Ephemeris
func ConvertUTCToJulianDay(utcTime time.Time) float64 {
    // Use Swiss Ephemeris to perform the Julian Day calculation properly
    jd := swephgo.JDFromUnix(utcTime.Unix())
    return jd
}

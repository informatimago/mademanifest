// package astronomy
//
// import (
// 	"fmt"
// 	"log"
// 	"time"
//
// 	"github.com/bxparks/acetimego/acetime"
// 	"github.com/bxparks/acetimego/zonedb2025"
// )

package astronomy

import (
    "fmt"
    "time"
    _ "time/tzdata" // embed TZ database
)

// ConvertLocalTimeToUTC converts a local time in a given IANA timezone
// to UTC. The input localTime should have the date and time in the local zone.
func ConvertLocalTimeToUTC(localTime time.Time, timezone string) (time.Time, error) {
    // Load the IANA location
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        return time.Time{}, fmt.Errorf("timezone not found: %s", timezone)
    }

    // Construct a time in the given location
    local := time.Date(
        localTime.Year(),
        localTime.Month(),
        localTime.Day(),
        localTime.Hour(),
        localTime.Minute(),
        localTime.Second(),
        localTime.Nanosecond(),
        loc,
    )

    // Convert to UTC
    return local.UTC(), nil
}

// // ConvertLocalTimeToUTC converts a local civil time in a named IANA timezone
// // to UTC using acetimego (TZDB 2025c).
// func ConvertLocalTimeToUTC(localTime time.Time, timezone string) (time.Time, error) {
// 	// Initialize ZoneManager (note the pointer)
// 	zm := acetime.ZoneManagerFromDataContext(&zonedb2025.DataContext)
//
// 	// Resolve timezone
// 	tz := zm.TimeZoneFromName(timezone)
// 	if tz.IsError() {
// 		return time.Time{}, fmt.Errorf("timezone not found: %s", timezone)
// 	}
//
// 	// Build PlainDateTime
// 	pdt := acetime.PlainDateTime{
// 		Year:   int16(localTime.Year()),
// 		Month:  uint8(localTime.Month()),
// 		Day:    uint8(localTime.Day()),
// 		Hour:   uint8(localTime.Hour()),
// 		Minute: uint8(localTime.Minute()),
// 		Second: uint8(localTime.Second()),
// 	}
//
// 	log.Printf("pdt = %v",pdt)
// 	// Resolve ZonedDateTime (explicit cast required)
// 	zdt := acetime.ZonedDateTimeFromPlainDateTime(
// 		&pdt, &tz, uint8(acetime.ResolvedOverlapLater))
//
// 	log.Printf("zdt = %v",zdt)
//
// 	if zdt.IsError() {
// 		return time.Time{}, fmt.Errorf("invalid local time for datetime %v timezone: %v", pdt, timezone)
// 	}
//
// 	// Convert to Unix seconds (acetime.Time → int64)
// 	unixSeconds := zdt.UnixSeconds()
//
// 	return time.Unix(int64(unixSeconds), int64(localTime.Nanosecond())).UTC(), nil
// }


// ConvertUTCToJulianDay converts UTC time to Julian Day (JD).
func ConvertUTCToJulianDay(utcTime time.Time) float64 {
    utc := utcTime.UTC()
    seconds := float64(utc.Unix()) + float64(utc.Nanosecond())/1e9
    daysSinceUnixEpoch := seconds / 86400.0
    return daysSinceUnixEpoch + 2440587.5
}

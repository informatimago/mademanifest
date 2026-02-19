package sweph

// Swiss Ephemeris object constants
const (
	SE_SUN = iota       // 0
	SE_MOON             // 1
	SE_MERCURY          // 2
	SE_VENUS            // 3
	SE_MARS             // 4
	SE_JUPITER          // 5
	SE_SATURN           // 6
	SE_URANUS           // 7
	SE_NEPTUNE          // 8
	SE_PLUTO            // 9
	SE_MEAN_NODE        // 10
	SE_TRUE_NODE        // 11
	SE_MEAN_APOG        // 12
	SE_OSCU_APOG        // 13
	SE_EARTH            // 14
	SE_CHIRON           // 15
	SE_PHOLUS           // 16
	SE_CERES            // 17
	SE_PALLAS           // 18
	SE_JUNO             // 19
	SE_VESTA            // 20
	SE_INTP_APOG        // 21
	SE_INTP_PERG        // 22
)


// Swiss Ephemeris calculation flags
const (
	SEFLG_JPLEPH    = 1          // use JPL ephemeris
	SEFLG_SWIEPH    = 2          // use SWISSEPH ephemeris
	SEFLG_MOSEPH    = 4          // use Moshier ephemeris

	SEFLG_HELCTR    = 8          // heliocentric position
	SEFLG_TRUEPOS   = 16         // true/geometric position, not apparent position
	SEFLG_J2000     = 32         // no precession, i.e. give J2000 equinox
	SEFLG_NONUT     = 64         // no nutation, i.e. mean equinox of date
	SEFLG_SPEED3    = 128        // speed from 3 positions (not recommended)
	SEFLG_SPEED     = 256        // high precision speed
	SEFLG_NOGDEFL   = 512        // turn off gravitational deflection
	SEFLG_NOABERR   = 1024       // turn off annual aberration of light

	SEFLG_ASTROMETRIC = SEFLG_NOABERR | SEFLG_NOGDEFL  // astrometric position
	SEFLG_EQUATORIAL  = 2 * 1024                        // equatorial positions
	SEFLG_XYZ         = 4 * 1024                        // cartesian coordinates
	SEFLG_RADIANS     = 8 * 1024                        // coordinates in radians
	SEFLG_BARYCTR     = 16 * 1024                       // barycentric position
	SEFLG_TOPOCTR     = 32 * 1024                       // topocentric position
	SEFLG_ORBEL_AA    = SEFLG_TOPOCTR                   // Astronomical Almanac mode

	SEFLG_TROPICAL    = 0          // tropical position (default)
	SEFLG_SIDEREAL    = 64 * 1024  // sidereal position
	SEFLG_ICRS        = 128 * 1024 // ICRS reference frame
	SEFLG_DPSIDEPS_1980     = 256 * 1024 // reproduce JPL Horizons 1962-today
	SEFLG_JPLHOR           = SEFLG_DPSIDEPS_1980
	SEFLG_JPLHOR_APPROX    = 512 * 1024   // approximate JPL Horizons
	SEFLG_CENTER_BODY       = 1024 * 1024 // position of center of body (COB)
	SEFLG_TEST_PLMOON       = (2*1024*1024 | SEFLG_J2000 | SEFLG_ICRS | SEFLG_HELCTR | SEFLG_TRUEPOS)
)

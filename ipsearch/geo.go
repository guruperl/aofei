// Package ipsearch provides functionality for IP address geolocation search.
package ipsearch

// Geo represents geographical information associated with an IP address.
// The structure includes various IDs for continent, country, state, DMA, city, ISP, and zip code,
// as well as latitude and longitude coordinates.
type Geo struct {
	ContinentID uint8   // ID of the continent
	CountryID   uint16  // ID of the country
	StateID     uint16  // ID of the state
	DmaID       uint16  // ID of the DMA (Designated Market Area)
	CityID      uint32  // ID of the city
	IspID       uint16  // ID of the ISP (Internet Service Provider)
	ZipID       uint32  // ID of the zip code
	Lat         float64 // Latitude coordinate
	Lon         float64 // Longitude coordinate
}

// ipIndex represents an index entry for an IP address range.
// It includes the start and end IP addresses, local offset and length, geographical information,
// and a local string in byte format.
type ipIndex struct {
	StartIP     uint32 // Start IP address of the range
	EndIP       uint32 // End IP address of the range
	LocalOffset uint32 // Offset for local data
	LocalLength uint32 // Length of local data
	Geo                // Embedded geographical information
	LocalString []byte // Local string in byte format
}

// PzGeo represents detailed geographical information associated with an IP address.
// It includes the basic geographical IDs and coordinates from the Geo struct,
// as well as additional descriptive fields for continent, country, state, metro, city, zip, and ISP.
type PzGeo struct {
	Geo              // Embedded geographical information
	Continent string // Name of the continent
	Country   string // Name of the country
	State     string // Name of the state
	Metro     string // Name of the metro area
	City      string // Name of the city
	Zip       string // Zip code
	Isp       string // Name of the ISP (Internet Service Provider)
}

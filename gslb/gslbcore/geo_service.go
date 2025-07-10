package gslbcore

import (
	"fmt"
	"math"
	"net/netip"

	"github.com/oschwald/geoip2-golang/v2"
)

type Location struct {
	Latitude  float64 `json:"latitude,omitzero"`
	Longitude float64 `json:"longitude,omitzero"`
}

var zeroLocation Location
var MAX_DISTANCE = 10000000.0

func NewLocationFromGeoIP2(record *geoip2.City) *Location {
	if record == nil || !record.Location.HasCoordinates() {
		return &zeroLocation
	}
	return &Location{
		Latitude:  *record.Location.Latitude,
		Longitude: *record.Location.Longitude,
	}
}

func (l Location) IsZero() bool {
	return l.Latitude == 0 && l.Longitude == 0
}

func (l Location) Distance(m *Location) float64 {
	if l.IsZero() || m.IsZero() {
		return MAX_DISTANCE
	}
	degToRad := func(deg float64) float64 {
		return deg * math.Pi / 180.0
	}
	// Haversine formula
	R := 6371.0
	lat1 := degToRad(l.Latitude)
	lat2 := degToRad(m.Latitude)
	dLat := degToRad(m.Latitude - l.Latitude)
	dLon := degToRad(m.Longitude - l.Longitude)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))

	return R * c
}

func (l Location) Nearest(locations []Location) (int, *Location) {
	if len(locations) == 0 {
		return -1, &zeroLocation
	}

	var nearest *Location
	var nearestIndex int
	minDistance := MAX_DISTANCE

	for i, loc := range locations {
		if loc.IsZero() {
			continue
		}
		distance := l.Distance(&loc)
		if distance < minDistance {
			minDistance = distance
			nearestIndex = i
			nearest = &loc
		}
	}

	if nearest == nil {
		return -1, &zeroLocation
	}
	return nearestIndex, nearest
}

func AverageLocation(locations []*Location) *Location {
	if len(locations) == 0 {
		return &zeroLocation
	}

	var sumLat, sumLon float64
	var num int
	for _, loc := range locations {
		if loc.IsZero() {
			continue
		}
		num++
		sumLat += loc.Latitude
		sumLon += loc.Longitude
	}
	if num == 0 {
		return &zeroLocation
	}
	fnum := float64(num)
	return &Location{
		Latitude:  sumLat / fnum,
		Longitude: sumLon / fnum,
	}
}

type GeoService struct {
	db *geoip2.Reader
}

func (g *GeoService) Init() error {
	var err error
	g.db, err = geoip2.Open("gslb/data/GeoLite2-City.mmdb")
	if err != nil {
		return err
	}
	return nil
}

func (g *GeoService) Close() error {
	if g.db != nil {
		return g.db.Close()
	}
	return nil
}

func (g *GeoService) City(ipAddress netip.Addr) (*geoip2.City, error) {
	if g.db == nil {
		return nil, fmt.Errorf("GeoService not initialized")
	}
	return g.db.City(ipAddress)
}

func (g *GeoService) GetLocations(Prefices []netip.Prefix) []*Location {
	locations := make([]*Location, len(Prefices))
	fmt.Println("  locations:")

	for i, prefix := range Prefices {
		record, err := g.db.City(prefix.Addr())
		if err != nil {
			locations[i] = &zeroLocation
			continue
		}
		fmt.Println("    ", record)
		locations[i] = NewLocationFromGeoIP2(record)
	}
	return locations
}

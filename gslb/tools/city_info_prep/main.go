package main

import (
	"encoding/csv"
	"encoding/gob"
	"fmt"
	"os"

	"github.com/jszwec/csvutil"
	"github.com/okitz/ncdn/gslb/gslbcore"
)

type Server struct {
	ID        uint64  `csv:"id"`
	Name      string  `csv:"name"`
	Title     string  `csv:"title"`
	Location  string  `csv:"location"`
	State     string  `csv:"state"`
	Country   string  `csv:"country"`
	StateAbbv string  `csv:"state_abbv"`
	Continent string  `csv:"continent"`
	Latitude  float64 `csv:"latitude"`
	Longitude float64 `csv:"longitude"`
}

type Ping struct {
	Source      uint64  `csv:"source"`
	Destination uint64  `csv:"destination"`
	Timestamp   string  `csv:"timestamp"`
	Min         float64 `csv:"min"`
	Avg         float64 `csv:"avg"`
	Max         float64 `csv:"max"`
	Mdev        float64 `csv:"mdev"`
}

func filterServerCSV(filePath string, names []string) ([]Server, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder, err := csvutil.NewDecoder(csv.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	var rows []Server
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("failed to decode CSV: %w", err)
	}

	nameSet := make(map[string]int)
	for i, value := range names {
		nameSet[value] = i
	}

	if len(nameSet) == 0 {
		return rows, nil // Return all rows if no names are provided
	}

	filteredRows := make([]Server, len(nameSet))
	for _, row := range rows {
		if index, exists := nameSet[row.Name]; exists {
			filteredRows[index] = row
		}
	}

	return filteredRows, nil
}

func filterPingsCSV(filePath string, values []uint64) ([]Ping, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder, err := csvutil.NewDecoder(csv.NewReader(file))
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	var rows []Ping
	if err := decoder.Decode(&rows); err != nil {
		return nil, fmt.Errorf("failed to decode CSV: %w", err)
	}

	valueSet := make(map[uint64]struct{})
	for _, value := range values {
		valueSet[value] = struct{}{}
	}

	var filteredRows []Ping
	for _, row := range rows {
		if _, exists := valueSet[row.Destination]; exists {
			filteredRows = append(filteredRows, row)
		}
	}

	return filteredRows, nil
}

func generateRTTData(serverNames []string) ([]uint64, []Ping) {
	serverFilePath := "./gslb/data/servers-2020-07-19.csv"
	pingFilePath := "./gslb/data/pings-2020-07-19-2020-07-20.csv"
	servers, err := filterServerCSV(serverFilePath, serverNames)
	if err != nil {
		panic(fmt.Errorf("failed to filter servers: %w", err))
	}
	serverIds := make([]uint64, len(servers))
	for i, server := range servers {
		serverIds[i] = server.ID
	}
	pings, err := filterPingsCSV(pingFilePath, serverIds)
	if err != nil {
		panic(fmt.Errorf("failed to filter pings: %w", err))
	}

	return serverIds, pings
}

func calculateAverageRTTMap(pings []Ping) map[uint64]map[uint64]float64 {
	// Group pings by Source and Destination
	grouped := make(map[uint64]map[uint64][]Ping)
	for _, ping := range pings {
		if _, exists := grouped[ping.Source]; !exists {
			grouped[ping.Source] = make(map[uint64][]Ping)
		}
		grouped[ping.Source][ping.Destination] = append(grouped[ping.Source][ping.Destination], ping)
	}

	// Calculate average RTT for each Source and Destination
	result := make(map[uint64]map[uint64]float64)
	for source, destinations := range grouped {
		result[source] = make(map[uint64]float64)
		for destination, group := range destinations {
			var sumRTT float64
			for _, ping := range group {
				sumRTT += ping.Avg
			}
			averageRTT := sumRTT / float64(len(group))
			result[source][destination] = averageRTT
		}
	}

	return result
}

func saveRTTArrayToFile(rttArray [][]float64, filePath string) {
	file, err := os.Create(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to create file: %w", err))
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(rttArray); err != nil {
		panic(fmt.Errorf("failed to encode RTT map: %w", err))
	}
}

func printRTTArray(filePath string, serverIds []uint64) {
	rttArray, _ := gslbcore.LoadCityRTTInfos(filePath)
	fmt.Printf("RTT Map from %s:\n", filePath)
	for i, row := range rttArray {
		fmt.Printf("Source %d: ", i)
		for j, rtt := range row {
			fmt.Printf("Dst %d: %f ", serverIds[j], rtt)
		}
		fmt.Println()
	}
}

func convertRTTMapToArray(rttMap map[uint64]map[uint64]float64, popServerIds []uint64) [][]float64 {
	// Determine the maximum source ID for array size
	maxSourceID := uint64(0)
	for source := range rttMap {
		if source > maxSourceID {
			maxSourceID = source
		}
	}

	// Initialize the result array
	result := make([][]float64, maxSourceID+1)
	for i := range result {
		result[i] = make([]float64, len(popServerIds))
	}

	// Populate the result array
	for source, destinations := range rttMap {
		for i, destination := range popServerIds {
			if rtt, exists := destinations[destination]; exists {
				result[source][i] = rtt
			} else {
				result[source][i] = 0
			}
		}
	}

	return result
}

func saveServerLocationArray() {
	serverRecords, err := filterServerCSV("./gslb/data/servers-2020-07-19.csv", nil)
	if err != nil {
		panic(fmt.Errorf("failed to filter server records: %w", err))
	}
	maxID := uint64(0)
	for _, record := range serverRecords {
		if record.ID > maxID {
			maxID = record.ID
		}
	}

	locations := make([]gslbcore.Location, maxID+1)
	for _, record := range serverRecords {
		idx := record.ID
		locations[idx] = gslbcore.Location{
			Latitude:  record.Latitude,
			Longitude: record.Longitude,
		}
	}

	file, err := os.Create("./gslb/data/server_locations.gob")
	if err != nil {
		panic(fmt.Errorf("failed to create file: %w", err))
	}
	defer file.Close()

	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(locations); err != nil {
		panic(fmt.Errorf("failed to encode server locations: %w", err))
	}
}

func printServerLocationArray(filePath string) {
	locations, _ := gslbcore.LoadCityLocations(filePath)
	fmt.Printf("Server Locations from %s:\n", filePath)
	for i, loc := range locations {
		fmt.Printf("Server %d: Latitude: %f, Longitude: %f\n", i, loc.Latitude, loc.Longitude)
	}
}

var (
	popServerNames   = []string{"Singapore", "NewYork", "SanFrancisco", "Toronto", "Frankfurt"}
	probeServerNames = []string{"Singapore", "SanFrancisco", "Bangalore", "Sydney", "Amsterdam"}
	popServerIds     []uint64
	probeServerIds   []uint64
	popPings         []Ping
	probePings       []Ping
	popRTTMap        map[uint64]map[uint64]float64
	probeRTTMap      map[uint64]map[uint64]float64
	popRTTArray      [][]float64
	probeRTTArray    [][]float64
)

func generate() error {
	popServerIds, popPings = generateRTTData(popServerNames)
	popRTTMap = calculateAverageRTTMap(popPings)
	popRTTArray = convertRTTMapToArray(popRTTMap, popServerIds)
	saveRTTArrayToFile(popRTTArray, "./gslb/data/pop_rtt_map.gob")

	probeServerIds, probePings = generateRTTData(probeServerNames)
	probeRTTMap = calculateAverageRTTMap(probePings)
	probeRTTArray = convertRTTMapToArray(probeRTTMap, probeServerIds)
	saveRTTArrayToFile(probeRTTArray, "./gslb/data/probe_rtt_map.gob")

	saveServerLocationArray()
	return nil
}

func main() {
	if err := generate(); err != nil {
		fmt.Printf("エラー: %v\n", err)
	}

	printRTTArray("./gslb/data/pop_rtt_map.gob", popServerIds)
	printRTTArray("./gslb/data/probe_rtt_map.gob", probeServerIds)
	printServerLocationArray("./gslb/data/server_locations.gob")
}

package gslbcore

import (
	"encoding/gob"
	"fmt"
	"os"
)

func LoadCityRTTInfos(filePath string) ([][]float64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var rttArray [][]float64
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&rttArray); err != nil {
		return nil, fmt.Errorf("failed to decode RTT map: %w", err)
	}

	return rttArray, nil
}

func LoadCityLocations(filePath string) ([]Location, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var locations []Location
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&locations); err != nil {
		return nil, fmt.Errorf("failed to decode server locations: %w", err)
	}

	return locations, nil
}

package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

type GeoResult struct {
	Lat string `json:"lat"`
	Lon string `json:"lon"`
}

type StationInfo struct {
	Name string
	Lat  string
	Lon  string
}

var cleanedNameCache = make(map[string]string)

func cleanName(name string) string {
	// Normalize common variants like Friedrichstr. vs Friedrichstraße
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "ß", "ss")
	name = strings.ReplaceAll(name, ".", "")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ToLower(name)
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ") // normalize whitespace
	return name
}

func geocodeStation(name string) (string, string, error) {
	client := resty.New()
	time.Sleep(1000 * time.Millisecond) // be polite to Nominatim

	query := name
	lower := strings.ToLower(name)

	if strings.HasPrefix(lower, "berlin") ||
		strings.HasPrefix(lower, "s ") ||
		strings.HasPrefix(lower, "u ") ||
		strings.HasPrefix(lower, "s+u ") {
		query += ", Berlin, Germany"
	} else {
		query += ", Germany"
	}

	resp, err := client.R().
		SetQueryParams(map[string]string{
			"format": "json",
			"q":      query,
		}).
		SetHeader("User-Agent", "GoStationGeocoder/1.0").
		Get("https://nominatim.openstreetmap.org/search")

	if err != nil {
		return "", "", err
	}

	var results []GeoResult
	err = json.Unmarshal(resp.Body(), &results)
	if err != nil || len(results) == 0 {
		return "", "", fmt.Errorf("no result for %s", name)
	}

	return results[0].Lat, results[0].Lon, nil
}

func main() {
	outputFile, err := os.Create("data/berlin.csv")
	if err != nil {
		log.Fatal("Failed to create output file:", err)
	}
	defer outputFile.Close()

	writer := csv.NewWriter(outputFile)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"line_name", "station_name", "lat", "lng"})

	stationCache := make(map[string]StationInfo) // cleaned name => geo info

	err = filepath.Walk("data/stations-berlin/", func(path string, info fs.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".csv") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		fmt.Printf("Reading file : %s", path)
		r := csv.NewReader(file)
		r.FieldsPerRecord = -1 // allow flexible row lengths

		records, err := r.ReadAll()
		if err != nil {
			return err
		}

		lineName := strings.TrimSuffix(filepath.Base(path), ".csv")

		for _, row := range records {
			if len(row) == 0 {
				continue
			}

			stationName := strings.TrimSpace(row[0])
			cleaned := cleanName(stationName)

			var lat, lon string
			if cached, ok := stationCache[cleaned]; ok {
				lat = cached.Lat
				lon = cached.Lon
			} else {
				fmt.Println("... Geocoding:", stationName)
				lat, lon, err = geocodeStation(stationName)
				if err != nil {
					fmt.Println("... Warning: could not geocode", stationName)
					continue
				}
				stationCache[cleaned] = StationInfo{stationName, lat, lon}
			}

			// Write line to output
			writer.Write([]string{lineName, stationName, lat, lon})
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Done! Output saved to data/berlin.csv")
}

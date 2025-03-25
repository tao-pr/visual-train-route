package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

// Data models
type Station struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

type Line struct {
	Name     string    `json:"name"`
	Color    string    `json:"color"`
	Stations []Station `json:"stations"`
}

func parseCSV(path string) ([]Line, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	lines := map[string]*Line{}
	// lineColors := map[string]string{}

	// Skip header
	for i, row := range records {
		if i == 0 {
			continue
		}
		if len(row) < 4 {
			continue
		}

		lineName := strings.ToLower(row[0])
		stationName := row[1]
		lat := parseFloat(row[2])
		lng := parseFloat(row[3])

		if _, exists := lines[lineName]; !exists {
			lines[lineName] = &Line{
				Name:  lineName,
				Color: randomColor(),
			}
		}
		lines[lineName].Stations = append(lines[lineName].Stations, Station{
			Name: stationName,
			Lat:  lat,
			Lng:  lng,
		})
	}

	// Convert to slice
	var allLines []Line
	for _, l := range lines {
		allLines = append(allLines, *l)
	}

	return allLines, nil
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0.0
	}
	return v
}

func randomColor() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("#%06x", rand.Intn(0xffffff))
}

func main() {
	engine := html.New("./views", ".html")
	engine.AddFunc("json", func(v interface{}) string {
		data, _ := json.Marshal(v)
		return string(data)
	})

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	lines, err := parseCSV("data/berlin.csv")
	if err != nil {
		log.Fatalf("Failed to parse CSV: %v", err)
	}

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Lines": lines,
		})
	})

	log.Fatal(app.Listen(":3000"))
}

package metadata

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	vdf "github.com/andygrunwald/vdf"
)

type MapMetadata struct {
	X      float64 `json:"pos_x,string"`
	Y      float64 `json:"pos_y,string"`
	Scale  float64 `json:"scale,string"`
	Rotate int     `json:"rotate,string"`
}

func GetMapMetadata(name string) MapMetadata {
	f, err := os.Open(fmt.Sprintf("assets/metadata/%s.txt", name))
	if err != nil {
		log.Panicf("Failed to open map metadata file: %v", err)
	}

	defer f.Close()

	m, err := vdf.NewParser(f).Parse()
	if err != nil {
		log.Panicf("Failed to parse map metadata file: %v", err)
	}

	b, err := json.Marshal(m)
	if err != nil {
		log.Panicf("Failed to marshal map metadata: %v", err)
	}

	var data map[string]MapMetadata

	err = json.Unmarshal(b, &data)
	if err != nil {
		log.Panicf("Failed to unmarshal map metadata: %v", err)
	}

	mapInfo, ok := data[name]
	if !ok {
		log.Panicf("Failed to get map info.json entry for %q", name)
	}

	return mapInfo
}

// TODO: Handle rotation
func (mapMetadata MapMetadata) WorldToRadarCoords(x, y float64) (float64, float64) {
	x, y = x-mapMetadata.X, mapMetadata.Y-y
	return x / mapMetadata.Scale, y / mapMetadata.Scale
}

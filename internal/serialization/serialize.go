package serialization

import (
	"encoding/json"
	"os"
	"topdown/internal/metadata"
	"topdown/internal/replay"
)

type PlayerPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type NadePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type FrameData struct {
	PlayerPositions map[int]PlayerPosition `json:"player_positions"` // Key: playerID, Val: PlayerPosition for that tick
	NadePositions   map[int64]NadePosition `json:"nade_positions"`   // Key: nadeID, Val: NadePosition for that tick
}

// type SerializedRound struct {
// 	StartTick int `json:"start_tick"`
// 	EndTick   int `json:"end_tick"`

// 	// PlayerPositions is a map where the key is the UserID and the value is a list of PlayerPosition during the round.
// 	// PlayerPositions map[int][]PlayerPosition `json:"player_positions"`
// 	FrameDatum []FrameData `json:"frame_data"` // Index: Tick, Val: FrameData for that tick
// }

type Replay struct {
	MapName        string                          `json:"mapName"`
	TickRate       float64                         `json:"tickRate"`
	PlayerMetadata map[int]metadata.PlayerMetadata `json:"playerMetadata"` // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[int64]metadata.NadeMetadata `json:"nadeMetadata"`   // Key: nadeId, Val: NadeMetadata
	Rounds         [][]FrameData                   `json:"rounds"`
}

func SerializeReplay(replay *replay.Replay, path string) error {
	file, _ := os.Create(path)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Create a SerializedReplay struct from the replay handler
	serializedReplay := Replay{
		MapName:        replay.MapName,
		TickRate:       replay.TickRate,
		PlayerMetadata: replay.PlayerMetadata,
		NadeMetadata:   replay.NadeMetadata,
		Rounds:         [][]FrameData{},
	}
	for _, round := range replay.Rounds {

		serializedRound := make([]FrameData, 0, len(round))

		for _, frameData := range round {

			newPlayerPositions := make(map[int]PlayerPosition)
			newNadePositions := make(map[int64]NadePosition)

			for playerID, pos := range frameData.PlayerPositions {
				newPlayerPositions[playerID] = PlayerPosition{
					X: pos.X,
					Y: pos.Y,
				}
			}

			for nadeID, pos := range frameData.NadePositions {
				newNadePositions[nadeID] = NadePosition{
					X: pos.X,
					Y: pos.Y,
				}
			}

			serializedRound = append(serializedRound, FrameData{
				PlayerPositions: newPlayerPositions,
				NadePositions:   newNadePositions,
			})
		}

		serializedReplay.Rounds = append(serializedReplay.Rounds, serializedRound)
	}

	err := encoder.Encode(serializedReplay)
	if err != nil {
		return err
	}

	return nil
}

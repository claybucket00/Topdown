package serialization

import (
	"encoding/json"
	"os"
	"topdown/internal/replayhandler"
)

type PlayerPosition struct {
	Tick int     `json:"tick"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type SerializedRound struct {
	Number    int `json:"number"`
	StartTick int `json:"start_tick"`
	EndTick   int `json:"end_tick"`

	// PlayerPositions is a map where the key is the UserID and the value is a list of PlayerPosition during the round.
	PlayerPositions map[int][]PlayerPosition `json:"player_positions"`
}

type Replay struct {
	MapName  string            `json:"mapName"`
	TickRate float64           `json:"tickRate"`
	Rounds   []SerializedRound `json:"rounds"`
}

func SerializeReplay(replay *replayhandler.ReplayHandler, path string) error {
	file, _ := os.Create(path)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Create a SerializedReplay struct from the replay handler
	serializedReplay := Replay{
		MapName:  replay.MapName,
		TickRate: replay.TickRate,
		Rounds:   []SerializedRound{},
	}

	for _, round := range replay.Rounds {
		serializedRound := SerializedRound{
			Number:          round.Number,
			StartTick:       round.StartTick,
			EndTick:         round.EndTick,
			PlayerPositions: map[int][]PlayerPosition{},
		}

		for playerID, playerPositions := range round.PlayerPositions {
			serializedPlayerPositions := []PlayerPosition{}
			for _, position := range playerPositions {
				serializedPlayerPositions = append(serializedPlayerPositions, PlayerPosition{
					Tick: position.Tick,
					X:    position.X,
					Y:    position.Y,
				})
			}
			serializedRound.PlayerPositions[playerID] = serializedPlayerPositions
		}

		serializedReplay.Rounds = append(serializedReplay.Rounds, serializedRound)
	}

	err := encoder.Encode(serializedReplay)
	if err != nil {
		return err
	}

	return nil
}

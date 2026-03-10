package replay

import (
	"encoding/json"
	"os"
	frames "topdown/internal/frames"
	metadata "topdown/internal/metadata"

	ulid "github.com/oklog/ulid/v2"
)

type Replay struct {
	MapName        string                              `json:"mapName"`
	TickRate       float64                             `json:"tickRate"`
	PlayerMetadata map[int]metadata.PlayerMetadata     `json:"playerMetadata"` // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[ulid.ULID]metadata.NadeMetadata `json:"nadeMetadata"`   // Key: nadeId, Val: NadeMetadata
	Rounds         [][]frames.FrameData                `json:"rounds"`
}

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:        rh.MapName,
		TickRate:       rh.TickRate,
		PlayerMetadata: rh.PlayerMetadata,
		NadeMetadata:   rh.NadeMetadata,
		Rounds:         make([][]frames.FrameData, len(rh.Rounds)),
	}

	for i, round := range rh.Rounds {
		// log.Printf("Processing round %d start=%d end=%d\n",
		// 	i, round.StartTick, round.EndTick)

		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			frameData, exists := rh.Frames[tick]
			if exists {
				roundFrames = append(roundFrames, *frameData)
			}
			// if frameData, exists := rh.Frames[tick]; exists {
			// 	roundFrames = append(roundFrames, *frameData)
			// }
		}

		// log.Printf("Round %d frames: %d\n", i, len(roundFrames))
		replay.Rounds[i] = roundFrames
	}
	return replay
}

func (r *Replay) SerializeReplay(path string) error {
	file, _ := os.Create(path)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Create a SerializedReplay struct from the replay handler
	// serializedReplay := Replay{
	// 	MapName:        r.MapName,
	// 	TickRate:       r.TickRate,
	// 	PlayerMetadata: r.PlayerMetadata,
	// 	NadeMetadata:   r.NadeMetadata,
	// 	Rounds:         [][]frames.FrameData{},
	// }
	// for _, round := range replay.Rounds {

	// 	serializedRound := make([]frames.FrameData, 0, len(round))

	// 	for _, frameData := range round {

	// 		newPlayerPositions := make(map[int]playerposition.PlayerPosition)
	// 		newNadePositions := make(map[int64]playerposition.NadePosition)

	// 		for playerID, pos := range frameData.PlayerPositions {
	// 			newPlayerPositions[playerID] = playerposition.PlayerPosition{
	// 				X: pos.X,
	// 				Y: pos.Y,
	// 			}
	// 		}

	// 		for nadeID, pos := range frameData.NadePositions {
	// 			// println("Serialized data for:", nadeID)
	// 			newNadePositions[nadeID] = playerposition.NadePosition{
	// 				X: pos.X,
	// 				Y: pos.Y,
	// 			}
	// 		}

	// 		serializedRound = append(serializedRound, frames.FrameData{
	// 			PlayerPositions: newPlayerPositions,
	// 			NadePositions:   newNadePositions,
	// 		})
	// 	}

	// 	serializedReplay.Rounds = append(serializedReplay.Rounds, serializedRound)
	// }

	err := encoder.Encode(r)
	if err != nil {
		return err
	}

	return nil
}

func (r *Replay) PrintNadeData() {
	for roundIndex, round := range r.Rounds {
		nadeData := 0
		for _, frameData := range round {
			nadeData += len(frameData.NadePositions)
		}
		println("Round", roundIndex, "Nade data:", nadeData)
	}
}

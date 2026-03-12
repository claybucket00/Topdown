package replay

import (
	"encoding/json"
	"os"
	events "topdown/internal/events"
	frames "topdown/internal/frames"
	metadata "topdown/internal/metadata"

	ulid "github.com/oklog/ulid/v2"
)

type Replay struct {
	MapName        string                              `json:"mapName"`
	TickRate       float64                             `json:"tickRate"`
	PlayerMetadata map[int]metadata.PlayerMetadata     `json:"playerMetadata"` // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[ulid.ULID]metadata.NadeMetadata `json:"nadeMetadata"`   // Key: nadeId, Val: NadeMetadata
	RoundMetadata  []metadata.RoundMetadata            `json:"roundMetadata"`  // Slice of RoundMetadata indexed by round number (0-based)
	Rounds         [][]frames.FrameData                `json:"rounds"`
	Events         [][]events.GameEvent                `json:"events"`
}

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:        rh.MapName,
		TickRate:       rh.TickRate,
		PlayerMetadata: rh.PlayerMetadata,
		NadeMetadata:   rh.NadeMetadata,
		Events:         make([][]events.GameEvent, len(rh.Rounds)),
		RoundMetadata:  make([]metadata.RoundMetadata, len(rh.Rounds)),
		Rounds:         make([][]frames.FrameData, len(rh.Rounds)),
	}

	eventIndex := 0
	for i, round := range rh.Rounds {
		// log.Printf("Processing round %d start=%d end=%d\n",
		// 	i, round.StartTick, round.EndTick)

		replay.RoundMetadata[i] = metadata.RoundMetadata{
			Score:         round.Score,
			PlayerToTeams: round.PlayerTeams, // TODO: Could convert to slices instead of maps, however players are not guaranteed to be 0-9 due to bots and spectators
		}
		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			frameData, exists := rh.Frames[tick]
			if exists {
				roundFrames = append(roundFrames, *frameData)
			}
		}

		roundEvents := make([]events.GameEvent, 0)
		if eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick < round.StartTick {
			eventIndex += (round.StartTick - rh.Events[eventIndex].Tick) // Fast forward to the start tick of the round
		}

		for eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick <= round.EndTick {
			rh.Events[eventIndex].Tick = rh.Events[eventIndex].Tick - round.StartTick // Convert to 0-based tick for the round
			roundEvents = append(roundEvents, rh.Events[eventIndex])
			eventIndex++
		}

		replay.Rounds[i] = roundFrames
		replay.Events[i] = roundEvents
	}
	return replay
}

func (r *Replay) SerializeReplay(path string) error {
	file, _ := os.Create(path)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

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

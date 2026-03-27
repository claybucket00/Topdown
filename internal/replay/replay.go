package replay

import (
	"encoding/json"
	"os"
	events "topdown/internal/events"
	frames "topdown/internal/frames"
	metadata "topdown/internal/metadata"

	ulid "github.com/oklog/ulid/v2"
)

// Need to create snapshots for replay scrubbing. Need to store:
// Players: health, armor, team
// Blooms: x, y, type, timeRemaining
// Infernos: x, y, timeRemaining
// Players Flashed: playerId, timeRemaining
// Players Equipment: playerId, equipment
// TODO: Maybe need to store killfeed state as well, if we want perfect scrubbing.

type playerSnapshot struct {
	PlayerID int
	Health   int
	Armor    int
	Team     int
}

type bloomSnapshot struct {
	NadeID        ulid.ULID
	X             float64
	Y             float64
	Type          string
	TimeRemaining int
}

type infernoSnapshot struct {
	X             float64
	Y             float64
	TimeRemaining int
}

type flashedSnapshot struct {
	PlayerID      int
	TimeRemaining int
}

type equipmentSnapshot struct {
	PlayerID  int
	Equipment []string
}

type Snapshot struct {
	playerSnapshots    []playerSnapshot
	bloomSnapshots     []bloomSnapshot
	infernoSnapshots   []infernoSnapshot
	flashedSnapshots   []flashedSnapshot
	equipmentSnapshots []equipmentSnapshot
}

func (r *Snapshot) updateSnapshot(event events.GameEvent) {
	switch event.Type {
	case events.EventFlash:
	// Update flashedSnapshots
	case events.EventSmokeStart:
		// Update bloomSnapshots
	case events.EventSmokeEnd:
		// Update bloomSnapshots
	case events.EventInferno:
		// Update infernoSnapshots
	case events.EventPlayerFlashed:
		// Update flashedSnapshots
	case events.EventEquipmentUpdate:
		// Update equipmentSnapshots
	}
}

type Replay struct {
	MapName        string                              `json:"mapName"`
	TickRate       float64                             `json:"tickRate"`
	PlayerMetadata map[int]metadata.PlayerMetadata     `json:"playerMetadata"` // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[ulid.ULID]metadata.NadeMetadata `json:"nadeMetadata"`   // Key: nadeId, Val: NadeMetadata
	RoundMetadata  []metadata.RoundMetadata            `json:"roundMetadata"`  // Slice of RoundMetadata indexed by round number (0-based)
	Rounds         [][]frames.FrameData                `json:"rounds"`
	Events         [][]events.GameEvent                `json:"events"`
	Snapshots      []Snapshot                          `json:"snapshots"`
}

var SNAPSHOT_INTERVAL = 256

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:        rh.MapName,
		TickRate:       rh.GetTickRate(),
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

		playerSnapshots := []playerSnapshot{}
		bloomSnapshots := []bloomSnapshot{}
		infernoSnapshots := []infernoSnapshot{}
		flashedSnapshots := []flashedSnapshot{}
		equipmentSnapshots := []equipmentSnapshot{}

		replay.RoundMetadata[i] = metadata.RoundMetadata{
			Score:             round.Score,
			PlayerToTeams:     round.PlayerTeams, // TODO: Could convert to slices instead of maps, however players are not guaranteed to be 0-9 due to bots and spectators
			PlayerToEquipment: round.PlayerToEquipment,
		}
		roundEvents := make([]events.GameEvent, 0)
		if eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick < round.StartTick {
			eventIndex += (round.StartTick - rh.Events[eventIndex].Tick) // Fast forward to the start tick of the round
		}
		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			frameData, exists := rh.Frames[tick]
			if exists {
				roundFrames = append(roundFrames, *frameData)
			}
			for eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick == tick {
				rh.Events[eventIndex].Tick = rh.Events[eventIndex].Tick - round.StartTick // Convert to 0-based tick for the round
				roundEvents = append(roundEvents, rh.Events[eventIndex])
				eventIndex++
			}
			if tick%SNAPSHOT_INTERVAL == 0 {
				// Append snapshot
			}
		}

		// for eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick <= round.EndTick {
		// 	rh.Events[eventIndex].Tick = rh.Events[eventIndex].Tick - round.StartTick // Convert to 0-based tick for the round
		// 	roundEvents = append(roundEvents, rh.Events[eventIndex])
		// 	eventIndex++
		// }

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

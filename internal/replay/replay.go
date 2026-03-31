package replay

import (
	"encoding/json"
	"os"
	events "topdown/internal/events"
	frames "topdown/internal/frames"
	metadata "topdown/internal/metadata"
	player "topdown/internal/playerposition"
	playerposition "topdown/internal/playerposition"
	utility "topdown/internal/utility"

	r2 "github.com/golang/geo/r2"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	ulid "github.com/oklog/ulid/v2"
)

// Need to create snapshots for replay scrubbing. Need to store:
// Players: health, armor, team
// Blooms: x, y, type, timeRemaining
// Infernos: x, y, timeRemaining
// Players Flashed: playerId, timeRemaining
// Players Equipment: playerId, equipment
// TODO: Maybe need to store killfeed state as well, if we want perfect scrubbing.

type PlayerSnapshot struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Yaw       float32 `json:"yaw"`
	Health    int
	Armor     int
	Team      common.Team
	Money     int
	Equipment []string
}

type BloomSnapshot struct {
	X        float64
	Y        float64
	Type     string
	Duration float64
}

// type infernoSnapshot struct {
// 	Points        []r2.Point
// 	TimeRemaining int
// }

type FlashedSnapshot struct {
	TimeRemaining float64 `json:"timeRemaining"`
}

type Snapshot struct {
	Tick             int
	PlayerSnapshots  map[player.PlayerID]PlayerSnapshot
	BloomSnapshots   map[ulid.ULID]BloomSnapshot
	InfernoSnapshots map[int64][]r2.Point
	FlashedSnapshots map[player.PlayerID]FlashedSnapshot
	BombSnapshot     *r2.Point `json:"bombSnapshot,omitempty"`
}

func (snap *Snapshot) updateSnapshot(event events.GameEvent) {
	switch event.Type {
	case events.EventFlash:
		// Update bloomSnapshots
		if eventData, ok := event.Data.(events.GrenadeEvent); ok {
			snap.BloomSnapshots[eventData.NadeId] = BloomSnapshot{
				X:        eventData.X,
				Y:        eventData.Y,
				Type:     "Flashbang",
				Duration: 500.0,
			}
		}
	case events.EventSmokeStart:
		// Update bloomSnapshots
		if eventData, ok := event.Data.(events.SmokeEvent); ok {
			snap.BloomSnapshots[eventData.NadeId] = BloomSnapshot{
				X:        eventData.X,
				Y:        eventData.Y,
				Type:     "Smoke Grenade", // TODO: Refactor smoke grenades into general nade structure.
				Duration: 18000.0,         // Used to track duration left in front end.
			}
		}
	case events.EventSmokeEnd:
		// Update bloomSnapshots
		if eventData, ok := event.Data.(events.SmokeEvent); ok {
			bloomToRemove := eventData.NadeId
			delete(snap.BloomSnapshots, bloomToRemove)
		}
	case events.EventHe:
		if eventData, ok := event.Data.(events.GrenadeEvent); ok {
			snap.BloomSnapshots[eventData.NadeId] = BloomSnapshot{
				X:        eventData.X,
				Y:        eventData.Y,
				Type:     "HE Grenade",
				Duration: 500.0,
			}
		}
	case events.EventTeamChange:
		if eventData, ok := event.Data.(events.TeamChangeEvent); ok {
			playerSnapshot := snap.PlayerSnapshots[playerposition.PlayerID(eventData.PlayerID)]
			playerSnapshot.Team = eventData.Team
			snap.PlayerSnapshots[playerposition.PlayerID(eventData.PlayerID)] = playerSnapshot
		}
	case events.EventInferno:
		// Update infernoSnapshots
		if eventData, ok := event.Data.(events.InfernoEvent); ok {
			snap.InfernoSnapshots[eventData.NadeId] = eventData.Points
		}
	case events.EventDamage:
		if eventData, ok := event.Data.(events.DamageEvent); ok {
			playerSnapshot := snap.PlayerSnapshots[*eventData.PlayerID]
			playerSnapshot.Health = eventData.Health
			playerSnapshot.Armor = eventData.Armor
			snap.PlayerSnapshots[*eventData.PlayerID] = playerSnapshot

		}
	case events.EventPlayerFlashed:
		// Update flashedSnapshots
		if eventData, ok := event.Data.(events.FlashEvent); ok {
			snap.FlashedSnapshots[*eventData.PlayerID] = FlashedSnapshot{
				TimeRemaining: float64(eventData.Duration),
			}
		}
	case events.EventEquipmentUpdate:
		// Update equipmentSnapshots
		if eventData, ok := event.Data.(events.EquipmentEvent); ok {
			playerSnapshot := snap.PlayerSnapshots[*eventData.PlayerID]
			playerSnapshot.Money = eventData.Money
			playerSnapshot.Equipment = *eventData.Equipment
			snap.PlayerSnapshots[*eventData.PlayerID] = playerSnapshot
		}
	case events.EventBombDropped:
		if eventData, ok := event.Data.(events.BombDroppedEvent); ok {
			snap.BombSnapshot = utility.Ptr[r2.Point](eventData.Position)
		}

	}
}

func (snap *Snapshot) tickBlooms(timeElapsed float64) {
	for id, bloom := range snap.BloomSnapshots {
		bloom.Duration -= timeElapsed
		if bloom.Duration <= 0 {
			delete(snap.BloomSnapshots, id)
		} else {
			snap.BloomSnapshots[id] = bloom
		}
	}
}

func (snap *Snapshot) tickFlashedPlayers(timeElapsed float64) {
	for id, flashedSnapshot := range snap.FlashedSnapshots {
		flashedSnapshot.TimeRemaining -= timeElapsed
		if flashedSnapshot.TimeRemaining <= 0 {
			delete(snap.FlashedSnapshots, id)
		} else {
			snap.FlashedSnapshots[id] = flashedSnapshot
		}
	}
}

func (snap *Snapshot) resetInfernos() {
	snap.InfernoSnapshots = make(map[int64][]r2.Point)
}

func (snap *Snapshot) Copy() Snapshot {
	newSnap := Snapshot{
		Tick:             snap.Tick,
		PlayerSnapshots:  make(map[player.PlayerID]PlayerSnapshot),
		BloomSnapshots:   make(map[ulid.ULID]BloomSnapshot),
		InfernoSnapshots: make(map[int64][]r2.Point),
		FlashedSnapshots: make(map[player.PlayerID]FlashedSnapshot),
		BombSnapshot:     snap.BombSnapshot,
	}
	for playerId, playerSnapshot := range snap.PlayerSnapshots {
		newSnap.PlayerSnapshots[playerId] = playerSnapshot
	}
	for bloomId, bloomSnapshot := range snap.BloomSnapshots {
		newSnap.BloomSnapshots[bloomId] = bloomSnapshot
	}
	for infernoId, infernoSnapshot := range snap.InfernoSnapshots {
		newSnap.InfernoSnapshots[infernoId] = infernoSnapshot
	}
	for flashedPlayerId, flashedSnapshot := range snap.FlashedSnapshots {
		newSnap.FlashedSnapshots[flashedPlayerId] = flashedSnapshot
	}
	return newSnap
}

type Replay struct {
	MapName        string                              `json:"mapName"`
	TickRate       float64                             `json:"tickRate"`
	PlayerMetadata map[int]metadata.PlayerMetadata     `json:"playerMetadata"` // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[ulid.ULID]metadata.NadeMetadata `json:"nadeMetadata"`   // Key: nadeId, Val: NadeMetadata
	RoundMetadata  []metadata.RoundMetadata            `json:"roundMetadata"`  // Slice of RoundMetadata indexed by round number (0-based)
	Rounds         [][]frames.FrameData                `json:"rounds"`
	Events         [][]events.GameEvent                `json:"events"`
	Snapshots      [][]Snapshot                        `json:"snapshots"`
}

var SNAPSHOT_INTERVAL = 256
var BLOOM_DURATION = 500 // Bloom duration in ms for flashes, HEs, decoys

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:        rh.MapName,
		TickRate:       rh.GetTickRate(),
		PlayerMetadata: rh.PlayerMetadata,
		NadeMetadata:   rh.NadeMetadata,
		Events:         make([][]events.GameEvent, len(rh.Rounds)),
		RoundMetadata:  make([]metadata.RoundMetadata, len(rh.Rounds)),
		Rounds:         make([][]frames.FrameData, len(rh.Rounds)),
		Snapshots:      make([][]Snapshot, len(rh.Rounds)),
	}

	tickDuration := 1000 / replay.TickRate // ms per tick.
	timeElasped := 0.0

	eventIndex := 0
	for i, round := range rh.Rounds {
		// log.Printf("Processing round %d start=%d end=%d\n",
		// 	i, round.StartTick, round.EndTick)

		// playerSnapshots := []playerSnapshot{}
		// bloomSnapshots := []bloomSnapshot{}
		// infernoSnapshots := make(map[int64]infernoSnapshot)
		// flashedSnapshots := []flashedSnapshot{}
		// equipmentSnapshots := []equipmentSnapshot{}
		replay.RoundMetadata[i] = metadata.RoundMetadata{
			Score:             round.Score,
			PlayerToTeams:     round.PlayerTeams, // TODO: Could convert to slices instead of maps, however players are not guaranteed to be 0-9 due to bots and spectators
			PlayerToEquipment: round.PlayerToEquipment,
		}

		snapshot := Snapshot{
			PlayerSnapshots:  make(map[playerposition.PlayerID]PlayerSnapshot),
			BloomSnapshots:   make(map[ulid.ULID]BloomSnapshot),
			InfernoSnapshots: make(map[int64][]r2.Point),
			FlashedSnapshots: make(map[player.PlayerID]FlashedSnapshot),
			BombSnapshot:     nil,
		}

		for playerID, playerEquipment := range round.PlayerToEquipment {
			playerSnapshot := snapshot.PlayerSnapshots[playerposition.PlayerID(playerID)]
			playerSnapshot.Equipment = playerEquipment.Equipment
			playerSnapshot.Money = playerEquipment.Money
			playerSnapshot.Health = 100 // Default health is 100 at the start of the round
			playerSnapshot.Armor = playerEquipment.Armor
			snapshot.PlayerSnapshots[playerposition.PlayerID(playerID)] = playerSnapshot
		}

		roundEvents := make([]events.GameEvent, 0)
		// if eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick < round.StartTick {
		// 	eventIndex += (round.StartTick - rh.Events[eventIndex].Tick) // Fast forward to the start tick of the round
		// }
		for eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick < round.StartTick {
			eventIndex++ // Fast forward to the start tick of the round
		}
		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			timeElasped += float64(tick-round.StartTick) * tickDuration
			frameData, exists := rh.Frames[tick]
			if exists {
				roundFrames = append(roundFrames, *frameData)
				for playerID, playerPosition := range frameData.PlayerPositions {
					playerSnapshot := snapshot.PlayerSnapshots[playerposition.PlayerID(playerID)]
					playerSnapshot.X = playerPosition.X
					playerSnapshot.Y = playerPosition.Y
					playerSnapshot.Yaw = playerPosition.Yaw
					snapshot.PlayerSnapshots[playerposition.PlayerID(playerID)] = playerSnapshot
				}
			}

			snapshot.resetInfernos() // Reset infernos every tick because we track them per tick. We don't need to accumulate them.
			for eventIndex < len(rh.Events) && rh.Events[eventIndex].Tick == tick {
				rh.Events[eventIndex].Tick = rh.Events[eventIndex].Tick - round.StartTick // Convert to 0-based tick for the round
				roundEvents = append(roundEvents, rh.Events[eventIndex])
				snapshot.updateSnapshot(rh.Events[eventIndex])
				eventIndex++
			}
			if (tick-round.StartTick)%SNAPSHOT_INTERVAL == 0 {
				// Append snapshot
				snapshot.Tick = tick - round.StartTick   // Convert to 0-based tick for the round
				snapshot.tickBlooms(timeElasped)         // Remove expired blooms
				snapshot.tickFlashedPlayers(timeElasped) // Remove expired flashed players
				replay.Snapshots[i] = append(replay.Snapshots[i], snapshot.Copy())
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

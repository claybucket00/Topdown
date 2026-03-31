package replay

import (
	"encoding/json"
	"os"
	events "topdown/internal/events"
	frames "topdown/internal/frames"
	metadata "topdown/internal/metadata"
	player "topdown/internal/playerposition"
	playerposition "topdown/internal/playerposition"
	serialization "topdown/internal/serialization"
	utility "topdown/internal/utility"

	r2 "github.com/golang/geo/r2"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	ulid "github.com/oklog/ulid/v2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	RemainingTime float64 `json:"remainingTime"`
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
				RemainingTime: float64(eventData.Duration),
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
		flashedSnapshot.RemainingTime -= timeElapsed
		if flashedSnapshot.RemainingTime <= 0 {
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

func (r *Replay) SerializeReplayJSON(path string) error {
	file, _ := os.Create(path)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	err := encoder.Encode(r)
	if err != nil {
		return err
	}

	return nil
}

func (r *Replay) SerializeReplayProtobuf(path string) error {
	// Create proto Replay message
	protoReplay := &serialization.Replay{
		MapName:     r.MapName,
		TickRate:    r.TickRate,
		LastUpdated: timestamppb.Now(),
	}

	// Convert PlayerMetadata map
	protoReplay.PlayersMetadata = make(map[int32]*serialization.PlayerMetadata)
	for playerID, playerMeta := range r.PlayerMetadata {
		protoReplay.PlayersMetadata[int32(playerID)] = &serialization.PlayerMetadata{
			Name: playerMeta.Name,
		}
	}

	// Convert NadeMetadata map (ULID to string)
	protoReplay.NadesMetadata = make(map[string]*serialization.NadeMetadata)
	for nadeID, nadeMeta := range r.NadeMetadata {
		protoReplay.NadesMetadata[nadeID.String()] = &serialization.NadeMetadata{
			Type:    nadeMeta.Type,
			Thrower: int32(nadeMeta.Thrower),
		}
	}

	// Convert RoundMetadata slice
	protoReplay.RoundsMetadata = make([]*serialization.RoundMetadata, len(r.RoundMetadata))
	for i, roundMeta := range r.RoundMetadata {
		score := &serialization.Score{
			Ct: int32(roundMeta.Score.CT),
			T:  int32(roundMeta.Score.T),
		}

		playerToTeam := make(map[int32]int32)
		for playID, team := range roundMeta.PlayerToTeams {
			playerToTeam[int32(playID)] = int32(team)
		}

		playerToEquipment := make(map[int32]*serialization.PlayerEquipment)
		for playID, equip := range roundMeta.PlayerToEquipment {
			playerToEquipment[int32(playID)] = &serialization.PlayerEquipment{
				Equipment: equip.Equipment,
				Money:     int32(equip.Money),
				Armor:     int32(equip.Armor),
			}
		}

		protoReplay.RoundsMetadata[i] = &serialization.RoundMetadata{
			Score:             score,
			PlayerToTeam:      playerToTeam,
			PlayerToEquipment: playerToEquipment,
		}
	}

	// Convert Rounds (2D slice of FrameData to RoundFrames)
	protoReplay.Rounds = make([]*serialization.RoundFrames, len(r.Rounds))
	for roundIdx, round := range r.Rounds {
		roundFrames := &serialization.RoundFrames{
			Frames: make([]*serialization.FrameData, len(round)),
		}
		for frameIdx, frameData := range round {
			protoFrameData := &serialization.FrameData{
				PlayerPositions: make(map[int32]*serialization.PlayerPosition),
				NadePositions:   make(map[string]*serialization.NadePosition),
			}

			// Convert player positions
			for playerID, playerPos := range frameData.PlayerPositions {
				protoFrameData.PlayerPositions[int32(playerID)] = &serialization.PlayerPosition{
					X:   playerPos.X,
					Y:   playerPos.Y,
					Yaw: playerPos.Yaw,
				}
			}

			// Convert nade positions (ULID to string)
			for nadeID, nadePos := range frameData.NadePositions {
				protoFrameData.NadePositions[nadeID.String()] = &serialization.NadePosition{
					X: nadePos.X,
					Y: nadePos.Y,
				}
			}

			roundFrames.Frames[frameIdx] = protoFrameData
		}
		protoReplay.Rounds[roundIdx] = roundFrames
	}

	// Convert Events (2D slice of GameEvent to RoundEvents)
	protoReplay.Events = make([]*serialization.RoundEvents, len(r.Events))
	for roundIdx, round := range r.Events {
		roundEvents := &serialization.RoundEvents{
			Events: make([]*serialization.GameEvent, len(round)),
		}
		for eventIdx, event := range round {
			protoEvent := &serialization.GameEvent{
				Tick: int32(event.Tick),
				Type: eventTypeToProto(event.Type),
			}

			// Convert the event data based on type
			switch eventData := event.Data.(type) {
			case events.TeamChangeEvent:
				protoEvent.Data = &serialization.GameEvent_TeamChangeEvent{
					TeamChangeEvent: &serialization.TeamChangeEvent{
						PlayerID: int32(eventData.PlayerID),
						Team:     int32(eventData.Team),
					},
				}
			case events.SmokeEvent:
				protoEvent.Data = &serialization.GameEvent_SmokeEvent{
					SmokeEvent: &serialization.SmokeEvent{
						X:      eventData.X,
						Y:      eventData.Y,
						NadeId: eventData.NadeId.String(),
					},
				}
			case events.GrenadeEvent:
				protoEvent.Data = &serialization.GameEvent_GrenadeEvent{
					GrenadeEvent: &serialization.GrenadeEvent{
						X:      eventData.X,
						Y:      eventData.Y,
						NadeId: eventData.NadeId.String(),
					},
				}
			case events.KillEvent:
				var attackerID, assisterID int32
				if eventData.AttackerID != nil {
					attackerID = int32(*eventData.AttackerID)
				}
				if eventData.AssisterID != nil {
					assisterID = int32(*eventData.AssisterID)
				}
				protoEvent.Data = &serialization.GameEvent_KillEvent{
					KillEvent: &serialization.KillEvent{
						VictimID:      int32(eventData.VictimID),
						AttackerID:    attackerID,
						AssisterID:    assisterID,
						Weapon:        eventData.Weapon,
						IsWallbang:    eventData.IsWallbang,
						IsHeadshot:    eventData.IsHeadshot,
						AssistedFlash: eventData.AssistedFlash,
						AttackerBlind: eventData.AttackerBlind,
						NoScope:       eventData.NoScope,
						ThroughSmoke:  eventData.ThroughSmoke,
					},
				}
			case events.DamageEvent:
				var playerID int32
				if eventData.PlayerID != nil {
					playerID = int32(*eventData.PlayerID)
				}
				protoEvent.Data = &serialization.GameEvent_DamageEvent{
					DamageEvent: &serialization.DamageEvent{
						PlayerID:          playerID,
						Health:            int32(eventData.Health),
						Armor:             int32(eventData.Armor),
						HealthDamageTaken: int32(eventData.HealthDamageTaken),
						ArmorDamageTaken:  int32(eventData.ArmorDamageTaken),
					},
				}
			case events.InfernoEvent:
				points := make([]*serialization.Point, len(eventData.Points))
				for i, p := range eventData.Points {
					points[i] = &serialization.Point{
						X: p.X,
						Y: p.Y,
					}
				}
				protoEvent.Data = &serialization.GameEvent_InfernoEvent{
					InfernoEvent: &serialization.InfernoEvent{
						Points: points,
						NadeId: eventData.NadeId,
					},
				}
			case events.FlashEvent:
				var playerID int32
				if eventData.PlayerID != nil {
					playerID = int32(*eventData.PlayerID)
				}
				protoEvent.Data = &serialization.GameEvent_FlashEvent{
					FlashEvent: &serialization.FlashEvent{
						PlayerID: playerID,
						Duration: eventData.Duration,
					},
				}
			case events.EquipmentEvent:
				var playerID int32
				var equipment []string
				if eventData.PlayerID != nil {
					playerID = int32(*eventData.PlayerID)
				}
				if eventData.Equipment != nil {
					equipment = *eventData.Equipment
				}
				protoEvent.Data = &serialization.GameEvent_EquipmentEvent{
					EquipmentEvent: &serialization.EquipmentEvent{
						PlayerID:  playerID,
						Money:     int32(eventData.Money),
						Equipment: equipment,
					},
				}
			case events.PickupEvent:
				protoEvent.Data = &serialization.GameEvent_PickupEvent{
					PickupEvent: &serialization.PickupEvent{
						EquipmentID: eventData.EquipmentID.String(),
					},
				}
			case events.DropEvent:
				protoEvent.Data = &serialization.GameEvent_DropEvent{
					DropEvent: &serialization.DropEvent{
						EquipmentID:   eventData.EquipmentID.String(),
						EquipmentName: eventData.EquipmentName,
						Position: &serialization.Point{
							X: eventData.Position.X,
							Y: eventData.Position.Y,
						},
					},
				}
			case events.BombDroppedEvent:
				protoEvent.Data = &serialization.GameEvent_BombDroppedEvent{
					BombDroppedEvent: &serialization.BombDroppedEvent{
						Position: &serialization.Point{
							X: eventData.Position.X,
							Y: eventData.Position.Y,
						},
					},
				}
			}

			roundEvents.Events[eventIdx] = protoEvent
		}
		protoReplay.Events[roundIdx] = roundEvents
	}

	// Convert Snapshots (2D slice of Snapshot to RoundSnapshots)
	protoReplay.Snapshots = make([]*serialization.RoundSnapshots, len(r.Snapshots))
	for roundIdx, round := range r.Snapshots {
		roundSnapshots := &serialization.RoundSnapshots{
			Snapshots: make([]*serialization.Snapshot, len(round)),
		}
		for snapIdx, snap := range round {
			protoSnapshot := &serialization.Snapshot{
				Tick:             int32(snap.Tick),
				PlayerSnapshots:  make(map[int32]*serialization.PlayerSnapshot),
				BloomSnapshots:   make(map[string]*serialization.BloomSnapshot),
				InfernoSnapshots: make(map[int64]*serialization.Points),
				FlashedSnapshots: make(map[int32]*serialization.FlashedSnapshot),
			}

			// Convert player snapshots
			for playerID, playerSnap := range snap.PlayerSnapshots {
				protoSnapshot.PlayerSnapshots[int32(playerID)] = &serialization.PlayerSnapshot{
					X:         playerSnap.X,
					Y:         playerSnap.Y,
					Yaw:       playerSnap.Yaw,
					Health:    int32(playerSnap.Health),
					Armor:     int32(playerSnap.Armor),
					Team:      int32(playerSnap.Team),
					Money:     int32(playerSnap.Money),
					Equipment: playerSnap.Equipment,
				}
			}

			// Convert bloom snapshots (ULID to string)
			for bloomID, bloomSnap := range snap.BloomSnapshots {
				protoSnapshot.BloomSnapshots[bloomID.String()] = &serialization.BloomSnapshot{
					X:        bloomSnap.X,
					Y:        bloomSnap.Y,
					Type:     bloomSnap.Type,
					Duration: bloomSnap.Duration,
				}
			}

			// Convert inferno snapshots
			for infernoID, infernoPoints := range snap.InfernoSnapshots {
				points := make([]*serialization.Point, len(infernoPoints))
				for i, p := range infernoPoints {
					points[i] = &serialization.Point{
						X: p.X,
						Y: p.Y,
					}
				}
				protoSnapshot.InfernoSnapshots[infernoID] = &serialization.Points{
					Points: points,
				}
			}

			// Convert flashed snapshots
			for playerID, flashedSnap := range snap.FlashedSnapshots {
				protoSnapshot.FlashedSnapshots[int32(playerID)] = &serialization.FlashedSnapshot{
					RemainingTime: flashedSnap.RemainingTime,
				}
			}

			// Convert bomb snapshot if present
			if snap.BombSnapshot != nil {
				protoSnapshot.BombSnapshot = &serialization.Point{
					X: snap.BombSnapshot.X,
					Y: snap.BombSnapshot.Y,
				}
			}

			roundSnapshots.Snapshots[snapIdx] = protoSnapshot
		}
		protoReplay.Snapshots[roundIdx] = roundSnapshots
	}

	// Marshal to bytes
	data, err := proto.Marshal(protoReplay)
	if err != nil {
		return err
	}

	// Write to disk
	return os.WriteFile(path, data, 0644)
}

// Helper function to convert EventType to protobuf EventType
func eventTypeToProto(eventType events.EventType) serialization.EventType {
	switch eventType {
	case events.EventFlash:
		return serialization.EventType_EVENT_FLASH
	case events.EventSmokeStart:
		return serialization.EventType_EVENT_SMOKE_START
	case events.EventSmokeEnd:
		return serialization.EventType_EVENT_SMOKE_END
	case events.EventKill:
		return serialization.EventType_EVENT_KILL
	case events.EventHe:
		return serialization.EventType_EVENT_HE
	case events.EventTeamChange:
		return serialization.EventType_EVENT_TEAM_CHANGE
	case events.EventInferno:
		return serialization.EventType_EVENT_INFERNO
	case events.EventDamage:
		return serialization.EventType_EVENT_DAMAGE
	case events.EventPlayerFlashed:
		return serialization.EventType_EVENT_PLAYER_FLASHED
	case events.EventEquipmentUpdate:
		return serialization.EventType_EVENT_EQUIPMENT_UPDATE
	case events.EventPickup:
		return serialization.EventType_EVENT_PICKUP
	case events.EventDrop:
		return serialization.EventType_EVENT_DROP
	case events.EventBombDropped:
		return serialization.EventType_EVENT_BOMB_DROPPED
	case events.EventBombPickup:
		return serialization.EventType_EVENT_BOMB_PICKUP
	default:
		return serialization.EventType_UNKNOWN
	}
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

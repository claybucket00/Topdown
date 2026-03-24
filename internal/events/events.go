package events

import (
	player "topdown/internal/playerposition"

	r2 "github.com/golang/geo/r2"
	common "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs/common"
	ulid "github.com/oklog/ulid/v2"
)

type GameEvent struct {
	Tick int
	Type EventType
	Data any // TODO: refactor this to specific types ... maybe
}

type EventType int

const (
	_ EventType = iota
	EventFlash
	EventSmokeStart
	EventSmokeEnd
	EventKill
	EventHe
	EventTeamChange
	EventInferno
	EventDamage
	EventPlayerFlashed
	EventEquipmentUpdate
	EventPickup
	EventDrop
	EventBombDropped
	EventBombPickup
)

type TeamChangeEvent struct {
	PlayerID int
	Team     common.Team
}

type SmokeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
}

type GrenadeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
}

type KillEvent struct {
	VictimID      int
	AttackerID    *player.PlayerID `json:"attacker,omitempty"`
	AssisterID    *player.PlayerID `json:"assister,omitempty"`
	Weapon        string
	IsWallbang    bool
	IsHeadshot    bool
	AssistedFlash bool
	AttackerBlind bool
	NoScope       bool
	ThroughSmoke  bool
}

type DamageEvent struct {
	PlayerID          *player.PlayerID `json:"playerID"`
	Health            int              `json:"health"`
	Armor             int              `json:"armor"`
	HealthDamageTaken int              `json:"healthdamage"`
	ArmorDamageTaken  int              `json:"armordamage"`
}

type InfernoEvent struct {
	Points []r2.Point // 2D convex hull of all the fires active in the inferno.
	NadeId int64      // We cannot use the usual ULID id as infernos are seperate entities than incendiary grenades.
}

type FlashEvent struct {
	PlayerID *player.PlayerID `json:"playerID"` // ID of the player who is flashed
	Duration int64            `json:"duration"` // Blindness duration (milliseconds)
}

type EquipmentEvent struct {
	PlayerID  *player.PlayerID `json:"playerID"`
	Money     int              `json:"money"`
	Equipment *[]string        `json:"equipment"`
}

type PickupEvent struct {
	EquipmentID ulid.ULID `json:"equipmentID"`
}

type DropEvent struct {
	EquipmentID   ulid.ULID `json:"equipmentID"`
	EquipmentName string    `json:"equipmentName"`
	Position      r2.Point  `json:"position"`
}

type BombDroppedEvent struct {
	Position r2.Point `json:"position"`
}

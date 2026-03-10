package events

import ulid "github.com/oklog/ulid/v2"

type GameEvent struct {
	Tick int
	Type EventType
	Data any
}

type EventType int

const (
	_ EventType = iota
	EventFlash
	EventSmokeStart
	EventSmokeEnd
	EventInfernoStart
	EvenInfernoEnd
)

type SmokeEvent struct {
	X      float64
	Y      float64
	NadeId ulid.ULID
	// NadeId int64
}

package round

import "topdown/internal/playerposition"

type FrameData struct {
	playerpositions map[int][]playerposition.PlayerPosition
	nadePaths       map[int][]playerposition.NadePath
}

type Round struct {
	Number          int
	StartTick       int
	EndTick         int
	PlayerPositions map[int][]playerposition.PlayerPosition // Key: UserID, Val: List of positions during the round
	NadePaths       map[int][]playerposition.NadePath       // Key: Tick, Val: List of nade paths starting at that tick
	FrameData       []FrameData
}

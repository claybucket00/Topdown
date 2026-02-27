package round

import "topdown/internal/playerposition"

type Round struct {
	Number          int
	StartTick       int
	EndTick         int
	PlayerPositions map[int][]playerposition.PlayerPosition // Key: UserID, Val: List of positions during the round
}

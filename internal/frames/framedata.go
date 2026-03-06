package frames

import "topdown/internal/playerposition"

type FrameData struct {
	PlayerPositions map[int]playerposition.PlayerPosition
	NadePositions   map[int64]playerposition.NadePosition
}

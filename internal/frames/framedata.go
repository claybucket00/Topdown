package frames

import (
	"topdown/internal/playerposition"

	"github.com/oklog/ulid/v2"
)

type FrameData struct {
	PlayerPositions map[int]playerposition.PlayerPosition     `json:"player_positions"` // Key: playerID, Val: PlayerPosition for that tick
	NadePositions   map[ulid.ULID]playerposition.NadePosition `json:"nade_positions"`   // Key: nadeID, Val: NadePosition for that tick
}

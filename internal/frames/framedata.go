package frames

import "topdown/internal/playerposition"

type FrameData struct {
	PlayerPositions map[int]playerposition.PlayerPosition `json:"player_positions"` // Key: playerID, Val: PlayerPosition for that tick
	NadePositions   map[int64]playerposition.NadePosition `json:"nade_positions"`   // Key: nadeID, Val: NadePosition for that tick
}

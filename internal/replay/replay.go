package replay

import (
	"topdown/internal/frames"
	metadata "topdown/internal/metadata"
)

type Replay struct {
	MapName        string
	TickRate       float64
	PlayerMetadata map[int]metadata.PlayerMetadata // Key: playerId, Val: PlayerMetadata
	NadeMetadata   map[int64]metadata.NadeMetadata // Key: nadeId, Val: NadeMetadata
	Rounds         [][]frames.FrameData
}

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:        rh.MapName,
		TickRate:       rh.TickRate,
		PlayerMetadata: rh.PlayerMetadata,
		NadeMetadata:   rh.NadeMetadata,
		Rounds:         make([][]frames.FrameData, len(rh.Rounds)),
	}

	for i, round := range rh.Rounds {
		// log.Printf("Processing round %d start=%d end=%d\n",
		// 	i, round.StartTick, round.EndTick)

		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			frameData, exists := rh.Frames[tick]
			if exists {
				roundFrames = append(roundFrames, *frameData)
			}
			// if frameData, exists := rh.Frames[tick]; exists {
			// 	roundFrames = append(roundFrames, *frameData)
			// }
		}

		// log.Printf("Round %d frames: %d\n", i, len(roundFrames))
		replay.Rounds[i] = roundFrames
	}
	return replay
}

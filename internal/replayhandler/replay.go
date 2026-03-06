package replayhandler

import (
	"topdown/internal/frames"
)

type Replay struct {
	MapName  string
	TickRate float64
	Rounds   [][]frames.FrameData
}

func (rh *ReplayHandler) GenerateReplay() Replay {
	replay := Replay{
		MapName:  rh.MapName,
		TickRate: rh.TickRate,
		Rounds:   make([][]frames.FrameData, len(rh.Rounds)),
	}

	for i, round := range rh.Rounds {
		roundFrames := make([]frames.FrameData, 0)
		for tick := round.StartTick; tick <= round.EndTick; tick++ {
			if frameData, exists := rh.Frames[tick]; exists {
				roundFrames = append(roundFrames, *frameData)
			}
		}
		replay.Rounds[i] = roundFrames
	}
	return replay
}

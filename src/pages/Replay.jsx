import { useEffect } from 'react'
import { init } from '../renderer/renderer'
import '../styles/replay.css'

export default function Replay({ demoId, demoMap, tickRate, roundCount, onBack }) {
  useEffect(() => {
    init(demoId, demoMap, tickRate, roundCount).catch(err => console.error('Renderer init failed:', err))
  }, [demoId, demoMap, tickRate, roundCount])

  return (
    <>
      <button
        type="button"
        onClick={onBack}
        style={{ position: 'fixed', top: 8, left: 8, zIndex: 100 }}
      >
        ← Back
      </button>

      <div className="main-layout">
        <div className="sidebar sidebar-left">
          <div className="team-header ct">CT</div>
          <div className="players-container" id="ct-players"></div>
        </div>

        <div className="center-area">
          <div className="canvas-container">
            <canvas id="map"></canvas>
          </div>
        </div>

        <div className="sidebar sidebar-right">
          <div className="team-header t">T</div>
          <div className="players-container" id="t-players"></div>
        </div>
      </div>

      <div className="replay-bar">
        <div className="replay-controls">
          <button id="play-pause-btn" className="play-pause-btn" title="Play/Pause">⏸</button>
          <input type="range" id="replay-progress" className="progress-slider" min="0" max="100" defaultValue="0" />
        </div>
        <div className="time-display">
          <span id="current-time">0:00</span> / <span id="total-time">0:00</span>
        </div>
        <div className="speed-controls">
          <button className="speed-btn active" data-speed="1">1x</button>
          <button className="speed-btn" data-speed="2">2x</button>
          <button className="speed-btn" data-speed="4">4x</button>
        </div>
      </div>
    </>
  )
}

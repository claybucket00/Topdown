import { useState } from 'react'
import Landing from './pages/Landing'
import Replay from './pages/Replay'

export default function App() {
  const [view, setView] = useState({ page: 'landing' })

  if (view.page === 'replay') {
    return (
      <Replay
        demoId={view.demoId}
        onBack={() => setView({ page: 'landing' })}
      />
    )
  }

  return (
    <Landing onViewReplay={(demoId) => setView({ page: 'replay', demoId })} />
  )
}

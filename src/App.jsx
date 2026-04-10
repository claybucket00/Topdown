import { useState } from 'react'
import Landing from './pages/Landing'
import Replay from './pages/Replay'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'

const queryClient = new QueryClient();

export default function App() {
  const [view, setView] = useState({ page: 'landing' })

  if (view.page === 'replay') {
    return (
      <Replay
        demoId={view.demoId}
        demoMap={view.demoMap}
        tickRate={view.tickRate}
        roundCount={view.roundCount}
        onBack={() => setView({ page: 'landing' })}
      />
    )
  }

  return (
    <QueryClientProvider client={queryClient}>
    <Landing onViewReplay={(demoId, demoMap, tickRate, roundCount) => setView({ page: 'replay', demoId, demoMap, tickRate, roundCount })} />
    </QueryClientProvider>
  )
}

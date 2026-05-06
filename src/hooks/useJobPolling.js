import { useEffect, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'

const API_URL = 'http://localhost:8080'
const POLL_INTERVAL_MS = 5000

export function useJobPolling(jobs, setJobs) {
  const queryClient = useQueryClient()
  const jobsRef = useRef(jobs)

  useEffect(() => { jobsRef.current = jobs }, [jobs])

  useEffect(() => {
    const id = setInterval(() => {
      const active = jobsRef.current.filter(
        j => j.status === 'pending' || j.status === 'parsing'
      )
      if (active.length === 0) return

      active.forEach(async (job) => {
        try {
          const data = await fetch(`${API_URL}/demos/jobs/${job.jobId}/status`).then(r => r.json())
          setJobs(prev => prev.map(j =>
            j.jobId === job.jobId
              ? { ...j, status: data.status, progress: data.progress, error: data.error }
              : j
          ))
          if (data.status === 'complete') {
            new Notification('Demo parsed', { body: `${job.demoName} is ready to view` })
            queryClient.invalidateQueries({ queryKey: ['demos'] })
          }
          if (data.status === 'failed') {
            new Notification('Parse failed', { body: `${job.demoName}: ${data.error || 'unknown error'}` })
          }
        } catch (_) {
          // network error — keep polling
        }
      })
    }, POLL_INTERVAL_MS)

    return () => clearInterval(id)
  }, []) // jobsRef keeps callback fresh without re-subscribing the interval
}

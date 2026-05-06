import { useState, useMemo } from 'react'
import {
  flexRender,
  useReactTable,
  getCoreRowModel,
} from '@tanstack/react-table'
import { useQuery } from '@tanstack/react-query'
import { useJobPolling } from '../hooks/useJobPolling'
import '../styles/Landing.css'

const API_URL = 'http://localhost:8080'

export default function Landing({ onViewReplay }) {
  const [jobs, setJobs] = useState([])
  useJobPolling(jobs, setJobs)

  const  {data: demoData, refetch, isFetching} = useQuery({
    queryKey: ['demos'],
    queryFn:  () => fetch(`${API_URL}/demos`).then(r => r.json()),
  })

  const tableColumns = useMemo(() => [
    {
      header: 'Name',
      accessorKey: 'name',
    },
    {
      header: 'Map',
      accessorKey: 'mapName',
    },
    {
      header: 'Rounds',
      accessorKey: 'roundCount',
    },
  ], [])

  const tableData = useMemo(() => demoData?.demos || [], [demoData])

  console.log('Fetched demos:', demoData)
  
  const demoTable = useReactTable({
    columns: tableColumns,
    data: tableData,
    getCoreRowModel: getCoreRowModel(),
  })

  async function handleUpload() {
    if (window.electronAPI?.selectFile) {
      const path = await window.electronAPI.selectFile()
      if (!path) return
      const file = await window.electronAPI.readFile(path)
      const formData = new FormData()
      formData.append('file', file)
      const jobMetadata = await fetch(`${API_URL}/demos`, {
        method: 'POST',
        body: formData,
      }).then(r => r.json())
      const demoName = path.split(/[\\/]/).pop().replace(/\.dem$/i, '')
      setJobs(prev => [...prev, {
        jobId: jobMetadata.jobId,
        demoName,
        status: 'pending',
        progress: 0,
        error: null,
      }])
      return
    }

    // Fallback for non-Electron environments
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = '.dem'
    input.onchange = (event) => {
      const file = event.target.files?.[0]
      if (file) console.log('Selected file (fallback):', file.name)
    }
    input.click()
  }

  async function handleDelete(demoId) {
    if (!window.confirm('Are you sure you want to delete this demo?')) return
    try {
      const response = await fetch(`${API_URL}/demos/${demoId}`, { method: 'DELETE' })
      if (response.ok) {
        refetch()
        new Notification('Demo deleted', { body: `Demo ${demoId} has been deleted` })
      }
      else {
        new Notification('Delete failed', { body: `Failed to delete demo ${demoId}` })
      }
    }
    catch {
      new Notification('Delete failed', { body: `Failed to delete demo ${demoId} (network error)` })
    }
  }

  return (
    <div className="landing">
      <div className="landing-actions">
        <button
          className="btn"
          onClick={() => refetch()}
          disabled={isFetching}
        >
          {isFetching ? 'Refreshing...' : 'Refresh'}
        </button>
        <button className="btn btn-primary" type="button" onClick={handleUpload}>
          Upload Demo
        </button>
      </div>
      {jobs.length > 0 && (
        <div className="parse-jobs">
          {jobs.map(job => (
            <div key={job.jobId} className={`parse-job parse-job--${job.status}`}>
              <span className="parse-job-name">{job.demoName}</span>
              <span className="parse-job-status">
                {job.status === 'pending' && 'Queued'}
                {job.status === 'parsing' && `Parsing… ${job.progress}%`}
                {job.status === 'complete' && 'Complete ✓'}
                {job.status === 'failed' && `Failed: ${job.error}`}
              </span>
              {(job.status === 'complete' || job.status === 'failed') && (
                <button
                  className="btn parse-job-dismiss"
                  onClick={() => setJobs(prev => prev.filter(j => j.jobId !== job.jobId))}
                >
                  ×
                </button>
              )}
            </div>
          ))}
        </div>
      )}
      <table className="demo-table">
        <thead>
          {demoTable.getHeaderGroups().map((headerGroup) => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map((header) => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(
                        header.column.columnDef.header,
                        header.getContext(),
                      )}
                </th>
              ))}
              <th />
            </tr>
          ))}
        </thead>
        <tbody>
          {demoTable.getRowModel().rows.map((row) => (
            <tr key={row.id}>
              {row.getVisibleCells().map((cell) => (
                <td key={cell.id}>
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
              <td>
                <button className="btn btn-primary" type="button" onClick={() => onViewReplay(row.original.id, row.original.mapName, row.original.tickRate, row.original.roundCount)}>
                  View Replay
                </button>
                <button className="btn btn-primary" type="button" onClick={() => handleDelete(row.original.id)}>
                  ×
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

import { useState, useEffect, useMemo } from 'react'
import {
  flexRender,
  useReactTable,
  getCoreRowModel,
} from '@tanstack/react-table'
import { useQuery } from '@tanstack/react-query'
import '../styles/Landing.css'

const API_URL = 'http://localhost:8080'

export default function Landing({ onViewReplay }) {
  // const [demos, setDemos] = useState([])

  // useEffect(() => {
  //   fetch(`${API_URL}/demos`)
  //     .then(r => r.json())
  //     .then(data => setDemos(data.demos || []))
  //     .catch(err => console.error('Error fetching demos:', err))
  // }, [])

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
      console.log('Job metadata:', jobMetadata)
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
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

import { useState, useEffect } from 'react'

const API_URL = 'http://localhost:8080'

export default function Landing({ onViewReplay }) {
  const [demos, setDemos] = useState([])

  useEffect(() => {
    fetch(`${API_URL}/demos`)
      .then(r => r.json())
      .then(data => setDemos(data.demos || []))
      .catch(err => console.error('Error fetching demos:', err))
  }, [])

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
    <div style={{ padding: 16 }}>
      <table border="1">
        <thead>
          <tr>
            <th>Name</th>
            <th>Map</th>
            <th>Rounds</th>
          </tr>
        </thead>
        <tbody>
          {demos.map(demo => (
            <tr key={demo.name}>
              <td>{demo.name}</td>
              <td>{demo.mapName}</td>
              <td>
                {demo.roundCount}{' '}
                <button type="button" onClick={() => onViewReplay(demo.name)}>
                  View Replay
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <button type="button" onClick={handleUpload}>
        Upload Demo
      </button>
    </div>
  )
}

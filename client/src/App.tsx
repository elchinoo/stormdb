import React, { useState, useEffect } from 'react'
import './App.css'

// Types for test runs
interface TestRunResponse {
  id: number
  plugin_name: string
  status: string
  result?: Record<string, any>
  error?: string
}

interface TestRun {
  id: number
  plugin_name: string
  status: string
}

interface HistoryResponse {
  test_runs: TestRun[]
  count: number
  limit: number
  offset: number
  error?: string
}

function App() {
  const [payload, setPayload] = useState<string>(`{
  "plugin_name": "bulk-load",
  "config": {
    "host": "localhost",
    "port": 5432,
    "database": "stormdb",
    "username": "postgres",
    "password": "postgres",
    "rebuild": true
  }
}`)
  const [response, setResponse] = useState<TestRunResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [history, setHistory] = useState<TestRun[]>([])
  const [loadingHistory, setLoadingHistory] = useState(false)
  const [historyError, setHistoryError] = useState<string | null>(null)
  const [selectedRun, setSelectedRun] = useState<TestRunResponse | null>(null)

  const sendTestRun = async () => {
    setResponse(null)
    setError(null)
    try {
      const res = await fetch('/test-runs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: payload,
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error || JSON.stringify(data))
      } else {
        setResponse(data)
      }
    } catch (err: any) {
      setError(err.message)
    }
  }

  const fetchHistory = async () => {
    setLoadingHistory(true)
    setHistoryError(null)
    try {
      const res = await fetch('/test-runs')
      const data: HistoryResponse = await res.json()
      if (!res.ok) throw new Error(data.error || JSON.stringify(data))
      setHistory(data.test_runs)
    } catch (err: any) {
      setHistoryError(err.message)
    } finally {
      setLoadingHistory(false)
    }
  }

  const selectRun = async (id: number) => {
    try {
      const res = await fetch(`/test-runs/${id}`)
      const data: TestRunResponse = await res.json()
      setSelectedRun(data)
    } catch (err: any) {
      console.error(err)
    }
  }

  // Poll for status updates when a run is active
  useEffect(() => {
    if (!selectedRun) return
    if (selectedRun.status !== 'scheduled' && selectedRun.status !== 'running') return
    const interval = setInterval(async () => {
      const res = await fetch(`/test-runs/${selectedRun.id}`)
      const data: TestRunResponse = await res.json()
      setSelectedRun(data)
      if (data.status !== 'scheduled' && data.status !== 'running') {
        clearInterval(interval)
        fetchHistory()
      }
    }, 5000)
    return () => clearInterval(interval)
  }, [selectedRun])

  useEffect(() => { fetchHistory() }, [])

  return (
    <div className="App">
      <h1>StormDB API Client</h1>
      <div className="container">
        <div className="section">
          <h2>New Test Run</h2>
          <textarea value={payload} onChange={e => setPayload(e.target.value)} rows={10} />
          <button onClick={sendTestRun}>Send</button>
          {response && <div className="alert success">Scheduled run ID: {response.id}</div>}
          {error && <div className="alert error">{error}</div>}
        </div>

        <div className="section">
          <h2>History</h2>
          {loadingHistory && <p>Loading...</p>}
          {historyError && <div className="alert error">{historyError}</div>}
          <table>
            <thead><tr><th>ID</th><th>Plugin</th><th>Status</th></tr></thead>
            <tbody>
              {history.map(run => (
                <tr key={run.id} onClick={() => selectRun(run.id)}>
                  <td>{run.id}</td>
                  <td>{run.plugin_name}</td>
                  <td>{run.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {selectedRun && (
        <div className="section">
          <h2>Details for Run {selectedRun.id}</h2>
          <p>Status: {selectedRun.status}</p>
          {selectedRun.result && <pre>{JSON.stringify(selectedRun.result, null, 2)}</pre>}
          {selectedRun.error && <div className="alert error">{selectedRun.error}</div>}
        </div>
      )}
    </div>
  )
}

export default App

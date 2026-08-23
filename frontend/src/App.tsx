import { useEffect, useState } from 'react'
import axios from 'axios'

type Healthz = {
  success: boolean
  data: { status: string; services: Record<string, { status: string }> }
}

export default function App() {
  const [health, setHealth] = useState<Healthz | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    axios
      .get<Healthz>('/healthz')
      .then((res) => setHealth(res.data))
      .catch((err) => setError(String(err)))
  }, [])

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-950 text-slate-100">
      <div className="w-full max-w-md rounded-xl border border-slate-800 bg-slate-900 p-8">
        <h1 className="text-xl font-semibold">Reimbursement System</h1>
        <p className="mt-1 text-sm text-slate-400">M0 skeleton — API proxy check</p>

        {error && (
          <p className="mt-6 rounded-lg bg-red-950 px-4 py-3 text-sm text-red-300">
            API unreachable: {error}
          </p>
        )}

        {health && (
          <ul className="mt-6 space-y-2 text-sm">
            {Object.entries(health.data.services).map(([name, svc]) => (
              <li key={name} className="flex items-center justify-between rounded-lg bg-slate-800/60 px-4 py-2">
                <span className="capitalize">{name}</span>
                <span className={svc.status === 'up' ? 'text-emerald-400' : 'text-red-400'}>
                  ● {svc.status}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </main>
  )
}

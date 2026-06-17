import { usePingStore } from './store/ping';

function App() {
  const { result, loading, ping } = usePingStore();

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-8 bg-slate-900 text-slate-100">
      <header className="text-center">
        <h1 className="text-4xl font-extrabold tracking-tight">The War Room</h1>
        <p className="mt-1 text-sm text-slate-400">
          B0 scaffold — Go &harr; JS bridge + SQLite WAL
        </p>
      </header>

      <button
        type="button"
        onClick={() => void ping()}
        disabled={loading}
        className="rounded-lg bg-emerald-600 px-6 py-3 font-semibold text-white transition hover:bg-emerald-500 disabled:opacity-50"
      >
        {loading ? 'Pinging…' : 'Ping backend'}
      </button>

      {result && (
        <div
          className={`w-96 rounded-lg border p-4 text-left text-sm ${
            result.ok
              ? 'border-emerald-700 bg-emerald-950/40'
              : 'border-rose-700 bg-rose-950/40'
          }`}
        >
          <div className="font-mono">
            <span className={result.ok ? 'text-emerald-400' : 'text-rose-400'}>
              {result.ok ? '✓' : '✗'} {result.message}
            </span>
          </div>
          {result.journalMode && (
            <div className="mt-2 text-slate-300">
              journal mode: <span className="font-mono">{result.journalMode}</span>
            </div>
          )}
          <div className="mt-1 text-slate-400">{result.detail}</div>
        </div>
      )}
    </div>
  );
}

export default App;

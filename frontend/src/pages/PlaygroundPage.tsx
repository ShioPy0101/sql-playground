import { FormEvent, useMemo, useState } from "react";
import { executeSQL } from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { ResultPanel } from "../components/ResultPanel";
import { formatTableLabel } from "../utils/csv";

const sampleCSV = `name,team,score
Alice,red,82
Bob,blue,91
Chika,red,88
Daichi,blue,76`;

const sampleQuery = `SELECT
  team,
  COUNT(*) AS members,
  AVG(CAST(score AS INTEGER)) AS average_score
FROM input
GROUP BY team
ORDER BY average_score DESC;`;

export function PlaygroundPage() {
  const [csv, setCSV] = useState(sampleCSV);
  const [query, setQuery] = useState(sampleQuery);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const tableLabel = useMemo(() => formatTableLabel(csv), [csv]);

  async function handleExecute(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsRunning(true);
    setError("");

    try {
      const payload = await executeSQL(csv, query);
      setResult(payload.csv);
    } catch (err) {
      setError(err instanceof Error ? err.message : "予期しないエラーが発生しました");
      setResult("");
    } finally {
      setIsRunning(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">SQLite CSV Runner</p>
          <h1>SQL Playground</h1>
        </div>
        <button form="sql-form" className="run-button" disabled={isRunning}>
          {isRunning ? "Running..." : "Run SQL"}
        </button>
      </header>

      <form id="sql-form" className="workspace" onSubmit={handleExecute}>
        <EditorPanel
          title="CSV"
          label={tableLabel}
          value={csv}
          ariaLabel="CSV input"
          onChange={setCSV}
        />
        <EditorPanel
          title="SQL"
          label="SQLite"
          value={query}
          ariaLabel="SQL query"
          onChange={setQuery}
        />
        <ResultPanel result={result} error={error} emptyText="CSV と SQL を編集して実行してください。" />
      </form>
    </main>
  );
}

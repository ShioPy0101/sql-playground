import { FormEvent, useMemo, useState } from "react";

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

type ExecuteResponse = {
  csv: string;
};

function App() {
  const [csv, setCSV] = useState(sampleCSV);
  const [query, setQuery] = useState(sampleQuery);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);

  const previewRows = useMemo(() => parseCSVPreview(result), [result]);

  async function executeSQL(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsRunning(true);
    setError("");

    try {
      const response = await fetch("/api/sqlite/execute", {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ csv, query })
      });

      const payload = await response.json();
      if (!response.ok) {
        throw new Error(payload.message ?? "SQL の実行に失敗しました");
      }

      setResult((payload as ExecuteResponse).csv);
    } catch (err) {
      setError(err instanceof Error ? err.message : "予期しないエラーが発生しました");
      setResult("");
    } finally {
      setIsRunning(false);
    }
  }

  return (
    <main className="app-shell">
      {/* ヘッダー */}
      <header className="app-header">
        <div>
          <p className="eyebrow">SQLite CSV Runner</p>
          <h1>SQL Playground</h1>
        </div>
        <button form="sql-form" className="run-button" disabled={isRunning}>
          {isRunning ? "Running..." : "Run SQL"}
        </button>
      </header>

      {/* データ入力欄 */}
      <form id="sql-form" className="workspace" onSubmit={executeSQL}>
        <section className="editor-panel" aria-label="CSV input">
          <div className="panel-heading">
            <h2>CSV</h2>
            <span>table: input</span>
          </div>
          <textarea
            value={csv}
            onChange={(event) => setCSV(event.target.value)}
            spellCheck={false}
            aria-label="CSV text"
          />
        </section>

        <section className="editor-panel" aria-label="SQL query">
          <div className="panel-heading">
            <h2>SQL</h2>
            <span>SQLite</span>
          </div>
          <textarea
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            spellCheck={false}
            aria-label="SQL query"
          />
        </section>

        <section className="result-panel" aria-label="Query result">
          <div className="panel-heading">
            <h2>Result</h2>
            <span>{result ? `${previewRows.length} rows` : "waiting"}</span>
          </div>

          {error ? <pre className="error-output">{error}</pre> : null}

          {!error && previewRows.length > 0 ? (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    {previewRows[0].map((cell) => (
                      <th key={cell}>{cell}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {previewRows.slice(1).map((row, rowIndex) => (
                    <tr key={`${row.join("-")}-${rowIndex}`}>
                      {row.map((cell, cellIndex) => (
                        <td key={`${cell}-${cellIndex}`}>{cell}</td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}

          {!error && !result ? (
            <div className="empty-state">
              <p>CSV と SQL を編集して実行してください。</p>
            </div>
          ) : null}

          {!error && result ? <pre className="raw-output">{result}</pre> : null}
        </section>
      </form>
    </main>
  );
}

function parseCSVPreview(csvText: string) {
  return csvText
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => line.split(",").map((cell) => cell.trim()));
}

export default App;

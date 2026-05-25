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

type ResultSection = {
  title: string;
  rows: string[][];
  status: string;
};

function App() {
  const [csv, setCSV] = useState(sampleCSV);
  const [query, setQuery] = useState(sampleQuery);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);

  const resultSections = useMemo(() => parseResultSections(result), [result]);
  const resultRowCount = useMemo(() => countResultRows(resultSections), [resultSections]);
  const tableLabel = useMemo(() => formatTableLabel(csv), [csv]);

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
            <span>{tableLabel}</span>
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
            <span>{result ? `${resultRowCount} rows` : "waiting"}</span>
          </div>

          {error ? <pre className="error-output">{error}</pre> : null}

          {!error && resultSections.length > 0 ? (
            <div className="result-content">
              {resultSections.map((section, sectionIndex) => (
                <div className="result-section" key={`${section.title}-${sectionIndex}`}>
                  {section.title ? <p className="query-title">{section.title}</p> : null}
                  {section.status ? (
                    <p className="status-output">{section.status}</p>
                  ) : (
                    <ResultTable rows={section.rows} />
                  )}
                </div>
              ))}
            </div>
          ) : null}

          {!error && !result ? (
            <div className="empty-state">
              <p>CSV と SQL を編集して実行してください。</p>
            </div>
          ) : null}
        </section>
      </form>
    </main>
  );
}

function ResultTable({ rows }: { rows: string[][] }) {
  if (rows.length === 0) {
    return null;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {rows[0].map((cell, cellIndex) => (
              <th key={`${cell}-${cellIndex}`}>{cell}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.slice(1).map((row, rowIndex) => (
            <tr key={`${row.join("-")}-${rowIndex}`}>
              {row.map((cell, cellIndex) => (
                <td key={`${cell}-${cellIndex}`}>{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function parseResultSections(resultText: string): ResultSection[] {
  const trimmed = resultText.trim();
  if (!trimmed) {
    return [];
  }

  const sections: Array<{ title: string; lines: string[] }> = [];
  let current = { title: "", lines: [] as string[] };

  for (const line of trimmed.split("\n")) {
    if (line.startsWith("-- Query ")) {
      if (current.title || current.lines.length > 0) {
        sections.push(current);
      }
      current = { title: line.replace(/^--\s*/, ""), lines: [] };
      continue;
    }

    current.lines.push(line);
  }

  if (current.title || current.lines.length > 0) {
    sections.push(current);
  }

  return sections.map((section) => {
    const body = section.lines.join("\n").trim();
    const status = body === "OK" ? body : "";

    return {
      title: section.title,
      rows: status ? [] : parseCSVPreview(body),
      status
    };
  });
}

function countResultRows(sections: ResultSection[]) {
  return sections.reduce((total, section) => {
    if (section.status) {
      return total;
    }
    return total + Math.max(section.rows.length - 1, 0);
  }, 0);
}

function parseCSVPreview(csvText: string) {
  return csvText
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => line.split(",").map((cell) => cell.trim()));
}

function formatTableLabel(csvText: string) {
  const names = parseTableNames(csvText);

  if (names.length === 1) {
    return `table: ${names[0]}`;
  }

  return `tables: ${names.join(", ")}`;
}

function parseTableNames(csvText: string) {
  const markerNames = csvText
    .split("\n")
    .map((line) => parseTableMarker(line))
    .filter((name): name is string => Boolean(name));

  if (markerNames.length === 0) {
    return ["input"];
  }

  return Array.from(new Set(markerNames));
}

function parseTableMarker(line: string) {
  const trimmed = line.trim();
  if (!trimmed) {
    return "";
  }

  if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
    return trimmed.slice(1, -1).trim();
  }

  const lower = trimmed.toLowerCase();
  for (const prefix of ["# table:", "-- table:"]) {
    if (lower.startsWith(prefix)) {
      return trimmed.slice(prefix.length).trim();
    }
  }

  return "";
}

export default App;

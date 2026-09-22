import { FormEvent, useEffect, useMemo, useState } from "react";
import { executeSQL } from "../api/client";
import type { ExecutionMetrics, InputFormat } from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { ResultPanel } from "../components/ResultPanel";
import { formatTableLabel } from "../utils/csv";
import { decodePlaygroundState, encodePlaygroundState } from "../utils/shareState";
import { schemaFromCSV, schemaFromDDL } from "../utils/sqlAutocomplete";

const sampleCSV = `name,team,score
Alice,red,82
Bob,blue,91
Chika,red,88
Daichi,blue,76`;

const sampleInputSQL = `CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  name TEXT NOT NULL
);

CREATE TABLE posts (
  id INTEGER PRIMARY KEY,
  tenant_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id)
);

INSERT INTO users (id, tenant_id, name) VALUES
  (1, 1, '吉田東洋'),
  (2, 1, '坂本龍馬'),
  (6, 2, '西郷隆盛');

INSERT INTO posts (id, tenant_id, user_id, title) VALUES
  (1, 1, 2, '藩政について');`;

const sampleQuery = `SELECT
  team,
  COUNT(*) AS members,
  AVG(CAST(score AS INTEGER)) AS average_score
FROM input
GROUP BY team
ORDER BY average_score DESC;`;

const defaultDialect = "sqlite";

function inputFormatFromURL() {
  const input = new URLSearchParams(window.location.search).get("input");
  return input === "csv" || input === "sql" ? input : null;
}

export function PlaygroundPage() {
  const [csv, setCSV] = useState(sampleCSV);
  const [inputSQL, setInputSQL] = useState(sampleInputSQL);
  const [inputFormat, setInputFormat] = useState<InputFormat>(inputFormatFromURL() ?? "csv");
  const [query, setQuery] = useState(sampleQuery);
  const [dialect, setDialect] = useState(defaultDialect);
  const [result, setResult] = useState("");
  const [metrics, setMetrics] = useState<ExecutionMetrics | null>(null);
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [isShareStateReady, setIsShareStateReady] = useState(false);
  const tableLabel = useMemo(
    () => (inputFormat === "csv" ? formatTableLabel(csv) : "SQLite DDL + INSERT"),
    [csv, inputFormat]
  );
  const autocompleteSchema = useMemo(
    () => (inputFormat === "sql" ? schemaFromDDL(inputSQL) : schemaFromCSV(csv)),
    [csv, inputFormat, inputSQL]
  );
  const emptySchema = useMemo(() => ({ tables: [] }), []);

  useEffect(() => {
    let cancelled = false;
    let restoreVersion = 0;

    async function restoreShareState() {
      const version = ++restoreVersion;
      setIsShareStateReady(false);
      const encoded = new URLSearchParams(window.location.hash.slice(1)).get("state");
      const urlInput = inputFormatFromURL();

      if (encoded) {
        const state = await decodePlaygroundState(encoded);
        if (!cancelled && version === restoreVersion && state) {
          setCSV(state.csv);
          setInputSQL(state.inputSql ?? sampleInputSQL);
          setInputFormat(urlInput ?? state.input ?? "csv");
          setQuery(state.sql);
          setDialect(state.dialect);
        }
      } else if (!cancelled && version === restoreVersion) {
        setInputFormat(urlInput ?? "csv");
      }

      if (!cancelled && version === restoreVersion) {
        setIsShareStateReady(true);
      }
    }

    void restoreShareState();
    window.addEventListener("popstate", restoreShareState);
    window.addEventListener("hashchange", restoreShareState);
    return () => {
      cancelled = true;
      window.removeEventListener("popstate", restoreShareState);
      window.removeEventListener("hashchange", restoreShareState);
    };
  }, []);

  useEffect(() => {
    if (!isShareStateReady) {
      return;
    }

    let cancelled = false;
    const timer = window.setTimeout(async () => {
      try {
        const encoded = await encodePlaygroundState({
          csv,
          inputSql: inputSQL,
          input: inputFormat,
          sql: query,
          dialect
        });
        if (cancelled) {
          return;
        }

        const hash = new URLSearchParams(window.location.hash.slice(1));
        hash.set("state", encoded);
        const search = new URLSearchParams(window.location.search);
        search.set("input", inputFormat);
        window.history.replaceState(
          null,
          "",
          `${window.location.pathname}?${search.toString()}#${hash.toString()}`
        );
      } catch (encodeError) {
        console.error("Failed to encode playground share state", encodeError);
      }
    }, 250);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [csv, dialect, inputFormat, inputSQL, isShareStateReady, query]);

  function handleInputFormatChange(nextFormat: InputFormat) {
    if (nextFormat === inputFormat) {
      return;
    }

    const search = new URLSearchParams(window.location.search);
    search.set("input", nextFormat);
    window.history.pushState(
      null,
      "",
      `${window.location.pathname}?${search.toString()}${window.location.hash}`
    );
    setInputFormat(nextFormat);
  }

  async function handleExecute(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setIsRunning(true);
    setError("");

    try {
      const input = inputFormat === "csv" ? csv : inputSQL;
      const payload = await executeSQL(input, query, "", inputFormat);
      setResult(payload.csv);
      setMetrics(payload.metrics ?? null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "予期しないエラーが発生しました");
      setResult("");
      setMetrics(null);
    } finally {
      setIsRunning(false);
    }
  }

  return (
    <main className="app-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">SQLite CSV実行環境</p>
          <h1 className="playground-title">SQLプレイグラウンド</h1>
        </div>
        <button form="sql-form" className="run-button" disabled={isRunning}>
          {isRunning ? "実行中..." : "SQLを実行"}
        </button>
      </header>

      <form id="sql-form" className="workspace" onSubmit={handleExecute}>
        <EditorPanel
          title="入力データ"
          label={tableLabel}
          value={inputFormat === "csv" ? csv : inputSQL}
          ariaLabel={inputFormat === "csv" ? "CSV入力" : "SQL入力"}
          sqlAutocomplete={inputFormat === "sql" ? emptySchema : undefined}
          onChange={inputFormat === "csv" ? setCSV : setInputSQL}
          headingAccessory={
            <div className="input-format-toggle" aria-label="入力形式">
              {(["csv", "sql"] as const).map((format) => (
                <button
                  key={format}
                  type="button"
                  className={inputFormat === format ? "is-active" : ""}
                  aria-pressed={inputFormat === format}
                  onClick={() => handleInputFormatChange(format)}
                >
                  {format.toUpperCase()}
                </button>
              ))}
            </div>
          }
        />
        <EditorPanel
          title="SQL"
          label={dialect === "sqlite" ? "SQLite" : dialect}
          value={query}
          ariaLabel="SQLクエリ"
          sqlAutocomplete={autocompleteSchema}
          onChange={setQuery}
        />
        <ResultPanel
          result={result}
          error={error}
          metrics={metrics}
          emptyText="入力データと SQL を編集して実行してください。"
        />
      </form>
    </main>
  );
}

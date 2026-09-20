import { FormEvent, useEffect, useMemo, useState } from "react";
import { executeSQL } from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { ResultPanel } from "../components/ResultPanel";
import { formatTableLabel } from "../utils/csv";
import { decodePlaygroundState, encodePlaygroundState } from "../utils/shareState";

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

const defaultDialect = "sqlite";

export function PlaygroundPage() {
  const [csv, setCSV] = useState(sampleCSV);
  const [query, setQuery] = useState(sampleQuery);
  const [dialect, setDialect] = useState(defaultDialect);
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [isShareStateReady, setIsShareStateReady] = useState(false);
  const tableLabel = useMemo(() => formatTableLabel(csv), [csv]);

  useEffect(() => {
    let cancelled = false;
    const encoded = new URLSearchParams(window.location.hash.slice(1)).get("state");

    async function restoreShareState() {
      if (encoded) {
        const state = await decodePlaygroundState(encoded);
        if (!cancelled && state) {
          setCSV(state.csv);
          setQuery(state.sql);
          setDialect(state.dialect);
        }
      }

      if (!cancelled) {
        setIsShareStateReady(true);
      }
    }

    void restoreShareState();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!isShareStateReady) {
      return;
    }

    let cancelled = false;
    const timer = window.setTimeout(async () => {
      try {
        const encoded = await encodePlaygroundState({ csv, sql: query, dialect });
        if (cancelled) {
          return;
        }

        const hash = new URLSearchParams(window.location.hash.slice(1));
        hash.set("state", encoded);
        window.history.replaceState(
          null,
          "",
          `${window.location.pathname}${window.location.search}#${hash.toString()}`
        );
      } catch (encodeError) {
        console.error("Failed to encode playground share state", encodeError);
      }
    }, 250);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [csv, dialect, isShareStateReady, query]);

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
          <p className="eyebrow">SQLite CSV実行環境</p>
          <h1 className="playground-title">SQLプレイグラウンド</h1>
        </div>
        <button form="sql-form" className="run-button" disabled={isRunning}>
          {isRunning ? "実行中..." : "SQLを実行"}
        </button>
      </header>

      <form id="sql-form" className="workspace" onSubmit={handleExecute}>
        <EditorPanel
          title="CSV"
          label={tableLabel}
          value={csv}
          ariaLabel="CSV入力"
          onChange={setCSV}
        />
        <EditorPanel
          title="SQL"
          label={dialect === "sqlite" ? "SQLite" : dialect}
          value={query}
          ariaLabel="SQLクエリ"
          onChange={setQuery}
        />
        <ResultPanel result={result} error={error} emptyText="CSV と SQL を編集して実行してください。" />
      </form>
    </main>
  );
}

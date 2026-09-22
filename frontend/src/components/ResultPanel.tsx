import { useMemo } from "react";
import type { ExecutionMetrics, StatementMetrics } from "../api/client";
import { countResultRows, parseResultSections } from "../utils/csv";
import { DataTable } from "./DataTable";

type ResultPanelProps = {
  result: string;
  error: string;
  emptyText: string;
  metrics?: ExecutionMetrics | null;
};

const metricDefinitions = [
  ["vmSteps", "実行ステップ数", "SQLite 内部で実行された処理量の目安"],
  ["fullScanSteps", "全表走査ステップ数", "インデックスを使わず、テーブルを順番に調べた回数"],
  ["sortOperations", "ソート回数", "SQLite 内部でソート処理が行われた回数"]
] as const;

function formatDuration(durationMs: number) {
  return `${durationMs.toFixed(2)} ms`;
}

function MetricsValues({ metrics }: { metrics: ExecutionMetrics | StatementMetrics }) {
  return (
    <dl className="metrics-grid">
      <div className="metric-item is-primary">
        <dt title="SQL の実行にかかった時間">実行時間</dt>
        <dd>{formatDuration(metrics.durationMs)}</dd>
      </div>
      {metricDefinitions.map(([key, label, description]) =>
        metrics[key] === null ? null : (
          <div className="metric-item" key={key}>
            <dt title={description}>{label}</dt>
            <dd>{metrics[key].toLocaleString("ja-JP")}</dd>
          </div>
        )
      )}
    </dl>
  );
}

function QueryPlan({ lines }: { lines: string[] }) {
  if (lines.length === 0) {
    return null;
  }

  return (
    <div className="query-plan">
      <h4>実行計画</h4>
      <div className="query-plan-lines">
        {lines.map((line, index) => (
          <code
            className={line.startsWith("SEARCH ") ? "is-search" : line.startsWith("SCAN ") ? "is-scan" : ""}
            key={`${line}-${index}`}
          >
            {line}
          </code>
        ))}
      </div>
    </div>
  );
}

function ExecutionMetricsPanel({ metrics }: { metrics: ExecutionMetrics }) {
  return (
    <section className="execution-metrics" aria-label="実行統計">
      <div className="execution-metrics-heading">
        <h3>実行統計</h3>
        <span>{metrics.statements.length} statement</span>
      </div>
      <MetricsValues metrics={metrics} />

      {metrics.statements.length === 1 ? (
        <QueryPlan lines={metrics.statements[0].queryPlan} />
      ) : (
        <div className="statement-metrics-list">
          {metrics.statements.map((statement) => (
            <details key={statement.statementIndex}>
              <summary>
                <span>Statement {statement.statementIndex}</span>
                <strong>{formatDuration(statement.durationMs)}</strong>
              </summary>
              <code className="statement-sql">{statement.statement}</code>
              <MetricsValues metrics={statement} />
              <QueryPlan lines={statement.queryPlan} />
            </details>
          ))}
        </div>
      )}
    </section>
  );
}

export function ResultPanel({ result, error, emptyText, metrics }: ResultPanelProps) {
  const resultSections = useMemo(() => parseResultSections(result), [result]);
  const resultRowCount = useMemo(() => countResultRows(resultSections), [resultSections]);

  return (
    <section className="result-panel" aria-label="実行結果">
      <div className="panel-heading">
        <h2>実行結果</h2>
        <span>{result ? `${resultRowCount} 行` : "待機中"}</span>
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
                <DataTable rows={section.rows} />
              )}
            </div>
          ))}
          {metrics ? <ExecutionMetricsPanel metrics={metrics} /> : null}
        </div>
      ) : null}

      {!error && !result ? (
        <div className="empty-state">
          <p>{emptyText}</p>
        </div>
      ) : null}
    </section>
  );
}

import type { BenchmarkReport } from "../api/client";

type BenchmarkPanelProps = {
  report: BenchmarkReport | null;
  error: string;
  running: boolean;
};

function formatBytes(bytes: number) {
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
}

const benchmarkColumns = [
  ["行数", "計測用の各テーブルに生成した行数"],
  ["実行時間", "提出したSQLの実行にかかった時間"],
  ["実行ステップ数", "SQLite内部で実行された処理ステップの数。処理量を比べる目安になります"],
  ["全表走査ステップ数", "インデックスを使わず、テーブルを先頭から調べたステップの数"],
  ["ソート回数", "SQLite内部でソート処理が行われた回数"],
  ["DBサイズ", "計測用データベースファイルの大きさ"],
  ["データ生成時間", "計測用のテーブルへデータを投入するまでにかかった時間"],
  ["実行計画", "SQLiteが選んだテーブルの読み方やインデックスの使い方"]
] as const;

function BenchmarkHeader({ label, description }: { label: string; description: string }) {
  return (
    <th>
      <span
        className="benchmark-column-heading"
        data-tooltip={description}
        tabIndex={0}
        aria-label={`${label}: ${description}`}
      >
        {label}
        <span className="benchmark-help-icon" aria-hidden="true">?</span>
      </span>
    </th>
  );
}

export function BenchmarkPanel({ report, error, running }: BenchmarkPanelProps) {
  if (!running && !report && !error) {
    return null;
  }

  return (
    <section className="benchmark-panel" aria-label="性能計測">
      <div className="panel-heading">
        <h2>性能計測</h2>
        <span>{running ? "計測中..." : report?.status === "completed" ? "完了" : "一部失敗"}</span>
      </div>
      {error ? <p className="benchmark-error">採点結果には影響しません: {error}</p> : null}
      {report?.error ? <p className="benchmark-error">{report.error}</p> : null}
      {report && report.results.length > 0 ? (
        <div className="table-wrap">
          <table className="benchmark-table">
            <thead>
              <tr>
                {benchmarkColumns.map(([label, description]) => (
                  <BenchmarkHeader key={label} label={label} description={description} />
                ))}
              </tr>
            </thead>
            <tbody>
              {report.results.map((result) => (
                <tr key={result.rowCount}>
                  <td>{result.rowCount.toLocaleString("ja-JP")}</td>
                  <td>{result.executionTimeMs.toFixed(2)} ms</td>
                  <td>{result.vmSteps.toLocaleString("ja-JP")}</td>
                  <td>{result.fullScanSteps.toLocaleString("ja-JP")}</td>
                  <td>{result.sortCount.toLocaleString("ja-JP")}</td>
                  <td>{formatBytes(result.databaseSizeBytes)}</td>
                  <td>{result.dataGenerationMs.toFixed(1)} ms</td>
                  <td>
                    {result.queryPlan.map((line, index) => (
                      <code className={line.startsWith("SEARCH ") ? "is-search" : line.startsWith("SCAN ") ? "is-scan" : ""} key={`${line}-${index}`}>
                        {line}
                      </code>
                    ))}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </section>
  );
}

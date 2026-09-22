import type { BenchmarkReport } from "../api/client";

type BenchmarkPanelProps = {
  report: BenchmarkReport | null;
  error: string;
  running: boolean;
};

function formatBytes(bytes: number) {
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
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
                <th>行数</th>
                <th title="固定SELECTの実行にかかった時間">実行時間</th>
                <th title="SQLite 内部で実行された処理量の目安">VM_STEP</th>
                <th title="インデックスを使わず順番に調べたステップ数">FULLSCAN_STEP</th>
                <th>ソート</th>
                <th>自動index行数</th>
                <th>DBサイズ</th>
                <th>データ生成</th>
                <th>実行計画</th>
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
                  <td>{result.autoIndexRows.toLocaleString("ja-JP")}</td>
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

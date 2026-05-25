import { TaskSubmitResult } from "../api/client";
import { parseCSVPreview } from "../utils/csv";
import { DataTable } from "./DataTable";

type JudgePanelProps = {
  result: TaskSubmitResult | null;
};

export function JudgePanel({ result }: JudgePanelProps) {
  return (
    <section className="judge-panel" aria-label="採点結果">
      <div className="panel-heading">
        <h2>採点結果</h2>
        <span>{result ? (result.passed ? "正解" : "不正解") : "待機中"}</span>
      </div>

      {!result ? (
        <div className="empty-state compact">
          <p>提出するとテストケースごとの結果が表示されます。</p>
        </div>
      ) : (
        <div className="case-list">
          {result.cases.map((testCase) => (
            <article className="case-item" key={testCase.name}>
              <div className="case-heading">
                <h3>{testCase.name}</h3>
                <span className={testCase.passed ? "badge accepted" : "badge failed"}>
                  {testCase.passed ? "正解" : "不正解"}
                </span>
              </div>
              {testCase.error ? <pre className="error-output inline">{testCase.error}</pre> : null}
              {!testCase.passed && !testCase.error ? (
                <div className="comparison-grid">
                  <div>
                    <p className="comparison-label">実行結果</p>
                    <DataTable rows={parseCSVPreview(testCase.actualCsv)} />
                  </div>
                  <div>
                    <p className="comparison-label">期待する結果</p>
                    <DataTable rows={parseCSVPreview(testCase.expectedCsv)} />
                  </div>
                </div>
              ) : null}
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

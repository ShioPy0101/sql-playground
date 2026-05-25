import { TaskSubmitResult } from "../api/client";
import { parseCSVPreview } from "../utils/csv";
import { DataTable } from "./DataTable";

type JudgePanelProps = {
  result: TaskSubmitResult | null;
};

export function JudgePanel({ result }: JudgePanelProps) {
  return (
    <section className="judge-panel" aria-label="Judge result">
      <div className="panel-heading">
        <h2>Judge</h2>
        <span>{result ? (result.passed ? "accepted" : "wrong answer") : "waiting"}</span>
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
                  {testCase.passed ? "AC" : "WA"}
                </span>
              </div>
              {testCase.error ? <pre className="error-output inline">{testCase.error}</pre> : null}
              {!testCase.passed && !testCase.error ? (
                <div className="comparison-grid">
                  <div>
                    <p className="comparison-label">Actual</p>
                    <DataTable rows={parseCSVPreview(testCase.actualCsv)} />
                  </div>
                  <div>
                    <p className="comparison-label">Expected</p>
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

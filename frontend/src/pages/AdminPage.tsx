import { useEffect, useMemo, useState } from "react";
import { fetchAdminSubmissions, TaskSubmission } from "../api/client";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", {
    dateStyle: "medium",
    timeStyle: "medium"
  }).format(new Date(value));
}

function shortUserID(userID: string) {
  return userID.replace(/^user_/, "").slice(0, 10);
}

export function AdminPage() {
  const [submissions, setSubmissions] = useState<TaskSubmission[]>([]);
  const [error, setError] = useState("");
  const [selectedID, setSelectedID] = useState<number | null>(null);

  useEffect(() => {
    let ignore = false;
    setError("");

    fetchAdminSubmissions()
      .then((payload) => {
        if (ignore) {
          return;
        }
        setSubmissions(payload.submissions);
        setSelectedID(payload.submissions[0]?.id ?? null);
      })
      .catch((err) => {
        if (!ignore) {
          setError(err instanceof Error ? err.message : "提出履歴の取得に失敗しました");
        }
      });

    return () => {
      ignore = true;
    };
  }, []);

  const selected = useMemo(
    () => submissions.find((submission) => submission.id === selectedID) ?? submissions[0],
    [selectedID, submissions]
  );

  return (
    <main className="app-shell admin-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">Admin</p>
          <h1>提出履歴</h1>
        </div>
        <a className="secondary-link" href="/">
          問題一覧へ
        </a>
      </header>

      {error ? <pre className="error-output inline">{error}</pre> : null}

      <section className="admin-summary" aria-label="提出サマリー">
        <div>
          <span>提出数</span>
          <strong>{submissions.length}</strong>
        </div>
        <div>
          <span>ユーザー数</span>
          <strong>{new Set(submissions.map((submission) => submission.userId)).size}</strong>
        </div>
        <div>
          <span>正解数</span>
          <strong>{submissions.filter((submission) => submission.passed).length}</strong>
        </div>
      </section>

      {submissions.length === 0 && !error ? (
        <div className="empty-state page">
          <p>まだ提出履歴がありません。</p>
        </div>
      ) : (
        <div className="admin-workspace">
          <section className="admin-table-panel" aria-label="提出一覧">
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>日時</th>
                    <th>ユーザー</th>
                    <th>問題</th>
                    <th>結果</th>
                  </tr>
                </thead>
                <tbody>
                  {submissions.map((submission) => (
                    <tr
                      className={submission.id === selected?.id ? "selected-row" : ""}
                      key={submission.id}
                      onClick={() => setSelectedID(submission.id)}
                    >
                      <td>{formatDate(submission.submittedAt)}</td>
                      <td>
                        <code>{shortUserID(submission.userId)}</code>
                      </td>
                      <td>
                        #{submission.taskNumber} {submission.taskTitle}
                      </td>
                      <td>
                        <span className={submission.passed ? "badge accepted" : "badge failed"}>
                          {submission.passed ? "正解" : "不正解"}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          {selected ? (
            <aside className="admin-detail" aria-label="提出詳細">
              <div className="panel-heading">
                <h2>提出詳細</h2>
                <span>{formatDate(selected.submittedAt)}</span>
              </div>
              <dl className="submission-meta">
                <div>
                  <dt>ユーザー</dt>
                  <dd>{selected.userId}</dd>
                </div>
                <div>
                  <dt>問題</dt>
                  <dd>
                    #{selected.taskNumber} {selected.taskTitle}
                  </dd>
                </div>
                <div>
                  <dt>結果</dt>
                  <dd>{selected.passed ? "正解" : "不正解"}</dd>
                </div>
              </dl>
              <pre className="sql-preview">{selected.query}</pre>
              <div className="admin-cases">
                {selected.cases.map((testCase) => (
                  <div className="case-heading" key={testCase.name}>
                    <h3>{testCase.name}</h3>
                    <span className={testCase.passed ? "badge accepted" : "badge failed"}>
                      {testCase.passed ? "正解" : "不正解"}
                    </span>
                  </div>
                ))}
              </div>
            </aside>
          ) : null}
        </div>
      )}
    </main>
  );
}

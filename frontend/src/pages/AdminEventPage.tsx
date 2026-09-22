import { useCallback, useEffect, useMemo, useState } from "react";
import { AdminEventDetail, EventSubmission, fetchAdminEvent } from "../api/client";
import { BenchmarkPanel } from "../components/BenchmarkPanel";
import { eventRefreshIntervalMs } from "../config";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", { dateStyle: "medium", timeStyle: "medium" }).format(new Date(value));
}

export function AdminEventPage({ slug }: { slug: string }) {
  const [detail, setDetail] = useState<AdminEventDetail | null>(null);
  const [selectedID, setSelectedID] = useState<number | null>(null);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const selected = useMemo(() => detail?.submissions.find((item) => item.id === selectedID) ?? detail?.submissions[0], [detail, selectedID]);

  const load = useCallback(async (background = false) => {
    if (!background) setRefreshing(true);
    try {
      const payload = await fetchAdminEvent(slug);
      setDetail(payload);
      setSelectedID((current) => current !== null && payload.submissions.some((item) => item.id === current) ? current : payload.submissions[0]?.id ?? null);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "イベントの取得に失敗しました");
    } finally {
      if (!background) setRefreshing(false);
    }
  }, [slug]);

  useEffect(() => {
    void load();
    if (!eventRefreshIntervalMs) return;
    const intervalID = window.setInterval(() => void load(true), eventRefreshIntervalMs);
    return () => window.clearInterval(intervalID);
  }, [load]);

  async function copyURL() {
    await navigator.clipboard.writeText(`${window.location.origin}/events/${slug}`);
  }

  if (!detail) return <main className="app-shell"><div className="empty-state page"><p>{error || "イベントを読み込んでいます。"}</p></div></main>;

  return (
    <main className="app-shell admin-shell">
      <header className="app-header"><div><p className="eyebrow">Admin Event</p><h1>{detail.title}</h1></div><div className="task-actions"><button className="secondary-button" disabled={refreshing} onClick={() => void load()}>{refreshing ? "更新中..." : "今すぐ更新"}</button><a className="secondary-link" href="/admin">Adminへ戻る</a></div></header>
      {error ? <pre className="error-output inline">{error}</pre> : null}
      <section className="event-admin-info"><div><span>参加URL</span><a className="text-link" href={`/events/${slug}`}>{window.location.origin}/events/{slug}</a></div><button className="secondary-button" onClick={() => void copyURL()}>URLをコピー</button><div><span>開始時刻</span><strong>{formatDate(detail.startsAt)}</strong></div><div><span>課題</span><strong>{detail.taskNumbers.map((number, index) => <span key={number}>{index ? " → " : ""}<a className="text-link" href={`/tasks/${number}`}>#{number}</a></span>)}</strong></div></section>
      <section className="admin-summary"><div><span>参加者数</span><strong>{detail.participantCount}</strong></div><div><span>提出者数</span><strong>{detail.submitterCount}</strong></div><div><span>総提出数</span><strong>{detail.submissionCount}</strong></div></section>
      <section className="solution-check-panel" aria-label="参加者一覧">
        <div className="panel-heading"><h2>参加者</h2><span>参加順</span></div>
        <div className="table-wrap"><table><thead><tr><th>#</th><th>表示名</th><th>参加日時</th><th>提出数</th></tr></thead><tbody>{detail.participants.map((participant, index) => <tr key={participant.id}><td>{index + 1}</td><td>{participant.username}</td><td>{formatDate(participant.joinedAt)}</td><td>{detail.submissions.filter((submission) => submission.userId === participant.userId).length}</td></tr>)}</tbody></table></div>
      </section>
      <div className="admin-workspace">
        <section className="admin-table-panel"><div className="panel-heading"><h2>提出順</h2><span>古い順</span></div><div className="table-wrap"><table><thead><tr><th>#</th><th>日時</th><th>参加者</th><th>問題</th><th>結果</th></tr></thead><tbody>{detail.submissions.map((submission: EventSubmission, index) => <tr className={submission.id === selected?.id ? "selected-row" : ""} key={submission.id} onClick={() => setSelectedID(submission.id)}><td>{index + 1}</td><td>{formatDate(submission.submittedAt)}</td><td>{submission.username || submission.userId}</td><td>#{submission.taskNumber} {submission.taskTitle}</td><td><span className={submission.passed ? "badge accepted" : "badge failed"}>{submission.passed ? "正解" : "不正解"}</span></td></tr>)}</tbody></table></div></section>
        {selected ? (
          <aside className="admin-detail">
            <div className="panel-heading"><h2>提出詳細</h2><span>{selected.username}</span></div>
            <pre className="sql-preview">{selected.query}</pre>
            {selected.cases.map((testCase) => <div className="case-heading" key={testCase.name}><h3>{testCase.name}</h3><span className={testCase.passed ? "badge accepted" : "badge failed"}>{testCase.passed ? "正解" : "不正解"}</span></div>)}
            <BenchmarkPanel report={selected.benchmarkReport ?? null} error="" running={false} />
          </aside>
        ) : null}
      </div>
    </main>
  );
}

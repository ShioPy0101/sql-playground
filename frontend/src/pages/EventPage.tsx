import { FormEvent, useCallback, useEffect, useState } from "react";
import { EventPageResponse, fetchEvent, joinEvent } from "../api/client";
import { eventRefreshIntervalMs } from "../config";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", { dateStyle: "long", timeStyle: "short" }).format(new Date(value));
}

export function EventPage({ slug }: { slug: string }) {
  const [page, setPage] = useState<EventPageResponse | null>(null);
  const [username, setUsername] = useState("");
  const [error, setError] = useState("");
  const [joining, setJoining] = useState(false);

  const load = useCallback(async () => {
    setError("");
    try {
      setPage(await fetchEvent(slug));
    } catch (err) {
      setError(err instanceof Error ? err.message : "イベントの取得に失敗しました");
    }
  }, [slug]);

  useEffect(() => {
    void load();
    if (!eventRefreshIntervalMs) return;
    const intervalID = window.setInterval(() => void load(), eventRefreshIntervalMs);
    return () => window.clearInterval(intervalID);
  }, [load]);

  async function handleJoin(event: FormEvent) {
    event.preventDefault();
    setJoining(true);
    setError("");
    try {
      await joinEvent(slug, username);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "イベントへの参加に失敗しました");
    } finally {
      setJoining(false);
    }
  }

  if (!page) {
    return <main className="app-shell"><div className="empty-state page"><p>{error || "イベントを読み込んでいます。"}</p></div></main>;
  }

  return (
    <main className="app-shell list-shell">
      <header className="app-header">
        <div><p className="eyebrow">Event</p><h1>{page.event.title}</h1></div>
        <a className="secondary-link" href="/">通常の問題一覧へ</a>
      </header>
      <p className="event-time">開始: {formatDate(page.event.startsAt)}</p>
      {error ? <pre className="error-output inline">{error}</pre> : null}

      {!page.participant ? (
        <section className="event-join-card">
          <h2>イベントに参加</h2>
          <p>イベント内で表示する名前を入力してください。</p>
          <form className="event-join-form" onSubmit={handleJoin}>
            <label>表示名<input value={username} maxLength={40} required onChange={(event) => setUsername(event.target.value)} /></label>
            <button className="run-button" disabled={joining}>{joining ? "参加中..." : "参加する"}</button>
          </form>
        </section>
      ) : !page.started ? (
        <section className="empty-state page">
          <p className="user-chip">参加者: {page.participant.username}</p>
          <h2>開始までお待ちください</h2>
          <p>{formatDate(page.event.startsAt)} に開始します。</p>
          <button className="secondary-button" onClick={() => void load()}>更新</button>
        </section>
      ) : (
        <>
          <p className="user-chip">参加者: {page.participant.username}</p>
          <section className="task-list" aria-label="イベント課題一覧">
            {page.tasks.map((task, index) => (
              <a className="task-list-item" href={`/events/${slug}/tasks/${task.number}`} key={task.number}>
                <div><div className="task-list-meta"><span className="task-number">#{index + 1}</span>{task.solved ? <span className="status-badge solved">正解済み</span> : task.answered ? <span className="status-badge answered">提出済み</span> : null}</div><h2>{task.title}</h2></div>
                <span className="case-count">{task.testCount || 1} cases</span>
              </a>
            ))}
          </section>
        </>
      )}
    </main>
  );
}

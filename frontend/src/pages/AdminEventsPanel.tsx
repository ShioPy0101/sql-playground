import { FormEvent, useEffect, useState } from "react";
import { createAdminEvent, EventSummary, fetchAdminEvents, fetchTasks, TaskSummary } from "../api/client";
import { eventRefreshIntervalMs } from "../config";

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

export function AdminEventsPanel() {
  const [events, setEvents] = useState<EventSummary[]>([]);
  const [tasks, setTasks] = useState<TaskSummary[]>([]);
  const [title, setTitle] = useState("");
  const [slug, setSlug] = useState("");
  const [startsAt, setStartsAt] = useState("");
  const [endsAt, setEndsAt] = useState("");
  const [selectedTasks, setSelectedTasks] = useState<number[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function loadEvents() {
    const payload = await fetchAdminEvents();
    setEvents(payload.events ?? []);
  }

  useEffect(() => {
    void Promise.all([loadEvents(), fetchTasks().then((payload) => setTasks(payload.tasks ?? []))]).catch((err) => setError(err instanceof Error ? err.message : "イベント情報の取得に失敗しました"));
    if (!eventRefreshIntervalMs) return;
    const intervalID = window.setInterval(() => void loadEvents().catch((err) => setError(err instanceof Error ? err.message : "イベント情報の取得に失敗しました")), eventRefreshIntervalMs);
    return () => window.clearInterval(intervalID);
  }, []);

  function toggleTask(number: number) {
    setSelectedTasks((current) => current.includes(number) ? current.filter((item) => item !== number) : [...current, number]);
  }

  function moveTask(index: number, direction: -1 | 1) {
    setSelectedTasks((current) => {
      const target = index + direction;
      if (target < 0 || target >= current.length) return current;
      const next = [...current];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  }

  async function handleCreate(event: FormEvent) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const created = await createAdminEvent({ title, slug, startsAt: new Date(startsAt).toISOString(), ...(endsAt ? { endsAt: new Date(endsAt).toISOString() } : {}), taskNumbers: selectedTasks });
      window.location.href = `/admin/events/${created.slug}`;
    } catch (err) {
      setError(err instanceof Error ? err.message : "イベントの作成に失敗しました");
      setSaving(false);
    }
  }

  return (
    <section className="admin-events" aria-label="イベント管理">
      <div className="panel-heading"><h2>イベント</h2><span>{events.length}件</span></div>
      {error ? <pre className="error-output inline">{error}</pre> : null}
      <div className="admin-event-grid">
        <div className="table-wrap">
          <table><thead><tr><th>イベント</th><th>開始</th><th>参加者</th><th>提出</th></tr></thead>
            <tbody>{events.map((item) => <tr key={item.id}><td><a className="text-link" href={`/admin/events/${item.slug}`}>{item.title}</a></td><td>{formatDate(item.startsAt)}</td><td>{item.participantCount}</td><td>{item.submissionCount}</td></tr>)}</tbody>
          </table>
        </div>
        <form className="event-create-form" onSubmit={handleCreate}>
          <h2>新しいイベント</h2>
          <label>タイトル<input required value={title} onChange={(event) => setTitle(event.target.value)} /></label>
          <label>slug<input required pattern="[a-z0-9]+(?:-[a-z0-9]+)*" placeholder="kc3-2026" value={slug} onChange={(event) => setSlug(event.target.value.toLowerCase())} /></label>
          <label>開始日時<input required type="datetime-local" value={startsAt} onChange={(event) => setStartsAt(event.target.value)} /></label>
          <label>終了日時（任意）<input type="datetime-local" value={endsAt} onChange={(event) => setEndsAt(event.target.value)} /></label>
          <fieldset><legend>課題と表示順</legend>
            <div className="event-task-picker">{tasks.map((task) => <label key={task.number}><input type="checkbox" checked={selectedTasks.includes(task.number)} onChange={() => toggleTask(task.number)} />#{task.number} {task.title}</label>)}</div>
            {selectedTasks.map((number, index) => <div className="event-task-order" key={number}><span>{index + 1}. #{number} {tasks.find((task) => task.number === number)?.title}</span><span><button type="button" className="text-button" disabled={index === 0} onClick={() => moveTask(index, -1)}>↑</button><button type="button" className="text-button" disabled={index === selectedTasks.length - 1} onClick={() => moveTask(index, 1)}>↓</button></span></div>)}
          </fieldset>
          <button className="run-button" disabled={saving || selectedTasks.length === 0}>{saving ? "作成中..." : "イベントを作成"}</button>
        </form>
      </div>
    </section>
  );
}

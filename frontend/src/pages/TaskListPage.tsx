import { useEffect, useState } from "react";
import { fetchTasks, TaskSummary } from "../api/client";

export function TaskListPage() {
  const [tasks, setTasks] = useState<TaskSummary[]>([]);
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let ignore = false;
    setError("");
    setIsLoading(true);

    fetchTasks()
      .then((payload) => {
        if (!ignore) {
          setTasks(payload.tasks);
        }
      })
      .catch((err) => {
        if (!ignore) {
          setError(err instanceof Error ? err.message : "問題一覧の取得に失敗しました");
        }
      })
      .finally(() => {
        if (!ignore) {
          setIsLoading(false);
        }
      });

    return () => {
      ignore = true;
    };
  }, []);

  return (
    <main className="app-shell list-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">SQL課題</p>
          <h1>問題一覧</h1>
        </div>
        <a className="secondary-link" href="/sqlite">
          Playground
        </a>
      </header>

      {error ? <pre className="error-output inline">{error}</pre> : null}

      {isLoading ? (
        <div className="empty-state page">
          <p>問題を読み込んでいます。</p>
        </div>
      ) : tasks.length === 0 && !error ? (
        <div className="empty-state page">
          <p>まだ問題がありません。</p>
        </div>
      ) : (
        <section className="task-list" aria-label="問題一覧">
          {tasks.map((task) => (
            <a className="task-list-item" href={`/tasks/${task.number}`} key={task.number}>
              <div>
                <div className="task-list-meta">
                  <p className="task-number">#{task.number.toString().padStart(3, "0")}</p>
                  {task.answered ? <span className="status-badge answered">解答済み</span> : null}
                  {task.solved ? <span className="status-badge solved">正解済み</span> : null}
                </div>
                <h2>{task.title}</h2>
                <p className="task-summary">{task.statement}</p>
              </div>
              <span className="case-count">{task.testCount} cases</span>
            </a>
          ))}
        </section>
      )}
    </main>
  );
}

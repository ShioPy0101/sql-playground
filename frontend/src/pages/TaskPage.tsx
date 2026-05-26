import { FormEvent, useEffect, useState } from "react";
import {
  fetchCurrentUser,
  fetchTask,
  fetchTasks,
  fetchTaskSubmissions,
  submitTask,
  Task,
  TaskSubmission,
  TaskSubmitResult
} from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { JudgePanel } from "../components/JudgePanel";
import { ResultPanel } from "../components/ResultPanel";
import { TaskDetails } from "../components/TaskDetails";
import { executeSQL } from "../api/client";

type TaskPageProps = {
  number: string;
};

type TaskNavigation = {
  previousNumber?: number;
  nextNumber?: number;
  total: number;
  position?: number;
};

function formatDate(value: string) {
  return new Intl.DateTimeFormat("ja-JP", {
    dateStyle: "medium",
    timeStyle: "medium"
  }).format(new Date(value));
}

export function TaskPage({ number }: TaskPageProps) {
  const [task, setTask] = useState<Task | null>(null);
  const [taskNavigation, setTaskNavigation] = useState<TaskNavigation>({ total: 0 });
  const [query, setQuery] = useState("");
  const [result, setResult] = useState("");
  const [judgeResult, setJudgeResult] = useState<TaskSubmitResult | null>(null);
  const [userId, setUserId] = useState("");
  const [error, setError] = useState("");
  const [historyError, setHistoryError] = useState("");
  const [submissions, setSubmissions] = useState<TaskSubmission[]>([]);
  const [showSolution, setShowSolution] = useState(false);
  const [isRunning, setIsRunning] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    let ignore = false;
    setError("");
    setTask(null);
    setJudgeResult(null);
    setResult("");
    setSubmissions([]);
    setShowSolution(false);

    fetchTask(number)
      .then((payload) => {
        if (ignore) {
          return;
        }
        setTask(payload);
        setQuery(payload.savedQuery || payload.starterSql);
      })
      .catch((err) => {
        if (!ignore) {
          setError(err instanceof Error ? err.message : "問題の取得に失敗しました");
        }
      });

    return () => {
      ignore = true;
    };
  }, [number]);

  useEffect(() => {
    let ignore = false;
    setHistoryError("");

    fetchTaskSubmissions(number)
      .then((payload) => {
        if (!ignore) {
          setSubmissions(payload.submissions ?? []);
        }
      })
      .catch((err) => {
        if (!ignore) {
          setHistoryError(err instanceof Error ? err.message : "解答履歴の取得に失敗しました");
        }
      });

    return () => {
      ignore = true;
    };
  }, [number]);

  useEffect(() => {
    let ignore = false;

    fetchTasks()
      .then((payload) => {
        if (ignore) {
          return;
        }

        const tasks = [...payload.tasks].sort((a, b) => a.number - b.number);
        const currentIndex = tasks.findIndex((summary) => String(summary.number) === number);
        setTaskNavigation({
          previousNumber: currentIndex > 0 ? tasks[currentIndex - 1].number : undefined,
          nextNumber: currentIndex >= 0 && currentIndex < tasks.length - 1 ? tasks[currentIndex + 1].number : undefined,
          total: tasks.length,
          position: currentIndex >= 0 ? currentIndex + 1 : undefined
        });
      })
      .catch(() => {
        if (!ignore) {
          setTaskNavigation({ total: 0 });
        }
      });

    return () => {
      ignore = true;
    };
  }, [number]);

  useEffect(() => {
    let ignore = false;

    fetchCurrentUser()
      .then((payload) => {
        if (!ignore) {
          setUserId(payload.userId);
        }
      })
      .catch(() => {
        if (!ignore) {
          setUserId("");
        }
      });

    return () => {
      ignore = true;
    };
  }, []);

  async function handleRun(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!task) {
      return;
    }

    setIsRunning(true);
    setError("");

    try {
      const payload = await executeSQL(task.csv, query, task.checkSql);
      setResult(payload.csv);
    } catch (err) {
      setError(err instanceof Error ? err.message : "SQL の実行に失敗しました");
      setResult("");
    } finally {
      setIsRunning(false);
    }
  }

  async function handleSubmit() {
    setIsSubmitting(true);
    setError("");

    try {
      const payload = await submitTask(number, query);
      setJudgeResult(payload);
      const history = await fetchTaskSubmissions(number);
      setSubmissions(history.submissions ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "提出に失敗しました");
      setJudgeResult(null);
    } finally {
      setIsSubmitting(false);
    }
  }

  function handleResetQuery() {
    if (!task) {
      return;
    }
    setQuery(task.starterSql);
    setResult("");
    setJudgeResult(null);
    setError("");
  }

  if (!task && !error) {
    return (
      <main className="app-shell">
        <div className="empty-state page">
          <p>問題を読み込んでいます。</p>
        </div>
      </main>
    );
  }

  if (!task) {
    return (
      <main className="app-shell">
        <div className="empty-state page">
          <p>{error}</p>
        </div>
      </main>
    );
  }

  return (
    <main className="app-shell task-shell">
      <nav className="task-nav" aria-label="問題ナビゲーション">
        <a className="text-link" href="/">
          タスク問題に戻る
        </a>
        <div className="task-nav-pager">
          {taskNavigation.previousNumber ? (
            <a className="secondary-link compact" href={`/tasks/${taskNavigation.previousNumber}`}>
              前の問題
            </a>
          ) : (
            <span className="secondary-link compact disabled" aria-disabled="true">
              前の問題
            </span>
          )}
          {taskNavigation.position ? (
            <span className="task-position">
              {taskNavigation.position} / {taskNavigation.total}
            </span>
          ) : null}
          {taskNavigation.nextNumber ? (
            <a className="secondary-link compact" href={`/tasks/${taskNavigation.nextNumber}`}>
              次の問題
            </a>
          ) : (
            <span className="secondary-link compact disabled" aria-disabled="true">
              次の問題
            </span>
          )}
        </div>
      </nav>

      <header className="app-header">
        <div>
          <p className="eyebrow">SQL課題</p>
          <h4>{task.title}</h4>
          {userId ? <p className="user-chip">User {userId.replace(/^user_/, "").slice(0, 10)}</p> : null}
        </div>
        <div className="task-actions">
          <button type="button" className="secondary-button" onClick={() => setShowSolution((current) => !current)}>
            {showSolution ? "答えを隠す" : "答えを見る"}
          </button>
          <button type="button" className="secondary-button" onClick={handleResetQuery} disabled={isRunning || isSubmitting}>
            入力を最初に戻す
          </button>
          <button form="task-form" className="secondary-button" disabled={isRunning || isSubmitting}>
            {isRunning ? "実行中..." : "実行"}
          </button>
          <button className="run-button" disabled={isRunning || isSubmitting} onClick={handleSubmit}>
            {isSubmitting ? "採点中..." : "提出"}
          </button>
        </div>
      </header>

      <TaskDetails task={task} />

      <form id="task-form" className="task-workspace" onSubmit={handleRun}>
        <EditorPanel
          title="解答SQL"
          label="編集できます"
          value={query}
          ariaLabel="解答SQL"
          className="answer-editor"
          onChange={setQuery}
        />
        {/* <EditorPanel title="CSV" label={tableLabel} value={task.csv} ariaLabel="Task CSV" readOnly /> */}
        {showSolution ? (
          <section className="solution-panel" aria-label="想定解答">
            <div className="panel-heading">
              <h2>想定解答</h2>
              <button type="button" className="text-button" onClick={() => setQuery(task.solutionSql)}>
                入力へ反映
              </button>
            </div>
            <pre className="sql-preview">{task.solutionSql || "想定解答が登録されていません。"}</pre>
          </section>
        ) : null}
        <ResultPanel result={result} error={error} emptyText="実行するとサンプルに対する結果を確認できます。" />
        <JudgePanel result={judgeResult} />
        <section className="submission-history" aria-label="解答履歴">
          <div className="panel-heading">
            <h2>解答履歴</h2>
            <span>{submissions.length}件</span>
          </div>
          {historyError ? <pre className="error-output inline">{historyError}</pre> : null}
          {submissions.length === 0 && !historyError ? (
            <div className="empty-state compact">
              <p>まだこの問題への提出はありません。</p>
            </div>
          ) : (
            <div className="history-list">
              {submissions.map((submission) => (
                <article className="history-item" key={submission.id}>
                  <div className="history-item-heading">
                    <div>
                      <span className={submission.passed ? "badge accepted" : "badge failed"}>
                        {submission.passed ? "正解" : "不正解"}
                      </span>
                      <time>{formatDate(submission.submittedAt)}</time>
                    </div>
                    <button type="button" className="text-button" onClick={() => setQuery(submission.query)}>
                      このSQLに戻す
                    </button>
                  </div>
                  <pre className="history-query">{submission.query}</pre>
                </article>
              ))}
            </div>
          )}
        </section>
      </form>

      <nav className="task-bottom-nav" aria-label="次の操作">
        <a className="secondary-link" href="/">
          タスク問題に戻る
        </a>
        {taskNavigation.nextNumber ? (
          <a className="run-button link-button" href={`/tasks/${taskNavigation.nextNumber}`}>
            次の問題へ
          </a>
        ) : (
          <span className="secondary-link disabled" aria-disabled="true">
            最後の問題です
          </span>
        )}
      </nav>
    </main>
  );
}

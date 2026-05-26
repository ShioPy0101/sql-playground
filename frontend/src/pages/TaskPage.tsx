import { FormEvent, useEffect, useMemo, useState } from "react";
import { fetchCurrentUser, fetchTask, fetchTasks, submitTask, Task, TaskSubmitResult } from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { JudgePanel } from "../components/JudgePanel";
import { ResultPanel } from "../components/ResultPanel";
import { TaskDetails } from "../components/TaskDetails";
import { executeSQL } from "../api/client";
import { formatTableLabel } from "../utils/csv";

type TaskPageProps = {
  number: string;
};

type TaskNavigation = {
  previousNumber?: number;
  nextNumber?: number;
  total: number;
  position?: number;
};

export function TaskPage({ number }: TaskPageProps) {
  const [task, setTask] = useState<Task | null>(null);
  const [taskNavigation, setTaskNavigation] = useState<TaskNavigation>({ total: 0 });
  const [query, setQuery] = useState("");
  const [result, setResult] = useState("");
  const [judgeResult, setJudgeResult] = useState<TaskSubmitResult | null>(null);
  const [userId, setUserId] = useState("");
  const [error, setError] = useState("");
  const [isRunning, setIsRunning] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const tableLabel = useMemo(() => formatTableLabel(task?.csv ?? ""), [task]);

  useEffect(() => {
    let ignore = false;
    setError("");
    setTask(null);
    setJudgeResult(null);
    setResult("");

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
      const payload = await executeSQL(task.csv, query);
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
    } catch (err) {
      setError(err instanceof Error ? err.message : "提出に失敗しました");
      setJudgeResult(null);
    } finally {
      setIsSubmitting(false);
    }
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
        <ResultPanel result={result} error={error} emptyText="実行するとサンプルに対する結果を確認できます。" />
        <JudgePanel result={judgeResult} />
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

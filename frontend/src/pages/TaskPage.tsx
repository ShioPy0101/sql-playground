import { FormEvent, useEffect, useMemo, useState } from "react";
import { fetchTask, submitTask, Task, TaskSubmitResult } from "../api/client";
import { EditorPanel } from "../components/EditorPanel";
import { JudgePanel } from "../components/JudgePanel";
import { ResultPanel } from "../components/ResultPanel";
import { TaskDetails } from "../components/TaskDetails";
import { executeSQL } from "../api/client";
import { formatTableLabel } from "../utils/csv";

type TaskPageProps = {
  number: string;
};

export function TaskPage({ number }: TaskPageProps) {
  const [task, setTask] = useState<Task | null>(null);
  const [query, setQuery] = useState("");
  const [result, setResult] = useState("");
  const [judgeResult, setJudgeResult] = useState<TaskSubmitResult | null>(null);
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
        setQuery(payload.starterSql);
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
      <header className="app-header">
        <div>
          <p className="eyebrow">SQL課題</p>
          <h4>{task.title}</h4>
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
    </main>
  );
}

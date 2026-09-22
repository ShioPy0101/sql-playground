import { FormEvent, useEffect, useMemo, useState } from "react";
import {
  fetchCurrentUser,
  fetchEvent,
  fetchEventTask,
  fetchEventTaskSubmissions,
  fetchTask,
  fetchTasks,
  fetchTaskSubmissions,
  benchmarkTask,
  submitTask,
  submitEventTask,
  Task,
  TaskSubmission,
  TaskSubmitResult
} from "../api/client";
import type { BenchmarkReport, ExecutionMetrics } from "../api/client";
import { BenchmarkPanel } from "../components/BenchmarkPanel";
import { EditorPanel } from "../components/EditorPanel";
import { JudgePanel } from "../components/JudgePanel";
import { ResultPanel } from "../components/ResultPanel";
import { TaskDetails } from "../components/TaskDetails";
import { executeSQL } from "../api/client";
import { schemaFromCSV, schemaFromDDL } from "../utils/sqlAutocomplete";

type TaskPageProps = {
  number: string;
  eventSlug?: string;
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

export function TaskPage({ number, eventSlug }: TaskPageProps) {
  const [task, setTask] = useState<Task | null>(null);
  const [taskNavigation, setTaskNavigation] = useState<TaskNavigation>({ total: 0 });
  const [query, setQuery] = useState("");
  const [result, setResult] = useState("");
  const [metrics, setMetrics] = useState<ExecutionMetrics | null>(null);
  const [judgeResult, setJudgeResult] = useState<TaskSubmitResult | null>(null);
  const [benchmarkReport, setBenchmarkReport] = useState<BenchmarkReport | null>(null);
  const [benchmarkError, setBenchmarkError] = useState("");
  const [userId, setUserId] = useState("");
  const [error, setError] = useState("");
  const [historyError, setHistoryError] = useState("");
  const [submissions, setSubmissions] = useState<TaskSubmission[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isBenchmarking, setIsBenchmarking] = useState(false);
  const autocompleteSchema = useMemo(() => {
    if (!task) {
      return { tables: [] };
    }
    return task.inputType === "sql"
      ? schemaFromDDL(task.input ?? "")
      : schemaFromCSV(task.input ?? task.csv);
  }, [task]);

  useEffect(() => {
    let ignore = false;
    setError("");
    setTask(null);
    setJudgeResult(null);
    setBenchmarkReport(null);
    setBenchmarkError("");
    setResult("");
    setMetrics(null);
    setSubmissions([]);

    const request = eventSlug ? fetchEventTask(eventSlug, number) : fetchTask(number);
    request
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
  }, [eventSlug, number]);

  useEffect(() => {
    let ignore = false;
    setHistoryError("");

    const request = eventSlug ? fetchEventTaskSubmissions(eventSlug, number) : fetchTaskSubmissions(number);
    request
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
  }, [eventSlug, number]);

  useEffect(() => {
    let ignore = false;

    const request = eventSlug ? fetchEvent(eventSlug).then((payload) => ({ tasks: payload.tasks })) : fetchTasks();
    request
      .then((payload) => {
        if (ignore) {
          return;
        }

        const tasks = eventSlug ? payload.tasks : [...payload.tasks].sort((a, b) => a.number - b.number);
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
  }, [eventSlug, number]);

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

  function handleFormSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void handleSubmit();
  }

  async function handleSubmit() {
    if (!task) {
      return;
    }

    setIsSubmitting(true);
    setError("");

    try {
      const payload = eventSlug ? await submitEventTask(eventSlug, number, query) : await submitTask(number, query);
      setJudgeResult(payload);
      setBenchmarkReport(null);
      setBenchmarkError("");

      try {
        const execution = await executeSQL(task.input ?? task.csv, query, task.checkSql, task.inputType ?? "csv");
        setResult(execution.csv);
        setMetrics(execution.metrics ?? null);
      } catch (executionFailure) {
        setResult("");
        setMetrics(null);
        setError(executionFailure instanceof Error ? executionFailure.message : "SQL の実行に失敗しました");
      }

      if (task.benchmark?.enabled && (payload.passed || task.benchmark.runOnFailed)) {
        setIsBenchmarking(true);
        void benchmarkTask(number, query, payload.submissionId)
          .then(setBenchmarkReport)
          .catch((benchmarkFailure) => {
            setBenchmarkError(
              benchmarkFailure instanceof Error ? benchmarkFailure.message : "性能計測に失敗しました"
            );
          })
          .finally(() => setIsBenchmarking(false));
      }
      const history = eventSlug
        ? await fetchEventTaskSubmissions(eventSlug, number)
        : await fetchTaskSubmissions(number);
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
    setMetrics(null);
    setJudgeResult(null);
    setBenchmarkReport(null);
    setBenchmarkError("");
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
        <a className="text-link" href={eventSlug ? `/events/${eventSlug}` : "/"}>
          {eventSlug ? "イベントに戻る" : "タスク問題に戻る"}
        </a>
        <div className="task-nav-pager">
          {taskNavigation.previousNumber ? (
            <a className="secondary-link compact" href={eventSlug ? `/events/${eventSlug}/tasks/${taskNavigation.previousNumber}` : `/tasks/${taskNavigation.previousNumber}`}>
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
            <a className="secondary-link compact" href={eventSlug ? `/events/${eventSlug}/tasks/${taskNavigation.nextNumber}` : `/tasks/${taskNavigation.nextNumber}`}>
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
          <button type="button" className="secondary-button" onClick={handleResetQuery} disabled={isSubmitting}>
            入力を最初に戻す
          </button>
          <button form="task-form" className="run-button" disabled={isSubmitting}>
            {isSubmitting ? "採点中..." : "提出"}
          </button>
        </div>
      </header>

      <TaskDetails task={task} />

      <form id="task-form" className="task-workspace" onSubmit={handleFormSubmit}>
        <EditorPanel
          title="解答SQL"
          label="編集できます"
          value={query}
          ariaLabel="解答SQL"
          className="answer-editor"
          sqlAutocomplete={autocompleteSchema}
          onChange={setQuery}
        />
        {/* <EditorPanel title="CSV" label={tableLabel} value={task.csv} ariaLabel="Task CSV" readOnly /> */}
        <details className="solution-panel">
          <summary className="solution-summary">想定解答</summary>
          <div className="solution-actions">
            <span>解答例</span>
            {task.solutionSql ? (
              <button type="button" className="text-button" onClick={() => setQuery(task.solutionSql)}>
                入力へ反映
              </button>
            ) : null}
          </div>
          <pre className="sql-preview">{task.solutionSql || "想定解答が登録されていません。"}</pre>
        </details>
        <ResultPanel
          result={result}
          error={error}
          metrics={metrics}
          emptyText="提出するとサンプルに対する結果を確認できます。"
        />
        <JudgePanel result={judgeResult} />
        <BenchmarkPanel report={benchmarkReport} error={benchmarkError} running={isBenchmarking} />
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
        <a className="secondary-link" href={eventSlug ? `/events/${eventSlug}` : "/"}>
          {eventSlug ? "イベントに戻る" : "タスク問題に戻る"}
        </a>
        {taskNavigation.nextNumber ? (
          <a className="run-button link-button" href={eventSlug ? `/events/${eventSlug}/tasks/${taskNavigation.nextNumber}` : `/tasks/${taskNavigation.nextNumber}`}>
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

export type ExecuteResponse = {
  csv: string;
  metrics?: ExecutionMetrics;
};

export type StatementMetrics = {
  statementIndex: number;
  statement: string;
  durationMs: number;
  vmSteps: number | null;
  fullScanSteps: number | null;
  sortOperations: number | null;
  autoIndexRows: number | null;
  queryPlan: string[];
};

export type ExecutionMetrics = {
  durationMs: number;
  vmSteps: number | null;
  fullScanSteps: number | null;
  sortOperations: number | null;
  autoIndexRows: number | null;
  statements: StatementMetrics[];
};

export type InputFormat = "csv" | "sql";

export type Constraint = {
  name: string;
  value: string;
};

export type Task = {
  number: number;
  mode?: "sql" | "ddl";
  benchmark?: BenchmarkConfig;
  slug: string;
  title: string;
  statement: string;
  constraints: Constraint[];
  csv: string;
  starterSql: string;
  solutionSql: string;
  checkSql: string;
  expectedCsv: string;
  testCount: number;
  savedQuery?: string;
  answered: boolean;
  solved: boolean;
};

export type BenchmarkConfig = {
  enabled: boolean;
  runOnFailed?: boolean;
  timeoutMs?: number;
  target: "submission" | "fixed-query";
  query?: string;
  rowCounts: number[];
  schemaSql?: string;
  dataset?: {
    table: string;
    columns: Array<{
      name: string;
      expression: string;
    }>;
  };
};

export type BenchmarkResult = {
  rowCount: number;
  executionTimeMs: number;
  vmSteps: number;
  fullScanSteps: number;
  sortCount: number;
  autoIndexRows: number;
  queryPlan: string[];
  databaseSizeBytes: number;
  dataGenerationMs: number;
};

export type BenchmarkReport = {
  status: "completed" | "failed";
  query: string;
  results: BenchmarkResult[];
  error?: string;
};

export type TaskSummary = {
  number: number;
  slug: string;
  title: string;
  statement: string;
  testCount: number;
  answered: boolean;
  solved: boolean;
  updatedAt?: string;
};

export type TaskListResponse = {
  tasks: TaskSummary[];
};

export type TaskSubmitResult = {
  passed: boolean;
  cases: TaskCaseResult[];
};

export type TaskCaseResult = {
  name: string;
  passed: boolean;
  actualCsv: string;
  expectedCsv: string;
  error?: string;
};

export type CurrentUser = {
  userId: string;
};

export type TaskSubmission = {
  id: number;
  eventId: number | null;
  userId: string;
  taskNumber: number;
  taskSlug: string;
  taskTitle: string;
  query: string;
  passed: boolean;
  cases: TaskCaseResult[];
  submittedAt: string;
};

export type Event = {
  id: number;
  slug: string;
  title: string;
  startsAt: string;
  endsAt: string | null;
  createdAt: string;
};

export type EventParticipant = {
  id: number;
  eventId: number;
  userId: string;
  username: string;
  joinedAt: string;
};

export type EventPageResponse = {
  event: Event;
  participant: EventParticipant | null;
  started: boolean;
  tasks: TaskSummary[];
};

export type EventSummary = Event & {
  participantCount: number;
  submissionCount: number;
};

export type EventSubmission = TaskSubmission & { username: string };

export type AdminEventDetail = Event & {
  taskNumbers: number[];
  participants: EventParticipant[];
  submissions: EventSubmission[];
  participantCount: number;
  submitterCount: number;
  submissionCount: number;
};

export type AdminSubmissionsResponse = {
  submissions: TaskSubmission[];
};

export type TaskSubmissionsResponse = {
  submissions: TaskSubmission[];
};

export type TaskSolutionCheck = {
  taskNumber: number;
  taskSlug: string;
  taskTitle: string;
  passed: boolean;
  cases: TaskCaseResult[];
  error?: string;
};

export type AdminSolutionChecksResponse = {
  checks: TaskSolutionCheck[];
};

export async function executeSQL(
  input: string,
  query: string,
  checkSql = "",
  inputType: InputFormat = "csv"
) {
  return postJSON<ExecuteResponse>("/api/sqlite/execute", { input, inputType, query, checkSql });
}

export async function fetchCurrentUser() {
  return getJSON<CurrentUser>("/api/me");
}

export async function fetchTask(number: string) {
  return getJSON<Task>(`/api/tasks/${number}`);
}

export async function fetchEvent(slug: string) {
  return getJSON<EventPageResponse>(`/api/events/${encodeURIComponent(slug)}`);
}

export async function joinEvent(slug: string, username: string) {
  return postJSON<EventParticipant>(`/api/events/${encodeURIComponent(slug)}/join`, { username });
}

export async function fetchEventTask(slug: string, number: string) {
  return getJSON<Task>(`/api/events/${encodeURIComponent(slug)}/tasks/${number}`);
}

export async function fetchTasks() {
  return getJSON<TaskListResponse>("/api/tasks");
}

export async function submitTask(number: string, query: string) {
  return postJSON<TaskSubmitResult>(`/api/tasks/${number}/submit`, { query });
}

export async function benchmarkTask(number: string, query: string) {
  return postJSON<BenchmarkReport>(`/api/tasks/${number}/benchmark`, { query });
}

export async function submitEventTask(slug: string, number: string, query: string) {
  return postJSON<TaskSubmitResult>(`/api/events/${encodeURIComponent(slug)}/tasks/${number}/submit`, { query });
}

export async function fetchTaskSubmissions(number: string) {
  return getJSON<TaskSubmissionsResponse>(`/api/tasks/${number}/history`);
}

export async function fetchEventTaskSubmissions(slug: string, number: string) {
  return getJSON<TaskSubmissionsResponse>(`/api/events/${encodeURIComponent(slug)}/tasks/${number}/history`);
}

export async function fetchAdminEvents() {
  return getJSON<{ events: EventSummary[] }>("/api/admin/events");
}

export async function createAdminEvent(input: {
  title: string;
  slug: string;
  startsAt: string;
  endsAt?: string;
  taskNumbers: number[];
}) {
  return postJSON<Event>("/api/admin/events", input);
}

export async function fetchAdminEvent(slug: string) {
  return getJSON<AdminEventDetail>(`/api/admin/events/${encodeURIComponent(slug)}`);
}

export async function fetchAdminSubmissions() {
  return getJSON<AdminSubmissionsResponse>("/api/admin/submissions");
}

export async function checkAdminSolutions() {
  return postJSON<AdminSolutionChecksResponse>("/api/admin/check-solutions", {});
}

async function getJSON<T>(url: string) {
  const response = await fetch(url);
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.message ?? "データの取得に失敗しました");
  }

  return payload as T;
}

async function postJSON<T>(url: string, body: unknown) {
  const response = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(body)
  });

  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.message ?? "SQL の実行に失敗しました");
  }

  return payload as T;
}

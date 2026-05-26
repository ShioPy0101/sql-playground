export type ExecuteResponse = {
  csv: string;
};

export type Constraint = {
  name: string;
  value: string;
};

export type Task = {
  number: number;
  slug: string;
  title: string;
  statement: string;
  constraints: Constraint[];
  csv: string;
  starterSql: string;
  solutionSql: string;
  expectedCsv: string;
  testCount: number;
  savedQuery?: string;
  answered: boolean;
  solved: boolean;
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
  userId: string;
  taskNumber: number;
  taskSlug: string;
  taskTitle: string;
  query: string;
  passed: boolean;
  cases: TaskCaseResult[];
  submittedAt: string;
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

export async function executeSQL(csv: string, query: string) {
  return postJSON<ExecuteResponse>("/api/sqlite/execute", { csv, query });
}

export async function fetchCurrentUser() {
  return getJSON<CurrentUser>("/api/me");
}

export async function fetchTask(number: string) {
  return getJSON<Task>(`/api/tasks/${number}`);
}

export async function fetchTasks() {
  return getJSON<TaskListResponse>("/api/tasks");
}

export async function submitTask(number: string, query: string) {
  return postJSON<TaskSubmitResult>(`/api/tasks/${number}/submit`, { query });
}

export async function fetchTaskSubmissions(number: string) {
  return getJSON<TaskSubmissionsResponse>(`/api/tasks/${number}/history`);
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

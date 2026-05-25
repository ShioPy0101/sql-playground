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
  expectedCsv: string;
  testCount: number;
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

export async function executeSQL(csv: string, query: string) {
  return postJSON<ExecuteResponse>("/api/sqlite/execute", { csv, query });
}

export async function fetchTask(number: string) {
  return getJSON<Task>(`/api/tasks/${number}`);
}

export async function submitTask(number: string, query: string) {
  return postJSON<TaskSubmitResult>(`/api/tasks/${number}/submit`, { query });
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

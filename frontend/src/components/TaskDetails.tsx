import { Task } from "../api/client";
import { parseCSVTables } from "../utils/csv";
import { DataTable } from "./DataTable";

type TaskDetailsProps = {
  task: Task;
};

export function TaskDetails({ task }: TaskDetailsProps) {
  const tables = parseCSVTables(task.csv);

  return (
    <section className="task-details" aria-label="問題詳細">
      <div className="task-statement">
        <div className="panel-heading">
          <h2>問題文</h2>
          <span>#{String(task.number).padStart(3, "0")}</span>
        </div>
        <div className="task-copy">
          <h3>{task.title}</h3>
          {task.note ? <p className="task-note">備考: {task.note}</p> : null}
          <p>{task.statement}</p>
        </div>
      </div>

      <div className="task-constraints">
        <div className="panel-heading">
          <h2>制約・メモ</h2>
          <span>{task.constraints.length} 件</span>
        </div>
        <dl>
          {task.constraints.map((constraint) => (
            <div key={constraint.name}>
              <dt>{constraint.name}</dt>
              <dd>{constraint.value}</dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="task-table">
        <div className="panel-heading">
          <h2>入力テーブル</h2>
          <span>{tables.length} 件</span>
        </div>
        <div className="input-table-list">
          {tables.map((table) => (
            <div className="input-table-item" key={table.name}>
              <p className="input-table-name">{table.name}</p>
              <DataTable rows={table.rows} />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

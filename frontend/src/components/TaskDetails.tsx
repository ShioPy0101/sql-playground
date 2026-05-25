import { Task } from "../api/client";
import { parseCSVPreview } from "../utils/csv";
import { DataTable } from "./DataTable";

type TaskDetailsProps = {
  task: Task;
};

export function TaskDetails({ task }: TaskDetailsProps) {
  return (
    <section className="task-details" aria-label="Task details">
      <div className="task-statement">
        <div className="panel-heading">
          <h2>Problem</h2>
          <span>#{String(task.number).padStart(3, "0")}</span>
        </div>
        <div className="task-copy">
          <h3>{task.title}</h3>
          <p>{task.statement}</p>
        </div>
      </div>

      <div className="task-constraints">
        <div className="panel-heading">
          <h2>Constraints</h2>
          <span>{task.constraints.length} items</span>
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
          <h2>CSV Preview</h2>
          <span>sample input</span>
        </div>
        <DataTable rows={parseCSVPreview(stripTableMarkers(task.csv))} />
      </div>
    </section>
  );
}

function stripTableMarkers(csvText: string) {
  return csvText
    .split("\n")
    .filter((line) => {
      const trimmed = line.trim();
      return !(trimmed.startsWith("[") && trimmed.endsWith("]"));
    })
    .filter((line) => !line.toLowerCase().startsWith("# table:"))
    .filter((line) => !line.toLowerCase().startsWith("-- table:"))
    .join("\n");
}

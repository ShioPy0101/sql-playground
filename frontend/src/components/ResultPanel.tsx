import { useMemo } from "react";
import { countResultRows, parseResultSections } from "../utils/csv";
import { DataTable } from "./DataTable";

type ResultPanelProps = {
  result: string;
  error: string;
  emptyText: string;
};

export function ResultPanel({ result, error, emptyText }: ResultPanelProps) {
  const resultSections = useMemo(() => parseResultSections(result), [result]);
  const resultRowCount = useMemo(() => countResultRows(resultSections), [resultSections]);

  return (
    <section className="result-panel" aria-label="Query result">
      <div className="panel-heading">
        <h2>Result</h2>
        <span>{result ? `${resultRowCount} rows` : "waiting"}</span>
      </div>

      {error ? <pre className="error-output">{error}</pre> : null}

      {!error && resultSections.length > 0 ? (
        <div className="result-content">
          {resultSections.map((section, sectionIndex) => (
            <div className="result-section" key={`${section.title}-${sectionIndex}`}>
              {section.title ? <p className="query-title">{section.title}</p> : null}
              {section.status ? (
                <p className="status-output">{section.status}</p>
              ) : (
                <DataTable rows={section.rows} />
              )}
            </div>
          ))}
        </div>
      ) : null}

      {!error && !result ? (
        <div className="empty-state">
          <p>{emptyText}</p>
        </div>
      ) : null}
    </section>
  );
}

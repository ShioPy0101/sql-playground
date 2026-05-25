type DataTableProps = {
  rows: string[][];
};

export function DataTable({ rows }: DataTableProps) {
  if (rows.length === 0) {
    return null;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {rows[0].map((cell, cellIndex) => (
              <th key={`${cell}-${cellIndex}`}>{cell}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.slice(1).map((row, rowIndex) => (
            <tr key={`${row.join("-")}-${rowIndex}`}>
              {row.map((cell, cellIndex) => (
                <td key={`${cell}-${cellIndex}`}>{cell}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

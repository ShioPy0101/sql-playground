import { useEffect, useRef, useState } from "react";
import { renderTableAsPng } from "../utils/tableImage";

type DataTableProps = {
  rows: string[][];
};

type CopyState = "idle" | "copying" | "success" | "error";

function CopyIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <rect x="4" y="4" width="12" height="12" rx="2" />
      <path d="M8 8h10a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H10a2 2 0 0 1-2-2V8Z" />
      <path d="m10.5 15 2.3-2.4 2 2 1.2-1.2 2 2.1" />
    </svg>
  );
}

function CheckIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24">
      <path d="m5 12.5 4.2 4.2L19 7" />
    </svg>
  );
}

export function DataTable({ rows }: DataTableProps) {
  const tableRef = useRef<HTMLTableElement>(null);
  const resetTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [copyState, setCopyState] = useState<CopyState>("idle");

  useEffect(() => {
    return () => {
      if (resetTimerRef.current) {
        clearTimeout(resetTimerRef.current);
      }
    };
  }, []);

  if (rows.length === 0) {
    return null;
  }

  async function handleCopyImage() {
    if (!tableRef.current || copyState === "copying") {
      return;
    }

    if (resetTimerRef.current) {
      clearTimeout(resetTimerRef.current);
    }
    setCopyState("copying");

    try {
      if (!navigator.clipboard?.write || typeof ClipboardItem === "undefined") {
        throw new Error("このブラウザは画像のコピーに対応していません");
      }

      // Start clipboard.write in the click event so browsers retain user activation
      // while the PNG is being rendered asynchronously.
      const png = renderTableAsPng(tableRef.current);
      await navigator.clipboard.write([new ClipboardItem({ "image/png": png })]);
      setCopyState("success");
    } catch (error) {
      console.error("Failed to copy result table as an image", error);
      setCopyState("error");
    }

    resetTimerRef.current = setTimeout(() => setCopyState("idle"), 1800);
  }

  return (
    <div className="data-table-result">
      <div className="table-copy-action">
        {copyState === "success" ? (
          <span className="copy-feedback success" role="status">
            画像をコピーしました
          </span>
        ) : null}
        {copyState === "error" ? (
          <span className="copy-feedback error" role="alert">
            画像のコピーに失敗しました
          </span>
        ) : null}
        <button
          className={`copy-image-button ${copyState === "success" ? "is-success" : ""}`}
          type="button"
          aria-label="画像としてコピー"
          data-tooltip="画像としてコピー"
          disabled={copyState === "copying"}
          onClick={handleCopyImage}
        >
          {copyState === "success" ? <CheckIcon /> : <CopyIcon />}
        </button>
      </div>
      <div className="table-wrap">
        <table ref={tableRef}>
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
    </div>
  );
}

import { KeyboardEvent, type ReactNode, useId, useMemo, useRef, useState } from "react";
import { getSqlCompletions, type SqlCompletion, type SqlSchema } from "../utils/sqlAutocomplete";

type EditorPanelProps = {
  title: string;
  label: string;
  value: string;
  ariaLabel: string;
  className?: string;
  readOnly?: boolean;
  sqlAutocomplete?: SqlSchema;
  headingAccessory?: ReactNode;
  onChange?: (value: string) => void;
};

export function EditorPanel({
  title,
  label,
  value,
  ariaLabel,
  className = "",
  readOnly = false,
  sqlAutocomplete,
  headingAccessory,
  onChange
}: EditorPanelProps) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const listboxId = useId();
  const [completions, setCompletions] = useState<SqlCompletion[]>([]);
  const [activeCompletion, setActiveCompletion] = useState(0);
  const completion = completions[activeCompletion];
  const autocompleteEnabled = Boolean(sqlAutocomplete && !readOnly && onChange);
  const activeDescendant = completion ? `${listboxId}-${activeCompletion}` : undefined;

  const visibleCompletions = useMemo(() => completions.slice(0, 12), [completions]);

  function updateCompletions(nextValue: string, cursor: number, trigger: "input" | "cursor" | "explicit") {
    if (!sqlAutocomplete) {
      return;
    }

    const next = getSqlCompletions(nextValue, cursor, sqlAutocomplete);
    const inserted = nextValue[cursor - 1] ?? "";
    const mayOpenAfterSpace = inserted === " " && next.some((item) => item.kind === "Table" || item.kind === "Column");
    const shouldOpen = trigger !== "input" || /[\w$.]/.test(inserted) || mayOpenAfterSpace;
    setCompletions(shouldOpen ? next : []);
    setActiveCompletion(0);
  }

  function applyCompletion(item: SqlCompletion) {
    const textarea = textareaRef.current;
    if (!textarea) {
      return;
    }

    textarea.focus();
    textarea.setSelectionRange(item.from, item.to);

    // insertText keeps the completion in the textarea's native undo history.
    // setRangeText is retained for browsers that do not support the command.
    let insertedWithUndo = false;
    try {
      insertedWithUndo = document.execCommand("insertText", false, item.label);
    } catch {
      insertedWithUndo = false;
    }
    if (!insertedWithUndo) {
      textarea.setRangeText(item.label, item.from, item.to, "end");
    }

    onChange?.(textarea.value);
    setCompletions([]);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (autocompleteEnabled && event.ctrlKey && event.code === "Space") {
      event.preventDefault();
      updateCompletions(value, event.currentTarget.selectionStart, "explicit");
      return;
    }
    if (!completion) {
      return;
    }

    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const direction = event.key === "ArrowDown" ? 1 : -1;
      setActiveCompletion((current) => (current + direction + visibleCompletions.length) % visibleCompletions.length);
    } else if (event.key === "Tab") {
      event.preventDefault();
      applyCompletion(completion);
    } else if (event.key === "Escape") {
      event.preventDefault();
      setCompletions([]);
    }
  }

  return (
    <section className={`editor-panel ${className}`.trim()} aria-label={ariaLabel}>
      <div className="panel-heading">
        <h2>{title}</h2>
        <div className="panel-heading-meta">
          <span>{label}</span>
          {headingAccessory}
        </div>
      </div>
      <div className="editor-input">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(event) => {
            const nextValue = event.target.value;
            onChange?.(nextValue);
            if (autocompleteEnabled) {
              updateCompletions(nextValue, event.target.selectionStart, "input");
            }
          }}
          onKeyDown={handleKeyDown}
          onClick={(event) => {
            if (autocompleteEnabled && completions.length > 0) {
              updateCompletions(value, event.currentTarget.selectionStart, "cursor");
            }
          }}
          onBlur={() => setCompletions([])}
          readOnly={readOnly}
          spellCheck={false}
          aria-label={ariaLabel}
          aria-autocomplete={autocompleteEnabled ? "list" : undefined}
          aria-controls={autocompleteEnabled ? listboxId : undefined}
          aria-expanded={autocompleteEnabled ? completions.length > 0 : undefined}
          aria-activedescendant={activeDescendant}
        />
        {visibleCompletions.length > 0 ? (
          <ul id={listboxId} className="sql-completions" role="listbox" aria-label="SQL入力候補">
            {visibleCompletions.map((item, index) => (
              <li
                id={`${listboxId}-${index}`}
                key={`${item.kind}-${item.label}-${item.detail ?? ""}`}
                className={index === activeCompletion ? "is-active" : ""}
                role="option"
                aria-selected={index === activeCompletion}
                onMouseDown={(event) => {
                  event.preventDefault();
                  applyCompletion(item);
                }}
                onMouseEnter={() => setActiveCompletion(index)}
              >
                <span className="sql-completion-label">{item.label}</span>
                <span className="sql-completion-detail">{item.detail ?? item.kind}</span>
                {item.detail ? <span className={`sql-completion-kind kind-${item.kind.toLowerCase()}`}>{item.kind}</span> : null}
              </li>
            ))}
          </ul>
        ) : null}
      </div>
    </section>
  );
}

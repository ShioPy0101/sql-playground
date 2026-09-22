import type { ReactNode } from "react";

type EditorPanelProps = {
  title: string;
  label: string;
  value: string;
  ariaLabel: string;
  className?: string;
  readOnly?: boolean;
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
  headingAccessory,
  onChange
}: EditorPanelProps) {
  return (
    <section className={`editor-panel ${className}`.trim()} aria-label={ariaLabel}>
      <div className="panel-heading">
        <h2>{title}</h2>
        <div className="panel-heading-meta">
          <span>{label}</span>
          {headingAccessory}
        </div>
      </div>
      <textarea
        value={value}
        onChange={(event) => onChange?.(event.target.value)}
        readOnly={readOnly}
        spellCheck={false}
        aria-label={ariaLabel}
      />
    </section>
  );
}

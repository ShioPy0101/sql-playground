type EditorPanelProps = {
  title: string;
  label: string;
  value: string;
  ariaLabel: string;
  readOnly?: boolean;
  onChange?: (value: string) => void;
};

export function EditorPanel({
  title,
  label,
  value,
  ariaLabel,
  readOnly = false,
  onChange
}: EditorPanelProps) {
  return (
    <section className="editor-panel" aria-label={ariaLabel}>
      <div className="panel-heading">
        <h2>{title}</h2>
        <span>{label}</span>
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

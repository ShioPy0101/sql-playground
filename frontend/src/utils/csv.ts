export type ResultSection = {
  title: string;
  rows: string[][];
  status: string;
};

export function parseResultSections(resultText: string): ResultSection[] {
  const trimmed = resultText.trim();
  if (!trimmed) {
    return [];
  }

  const sections: Array<{ title: string; lines: string[] }> = [];
  let current = { title: "", lines: [] as string[] };

  for (const line of trimmed.split("\n")) {
    if (line.startsWith("-- Query ")) {
      if (current.title || current.lines.length > 0) {
        sections.push(current);
      }
      current = { title: line.replace(/^--\s*/, ""), lines: [] };
      continue;
    }

    current.lines.push(line);
  }

  if (current.title || current.lines.length > 0) {
    sections.push(current);
  }

  return sections.map((section) => {
    const body = section.lines.join("\n").trim();
    const status = body === "OK" ? body : "";

    return {
      title: section.title,
      rows: status ? [] : parseCSVPreview(body),
      status
    };
  });
}

export function countResultRows(sections: ResultSection[]) {
  return sections.reduce((total, section) => {
    if (section.status) {
      return total;
    }
    return total + Math.max(section.rows.length - 1, 0);
  }, 0);
}

export function parseCSVPreview(csvText: string) {
  return csvText
    .trim()
    .split("\n")
    .filter(Boolean)
    .map((line) => line.split(",").map((cell) => cell.trim()));
}

export function formatTableLabel(csvText: string) {
  const names = parseTableNames(csvText);

  if (names.length === 1) {
    return `table: ${names[0]}`;
  }

  return `tables: ${names.join(", ")}`;
}

export function parseTableNames(csvText: string) {
  const markerNames = csvText
    .split("\n")
    .map((line) => parseTableMarker(line))
    .filter((name): name is string => Boolean(name));

  if (markerNames.length === 0) {
    return ["input"];
  }

  return Array.from(new Set(markerNames));
}

function parseTableMarker(line: string) {
  const trimmed = line.trim();
  if (!trimmed) {
    return "";
  }

  if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
    return trimmed.slice(1, -1).trim();
  }

  const lower = trimmed.toLowerCase();
  for (const prefix of ["# table:", "-- table:"]) {
    if (lower.startsWith(prefix)) {
      return trimmed.slice(prefix.length).trim();
    }
  }

  return "";
}

import { parseCSVTables } from "./csv";

export type SqlSchema = {
  tables: Array<{
    name: string;
    columns: Array<{
      name: string;
      type?: string;
    }>;
  }>;
};

export type SqlCompletionKind = "Table" | "Column" | "Keyword" | "Function";

export type SqlCompletion = {
  label: string;
  kind: SqlCompletionKind;
  detail?: string;
  from: number;
  to: number;
};

const KEYWORDS = [
  "SELECT",
  "FROM",
  "WHERE",
  "JOIN",
  "LEFT JOIN",
  "INNER JOIN",
  "ON",
  "GROUP BY",
  "ORDER BY",
  "HAVING",
  "LIMIT",
  "INSERT INTO",
  "UPDATE",
  "DELETE FROM",
  "CREATE TABLE",
  "CREATE INDEX",
  "UNIQUE",
  "PRIMARY KEY",
  "FOREIGN KEY",
  "REFERENCES",
  "AND",
  "OR",
  "NOT",
  "NULL",
  "IS NULL",
  "ASC",
  "DESC"
] as const;

const FUNCTIONS = ["COUNT", "SUM", "AVG", "MIN", "MAX"] as const;
const IDENTIFIER_SOURCE = String.raw`(?:"(?:[^"]|"")+"|\[[^\]]+\]|` + "`(?:[^`]|``)+`" + String.raw`|[A-Za-z_][\w$]*)`;
const RESERVED_WORDS = new Set([...KEYWORDS.flatMap((keyword) => keyword.split(" ")), ...FUNCTIONS, "AS"].map((word) => word.toUpperCase()));

export function schemaFromCSV(csv: string): SqlSchema {
  return {
    tables: parseCSVTables(csv).map((table) => {
      const [headers = [], ...records] = table.rows;
      return {
        name: table.name,
        columns: headers.map((header, index) => ({
          name: header || `column_${index + 1}`,
          type: inferCSVColumnType(records, index)
        }))
      };
    })
  };
}

export function schemaFromDDL(sql: string): SqlSchema {
  const tables: SqlSchema["tables"] = [];
  const sanitized = sanitizeSql(sql, { keepQuotedIdentifiers: true });
  const createTable = new RegExp(
    String.raw`\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(${IDENTIFIER_SOURCE})\s*\(`,
    "gi"
  );

  for (let match = createTable.exec(sanitized); match; match = createTable.exec(sanitized)) {
    const openingParenthesis = createTable.lastIndex - 1;
    const closingParenthesis = findClosingParenthesis(sql, openingParenthesis);
    if (closingParenthesis < 0) {
      continue;
    }

    const columns = splitTopLevel(sql.slice(openingParenthesis + 1, closingParenthesis), ",")
      .map(parseColumnDefinition)
      .filter((column): column is { name: string; type?: string } => column !== null);
    tables.push({ name: unquoteIdentifier(match[1]), columns });
    createTable.lastIndex = closingParenthesis + 1;
  }

  return { tables };
}

export function mergeSqlSchemas(...schemas: SqlSchema[]): SqlSchema {
  const tables = new Map<string, SqlSchema["tables"][number]>();

  for (const schema of schemas) {
    for (const table of schema.tables) {
      const key = table.name.toLowerCase();
      const existing = tables.get(key);
      if (!existing) {
        tables.set(key, { name: table.name, columns: [...table.columns] });
        continue;
      }

      const columns = new Map(existing.columns.map((column) => [column.name.toLowerCase(), column]));
      for (const column of table.columns) {
        columns.set(column.name.toLowerCase(), column);
      }
      existing.columns = [...columns.values()];
    }
  }

  return { tables: [...tables.values()] };
}

export function getSqlCompletions(sql: string, cursor: number, baseSchema: SqlSchema): SqlCompletion[] {
  const beforeCursor = sql.slice(0, cursor);
  const sanitized = sanitizeSql(beforeCursor);
  const schema = mergeSqlSchemas(baseSchema, schemaFromDDL(beforeCursor));
  const wordMatch = sanitized.match(/[A-Za-z_][\w$]*$/);
  const word = wordMatch?.[0] ?? "";
  const wordStart = cursor - word.length;
  const beforeWord = sanitized.slice(0, wordStart);

  const qualified = beforeCursor.match(new RegExp(`(${IDENTIFIER_SOURCE})\\.([A-Za-z_][\\w$]*)?$`, "i"));
  if (qualified) {
    const qualifier = unquoteIdentifier(qualified[1]);
    const prefix = qualified[2] ?? "";
    const table = resolveTable(qualifier, sanitized, schema);
    return table
      ? filterAndSort(
          table.columns.map((column) => columnCompletion(column, table.name, cursor - prefix.length, cursor)),
          prefix
        )
      : [];
  }

  const indexedTable = tableForCreateIndex(sanitized, schema);
  if (indexedTable) {
    return filterAndSort(
      indexedTable.columns.map((column) => columnCompletion(column, indexedTable.name, wordStart, cursor)),
      word
    );
  }

  if (expectsTable(beforeWord)) {
    return filterAndSort(
      schema.tables.map((table) => ({ label: table.name, kind: "Table", from: wordStart, to: cursor })),
      word
    );
  }

  const showEmptyPrefix = expectsColumn(beforeWord);
  if (!word && !showEmptyPrefix) {
    return [];
  }

  const seenColumns = new Set<string>();
  const columns = schema.tables.flatMap((table) =>
    table.columns.flatMap((column) => {
      const key = column.name.toLowerCase();
      if (seenColumns.has(key)) {
        return [];
      }
      seenColumns.add(key);
      return [columnCompletion(column, table.name, wordStart, cursor)];
    })
  );
  const keywords: SqlCompletion[] = KEYWORDS.map((label) => ({
    label,
    kind: "Keyword",
    ...keywordReplacementRange(label, sanitized, wordStart, cursor)
  }));
  const functions: SqlCompletion[] = FUNCTIONS.map((label) => ({
    label,
    kind: "Function",
    from: wordStart,
    to: cursor
  }));

  return filterAndSort([...columns, ...keywords, ...functions], word);
}

function inferCSVColumnType(records: string[][], column: number) {
  if (records.length === 0) {
    return "TEXT";
  }

  return records.every((record) => isIntegerLiteral(record[column] ?? "")) ? "INTEGER" : "TEXT";
}

function isIntegerLiteral(value: string) {
  const trimmed = value.trim();
  if (!/^[+-]?\d+$/.test(trimmed)) {
    return false;
  }
  const unsigned = trimmed.replace(/^[+-]/, "");
  return unsigned.length === 1 || !unsigned.startsWith("0");
}

function parseColumnDefinition(definition: string) {
  const trimmed = definition.trim();
  const identifier = trimmed.match(new RegExp(`^(${IDENTIFIER_SOURCE})(?:\\s+|$)`, "i"));
  if (!identifier) {
    return null;
  }

  const name = unquoteIdentifier(identifier[1]);
  if (["CONSTRAINT", "PRIMARY", "FOREIGN", "UNIQUE", "CHECK"].includes(name.toUpperCase())) {
    return null;
  }

  const remainder = trimmed.slice(identifier[0].length);
  const type = remainder.match(/^([A-Za-z]+(?:\s*\([^)]*\))?)/)?.[1]?.toUpperCase();
  return { name, ...(type ? { type } : {}) };
}

function findClosingParenthesis(sql: string, openingIndex: number) {
  let depth = 0;
  let quote = "";

  for (let index = openingIndex; index < sql.length; index += 1) {
    const character = sql[index];
    if (quote) {
      if (character === quote) {
        if (sql[index + 1] === quote && quote !== "]") {
          index += 1;
        } else {
          quote = "";
        }
      }
      continue;
    }
    if (character === "'" || character === '"' || character === "`" || character === "[") {
      quote = character === "[" ? "]" : character;
    } else if (character === "(") {
      depth += 1;
    } else if (character === ")" && --depth === 0) {
      return index;
    }
  }

  return -1;
}

function splitTopLevel(value: string, separator: string) {
  const parts: string[] = [];
  let start = 0;
  let depth = 0;
  let quote = "";

  for (let index = 0; index < value.length; index += 1) {
    const character = value[index];
    if (quote) {
      if (character === quote) {
        if (value[index + 1] === quote && quote !== "]") {
          index += 1;
        } else {
          quote = "";
        }
      }
      continue;
    }
    if (character === "'" || character === '"' || character === "`" || character === "[") {
      quote = character === "[" ? "]" : character;
    } else if (character === "(") {
      depth += 1;
    } else if (character === ")") {
      depth -= 1;
    } else if (character === separator && depth === 0) {
      parts.push(value.slice(start, index));
      start = index + 1;
    }
  }
  parts.push(value.slice(start));
  return parts;
}

function sanitizeSql(sql: string, options: { keepQuotedIdentifiers?: boolean } = {}) {
  let result = "";
  let state: "normal" | "single" | "double" | "backtick" | "bracket" | "line-comment" | "block-comment" = "normal";

  for (let index = 0; index < sql.length; index += 1) {
    const character = sql[index];
    const next = sql[index + 1];
    if (state === "normal") {
      if (character === "-" && next === "-") {
        state = "line-comment";
        result += "  ";
        index += 1;
      } else if (character === "/" && next === "*") {
        state = "block-comment";
        result += "  ";
        index += 1;
      } else if (character === "'") {
        state = "single";
        result += " ";
      } else if (character === '"' || character === "`" || character === "[") {
        state = character === '"' ? "double" : character === "`" ? "backtick" : "bracket";
        result += options.keepQuotedIdentifiers ? character : " ";
      } else {
        result += character;
      }
      continue;
    }

    const closing = state === "single" ? "'" : state === "double" ? '"' : state === "backtick" ? "`" : state === "bracket" ? "]" : "";
    if (state === "line-comment") {
      if (character === "\n") {
        state = "normal";
        result += "\n";
      } else {
        result += " ";
      }
    } else if (state === "block-comment") {
      if (character === "*" && next === "/") {
        state = "normal";
        result += "  ";
        index += 1;
      } else {
        result += character === "\n" ? "\n" : " ";
      }
    } else if (character === closing) {
      result += options.keepQuotedIdentifiers ? character : " ";
      if (next === closing && state !== "bracket") {
        result += options.keepQuotedIdentifiers ? next : " ";
        index += 1;
      } else {
        state = "normal";
      }
    } else {
      result += options.keepQuotedIdentifiers ? character : character === "\n" ? "\n" : " ";
    }
  }

  return result;
}

function unquoteIdentifier(identifier: string) {
  const first = identifier[0];
  const last = identifier[identifier.length - 1];
  if ((first === '"' && last === '"') || (first === "`" && last === "`") || (first === "[" && last === "]")) {
    const inner = identifier.slice(1, -1);
    return first === "[" ? inner : inner.replaceAll(first + first, first);
  }
  return identifier;
}

function resolveTable(qualifier: string, sql: string, schema: SqlSchema) {
  const aliases = new Map<string, string>();
  const tableReference = new RegExp(
    String.raw`\b(?:FROM|JOIN|UPDATE|INTO)\s+(${IDENTIFIER_SOURCE})(?:\s+(?:AS\s+)?(${IDENTIFIER_SOURCE}))?`,
    "gi"
  );
  for (let match = tableReference.exec(sql); match; match = tableReference.exec(sql)) {
    const table = unquoteIdentifier(match[1]);
    aliases.set(table.toLowerCase(), table);
    if (match[2]) {
      const alias = unquoteIdentifier(match[2]);
      if (!RESERVED_WORDS.has(alias.toUpperCase())) {
        aliases.set(alias.toLowerCase(), table);
      }
    }
  }

  const tableName = aliases.get(qualifier.toLowerCase()) ?? qualifier;
  return schema.tables.find((table) => table.name.toLowerCase() === tableName.toLowerCase());
}

function tableForCreateIndex(sql: string, schema: SqlSchema) {
  const pattern = new RegExp(
    String.raw`\bCREATE\s+(?:UNIQUE\s+)?INDEX\b[^;]*?\bON\s+(${IDENTIFIER_SOURCE})\s*\([^)]*$`,
    "gi"
  );
  let tableName = "";
  for (let match = pattern.exec(sql); match; match = pattern.exec(sql)) {
    tableName = unquoteIdentifier(match[1]);
  }
  return schema.tables.find((table) => table.name.toLowerCase() === tableName.toLowerCase());
}

function expectsTable(beforeWord: string) {
  return (
    /\b(?:FROM|JOIN|UPDATE|INSERT\s+INTO|DELETE\s+FROM)\s*$/i.test(beforeWord) ||
    /\bCREATE\s+(?:UNIQUE\s+)?INDEX\b[^;]*\bON\s*$/i.test(beforeWord)
  );
}

function expectsColumn(beforeWord: string) {
  return /\b(?:SELECT|WHERE|HAVING|ON|ORDER\s+BY|GROUP\s+BY)\s*$/i.test(beforeWord);
}

function columnCompletion(
  column: SqlSchema["tables"][number]["columns"][number],
  tableName: string,
  from: number,
  to: number
): SqlCompletion {
  return {
    label: column.name,
    kind: "Column",
    detail: column.type ? `${column.type} · ${tableName}` : tableName,
    from,
    to
  };
}

function keywordReplacementRange(label: string, sql: string, wordStart: number, cursor: number) {
  if (!label.includes(" ")) {
    return { from: wordStart, to: cursor };
  }

  const lowerLabel = label.toLowerCase();
  const earliest = Math.max(0, cursor - label.length - 8);
  for (let start = earliest; start < wordStart; start += 1) {
    if (!/[A-Za-z_]/.test(sql[start])) {
      continue;
    }
    if (start > 0 && /[\w$]/.test(sql[start - 1])) {
      continue;
    }
    const typed = sql.slice(start, cursor).trim().replace(/\s+/g, " ").toLowerCase();
    if (typed && lowerLabel.startsWith(typed)) {
      return { from: start, to: cursor };
    }
  }
  return { from: wordStart, to: cursor };
}

function filterAndSort(items: SqlCompletion[], prefix: string) {
  const normalized = prefix.toLowerCase();
  const kindOrder: Record<SqlCompletionKind, number> = { Table: 0, Column: 1, Keyword: 2, Function: 3 };
  return items
    .filter((item) => !normalized || item.label.toLowerCase().startsWith(normalized))
    .sort((left, right) => kindOrder[left.kind] - kindOrder[right.kind])
    .slice(0, 80);
}

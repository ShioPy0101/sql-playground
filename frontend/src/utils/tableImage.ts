const MIN_EXPORT_SCALE = 2;
const MAX_EXPORT_SCALE = 4;
const MIN_EXPORT_PIXEL_WIDTH = 1800;
const EXPORT_PADDING = 80;
const EXPORT_TABLE_MAX_WIDTH = 2400;
const NUMERIC_COLUMN_MIN_WIDTH = 128;
const NUMERIC_COLUMN_MAX_WIDTH = 300;
const TEXT_COLUMN_MIN_WIDTH = 200;
const TEXT_COLUMN_MAX_WIDTH = 680;
const CELL_HORIZONTAL_PADDING = 56;

function isNumericColumn(table: HTMLTableElement, columnIndex: number) {
  const values = Array.from(table.tBodies[0]?.rows ?? [])
    .map((row) => row.cells[columnIndex]?.textContent?.trim() ?? "")
    .filter(Boolean);

  return (
    values.length > 0 &&
    values.every((value) => /^[-+]?(?:\d+\.?\d*|\.\d+)(?:e[-+]?\d+)?$/i.test(value))
  );
}

function loadImage(source: string) {
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("画像の生成に失敗しました"));
    image.src = source;
  });
}

function canvasToBlob(canvas: HTMLCanvasElement) {
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((blob) => {
      if (blob) {
        resolve(blob);
      } else {
        reject(new Error("PNGの生成に失敗しました"));
      }
    }, "image/png");
  });
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.min(Math.max(value, minimum), maximum);
}

function getColumnWidths(table: HTMLTableElement, fontFamily: string) {
  const columnCount = Math.max(...Array.from(table.rows, (row) => row.cells.length));
  const measuringCanvas = document.createElement("canvas");
  const context = measuringCanvas.getContext("2d");
  const numericColumns = Array.from({ length: columnCount }, (_, index) =>
    isNumericColumn(table, index)
  );

  const widths = Array.from({ length: columnCount }, (_, columnIndex) => {
    const isNumeric = numericColumns[columnIndex];
    let widestContent = 0;

    Array.from(table.rows).forEach((row, rowIndex) => {
      const value = row.cells[columnIndex]?.textContent ?? "";
      if (context) {
        context.font = `${rowIndex === 0 ? "700 34px" : "400 30px"} ${fontFamily}`;
        widestContent = Math.max(widestContent, context.measureText(value).width);
      } else {
        widestContent = Math.max(widestContent, Array.from(value).length * 30);
      }
    });

    return Math.ceil(
      clamp(
        widestContent + CELL_HORIZONTAL_PADDING,
        isNumeric ? NUMERIC_COLUMN_MIN_WIDTH : TEXT_COLUMN_MIN_WIDTH,
        isNumeric ? NUMERIC_COLUMN_MAX_WIDTH : TEXT_COLUMN_MAX_WIDTH
      )
    );
  });

  const totalWidth = widths.reduce((total, width) => total + width, 0);

  const minimumWidths = numericColumns.map((isNumeric) =>
    isNumeric ? NUMERIC_COLUMN_MIN_WIDTH : TEXT_COLUMN_MIN_WIDTH
  );
  const minimumTotal = minimumWidths.reduce((total, width) => total + width, 0);

  if (totalWidth > EXPORT_TABLE_MAX_WIDTH && minimumTotal < EXPORT_TABLE_MAX_WIDTH) {
    const reducibleWidth = totalWidth - minimumTotal;
    const targetReduction = totalWidth - EXPORT_TABLE_MAX_WIDTH;
    widths.forEach((width, index) => {
      const reducibleColumnWidth = width - minimumWidths[index];
      widths[index] -= targetReduction * (reducibleColumnWidth / reducibleWidth);
    });
  }

  return widths.map(Math.ceil);
}

/**
 * Builds a presentation-sized copy of the table instead of capturing its
 * scroll viewport. The temporary DOM is also useful for resolving Japanese
 * system fonts before the table is serialized into an SVG foreignObject.
 */
export async function renderTableAsPng(table: HTMLTableElement) {
  await document.fonts?.ready;

  const tableStyle = getComputedStyle(table);

  const stage = document.createElement("div");
  stage.className = "table-image-stage";

  const exportRoot = document.createElement("div");
  exportRoot.className = "table-image-export";
  exportRoot.style.setProperty("--export-color", "#202124");
  exportRoot.style.setProperty("--export-font", tableStyle.fontFamily);

  const style = document.createElement("style");
  style.textContent = `
    .table-image-export {
      display: inline-flex;
      padding: ${EXPORT_PADDING}px;
      box-sizing: border-box;
      color: var(--export-color);
      background: #ffffff;
      font-family: var(--export-font);
      font-synthesis: none;
      text-rendering: geometricPrecision;
    }
    .table-image-export__card {
      overflow: hidden;
      width: max-content;
      border: 2px solid #dfe3e8;
      background: #ffffff;
    }
    .table-image-export table {
      width: auto;
      border: 0;
      border-collapse: separate;
      border-spacing: 0;
      table-layout: fixed;
      color: var(--export-color);
      background: #ffffff;
      font-family: var(--export-font);
      font-size: 30px;
      line-height: 1.5;
    }
    .table-image-export th,
    .table-image-export td {
      position: static;
      padding: 20px 28px;
      border: 0;
      border-right: 1px solid #e1e5ea;
      border-bottom: 1px solid #e1e5ea;
      box-sizing: border-box;
      text-align: left;
      vertical-align: top;
      white-space: pre-wrap;
      overflow-wrap: anywhere;
    }
    .table-image-export th {
      color: #3c4043;
      background: #f7f8fa;
      font-size: 34px;
      font-weight: 700;
      letter-spacing: 0.01em;
    }
    .table-image-export td { background: #ffffff; }
    .table-image-export .is-numeric {
      text-align: right;
      font-variant-numeric: tabular-nums;
    }
    .table-image-export tr > :last-child { border-right: 0; }
    .table-image-export tbody tr:last-child > * { border-bottom: 0; }
  `;

  const card = document.createElement("div");
  card.className = "table-image-export__card";
  const tableCopy = table.cloneNode(true) as HTMLTableElement;
  const columnWidths = getColumnWidths(table, tableStyle.fontFamily);
  const colgroup = document.createElement("colgroup");

  columnWidths.forEach((width) => {
    const column = document.createElement("col");
    column.style.width = `${width}px`;
    colgroup.append(column);
  });
  tableCopy.style.width = `${columnWidths.reduce((total, width) => total + width, 0)}px`;
  tableCopy.prepend(colgroup);

  Array.from(tableCopy.rows).forEach((row) => {
    Array.from(row.cells).forEach((cell, columnIndex) => {
      if (isNumericColumn(table, columnIndex)) {
        cell.classList.add("is-numeric");
      }
    });
  });

  card.append(tableCopy);
  exportRoot.append(style, card);
  stage.append(exportRoot);
  document.body.append(stage);

  try {
    const width = Math.ceil(exportRoot.scrollWidth);
    const height = Math.ceil(exportRoot.scrollHeight);
    const scale = clamp(MIN_EXPORT_PIXEL_WIDTH / width, MIN_EXPORT_SCALE, MAX_EXPORT_SCALE);
    const serialized = new XMLSerializer().serializeToString(exportRoot);
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><foreignObject width="100%" height="100%">${serialized}</foreignObject></svg>`;
    // A blob: URL containing foreignObject is treated as cross-origin by some
    // browsers and taints the canvas. The SVG is fully self-contained, so load
    // it as a data URL to keep the resulting canvas exportable.
    const dataUrl = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
    const image = await loadImage(dataUrl);
    const canvas = document.createElement("canvas");
    canvas.width = Math.ceil(width * scale);
    canvas.height = Math.ceil(height * scale);

    const context = canvas.getContext("2d");
    if (!context) {
      throw new Error("Canvasを初期化できませんでした");
    }

    context.scale(scale, scale);
    context.drawImage(image, 0, 0, width, height);
    return await canvasToBlob(canvas);
  } finally {
    stage.remove();
  }
}

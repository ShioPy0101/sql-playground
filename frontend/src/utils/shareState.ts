export type PlaygroundShareState = {
  csv: string;
  sql: string;
  dialect: string;
};

function bytesToBase64Url(bytes: Uint8Array) {
  let binary = "";
  const chunkSize = 0x8000;

  for (let offset = 0; offset < bytes.length; offset += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + chunkSize));
  }

  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "");
}

function base64UrlToBytes(encoded: string) {
  if (!encoded || !/^[A-Za-z0-9_-]+$/.test(encoded)) {
    throw new Error("共有URLのstateが不正です");
  }

  const base64 = encoded.replace(/-/g, "+").replace(/_/g, "/");
  const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
  const binary = atob(padded);
  return Uint8Array.from(binary, (character) => character.charCodeAt(0));
}

async function transformBytes(
  bytes: Uint8Array,
  stream: CompressionStream | DecompressionStream
) {
  const input = new Uint8Array(bytes.byteLength);
  input.set(bytes);
  const transformed = new Blob([input.buffer]).stream().pipeThrough(stream);
  return new Uint8Array(await new Response(transformed).arrayBuffer());
}

function isPlaygroundShareState(value: unknown): value is PlaygroundShareState {
  if (!value || typeof value !== "object") {
    return false;
  }

  const state = value as Record<string, unknown>;
  return (
    typeof state.csv === "string" &&
    typeof state.sql === "string" &&
    typeof state.dialect === "string" &&
    state.dialect.length > 0
  );
}

export async function encodePlaygroundState(state: PlaygroundShareState) {
  const json = JSON.stringify(state);
  const compressed = await transformBytes(
    new TextEncoder().encode(json),
    new CompressionStream("gzip")
  );
  return bytesToBase64Url(compressed);
}

export async function decodePlaygroundState(encoded: string) {
  try {
    const compressed = base64UrlToBytes(encoded);
    const jsonBytes = await transformBytes(compressed, new DecompressionStream("gzip"));
    const value: unknown = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(jsonBytes));
    return isPlaygroundShareState(value) ? value : null;
  } catch {
    return null;
  }
}

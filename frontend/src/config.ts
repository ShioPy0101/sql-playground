const configuredSeconds = Number(import.meta.env.VITE_EVENT_REFRESH_INTERVAL_SECONDS ?? "0");

export const eventRefreshIntervalMs =
  Number.isFinite(configuredSeconds) && configuredSeconds > 0 ? configuredSeconds * 1000 : 0;

export function normalizeMonitorName(raw) {
  const name = String(raw || "").trim();
  if (!name || name === "-") return "";
  return name;
}

export function monitorAliasLabel() {
  return "";
}

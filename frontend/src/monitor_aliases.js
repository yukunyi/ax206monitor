const UPPERCASE_TOKEN_MAP = {
  cpu: "CPU",
  gpu: "GPU",
  ip: "IP",
  fps: "FPS",
  vram: "VRAM",
};

export function normalizeMonitorName(raw) {
  const name = String(raw || "").trim();
  if (!name || name === "-") return "";
  return name;
}

export function monitorAliasLabel(raw, aliasLabelMap = null) {
  const name = String(raw || "").trim();
  if (!name) return "";
  const labels = aliasLabelMap && typeof aliasLabelMap === "object" ? aliasLabelMap : null;
  if (labels && typeof labels[name] === "string" && labels[name].trim()) {
    return labels[name].trim();
  }
  if (!name.startsWith("alias.")) return "";
  const text = name
    .slice(6)
    .split(".")
    .filter(Boolean)
    .map((part) => {
      const lower = String(part || "").trim().toLowerCase();
      if (!lower) return "";
      if (UPPERCASE_TOKEN_MAP[lower]) return UPPERCASE_TOKEN_MAP[lower];
      return lower.charAt(0).toUpperCase() + lower.slice(1);
    })
    .filter(Boolean)
    .join(" ");
  return text || "Alias";
}

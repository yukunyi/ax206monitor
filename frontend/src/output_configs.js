export const OUTPUT_TYPE_MEMIMG = "memimg";
export const OUTPUT_TYPE_AX206USB = "ax206usb";
export const OUTPUT_TYPE_HTTPPUSH = "httppush";

export const CONFIGURABLE_OUTPUT_TYPES = [OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH];
export const DEFAULT_OUTPUT_TYPES = [...CONFIGURABLE_OUTPUT_TYPES];
const DEFAULT_OUTPUTS = [];
const OUTPUT_SINGLETON_TYPES = new Set([OUTPUT_TYPE_MEMIMG, OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH]);
const OUTPUT_ALLOWED_TYPES = new Set([OUTPUT_TYPE_MEMIMG, OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH]);

export const OUTPUT_FORMAT_OPTIONS = [
  { label: "jpeg", value: "jpeg" },
  { label: "png", value: "png" },
];

export function normalizeOutputType(value) {
  return String(value || "").trim().toLowerCase();
}

export function isHttpPushType(type) {
  return normalizeOutputType(type) === OUTPUT_TYPE_HTTPPUSH;
}

export function isAX206Type(type) {
  return normalizeOutputType(type) === OUTPUT_TYPE_AX206USB;
}

export function normalizeAX206ReconnectMS(value) {
  const reconnectMS = Number(value || 3000);
  if (!Number.isFinite(reconnectMS)) return 3000;
  return Math.max(100, Math.min(60000, Math.round(reconnectMS)));
}

export function normalizeOutputEntry(raw) {
  const item = raw && typeof raw === "object" ? raw : { type: raw };
  const type = normalizeOutputType(item.type);
  if (!type || !OUTPUT_ALLOWED_TYPES.has(type)) return null;
  const entry = { type };
  if (type === OUTPUT_TYPE_AX206USB) {
    entry.reconnect_ms = normalizeAX206ReconnectMS(item.reconnect_ms);
  }
  if (type === OUTPUT_TYPE_HTTPPUSH) {
    entry.url = String(item.url || "").trim();
    let format = String(item.format || "jpeg").trim().toLowerCase();
    if (format === "jpg") format = "jpeg";
    if (format !== "png") format = "jpeg";
    entry.format = format;
    const qualityRaw = Number(item.quality || 80);
    entry.quality = Math.max(1, Math.min(100, Number.isFinite(qualityRaw) ? Math.round(qualityRaw) : 80));
  }
  return entry;
}

export function normalizeOutputs(rawOutputs, rawTypes) {
  const source = Array.isArray(rawOutputs)
    ? rawOutputs
    : Array.isArray(rawTypes)
      ? rawTypes.map((type) => ({ type }))
      : DEFAULT_OUTPUTS;
  const list = [];
  const singleton = new Set();
  source.forEach((raw) => {
    const entry = normalizeOutputEntry(raw);
    if (!entry) return;
    if (entry.type === OUTPUT_TYPE_MEMIMG) return;
    if (OUTPUT_SINGLETON_TYPES.has(entry.type)) {
      if (singleton.has(entry.type)) return;
      singleton.add(entry.type);
    }
    list.push(entry);
  });
  return list;
}

export function buildOutputTypeOptions(metaTypes, outputs) {
  const set = new Set(CONFIGURABLE_OUTPUT_TYPES);
  (Array.isArray(metaTypes) ? metaTypes : DEFAULT_OUTPUT_TYPES).forEach((item) => {
    const type = normalizeOutputType(item);
    if (!type || type === OUTPUT_TYPE_MEMIMG) return;
    set.add(type);
  });
  (Array.isArray(outputs) ? outputs : []).forEach((item) => {
    const type = normalizeOutputType(item?.type);
    if (!type || type === OUTPUT_TYPE_MEMIMG) return;
    set.add(type);
  });
  return CONFIGURABLE_OUTPUT_TYPES
    .filter((item) => set.has(item))
    .map((item) => ({ label: item, value: item }));
}

export function createDefaultOutputEntry(type = OUTPUT_TYPE_AX206USB) {
  const normalized = normalizeOutputType(type) || OUTPUT_TYPE_AX206USB;
  if (normalized === OUTPUT_TYPE_HTTPPUSH) {
    return { type: OUTPUT_TYPE_HTTPPUSH, url: "", format: "jpeg", quality: 80 };
  }
  if (normalized === OUTPUT_TYPE_AX206USB) {
    return { type: OUTPUT_TYPE_AX206USB, reconnect_ms: 3000 };
  }
  return { type: OUTPUT_TYPE_AX206USB, reconnect_ms: 3000 };
}

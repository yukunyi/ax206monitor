export const OUTPUT_TYPE_MEMIMG = "memimg";
export const OUTPUT_TYPE_AX206USB = "ax206usb";
export const OUTPUT_TYPE_HTTPPUSH = "httppush";
export const OUTPUT_TYPE_TCPPUSH = "tcppush";

export const CONFIGURABLE_OUTPUT_TYPES = [OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH, OUTPUT_TYPE_TCPPUSH];
export const DEFAULT_OUTPUT_TYPES = [...CONFIGURABLE_OUTPUT_TYPES];
const DEFAULT_OUTPUTS = [];
const OUTPUT_SINGLETON_TYPES = new Set([OUTPUT_TYPE_MEMIMG, OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH, OUTPUT_TYPE_TCPPUSH]);
const OUTPUT_ALLOWED_TYPES = new Set([OUTPUT_TYPE_MEMIMG, OUTPUT_TYPE_AX206USB, OUTPUT_TYPE_HTTPPUSH, OUTPUT_TYPE_TCPPUSH]);

export const OUTPUT_FORMAT_OPTIONS = [
  { label: "jpeg", value: "jpeg" },
  { label: "jpeg baseline", value: "jpeg_baseline" },
  { label: "png", value: "png" },
];

export const OUTPUT_TCP_FORMAT_OPTIONS = [
  { label: "jpeg", value: "jpeg" },
  { label: "rgb565le", value: "rgb565le" },
  { label: "rgb565le_rle", value: "rgb565le_rle" },
  { label: "index8_rle", value: "index8_rle" },
];

export const OUTPUT_HTTP_METHOD_OPTIONS = [
  { label: "POST", value: "POST" },
  { label: "PUT", value: "PUT" },
  { label: "PATCH", value: "PATCH" },
];

export const OUTPUT_HTTP_BODY_MODE_OPTIONS = [
  { label: "binary", value: "binary" },
  { label: "multipart", value: "multipart" },
];

export const OUTPUT_HTTP_AUTH_OPTIONS = [
  { label: "none", value: "none" },
  { label: "basic", value: "basic" },
  { label: "bearer", value: "bearer" },
];

export function normalizeOutputType(value) {
  return String(value || "").trim().toLowerCase();
}

export function isHttpPushType(type) {
  return normalizeOutputType(type) === OUTPUT_TYPE_HTTPPUSH;
}

export function isTcpPushType(type) {
  return normalizeOutputType(type) === OUTPUT_TYPE_TCPPUSH;
}

export function isAX206Type(type) {
  return normalizeOutputType(type) === OUTPUT_TYPE_AX206USB;
}

export function normalizeAX206ReconnectMS(value) {
  const reconnectMS = Number(value || 3000);
  if (!Number.isFinite(reconnectMS)) return 3000;
  return Math.max(100, Math.min(60000, Math.round(reconnectMS)));
}

export function normalizeHTTPMethod(value) {
  const method = String(value || "POST").trim().toUpperCase();
  return method || "POST";
}

export function normalizeHTTPBodyMode(value) {
  const mode = String(value || "binary").trim().toLowerCase();
  if (mode === "form" || mode === "formdata" || mode === "multipart_form") return "multipart";
  return mode === "multipart" ? "multipart" : "binary";
}

export function normalizeHTTPAuthType(value) {
  const authType = String(value || "none").trim().toLowerCase();
  if (authType === "basic" || authType === "bearer") return authType;
  return "none";
}

export function normalizeHTTPTimeoutMS(value) {
  const timeoutMS = Number(value || 5000);
  if (!Number.isFinite(timeoutMS)) return 5000;
  return Math.max(100, Math.min(600000, Math.round(timeoutMS)));
}

export function normalizeTCPIdleTimeoutSec(value) {
  const idleTimeoutSec = Number(value || 120);
  if (!Number.isFinite(idleTimeoutSec)) return 120;
  return Math.max(5, Math.min(3600, Math.round(idleTimeoutSec)));
}

export function normalizeHTTPKeyValueList(raw) {
  if (!Array.isArray(raw)) return [];
  return raw
    .map((item) => ({
      key: String(item?.key || "").trim(),
      value: String(item?.value || "").trim(),
    }))
    .filter((item) => item.key);
}

export function normalizeHTTPSuccessCodes(raw) {
  if (!Array.isArray(raw)) return [];
  const seen = new Set();
  return raw
    .map((value) => Number(value))
    .filter((value) => Number.isFinite(value))
    .map((value) => Math.round(value))
    .filter((value) => value >= 100 && value <= 599)
    .sort((left, right) => left - right)
    .filter((value) => {
      if (seen.has(value)) return false;
      seen.add(value);
      return true;
    });
}

export function normalizeOutputEntry(raw) {
  const item = raw && typeof raw === "object" ? raw : { type: raw };
  const type = normalizeOutputType(item.type);
  if (!type || !OUTPUT_ALLOWED_TYPES.has(type)) return null;
  const entry = { type, enabled: item.enabled !== false };
  if (type === OUTPUT_TYPE_AX206USB) {
    entry.reconnect_ms = normalizeAX206ReconnectMS(item.reconnect_ms);
  }
  if (type === OUTPUT_TYPE_HTTPPUSH) {
    entry.url = String(item.url || "").trim();
    entry.method = normalizeHTTPMethod(item.method);
    entry.body_mode = normalizeHTTPBodyMode(item.body_mode);
    let format = String(item.format || "jpeg").trim().toLowerCase();
    if (format === "jpg") format = "jpeg";
    if (format === "jpg_baseline" || format === "baseline_jpg" || format === "baseline_jpeg") format = "jpeg_baseline";
    if (format !== "png" && format !== "jpeg_baseline") format = "jpeg";
    entry.format = format;
    const qualityRaw = Number(item.quality || 80);
    entry.quality = Math.max(1, Math.min(100, Number.isFinite(qualityRaw) ? Math.round(qualityRaw) : 80));
    entry.content_type = String(item.content_type || "").trim();
    entry.headers = normalizeHTTPKeyValueList(item.headers);
    entry.auth_type = normalizeHTTPAuthType(item.auth_type);
    entry.auth_username = String(item.auth_username || "").trim();
    entry.auth_password = String(item.auth_password || "").trim();
    entry.auth_token = String(item.auth_token || "").trim();
    entry.timeout_ms = normalizeHTTPTimeoutMS(item.timeout_ms);
    entry.file_field = String(item.file_field || "file").trim() || "file";
    entry.file_name = String(item.file_name || "").trim();
    entry.form_fields = normalizeHTTPKeyValueList(item.form_fields);
    entry.success_codes = normalizeHTTPSuccessCodes(item.success_codes);
  }
  if (type === OUTPUT_TYPE_TCPPUSH) {
    entry.url = String(item.url || "").trim();
    let format = String(item.format || "jpeg").trim().toLowerCase();
    if (format === "jpg" || format === "jpg_baseline" || format === "baseline_jpg" || format === "baseline_jpeg" || format === "jpeg_baseline") format = "jpeg";
    if (format === "rgb565") format = "rgb565le";
    if (format === "rgb565_rle") format = "rgb565le_rle";
    if (format === "index8" || format === "palette8_rle") format = "index8_rle";
    if (format !== "jpeg" && format !== "rgb565le" && format !== "rgb565le_rle" && format !== "index8_rle") format = "jpeg";
    entry.format = format;
    const qualityRaw = Number(item.quality || 80);
    entry.quality = Math.max(1, Math.min(100, Number.isFinite(qualityRaw) ? Math.round(qualityRaw) : 80));
    entry.upload_token = String(item.upload_token || "").trim();
    entry.timeout_ms = normalizeHTTPTimeoutMS(item.timeout_ms);
    entry.idle_timeout_sec = normalizeTCPIdleTimeoutSec(item.idle_timeout_sec);
    entry.file_name = String(item.file_name || "").trim();
    entry.success_codes = normalizeHTTPSuccessCodes(item.success_codes);
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
    return {
      type: OUTPUT_TYPE_HTTPPUSH,
      enabled: true,
      url: "",
      method: "POST",
      body_mode: "binary",
      format: "jpeg",
      quality: 80,
      content_type: "",
      headers: [],
      auth_type: "none",
      auth_username: "",
      auth_password: "",
      auth_token: "",
      timeout_ms: 5000,
      file_field: "file",
      file_name: "",
      form_fields: [],
      success_codes: [],
    };
  }
  if (normalized === OUTPUT_TYPE_TCPPUSH) {
    return {
      type: OUTPUT_TYPE_TCPPUSH,
      enabled: true,
      url: "tcp://127.0.0.1:9100",
      format: "jpeg",
      quality: 80,
      upload_token: "",
      timeout_ms: 5000,
      idle_timeout_sec: 120,
      file_name: "",
      success_codes: [],
    };
  }
  if (normalized === OUTPUT_TYPE_AX206USB) {
    return { type: OUTPUT_TYPE_AX206USB, enabled: true, reconnect_ms: 3000 };
  }
  return { type: OUTPUT_TYPE_AX206USB, enabled: true, reconnect_ms: 3000 };
}

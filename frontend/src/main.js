import "./style.css";
import Pickr from "@simonwep/pickr";
import "@simonwep/pickr/dist/themes/nano.min.css";

const app = document.querySelector("#app");

const SIMPLE_ELEMENT_TYPES = [
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  "simple_label",
  "simple_rect",
  "simple_circle",
];
const FULL_ELEMENT_TYPES = [
  "full_chart",
  "full_progress",
  "full_gauge",
  "full_ring",
  "full_minmax",
  "full_delta",
  "full_status",
  "full_meter_h",
  "full_meter_v",
  "full_heat_strip",
];
const ELEMENT_TYPES = [...SIMPLE_ELEMENT_TYPES, ...FULL_ELEMENT_TYPES];
const LEGACY_TYPE_MAP = {
  value: "simple_value",
  progress: "simple_progress",
  line_chart: "simple_line_chart",
  label: "simple_label",
  rect: "simple_rect",
  circle: "simple_circle",
};
const ITEM_TYPE_DISPLAY_NAMES = {
  simple_value: "简单数值",
  simple_progress: "简单进度条",
  simple_line_chart: "简单折线图",
  simple_label: "简单文本",
  simple_rect: "简单矩形",
  simple_circle: "简单圆形",
  full_chart: "复杂图表",
  full_progress: "复杂进度条",
  full_gauge: "复杂仪表盘",
  full_ring: "复杂环形图",
  full_minmax: "复杂最值图",
  full_delta: "复杂变化趋势",
  full_status: "复杂状态卡",
  full_meter_h: "复杂水平刻度",
  full_meter_v: "复杂垂直刻度",
  full_heat_strip: "复杂热度条",
};
const ITEM_FIELD_DISPLAY_NAMES = {
  edit_ui_name: "名称",
  type: "类型",
  monitor: "采集项",
  text: "文本",
  x: "X",
  y: "Y",
  width: "宽度",
  height: "高度",
  font_size: "字号",
  unit: "单位",
  unit_font_size: "单位字号",
  color: "主颜色",
  unit_color: "单位颜色",
  bg: "背景色",
  border_color: "边框色",
  border_width: "边框宽度",
  radius: "圆角",
  min_value: "最小值",
  max_value: "最大值",
  max: "最大刻度",
  point_size: "历史点数",
};
const AUTO_APPLY_DELAY_OPTIONS = [100, 300, 500, 1000];
const MONITOR_ELEMENT_TYPES = new Set([
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  ...FULL_ELEMENT_TYPES,
]);
const RANGE_ELEMENT_TYPES = new Set([
  "simple_progress",
  "simple_line_chart",
  ...FULL_ELEMENT_TYPES,
]);
const LABEL_ELEMENT_TYPES = new Set(["simple_label"]);
const state = {
  config: null,
  meta: null,
  profiles: [],
  editingProfile: "default",
  snapshot: null,
  collectors: [],
  collectorsLoading: false,
  monitorOptions: [],
  monitorLabels: {},
  selectedItem: -1,
  error: "",
  dirty: false,
  drag: null,
  previewScale: 1,
  previewImageBitmap: null,
  previewRefreshPending: false,
  previewError: "",
  previewSyncing: false,
  previewSyncPending: false,
  saving: false,
  savePending: false,
  savePromise: null,
  polling: false,
  pollTimer: null,
  runtimeStream: null,
  runtimeStreamReady: false,
  runtimeStreamReqSeq: 1,
  runtimeStreamPending: {},
  runtimeStreamReconnectTimer: null,
  activeTab: "basic",
  addItemType: "simple_value",
  addItemMonitor: "",
  ui: {
    purePreview: false,
    showGrid: true,
    snapEnabled: true,
    snapSize: 10,
    autoApply: false,
    autoApplyDelay: 300,
    previewZoomAuto: true,
    previewZoom: 100,
    previewFitScale: 1,
    elementListScrollTop: 0,
    centerGuides: null,
    colorPickers: [],
  },
  autoApplyTimer: null,
  previewFetchTimer: null,
  previewResizeObserver: null,
  previewWrapperSize: { width: 0, height: 0 },
};

const DEFAULT_COLLECTOR_ENABLED = {
  "go_native.cpu": true,
  "go_native.memory": true,
  "go_native.system": true,
  "go_native.disk": true,
  "go_native.network": true,
  "custom.all": true,
  "external.coolercontrol": false,
  "external.librehardwaremonitor": false,
  "external.rtss": false,
};

const COLLECTOR_NAME_ALIAS = {
  "go_native.cpu": "CPU 原生采集器",
  "go_native.memory": "Memory 原生采集器",
  "go_native.system": "System 原生采集器",
  "go_native.disk": "Disk 原生采集器",
  "go_native.network": "Network 原生采集器",
  "custom.all": "自定义采集器",
  "external.coolercontrol": "CoolerControl 采集器",
  "external.librehardwaremonitor": "LibreHardwareMonitor 采集器",
  "external.rtss": "RTSS 采集器",
};

function deepClone(obj) {
  return JSON.parse(JSON.stringify(obj));
}

function num(value, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function maybeNumber(value) {
  if (value === "" || value === null || value === undefined) return null;
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return null;
  return parsed;
}

function ensureFourNumbers(values, fallback) {
  const out = [];
  for (const value of Array.isArray(values) ? values : []) {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      out.push(parsed);
    }
    if (out.length >= 4) break;
  }
  const fallbackList = Array.isArray(fallback) ? fallback : [25, 50, 75, 100];
  while (out.length < 4) {
    out.push(fallbackList[Math.min(out.length, fallbackList.length - 1)] ?? 100);
  }
  return out.slice(0, 4);
}

function ensureFourColors(values, fallback) {
  const out = [];
  for (const value of Array.isArray(values) ? values : []) {
    const text = String(value || "").trim();
    if (text) out.push(text);
    if (out.length >= 4) break;
  }
  const fallbackList = Array.isArray(fallback)
    ? fallback
    : ["#22c55e", "#eab308", "#f97316", "#ef4444"];
  while (out.length < 4) {
    out.push(fallbackList[Math.min(out.length, fallbackList.length - 1)] || "#ef4444");
  }
  return out.slice(0, 4);
}

const FULL_TYPE_ATTR_SCHEMAS = {
  full_chart: [
    { key: "history_points", label: "历史点数", kind: "number" },
    { key: "show_avg_line", label: "平均线", kind: "bool" },
    { key: "fill_area", label: "填充区域", kind: "bool" },
    { key: "line_width", label: "线条宽度", kind: "number" },
  ],
  full_progress: [
    {
      key: "progress_style",
      label: "进度条样式",
      kind: "select",
      options: ["gradient", "solid", "segmented", "stripes", "glow"],
    },
    { key: "segments", label: "分段数量", kind: "number" },
    { key: "track_color", label: "轨道颜色", kind: "color" },
    { key: "bar_radius", label: "条形圆角", kind: "number" },
  ],
  full_gauge: [
    { key: "gauge_thickness", label: "仪表厚度", kind: "number" },
  ],
  full_ring: [
    { key: "ring_thickness", label: "环形厚度", kind: "number" },
  ],
  full_minmax: [
    { key: "history_points", label: "历史点数", kind: "number" },
  ],
  full_delta: [
    { key: "history_points", label: "历史点数", kind: "number" },
    { key: "main_font_size", label: "主值字号", kind: "number" },
  ],
  full_status: [
  ],
  full_meter_h: [
    { key: "ticks", label: "刻度数量", kind: "number" },
  ],
  full_meter_v: [
    { key: "segments", label: "分段数量", kind: "number" },
  ],
  full_heat_strip: [
    { key: "cells", label: "格子数量", kind: "number" },
    { key: "cell_gap", label: "格子间距", kind: "number" },
  ],
};
const FULL_COMMON_ATTR_SCHEMAS = [
  { key: "header_divider", label: "显示分割线", kind: "bool" },
  { key: "header_divider_offset", label: "分割线高度", kind: "number" },
  { key: "header_divider_color", label: "分割线颜色", kind: "color" },
];

const DEFAULT_FULL_RENDER_ATTRS = {
  full_chart: { history_points: 90, show_avg_line: true, fill_area: true, line_width: 2 },
  full_progress: { progress_style: "gradient", segments: 12, track_color: "#1f2937", bar_radius: 8 },
  full_gauge: { gauge_thickness: 9 },
  full_ring: { ring_thickness: 8 },
  full_minmax: { history_points: 120 },
  full_delta: { history_points: 60, main_font_size: 0 },
  full_status: {},
  full_meter_h: { ticks: 8 },
  full_meter_v: { segments: 12 },
  full_heat_strip: { cells: 12, cell_gap: 2 },
};
const FULL_COMMON_RENDER_ATTRS = {
  content_padding: 1,
  title_font_size: 0,
  header_divider: true,
  header_divider_offset: 3,
  header_divider_color: "#94a3b840",
};
const FULL_ATTR_OPTION_DISPLAY_NAMES = {
  gradient: "渐变",
  solid: "纯色",
  segmented: "分段",
  stripes: "斜纹",
  glow: "发光",
};

function isFullElementType(type) {
  return String(type || "").startsWith("full_");
}

function normalizeRenderAttrsMap(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return {};
  }
  return { ...raw };
}

function ensureItemRenderAttrs(item) {
  if (!item) return {};
  if (!item.render_attrs_map || typeof item.render_attrs_map !== "object" || Array.isArray(item.render_attrs_map)) {
    item.render_attrs_map = {};
  }
  return item.render_attrs_map;
}

function defaultRenderAttrsForType(type) {
  const normalized = normalizeItemType(type);
  const defaults = DEFAULT_FULL_RENDER_ATTRS[normalized];
  return defaults ? { ...deepClone(FULL_COMMON_RENDER_ATTRS), ...deepClone(defaults) } : deepClone(FULL_COMMON_RENDER_ATTRS);
}

function normalizeOutputTypes(values) {
  const allowed = new Set(state.meta?.output_types || ["memimg", "ax206usb"]);
  const out = [];
  for (const value of Array.isArray(values) ? values : []) {
    const text = String(value || "").trim();
    if (!allowed.has(text) || out.includes(text)) continue;
    out.push(text);
  }
  if (out.length === 0) {
    out.push("memimg");
  }
  return out;
}

function normalizeItemType(type) {
  const raw = String(type || "").trim();
  if (!raw) return "simple_value";
  const lowered = raw.toLowerCase();
  const mapped = LEGACY_TYPE_MAP[lowered] || lowered;
  return ELEMENT_TYPES.includes(mapped) ? mapped : "simple_value";
}

function itemTypeDisplayName(type) {
  const normalized = normalizeItemType(type);
  return ITEM_TYPE_DISPLAY_NAMES[normalized] || normalized;
}

function itemFieldDisplayName(field) {
  return ITEM_FIELD_DISPLAY_NAMES[field] || field;
}

function fullAttrOptionDisplayName(value) {
  const key = String(value ?? "").trim();
  return FULL_ATTR_OPTION_DISPLAY_NAMES[key] || key;
}

function defaultEditUIName(index, item) {
  const monitor = String(item?.monitor || "").trim();
  const type = normalizeItemType(item?.type);
  const base = monitor || type || "item";
  return `${index + 1}_${base}`;
}

function ensureItemName(item, index) {
  if (!item) return;
  const current = String(item.edit_ui_name || "").trim();
  if (!current) {
    item.edit_ui_name = defaultEditUIName(index, item);
  }
}

function normalizeItem(item, index = 0, defaultPointSize = 150) {
  const itemType = normalizeItemType(item?.type);
  const renderAttrs = normalizeRenderAttrsMap(item?.render_attrs_map || item?.renderAttrsMap);
  const result = {
    type: itemType,
    edit_ui_name: String(item?.edit_ui_name || "").trim(),
    monitor: String(item?.monitor || ""),
    unit: String(item?.unit || "auto"),
    unit_color: String(item?.unit_color || ""),
    x: num(item?.x, 10),
    y: num(item?.y, 10),
    width: Math.max(10, num(item?.width, 120)),
    height: Math.max(10, num(item?.height, 40)),
    text: String(item?.text || (itemType === "simple_label" ? "Label" : "")),
    color: String(item?.color || ""),
    bg: String(item?.bg || ""),
    border_color: String(item?.border_color || ""),
    border_width: Math.max(0, num(item?.border_width, 0)),
    radius: Math.max(0, num(item?.radius, isFullElementType(itemType) ? 2 : 0)),
    history: !!item?.history,
    point_size: Math.max(10, num(item?.point_size, defaultPointSize)),
    render_attrs_map: renderAttrs,
  };

  if (item?.font_size !== undefined && item?.font_size !== null && item?.font_size !== "") {
    result.font_size = Math.max(0, num(item.font_size, 0));
  }
  if (item?.unit_font_size !== undefined && item?.unit_font_size !== null && item?.unit_font_size !== "") {
    result.unit_font_size = Math.max(0, num(item.unit_font_size, 0));
  }
  if (RANGE_ELEMENT_TYPES.has(result.type)) {
    if (item?.max !== undefined && item?.max !== null && item?.max !== "") {
      result.max = num(item.max, 100);
    }
    if (item?.max_value !== undefined && item?.max_value !== null && item?.max_value !== "") {
      result.max_value = num(item.max_value, 100);
    }
    if (item?.min_value !== undefined && item?.min_value !== null && item?.min_value !== "") {
      result.min_value = num(item.min_value, 0);
    }
  }
  if (Array.isArray(item?.thresholds) && item.thresholds.length > 0) {
    result.thresholds = ensureFourNumbers(item.thresholds, [25, 50, 75, 100]);
  }
  if (Array.isArray(item?.level_colors) && item.level_colors.length > 0) {
    result.level_colors = ensureFourColors(item.level_colors, ["#22c55e", "#eab308", "#f97316", "#ef4444"]);
  }
  if (!MONITOR_ELEMENT_TYPES.has(result.type)) {
    result.monitor = "";
    result.unit = "";
    result.unit_color = "";
    delete result.unit_font_size;
    delete result.max;
    delete result.max_value;
    delete result.min_value;
  } else if (!result.unit) {
    result.unit = "auto";
  }
  if (!RANGE_ELEMENT_TYPES.has(result.type)) {
    delete result.max;
    delete result.max_value;
    delete result.min_value;
  }
  if (result.type === "simple_line_chart") {
    result.history = true;
    result.point_size = Math.max(10, num(result.point_size, defaultPointSize));
  } else {
    result.history = false;
    delete result.point_size;
  }
  if (isFullElementType(result.type)) {
    result.render_attrs_map = {
      ...defaultRenderAttrsForType(result.type),
      ...result.render_attrs_map,
    };
  }
  if (!result.edit_ui_name) {
    result.edit_ui_name = defaultEditUIName(index, result);
  }

  return result;
}

function normalizeCustomMonitor(item) {
  const monitor = item || {};
  const type = ["file", "mixed", "coolercontrol", "librehardwaremonitor"].includes(String(monitor.type || ""))
    ? String(monitor.type)
    : "file";

  return {
    name: String(monitor.name || ""),
    label: String(monitor.label || ""),
    type,
    unit: String(monitor.unit || ""),
    precision: maybeNumber(monitor.precision),
    min: maybeNumber(monitor.min),
    max: maybeNumber(monitor.max),
    path: String(monitor.path || ""),
    scale: maybeNumber(monitor.scale),
    offset: maybeNumber(monitor.offset) ?? 0,
    sources: Array.isArray(monitor.sources) ? monitor.sources.map((x) => String(x)).filter(Boolean) : [],
    aggregate: ["max", "min", "avg"].includes(String(monitor.aggregate || "")) ? String(monitor.aggregate) : "max",
    source: String(monitor.source || ""),
  };
}

function collectorDefaultEnabled(name) {
  if (Object.prototype.hasOwnProperty.call(DEFAULT_COLLECTOR_ENABLED, name)) {
    return !!DEFAULT_COLLECTOR_ENABLED[name];
  }
  return true;
}

function ensureCollectorConfigEntry(config, collectorName) {
  if (!config.collector_config || typeof config.collector_config !== "object") {
    config.collector_config = {};
  }
  const name = String(collectorName || "").trim();
  if (!name) {
    return { enabled: true, options: {} };
  }
  const prev = config.collector_config[name] || {};
  const enabled = typeof prev.enabled === "boolean" ? prev.enabled : collectorDefaultEnabled(name);
  const options = prev.options && typeof prev.options === "object" ? { ...prev.options } : {};
  const next = { enabled, options };
  config.collector_config[name] = next;
  return next;
}

function knownCollectorNames(config) {
  const seen = new Set();
  const names = [];
  const push = (value) => {
    const name = String(value || "").trim();
    if (!name || seen.has(name)) return;
    seen.add(name);
    names.push(name);
  };

  Object.keys(DEFAULT_COLLECTOR_ENABLED).forEach(push);
  (state.meta?.collectors || []).forEach(push);
  (state.collectors || []).forEach((item) => push(item?.name));
  if (config?.collector_config && typeof config.collector_config === "object") {
    Object.keys(config.collector_config).forEach(push);
  }

  const defaultOrder = Object.keys(DEFAULT_COLLECTOR_ENABLED);
  const orderScore = new Map(defaultOrder.map((name, index) => [name, index]));
  names.sort((a, b) => {
    const sa = orderScore.has(a) ? orderScore.get(a) : 9999;
    const sb = orderScore.has(b) ? orderScore.get(b) : 9999;
    if (sa !== sb) return sa - sb;
    return a.localeCompare(b);
  });
  return names;
}

function collectorOptionValue(config, collectorName, optionKey, fallback = "") {
  const entry = ensureCollectorConfigEntry(config, collectorName);
  const value = entry.options?.[optionKey];
  if (value === null || value === undefined) {
    return String(fallback ?? "");
  }
  return String(value);
}

function collectorEnabledValue(config, collectorName) {
  return !!ensureCollectorConfigEntry(config, collectorName).enabled;
}

function syncLegacyCollectorFields(config) {
  const cc = ensureCollectorConfigEntry(config, "external.coolercontrol");
  const lhm = ensureCollectorConfigEntry(config, "external.librehardwaremonitor");
  const rtss = ensureCollectorConfigEntry(config, "external.rtss");
  config.coolercontrol_url = String(cc.options.url || "");
  config.coolercontrol_username = String(cc.options.username || "");
  config.coolercontrol_password = String(cc.options.password || "");
  config.libre_hardware_monitor_url = String(lhm.options.url || "");
  config.enable_rtss_collect = !!rtss.enabled;
}

function ensureConfig(cfg) {
  const config = cfg || {};
  config.name = String(config.name || "web");
  config.width = Math.max(10, num(config.width, 480));
  config.height = Math.max(10, num(config.height, 320));
  config.layout_padding = normalizeLayoutPadding(config.layout_padding, config.width, config.height);
  config.default_font_size = Math.max(1, num(config.default_font_size, 16));
  config.default_color = String(config.default_color || "#f8fafc");
  config.default_background = String(config.default_background || "#0b1220");

  config.font_families = Array.isArray(config.font_families) && config.font_families.length > 0
    ? config.font_families.map((name) => String(name)).filter(Boolean)
    : ["DejaVu Sans Mono", "Liberation Mono", "monospace"];
  config.default_font = String(config.default_font || config.font_families[0] || "DejaVu Sans Mono");

  config.level_colors = ensureFourColors(config.level_colors, ["#22c55e", "#eab308", "#f97316", "#ef4444"]);
  config.default_thresholds = ensureFourNumbers(config.default_thresholds, [25, 50, 75, 100]);

  config.output_types = normalizeOutputTypes(config.output_types);
  config.refresh_interval = Math.max(100, num(config.refresh_interval, 1000));
  config.history_size = Math.max(10, num(config.history_size, 150));
  config.network_interface = String(config.network_interface || "").trim();
  if (config.network_interface.toLowerCase() === "auto") {
    config.network_interface = "";
  }
  config.enable_rtss_collect = !!config.enable_rtss_collect;
  config.libre_hardware_monitor_url = String(config.libre_hardware_monitor_url ?? "");
  config.coolercontrol_url = String(config.coolercontrol_url ?? "");
  config.coolercontrol_username = String(config.coolercontrol_username || "");
  config.coolercontrol_password = String(config.coolercontrol_password || "");

  const allCollectors = knownCollectorNames(config);
  allCollectors.forEach((name) => {
    ensureCollectorConfigEntry(config, name);
  });

  const cc = ensureCollectorConfigEntry(config, "external.coolercontrol");
  if (!cc.options.url && config.coolercontrol_url) cc.options.url = config.coolercontrol_url;
  if (!cc.options.username && config.coolercontrol_username) cc.options.username = config.coolercontrol_username;
  if (!cc.options.password && config.coolercontrol_password) cc.options.password = config.coolercontrol_password;

  const lhm = ensureCollectorConfigEntry(config, "external.librehardwaremonitor");
  if (!lhm.options.url && config.libre_hardware_monitor_url) lhm.options.url = config.libre_hardware_monitor_url;

  const rtss = ensureCollectorConfigEntry(config, "external.rtss");
  if (config.enable_rtss_collect) {
    rtss.enabled = true;
  }

  syncLegacyCollectorFields(config);

  config.custom_monitors = Array.isArray(config.custom_monitors)
    ? config.custom_monitors.map((item) => normalizeCustomMonitor(item))
    : [];
  config.items = Array.isArray(config.items)
    ? config.items.map((item, index) => normalizeItem(item, index, config.history_size))
    : [];

  return config;
}

function createItemByType(type) {
  const itemType = normalizeItemType(type);
  const monitor = state.addItemMonitor || state.monitorOptions[0] || "";
  const defaultPointSize = Math.max(10, num(state.config?.history_size, 150));
  const padding = getLayoutPadding(state.config);
  const base = {
    type: itemType,
    edit_ui_name: "",
    monitor: MONITOR_ELEMENT_TYPES.has(itemType) ? monitor : "",
    unit: MONITOR_ELEMENT_TYPES.has(itemType) ? "auto" : "",
    unit_color: "",
    x: padding,
    y: padding,
    width: itemType === "simple_label" ? 140 : 120,
    height: itemType === "simple_label" ? 34 : 40,
    text: itemType === "simple_label" ? "Label" : "",
    color: "",
    bg: "",
    border_color: "",
    border_width: 0,
    radius: itemType === "simple_rect" ? 8 : 0,
    history: itemType === "simple_line_chart",
    point_size: itemType === "simple_line_chart" ? defaultPointSize : undefined,
    render_attrs_map: defaultRenderAttrsForType(itemType),
  };
  if (itemType === "simple_circle") {
    base.width = 60;
    base.height = 60;
  }
  if (isFullElementType(itemType)) {
    base.width = 220;
    base.height = 120;
    base.radius = 2;
  }
  return base;
}

function resolveAddItemMonitor() {
  const options = state.monitorOptions || [];
  if (options.length === 0) {
    state.addItemMonitor = "";
    return "";
  }
  if (!options.includes(state.addItemMonitor)) {
    state.addItemMonitor = options[0];
  }
  return state.addItemMonitor;
}

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
      ...(options.headers || {}),
    },
    ...options,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`${res.status} ${text || res.statusText}`);
  }
  return res.json();
}

function runtimeStreamURL() {
  const proto = window.location.protocol === "https:" ? "wss" : "ws";
  return `${proto}://${window.location.host}/api/ws`;
}

function isRuntimeStreamReady() {
  return !!state.runtimeStream && state.runtimeStreamReady && state.runtimeStream.readyState === WebSocket.OPEN;
}

function clearRuntimeStreamPending(errorMessage = "实时连接已断开") {
  const entries = Object.entries(state.runtimeStreamPending || {});
  state.runtimeStreamPending = {};
  for (const [, pending] of entries) {
    if (pending?.timer) {
      window.clearTimeout(pending.timer);
    }
    if (pending?.reject) {
      pending.reject(new Error(errorMessage));
    }
  }
}

async function applyPreviewImageBase64(base64Text) {
  const raw = String(base64Text || "").trim();
  if (!raw) return;
  const binary = window.atob(raw);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i);
  }
  const blob = new Blob([bytes], { type: "image/png" });
  const bitmap = await decodePreviewBitmap(blob);
  if (state.previewImageBitmap && typeof state.previewImageBitmap.close === "function") {
    state.previewImageBitmap.close();
  }
  state.previewImageBitmap = bitmap;
  state.previewError = "";
  updatePreviewDOM();
  const errorEl = document.getElementById("preview-render-error");
  if (errorEl) {
    errorEl.textContent = "";
  }
}

function handleRuntimeStreamMessage(message) {
  if (!message || typeof message !== "object") return;

  if (message.type === "response") {
    const id = String(message.id || "");
    if (!id || !state.runtimeStreamPending[id]) return;
    const pending = state.runtimeStreamPending[id];
    delete state.runtimeStreamPending[id];
    if (pending.timer) {
      window.clearTimeout(pending.timer);
    }
    if (message.ok) {
      pending.resolve(message.result || {});
    } else {
      pending.reject(new Error(message.error || "请求失败"));
    }
    return;
  }

  if (message.type === "runtime") {
    if (message.snapshot) {
      applySnapshotUpdate(message.snapshot, { fromWS: true });
    }
    if (message.preview_png) {
      void applyPreviewImageBase64(message.preview_png).catch(() => {});
    }
  }
}

function connectRuntimeStream() {
  if (state.runtimeStream && state.runtimeStream.readyState === WebSocket.OPEN) {
    return;
  }
  if (state.runtimeStream && state.runtimeStream.readyState === WebSocket.CONNECTING) {
    return;
  }

  if (state.runtimeStreamReconnectTimer) {
    window.clearTimeout(state.runtimeStreamReconnectTimer);
    state.runtimeStreamReconnectTimer = null;
  }

  try {
    const ws = new WebSocket(runtimeStreamURL());
    state.runtimeStream = ws;
    state.runtimeStreamReady = false;

    ws.onopen = () => {
      state.runtimeStreamReady = true;
      setError("");
      stopPolling();
      void runtimeStreamCall("request_runtime", {}, 4000).catch(() => {});
    };

    ws.onmessage = (event) => {
      if (!event?.data) return;
      try {
        const message = JSON.parse(event.data);
        handleRuntimeStreamMessage(message);
      } catch (_) {
        // ignore malformed message
      }
    };

    ws.onerror = () => {
      // will be handled by onclose
    };

    ws.onclose = () => {
      state.runtimeStreamReady = false;
      if (state.runtimeStream === ws) {
        state.runtimeStream = null;
      }
      clearRuntimeStreamPending("实时连接已关闭");
      startPolling();
      state.runtimeStreamReconnectTimer = window.setTimeout(() => {
        state.runtimeStreamReconnectTimer = null;
        connectRuntimeStream();
      }, 1000);
    };
  } catch (_) {
    startPolling();
  }
}

function closeRuntimeStream() {
  if (state.runtimeStreamReconnectTimer) {
    window.clearTimeout(state.runtimeStreamReconnectTimer);
    state.runtimeStreamReconnectTimer = null;
  }
  const ws = state.runtimeStream;
  state.runtimeStream = null;
  state.runtimeStreamReady = false;
  clearRuntimeStreamPending("实时连接已关闭");
  if (ws) {
    try {
      ws.close();
    } catch (_) {
      // ignore
    }
  }
}

function runtimeStreamCall(type, payload = {}, timeoutMs = 6000) {
  if (!isRuntimeStreamReady()) {
    return Promise.reject(new Error("实时连接未就绪"));
  }
  const id = `${Date.now()}_${state.runtimeStreamReqSeq++}`;
  const request = { type, id, ...payload };
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(() => {
      if (state.runtimeStreamPending[id]) {
        delete state.runtimeStreamPending[id];
      }
      reject(new Error("实时请求超时"));
    }, timeoutMs);
    state.runtimeStreamPending[id] = { resolve, reject, timer };
    try {
      state.runtimeStream.send(JSON.stringify(request));
    } catch (error) {
      window.clearTimeout(timer);
      delete state.runtimeStreamPending[id];
      reject(error instanceof Error ? error : new Error("实时发送失败"));
    }
  });
}

function applyProfilesPayload(payload) {
  state.profiles = payload?.items || [];
  state.meta ??= {};
  if (payload?.active) {
    state.meta.active_profile = payload.active;
  } else if (!state.meta.active_profile && state.profiles.length > 0) {
    state.meta.active_profile = state.profiles[0].name;
  }
}

function selectedProfileName() {
  const select = document.getElementById("profile-select");
  return select?.value || state.editingProfile || state.meta?.active_profile || "default";
}

function getProfileInfo(name) {
  const profileName = String(name || "").trim();
  return (state.profiles || []).find((item) => item.name === profileName) || null;
}

function isEditingProfileReadOnly() {
  const info = getProfileInfo(state.editingProfile);
  return !!info?.readonly;
}

function refreshSelectedItem() {
  if (!state.config || state.config.items.length === 0) {
    state.selectedItem = -1;
    return;
  }
  if (state.selectedItem < 0 || state.selectedItem >= state.config.items.length) {
    state.selectedItem = 0;
  }
}

function getAllMonitorOptions() {
  const monitorSet = new Set();
  (state.meta?.monitors || []).forEach((name) => monitorSet.add(name));
  (state.snapshot?.monitors || []).forEach((name) => monitorSet.add(name));
  Object.keys(state.snapshot?.values || {}).forEach((name) => monitorSet.add(name));

  (state.config?.items || []).forEach((item) => {
    if (item.monitor) monitorSet.add(item.monitor);
  });
  (state.config?.custom_monitors || []).forEach((item) => {
    if (item.name) monitorSet.add(item.name);
    (item.sources || []).forEach((name) => monitorSet.add(name));
  });

  return Array.from(monitorSet).sort();
}

function updateMonitorOptions() {
  const next = getAllMonitorOptions();
  const prev = state.monitorOptions;
  const changed = next.length !== prev.length || next.some((item, index) => item !== prev[index]);
  if (changed) {
    state.monitorOptions = next;
  }
  return changed;
}

function labelsChanged(nextLabels) {
  const prev = state.monitorLabels || {};
  const prevKeys = Object.keys(prev);
  const nextKeys = Object.keys(nextLabels || {});
  if (prevKeys.length !== nextKeys.length) return true;
  for (const key of nextKeys) {
    if (prev[key] !== nextLabels[key]) return true;
  }
  return false;
}

function monitorLabel(name) {
  const key = String(name || "");
  return String(state.monitorLabels?.[key] || "").trim();
}

function monitorDisplayName(name) {
  const key = String(name || "");
  const label = monitorLabel(key);
  if (label) {
    return label;
  }
  return key;
}

function setError(text) {
  state.error = text || "";
  const errEl = document.getElementById("global-error");
  if (errEl) {
    errEl.textContent = state.error;
  }
}

function normalizeAutoApplyDelay(value) {
  const parsed = Math.round(num(value, 300));
  if (AUTO_APPLY_DELAY_OPTIONS.includes(parsed)) {
    return parsed;
  }
  return 300;
}

function getSaveStateView() {
  const text = state.saving ? "保存中..." : (state.dirty ? "未保存更改" : "已保存");
  const cls = state.saving ? "saving" : (state.dirty ? "dirty" : "clean");
  return { text, cls };
}

function updateSaveStateDOM() {
  const badge = document.getElementById("save-state-badge");
  if (!badge) return;
  const view = getSaveStateView();
  badge.textContent = view.text;
  badge.className = `save-state ${view.cls}`;
}

function clearAutoApplyTimer() {
  if (!state.autoApplyTimer) return;
  window.clearTimeout(state.autoApplyTimer);
  state.autoApplyTimer = null;
}

function scheduleAutoApply(delay = null) {
  if (!state.ui.autoApply) return;
  clearAutoApplyTimer();
  const effectiveDelay = delay === null ? normalizeAutoApplyDelay(state.ui.autoApplyDelay) : Math.max(0, delay);
  state.autoApplyTimer = window.setTimeout(async () => {
    state.autoApplyTimer = null;
    if (!state.ui.autoApply || !state.dirty) return;
    try {
      await saveAndApplyPreview();
      setError("");
    } catch (error) {
      setError(`自动应用失败: ${error.message}`);
    } finally {
      updateSaveStateDOM();
    }
  }, effectiveDelay);
}

function markDirty() {
  if (isEditingProfileReadOnly()) {
    setError("内置只读配置，请先复制");
    render();
    return;
  }
  state.dirty = true;
  updateSaveStateDOM();
  scheduleAutoApply();
}

function clearDirty() {
  state.dirty = false;
  updateSaveStateDOM();
}

function confirmDiscardUnsavedChanges() {
  if (!state.dirty || state.saving) {
    return true;
  }
  return window.confirm("当前有未保存更改，继续操作将丢失这些修改。是否继续？");
}

async function saveEditingProfile() {
  if (isEditingProfileReadOnly()) return;
  if (!state.config) return;
  const profileName = state.editingProfile || state.meta?.active_profile || "default";
  if (!profileName) return;
  if (!state.dirty && !state.saving) {
    return;
  }

  if (state.saving) {
    state.savePending = true;
    return state.savePromise;
  }

  state.saving = true;
  updateSaveStateDOM();
  const payloadConfig = ensureConfig(deepClone(state.config));
  state.savePromise = (async () => {
    try {
      let response;
      if (isRuntimeStreamReady()) {
        response = await runtimeStreamCall("save_profile_config", {
          profile: profileName,
          config: payloadConfig,
        });
      } else {
        response = await api(`/api/profiles/${encodeURIComponent(profileName)}`, {
          method: "PUT",
          body: JSON.stringify({ config: payloadConfig }),
        });
      }
      applyProfilesPayload(response);
      state.config = ensureConfig(payloadConfig);
      clearDirty();
      setError("");
    } catch (error) {
      state.dirty = true;
      setError(`保存失败: ${error.message}`);
      throw error;
    } finally {
      state.saving = false;
      state.savePromise = null;
      state.savePending = false;
      updateSaveStateDOM();
    }
  })();
  return state.savePromise;
}

async function flushAutoSave() {
  if (state.autoApplyTimer) {
    clearAutoApplyTimer();
    if (state.ui.autoApply && state.dirty) {
      await saveAndApplyPreview();
      return;
    }
  }
  if (state.saving && state.savePromise) {
    await state.savePromise;
  }
}

async function saveAndApplyPreview() {
  await saveEditingProfile();
  await syncPreviewConfig();
}

async function syncPreviewConfig() {
  if (!state.config) return;
  if (state.previewSyncing) {
    state.previewSyncPending = true;
    return;
  }

  state.previewSyncing = true;
  try {
    const cfg = ensureConfig(deepClone(state.config));
    if (isRuntimeStreamReady()) {
      await runtimeStreamCall("preview_config", { config: cfg }, 5000);
      await runtimeStreamCall("request_runtime", {}, 3000).catch(() => {});
    } else if (state.runtimeStream && state.runtimeStream.readyState === WebSocket.CONNECTING) {
      return;
    } else {
      await api("/api/preview/config", {
        method: "POST",
        body: JSON.stringify({ config: cfg }),
      });
      await refreshPreviewImage();
    }
  } catch (error) {
    state.previewError = `预览同步失败: ${error.message}`;
    const errorEl = document.getElementById("preview-render-error");
    if (errorEl) {
      errorEl.textContent = state.previewError;
    }
  } finally {
    state.previewSyncing = false;
    if (state.previewSyncPending) {
      state.previewSyncPending = false;
      void syncPreviewConfig();
    }
  }
}

function render() {
  destroyColorPickers();

  const prevList = document.getElementById("elements-list");
  if (prevList) {
    state.ui.elementListScrollTop = prevList.scrollTop;
  }

  const cfg = state.config;
  if (!cfg) {
    app.innerHTML = `<div class="panel"><h2>加载失败</h2><p class="error">${escapeHTML(state.error)}</p></div>`;
    return;
  }
  if (state.editingProfile) {
    cfg.name = state.editingProfile;
  }

  app.innerHTML = `
    ${renderTopBar()}
    <div id="global-error" class="global-error">${escapeHTML(state.error || "")}</div>
    <main class="main-content">
      <div class="tab-nav">
        <button data-action="switch-tab" data-tab="basic" class="${state.activeTab === "basic" ? "active" : ""}">基础配置</button>
        <button data-action="switch-tab" data-tab="elements" class="${state.activeTab === "elements" ? "active" : ""}">屏幕元素</button>
        <button data-action="switch-tab" data-tab="custom" class="${state.activeTab === "custom" ? "active" : ""}">自定义采集项</button>
        <button data-action="switch-tab" data-tab="collection" class="${state.activeTab === "collection" ? "active" : ""}">采集运行态</button>
      </div>
      <div class="tab-content">
        ${isEditingProfileReadOnly() ? `<div class="global-error">当前为内置只读配置，不能直接修改。请使用顶部“复制”创建可编辑配置。</div>` : ""}
        ${renderActiveTab(cfg)}
      </div>
    </main>
  `;

  ensurePreviewResizeObserver();
  attachPreviewDragHandlers();
  updateRuntimePanelDOM();
  initColorPickers();

  const nextList = document.getElementById("elements-list");
  if (nextList) {
    nextList.scrollTop = Math.max(0, num(state.ui.elementListScrollTop, 0));
  }
}

function updateElementListSelectionDOM() {
  const list = document.getElementById("elements-list");
  if (!list) return;
  const buttons = list.querySelectorAll("button[data-action='select-item']");
  for (const button of buttons) {
    const index = Number(button.dataset.index);
    const active = Number.isFinite(index) && index === state.selectedItem;
    button.classList.toggle("active", active);
  }
}

function updatePreviewSelectedNoticeDOM() {
  const selected = state.config?.items?.[state.selectedItem];
  const indexEl = document.getElementById("preview-selected-index");
  if (indexEl) {
    indexEl.textContent = selected ? String(state.selectedItem + 1) : "无";
  }
}

function rerenderEditorPanelDOM() {
  const editorPanel = document.getElementById("editor-panel");
  if (!editorPanel || !state.config) return false;
  const currentScrollTop = editorPanel.scrollTop;
  destroyColorPickers();
  editorPanel.innerHTML = renderElementEditor(state.config.items[state.selectedItem]);
  initColorPickers();
  editorPanel.scrollTop = currentScrollTop;
  return true;
}

function syncElementSelectionUI(options = {}) {
  const rerenderEditor = options.rerenderEditor !== false;
  if (state.activeTab !== "elements") return;
  refreshSelectedItem();
  updateElementListSelectionDOM();
  if (rerenderEditor && !rerenderEditorPanelDOM()) {
    render();
    return;
  }
  updatePreviewSelectedNoticeDOM();
  updatePreviewDOM();
}

function renderTopBar() {
  const editing = state.editingProfile || state.meta?.active_profile || "default";
  const editingInfo = getProfileInfo(editing);
  const saveStateView = getSaveStateView();
  const options = (state.profiles || [])
    .map((item) => {
      const suffix = item.readonly ? " [内置]" : "";
      return `<option value="${escapeAttr(item.name)}" ${item.name === editing ? "selected" : ""}>${escapeHTML(item.name + suffix)}</option>`;
    })
    .join("");
  const autoApplyDelay = normalizeAutoApplyDelay(state.ui.autoApplyDelay);
  state.ui.autoApplyDelay = autoApplyDelay;
  const delayOptions = AUTO_APPLY_DELAY_OPTIONS
    .map((ms) => `<option value="${ms}" ${autoApplyDelay === ms ? "selected" : ""}>${ms}ms</option>`)
    .join("");

  return `
    <div class="topbar">
      <div class="topbar-row">
        <div class="topbar-actions-left">
          <label>配置文件
            <select id="profile-select">${options}</select>
          </label>
          <button data-action="activate-profile">设为激活</button>
          <button data-action="create-profile">新建</button>
          <button data-action="rename-profile" ${editingInfo?.readonly ? "disabled" : ""}>重命名</button>
          <button data-action="save-profile-as">复制</button>
          <button data-action="delete-profile" ${editingInfo?.readonly ? "disabled" : ""}>删除</button>
          <button data-action="refresh-profiles">刷新</button>
          <button data-action="export-json">导出 JSON</button>
          <button data-action="import-json">导入 JSON</button>
        </div>
        <div class="topbar-actions-right">
          <label class="topbar-check">
            <input type="checkbox" data-ui-field="autoApply" ${state.ui.autoApply ? "checked" : ""} />
            自动应用修改
          </label>
          <label class="topbar-delay">
            延迟
            <select data-ui-field="autoApplyDelay">
              ${delayOptions}
            </select>
          </label>
          <button data-action="save-profile" ${editingInfo?.readonly ? "disabled" : ""}>保存</button>
          <span id="save-state-badge" class="save-state ${saveStateView.cls}">${saveStateView.text}</span>
        </div>
      </div>
      <input id="import-file-input" type="file" accept=".json,application/json" style="display:none" />
    </div>
  `;
}

function renderActiveTab(cfg) {
  switch (state.activeTab) {
    case "elements":
      return renderElementsTab(cfg);
    case "custom":
      return renderCustomTab(cfg);
    case "collection":
      return renderCollectionTab();
    case "basic":
    default:
      return renderBasicTab(cfg);
  }
}

function collectorDisplayName(name) {
  return COLLECTOR_NAME_ALIAS[name] || name;
}

function collectorToggleField(cfg, collectorName) {
  const enabled = collectorEnabledValue(cfg, collectorName);
  return `
    <label class="collector-toggle">
      <input
        type="checkbox"
        data-collector-name="${escapeAttr(collectorName)}"
        data-collector-kind="enabled"
        ${enabled ? "checked" : ""}
      />
      启用
    </label>
  `;
}

function renderCollectorEnabledSection(cfg) {
  const collectorNames = knownCollectorNames(cfg);
  if (collectorNames.length === 0) {
    return "";
  }
  const rows = collectorNames
    .map((name) => `
      <div class="collector-enable-row">
        <div class="collector-enable-meta">
          <strong>${escapeHTML(collectorDisplayName(name))}</strong>
          <span class="collector-enable-key">${escapeHTML(name)}</span>
        </div>
        ${collectorToggleField(cfg, name)}
      </div>
    `)
    .join("");
  return `
    <div class="panel">
      <h3>采集器启用控制</h3>
      <div class="collector-enable-list">
        ${rows}
      </div>
    </div>
  `;
}

function collectorOptionField(cfg, collectorName, optionKey, label, type = "text") {
  const value = collectorOptionValue(cfg, collectorName, optionKey, "");
  return `
    <div class="field">
      <label>${label}</label>
      <input
        type="${type}"
        data-collector-name="${escapeAttr(collectorName)}"
        data-collector-kind="option"
        data-collector-option="${escapeAttr(optionKey)}"
        value="${escapeAttr(value)}"
      />
    </div>
  `;
}

function renderCollectorConfigSection(cfg) {
  const collectorNames = knownCollectorNames(cfg)
    .filter((name) => name === "external.coolercontrol" || name === "external.librehardwaremonitor");
  if (collectorNames.length === 0) {
    return "";
  }

  const blocks = collectorNames
    .map((name) => {
      let optionsHTML = "";
      if (name === "external.coolercontrol") {
        optionsHTML = `
          <div class="grid">
            ${collectorOptionField(cfg, name, "url", "CoolerControl URL")}
            ${collectorOptionField(cfg, name, "username", "CoolerControl 用户名")}
            ${collectorOptionField(cfg, name, "password", "CoolerControl 密码", "password")}
          </div>
        `;
      } else if (name === "external.librehardwaremonitor") {
        optionsHTML = `
          <div class="grid">
            ${collectorOptionField(cfg, name, "url", "Libre URL")}
          </div>
        `;
      }

      return `
        <div class="collector-config-card">
          <div class="collector-config-head">
            <strong title="${escapeAttr(name)}">${escapeHTML(collectorDisplayName(name))}</strong>
            <span class="collector-enable-key">${escapeHTML(name)}</span>
          </div>
          ${optionsHTML}
        </div>
      `;
    })
    .join("");

  return `
    <div class="panel">
      <h3>采集器配置</h3>
      <div class="collector-config-cards">
        ${blocks}
      </div>
      <div class="notice">采集器 URL/认证等连接参数在这里维护；启用状态请在“采集器启用控制”中设置。</div>
    </div>
  `;
}

function renderBasicTab(cfg) {
  const outputOptions = (state.meta?.output_types || ["memimg", "ax206usb"])
    .map((value) => `<option value="${escapeAttr(value)}" ${cfg.output_types.includes(value) ? "selected" : ""}>${escapeHTML(value)}</option>`)
    .join("");
  const networkInterfaceValues = (state.meta?.network_interfaces || []);
  const networkOptions = [
    `<option value="" ${cfg.network_interface === "" ? "selected" : ""}>未设置</option>`,
    ...networkInterfaceValues.map((value) => `<option value="${escapeAttr(value)}" ${cfg.network_interface === value ? "selected" : ""}>${escapeHTML(value)}</option>`),
  ].join("");
  const fontOptions = collectFontOptions(cfg)
    .map((name) => `<option value="${escapeAttr(name)}" ${cfg.default_font === name ? "selected" : ""}>${escapeHTML(name)}</option>`)
    .join("");

  return `
    <div class="layout-single">
      <div class="panel">
        <h3>基础参数</h3>
        <div class="grid">
          ${fieldNumber("width", "宽度", cfg.width)}
          ${fieldNumber("height", "高度", cfg.height)}
          ${fieldNumber("layout_padding", "布局内边距", cfg.layout_padding)}
          ${fieldSelectMultiple("output_types", "输出类型(多选)", outputOptions, 2)}
          ${fieldNumber("refresh_interval", "刷新间隔(ms)", cfg.refresh_interval)}
          ${fieldNumber("history_size", "折线历史长度", cfg.history_size)}
          ${fieldSelect("network_interface", "网络接口", networkOptions)}
          ${fieldSelect("default_font", "默认字体", fontOptions)}
          ${fieldNumber("default_font_size", "默认字号", cfg.default_font_size)}
          ${fieldColor("default_color", "默认颜色", cfg.default_color)}
          ${fieldColor("default_background", "默认背景", cfg.default_background)}
        </div>

        <h3>默认等级颜色（4级）</h3>
        <div class="grid-4">
          ${levelColorField("default-level-color", 0, cfg.level_colors[0])}
          ${levelColorField("default-level-color", 1, cfg.level_colors[1])}
          ${levelColorField("default-level-color", 2, cfg.level_colors[2])}
          ${levelColorField("default-level-color", 3, cfg.level_colors[3])}
        </div>

        <h3>默认阈值（4级）</h3>
        <div class="grid-4">
          ${levelThresholdField("default-threshold", 0, cfg.default_thresholds[0])}
          ${levelThresholdField("default-threshold", 1, cfg.default_thresholds[1])}
          ${levelThresholdField("default-threshold", 2, cfg.default_thresholds[2])}
          ${levelThresholdField("default-threshold", 3, cfg.default_thresholds[3])}
        </div>
      </div>
      ${renderCollectorEnabledSection(cfg)}
      ${renderCollectorConfigSection(cfg)}
    </div>
  `;
}

function renderElementsTab(cfg) {
  const selected = cfg.items[state.selectedItem];
  const typeOptions = (state.meta?.item_types || ELEMENT_TYPES)
    .map((itemType) => `<option value="${escapeAttr(itemType)}" ${state.addItemType === itemType ? "selected" : ""}>${escapeHTML(itemTypeDisplayName(itemType))}</option>`)
    .join("");

  return `
    <div class="elements-layout">
      <div class="panel list-panel">${renderElementList(cfg, selected, typeOptions)}</div>
      <div class="panel preview-panel">${renderPreview(cfg)}</div>
      <div class="panel editor-panel" id="editor-panel">${renderElementEditor(selected)}</div>
    </div>
  `;
}

function renderElementList(cfg, selected, typeOptions) {
  const monitorForAdd = resolveAddItemMonitor();
  const addMonitorOptions = state.monitorOptions
    .map((name) => `<option value="${escapeAttr(name)}" ${monitorForAdd === name ? "selected" : ""}>${escapeHTML(monitorDisplayName(name))}</option>`)
    .join("");
  const addMonitorSelect = MONITOR_ELEMENT_TYPES.has(state.addItemType)
    ? `
      <label>采集项
        <select data-ui-field="addItemMonitor" ${state.monitorOptions.length === 0 ? "disabled" : ""}>
          ${addMonitorOptions || '<option value="">（无）</option>'}
        </select>
      </label>
    `
    : "";

  const list = (cfg.items || [])
    .map((item, index) => {
      const active = index === state.selectedItem ? "active" : "";
      return `<button class="${active}" data-action="select-item" data-index="${index}">${escapeHTML(item.edit_ui_name || defaultEditUIName(index, item))}</button>`;
    })
    .join("");

  return `
    <h3>屏幕元素列表</h3>
    <div class="inline-actions">
      <label>类型
        <select data-ui-field="addItemType">${typeOptions}</select>
      </label>
      ${addMonitorSelect}
      <button data-action="add-item">添加</button>
      <button data-action="clone-item" ${selected ? "" : "disabled"}>复制</button>
      <button data-action="remove-item" ${selected ? "" : "disabled"}>删除</button>
      <button data-action="move-item-up" ${selected && state.selectedItem > 0 ? "" : "disabled"}>上移</button>
      <button data-action="move-item-down" ${selected && state.selectedItem < cfg.items.length - 1 ? "" : "disabled"}>下移</button>
    </div>
    <div id="elements-list" class="list">${list || '<button disabled>暂无屏幕元素</button>'}</div>
  `;
}

function renderElementEditor(item) {
  if (!item) {
    return `<h3>元素编辑</h3><div class="notice">请先在左侧元素列表中选择一个屏幕元素</div>`;
  }

  const typeOptions = (state.meta?.item_types || ELEMENT_TYPES)
    .map((itemType) => `<option value="${escapeAttr(itemType)}" ${item.type === itemType ? "selected" : ""}>${escapeHTML(itemTypeDisplayName(itemType))}</option>`)
    .join("");

  const monitorOptions = ["<option value=\"\">（无）</option>"]
    .concat(
      state.monitorOptions.map(
        (name) => `<option value=\"${escapeAttr(name)}\" ${item.monitor === name ? "selected" : ""}>${escapeHTML(monitorDisplayName(name))}</option>`,
      ),
    )
    .join("");

  const isMonitorType = MONITOR_ELEMENT_TYPES.has(item.type);
  const isLabelType = LABEL_ELEMENT_TYPES.has(item.type);

  return `
    <h3>元素编辑</h3>
    <div class="editor-block">
      <h4>基础属性</h4>
      <div class="grid compact-row base-compact-row">
        ${itemTextField("edit_ui_name", item.edit_ui_name, "compact-grow")}
        <div class="field compact-md"><label>${itemFieldDisplayName("type")}</label><select data-item-field="type">${typeOptions}</select></div>
        ${isMonitorType ? `<div class="field compact-grow"><label>${itemFieldDisplayName("monitor")}</label><select data-item-field="monitor">${monitorOptions}</select></div>` : ""}
        ${isLabelType ? itemTextField("text", item.text, "compact-grow") : ""}
        ${itemNumberField("x", item.x, "compact-xs")}
        ${itemNumberField("y", item.y, "compact-xs")}
        ${itemNumberField("width", item.width, "compact-xs")}
        ${itemNumberField("height", item.height, "compact-xs")}
        ${itemNumberField("font_size", item.font_size, "compact-sm")}
        ${isMonitorType ? itemTextField("unit", item.unit || "auto", "compact-md") : ""}
        ${isMonitorType ? itemNumberField("unit_font_size", item.unit_font_size, "compact-sm") : ""}
        ${itemColorField("color", item.color, "compact-color")}
        ${isMonitorType ? itemColorField("unit_color", item.unit_color, "compact-color") : ""}
        ${itemColorField("bg", item.bg, "compact-color")}
        ${itemColorField("border_color", item.border_color, "compact-color")}
        ${itemNumberField("border_width", item.border_width, "compact-sm")}
        ${itemNumberField("radius", item.radius, "compact-sm")}
      </div>
    </div>
    ${renderExtensionEditor(item)}
  `;
}

function renderExtensionEditor(item) {
  const thresholdMode = Array.isArray(item.thresholds) && item.thresholds.length === 4 ? "custom" : "default";
  const colorMode = Array.isArray(item.level_colors) && item.level_colors.length === 4 ? "custom" : "default";
  const isSimpleChart = item.type === "simple_line_chart";
  const extensionParts = [];

  if (RANGE_ELEMENT_TYPES.has(item.type)) {
    extensionParts.push(`
      <div class="grid compact-row ext-compact-row">
        ${itemNumberField("min_value", item.min_value, "compact-sm")}
        ${itemNumberField("max_value", item.max_value, "compact-sm")}
        ${isSimpleChart
          ? itemNumberField("point_size", item.point_size ?? state.config?.history_size ?? 150, "compact-sm")
          : itemNumberField("max", item.max, "compact-sm")}
      </div>
    `);
  }

  if (MONITOR_ELEMENT_TYPES.has(item.type)) {
    extensionParts.push(`
      <div class="grid compact-row ext-compact-row">
        <div class="field compact-md"><label>阈值配置</label>
          <select data-item-field="threshold_mode">
            <option value="default" ${thresholdMode === "default" ? "selected" : ""}>使用默认</option>
            <option value="custom" ${thresholdMode === "custom" ? "selected" : ""}>自定义</option>
          </select>
        </div>
        <div class="field compact-md"><label>等级色配置</label>
          <select data-item-field="level_color_mode">
            <option value="default" ${colorMode === "default" ? "selected" : ""}>使用默认</option>
            <option value="custom" ${colorMode === "custom" ? "selected" : ""}>自定义</option>
          </select>
        </div>
      </div>
    `);
    if (thresholdMode === "custom") {
      extensionParts.push(`
        <div class="grid compact-row ext-compact-row">
          ${itemThresholdField(0, item.thresholds?.[0])}
          ${itemThresholdField(1, item.thresholds?.[1])}
          ${itemThresholdField(2, item.thresholds?.[2])}
          ${itemThresholdField(3, item.thresholds?.[3])}
        </div>
      `);
    }
    if (colorMode === "custom") {
      extensionParts.push(`
        <div class="grid compact-row ext-compact-row">
          ${itemLevelColorField(0, item.level_colors?.[0])}
          ${itemLevelColorField(1, item.level_colors?.[1])}
          ${itemLevelColorField(2, item.level_colors?.[2])}
          ${itemLevelColorField(3, item.level_colors?.[3])}
        </div>
      `);
    }
  }

  if (isFullElementType(item.type)) {
    extensionParts.push(renderFullExtensionContent(item));
  }

  if (extensionParts.length === 0) {
    return "";
  }

  return `
    <div class="editor-block">
      <h4>扩展属性</h4>
      ${extensionParts.join("")}
    </div>
  `;
}

function renderFullExtensionContent(item) {
  const attrs = ensureItemRenderAttrs(item);
  const schema = [...FULL_COMMON_ATTR_SCHEMAS, ...(FULL_TYPE_ATTR_SCHEMAS[item.type] || [])];
  const attrFields = schema.map((spec) => renderFullAttrField(spec, attrs[spec.key])).join("");
  const titleValue = String(attrs.title ?? "");
  const titleFontSize = attrs.title_font_size ?? "";
  return `
    <div class="grid compact-row full-attrs-grid ext-compact-row">
      <div class="field compact-grow">
        <label>标题</label>
        <input data-item-attr-field="title" data-item-attr-kind="text" value="${escapeAttr(titleValue)}" placeholder="留空则使用采集项名称" />
      </div>
      <div class="field compact-sm">
        <label>标题字号</label>
        <input type="number" data-item-attr-field="title_font_size" data-item-attr-kind="number" value="${escapeAttr(titleFontSize)}" />
      </div>
      ${attrFields}
    </div>
  `;
}

function renderFullAttrField(spec, value) {
  const key = String(spec?.key || "").trim();
  if (!key) return "";
  const label = escapeHTML(spec.label || key);
  const kind = spec.kind || "text";
  if (kind === "bool") {
    const boolValue = value === true || String(value).toLowerCase() === "true";
    return `
      <div class="field compact-md">
        <label>${label}</label>
        <select data-item-attr-field="${escapeAttr(key)}" data-item-attr-kind="bool">
          <option value="false" ${boolValue ? "" : "selected"}>关闭</option>
          <option value="true" ${boolValue ? "selected" : ""}>开启</option>
        </select>
      </div>
    `;
  }
  if (kind === "select") {
    const options = Array.isArray(spec.options) ? spec.options : [];
    const selected = String(value ?? "");
    const optionHTML = options
      .map((opt) => `<option value="${escapeAttr(opt)}" ${selected === String(opt) ? "selected" : ""}>${escapeHTML(fullAttrOptionDisplayName(opt))}</option>`)
      .join("");
    return `
      <div class="field compact-md">
        <label>${label}</label>
        <select data-item-attr-field="${escapeAttr(key)}" data-item-attr-kind="select">
          ${optionHTML}
        </select>
      </div>
    `;
  }
  if (kind === "number") {
    return `
      <div class="field compact-sm">
        <label>${label}</label>
        <input type="number" data-item-attr-field="${escapeAttr(key)}" data-item-attr-kind="number" value="${escapeAttr(value ?? "")}" />
      </div>
    `;
  }
  if (kind === "color") {
    const colorValue = String(value || "");
    return `
      <div class="field compact-color">
        <label>${label}</label>
        <input data-item-attr-field="${escapeAttr(key)}" data-item-attr-kind="text" value="${escapeAttr(colorValue)}" placeholder="#1f2937" />
      </div>
    `;
  }
  return `
    <div class="field compact-md">
      <label>${label}</label>
      <input data-item-attr-field="${escapeAttr(key)}" data-item-attr-kind="text" value="${escapeAttr(value ?? "")}" />
    </div>
  `;
}

function renderPreview(cfg) {
  const selected = cfg.items[state.selectedItem];
  const purePreview = !!state.ui.purePreview;
  const showGrid = !!state.ui.showGrid;
  const snapEnabled = !!state.ui.snapEnabled;
  const snapSize = clamp(Math.round(num(state.ui.snapSize, 10)), 2, 100);
  const previewZoomAuto = state.ui.previewZoomAuto !== false;
  const previewZoom = clamp(Math.round(num(state.ui.previewZoom, 100)), 25, 400);
  return `
    <h3>预览</h3>
    <div class="inline-actions preview-toolbar">
      <label><input type="checkbox" data-ui-field="purePreview" ${purePreview ? "checked" : ""} /> 纯预览模式</label>
      <label><input type="checkbox" data-ui-field="previewZoomAuto" ${previewZoomAuto ? "checked" : ""} /> 自动缩放</label>
      <label>缩放
        <input class="zoom-range" type="range" min="25" max="400" step="5" data-ui-field="previewZoom" value="${previewZoom}" ${previewZoomAuto ? "disabled" : ""} />
        <span id="preview-zoom-value">${previewZoom}%</span>
      </label>
      <button data-action="preview-zoom-reset" type="button">100%</button>
      <label><input type="checkbox" data-ui-field="showGrid" ${showGrid ? "checked" : ""} /> 网格</label>
      <label><input type="checkbox" data-ui-field="snapEnabled" ${snapEnabled ? "checked" : ""} /> 吸附</label>
      <label>网格尺寸
        <input class="snap-size-input" type="number" min="2" max="100" step="1" data-ui-field="snapSize" value="${snapSize}" />
      </label>
    </div>
    <div class="preview-wrapper" id="preview-wrapper">
      <canvas class="preview-canvas" id="preview-canvas" width="1" height="1"></canvas>
    </div>
    <div id="preview-render-error" class="error">${escapeHTML(state.previewError || "")}</div>
    <div class="notice">Go 渲染图实时预览；仅选中元素可拖拽/缩放。当前选中：<span id="preview-selected-index">${selected ? `${state.selectedItem + 1}` : "无"}</span></div>
  `;
}

function renderCustomTab(cfg) {
  const typeOptions = state.meta?.custom_monitor_types || ["file", "mixed", "coolercontrol", "librehardwaremonitor"];
  const aggregateOptions = state.meta?.custom_aggregate_types || ["max", "min", "avg"];

  const content = (cfg.custom_monitors || [])
    .map((monitor, index) => {
      const typeSelect = typeOptions
        .map((value) => `<option value="${escapeAttr(value)}" ${monitor.type === value ? "selected" : ""}>${escapeHTML(value)}</option>`)
        .join("");

      const aggregateSelect = aggregateOptions
        .map((value) => `<option value="${escapeAttr(value)}" ${monitor.aggregate === value ? "selected" : ""}>${escapeHTML(value)}</option>`)
        .join("");

      const sourceOptions = state.monitorOptions
        .filter((name) => name !== monitor.name)
        .map((name) => `<option value="${escapeAttr(name)}" ${monitor.sources.includes(name) ? "selected" : ""}>${escapeHTML(monitorDisplayName(name))}</option>`)
        .join("");

      return `
        <div class="custom-card">
          <div class="inline-actions"><strong>#${index + 1}</strong><button data-action="remove-custom" data-index="${index}">删除</button></div>
          <div class="grid">
            ${customField(index, "name", "name", monitor.name)}
            ${customField(index, "label", "label", monitor.label)}
            ${customSelectField(index, "type", "type", typeSelect)}
            ${customField(index, "unit", "unit", monitor.unit)}
            ${customField(index, "precision", "precision", nullableNumber(monitor.precision), "number")}
            ${customField(index, "min", "min", nullableNumber(monitor.min), "number")}
            ${customField(index, "max", "max", nullableNumber(monitor.max), "number")}
          </div>

          ${monitor.type === "file" ? `
            <div class="grid">
              ${customField(index, "path", "path", monitor.path)}
              ${customField(index, "scale", "scale", nullableNumber(monitor.scale), "number")}
              ${customField(index, "offset", "offset", nullableNumber(monitor.offset), "number")}
            </div>
          ` : ""}

          ${monitor.type === "mixed" ? `
            <div class="grid">
              <div class="field">
                <label>sources（多选）</label>
                <select multiple size="6" data-custom-index="${index}" data-custom-field="sources">${sourceOptions}</select>
              </div>
              ${customSelectField(index, "aggregate", "aggregate", aggregateSelect)}
            </div>
          ` : ""}

          ${monitor.type === "coolercontrol" ? renderCoolerControlCustom(index, monitor) : ""}
          ${monitor.type === "librehardwaremonitor" ? renderLibreCustom(index, monitor) : ""}
        </div>
      `;
    })
    .join("");

  return `
    <div class="layout-single">
      <div class="panel">
        <h3>自定义采集项</h3>
        <div class="inline-actions">
          <button data-action="add-custom">添加自定义采集项</button>
        </div>
        ${content || '<div class="notice">暂无自定义采集项</div>'}
      </div>
    </div>
  `;
}

function renderLibreCustom(index, monitor) {
  const options = (state.monitorOptions || [])
    .filter((name) => String(name).startsWith("libre_"))
    .map((name) => ({ name, label: monitorDisplayName(name) }));
  const selected = String(monitor.source || "");
  const sensorOptions = ["<option value=\"\">（无）</option>"]
    .concat(
      options.map((item) => {
        const optionLabel = item.label || item.name || "unnamed";
        return `<option value=\"${escapeAttr(item.name)}\" ${selected === item.name ? "selected" : ""}>${escapeHTML(optionLabel)}</option>`;
      }),
    )
    .join("");
  return `
    <div class="grid">
      ${customSelectField(index, "source", "source", sensorOptions)}
    </div>
  `;
}

function renderCoolerControlCustom(index, monitor) {
  const options = (state.monitorOptions || [])
    .filter((name) => String(name).startsWith("coolercontrol_"))
    .map((name) => ({ name, label: monitorDisplayName(name) }));
  const selected = String(monitor.source || "");
  const sourceOptions = ["<option value=\"\">（无）</option>"]
    .concat(
      options.map((item) => {
        const optionLabel = item.label || item.name || "unnamed";
        return `<option value=\"${escapeAttr(item.name)}\" ${selected === item.name ? "selected" : ""}>${escapeHTML(optionLabel)}</option>`;
      }),
    )
    .join("");

  return `
    <div class="grid">
      ${customSelectField(index, "source", "source", sourceOptions)}
    </div>
  `;
}

function renderCollectionTab() {
  return `
    <div class="layout-single collection-layout">
      <div class="panel collection-panel">
        <div class="collection-top">
          <h3>采集运行态</h3>
          <div class="runtime-meta">
            <span>模式: <strong id="runtime-mode">${escapeHTML(getRuntimeModeText())}</strong></span>
            <span>更新时间: <strong id="runtime-updated">${escapeHTML(formatTimestamp(state.snapshot?.updated_at))}</strong></span>
            <span>采集项: <strong id="runtime-count">${Object.keys(state.snapshot?.values || {}).length}</strong></span>
          </div>
          <div class="inline-actions">
            <button data-action="refresh-snapshot">立即刷新</button>
          </div>
          <div class="notice">页面活跃时每 2 秒抓全量采集项，30 秒无请求自动回到按配置采集。</div>
        </div>
        <div class="collector-panel" id="collector-panel">${renderCollectorRows()}</div>
        <div class="runtime-monitor-stats" id="runtime-monitor-stats">${renderRuntimeMonitorStats()}</div>
        <div class="collection-list" id="runtime-values">${renderRuntimeValueRows()}</div>
      </div>
    </div>
  `;
}

function normalizeCollectors(payload) {
  const list = Array.isArray(payload?.items) ? payload.items : [];
  return list
    .map((item) => ({
      name: String(item?.name || "").trim(),
      enabled: !!item?.enabled,
    }))
    .filter((item) => item.name)
    .sort((a, b) => a.name.localeCompare(b.name));
}

async function refreshCollectors() {
  state.collectorsLoading = true;
  try {
    const payload = await api("/api/collectors");
    state.collectors = normalizeCollectors(payload);
  } catch (_) {
    state.collectors = [];
  } finally {
    state.collectorsLoading = false;
  }
}

function renderCollectorRows() {
  if (state.collectorsLoading && (!state.collectors || state.collectors.length === 0)) {
    return '<div class="notice">采集器状态加载中...</div>';
  }
  const items = Array.isArray(state.collectors) ? state.collectors : [];
  if (items.length === 0) {
    return '<div class="notice">暂无采集器信息</div>';
  }
  const enabledCount = items.filter((item) => item.enabled).length;
  const rows = items
    .map((item) => {
      const stateText = item.enabled ? "enabled" : "disabled";
      return `<div class="collector-row">
        <span class="collector-name" title="${escapeAttr(item.name)}">${escapeHTML(collectorDisplayName(item.name))}</span>
        <span class="collector-key">${escapeHTML(item.name)}</span>
        <span class="collector-state ${item.enabled ? "on" : "off"}">${stateText}</span>
      </div>`;
    })
    .join("");
  return `
    <div class="collector-header">
      <strong>采集器状态</strong>
      <span>${enabledCount}/${items.length} 已启用</span>
    </div>
    <div class="collector-list">${rows}</div>
  `;
}

function monitorRuntimeStats() {
  return state.snapshot?.monitor_runtime || null;
}

function formatRatio(numerator, denominator) {
  if (!Number.isFinite(numerator) || !Number.isFinite(denominator) || denominator <= 0) {
    return "0.0%";
  }
  return `${((numerator / denominator) * 100).toFixed(1)}%`;
}

function renderRuntimeMonitorStats() {
  const stats = monitorRuntimeStats();
  if (!stats) {
    return '<div class="notice">调度器运行态数据暂不可用</div>';
  }

  const windowDropRatio = formatRatio(stats.last_window_dropped || 0, stats.last_window_scheduled || 0);
  const windowSlowRatio = formatRatio(stats.last_window_slow || 0, stats.last_window_completed || 0);
  const totalDropRatio = formatRatio(stats.dropped_total || 0, stats.scheduled_total || 0);
  const totalSlowRatio = formatRatio(stats.slow_total || 0, stats.completed_total || 0);
  return `
    <div class="runtime-stat-grid">
      <div class="runtime-stat-row"><span>workers</span><strong>${escapeHTML(String(stats.worker_count || 0))}</strong></div>
      <div class="runtime-stat-row"><span>queue</span><strong>${escapeHTML(`${stats.queue_len || 0}/${stats.queue_size || 0}`)}</strong></div>
      <div class="runtime-stat-row"><span>scale</span><strong>x${escapeHTML(String(stats.interval_scale || 1))}</strong></div>
      <div class="runtime-stat-row"><span>auto tune</span><strong>${stats.auto_tune ? "on" : "off"}</strong></div>
      <div class="runtime-stat-row"><span>window dropped</span><strong>${escapeHTML(`${stats.last_window_dropped || 0}/${stats.last_window_scheduled || 0} (${windowDropRatio})`)}</strong></div>
      <div class="runtime-stat-row"><span>window slow</span><strong>${escapeHTML(`${stats.last_window_slow || 0}/${stats.last_window_completed || 0} (${windowSlowRatio})`)}</strong></div>
      <div class="runtime-stat-row"><span>total dropped</span><strong>${escapeHTML(`${stats.dropped_total || 0}/${stats.scheduled_total || 0} (${totalDropRatio})`)}</strong></div>
      <div class="runtime-stat-row"><span>total slow</span><strong>${escapeHTML(`${stats.slow_total || 0}/${stats.completed_total || 0} (${totalSlowRatio})`)}</strong></div>
    </div>
  `;
}

function renderRuntimeValueRows() {
  const values = state.snapshot?.values || {};
  const names = Object.keys(values).sort();
  if (names.length === 0) {
    return '<div class="notice">暂无采集数据</div>';
  }

  return names
    .map((name) => {
      const item = values[name] || {};
      const cls = item.available ? "" : " unavailable";
      const label = String(item.label || "").trim();
      const display = label || name;
      return `<div class="runtime-row${cls}"><span title="${escapeAttr(name)}">${escapeHTML(display)}</span><span>${escapeHTML(item.text || "-")}</span></div>`;
    })
    .join("");
}

function getRuntimeModeText() {
  if (!state.snapshot) return "waiting";
  return state.snapshot.mode === "full" ? "full" : "required";
}

function formatTimestamp(value) {
  if (!value) return "-";
  const ts = new Date(value);
  if (Number.isNaN(ts.getTime())) return value;
  return ts.toLocaleString();
}

function nullableNumber(value) {
  if (value === null || value === undefined) return "";
  return String(value);
}

function collectFontOptions(cfg) {
  const set = new Set();
  (state.meta?.font_families || []).forEach((name) => {
    const text = String(name || "").trim();
    if (text) set.add(text);
  });
  (cfg.font_families || []).forEach((name) => {
    const text = String(name || "").trim();
    if (text) set.add(text);
  });
  if (cfg.default_font) set.add(cfg.default_font);
  if (set.size === 0) {
    set.add("DejaVu Sans Mono");
  }
  return Array.from(set).sort();
}

function fieldText(field, label, value = "", readonly = false, title = "") {
  const readonlyAttr = readonly ? " readonly" : "";
  const titleAttr = title ? ` title="${escapeAttr(title)}"` : "";
  return `<div class="field"><label>${label}</label><input data-field="${field}" value="${escapeAttr(value ?? "")}"${readonlyAttr}${titleAttr} /></div>`;
}

function fieldNumber(field, label, value = 0) {
  return `<div class="field"><label>${label}</label><input type="number" data-field="${field}" data-kind="number" value="${escapeAttr(value ?? "")}" /></div>`;
}

function fieldColor(field, label, value = "") {
  return `
    <div class="field">
      <label>${label}</label>
      <div class="color-input-row color-picker-row">
        <button type="button" class="color-picker-trigger" aria-label="${escapeAttr(label)}"></button>
        <input type="color" class="native-color-input" data-field="${field}" data-kind="color" value="${escapeColor(value)}" />
        <input type="number" class="native-alpha-input" min="0" max="1" step="0.01" data-field-alpha="${field}" data-kind="alpha" value="${escapeAlpha(value)}" />
      </div>
    </div>
  `;
}

function fieldSelect(field, label, optionsHTML) {
  return `<div class="field"><label>${label}</label><select data-field="${field}">${optionsHTML}</select></div>`;
}

function fieldSelectMultiple(field, label, optionsHTML, size = 3) {
  return `<div class="field"><label>${label}</label><select data-field="${field}" data-kind="multi-select" multiple size="${size}">${optionsHTML}</select></div>`;
}

function fieldCheckbox(field, label, checked = false) {
  return `<div class="field"><label><input type="checkbox" data-field="${field}" data-kind="checkbox" ${checked ? "checked" : ""} /> ${label}</label></div>`;
}

function itemTextField(field, value, className = "") {
  const cls = className ? `field ${className}` : "field";
  const label = itemFieldDisplayName(field);
  return `<div class="${cls}"><label>${label}</label><input data-item-field="${field}" data-item-kind="text" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemNumberField(field, value, className = "") {
  const cls = className ? `field ${className}` : "field";
  const label = itemFieldDisplayName(field);
  return `<div class="${cls}"><label>${label}</label><input type="number" data-item-field="${field}" data-item-kind="nullable-number" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemColorField(field, value, className = "") {
  const cls = className ? `field ${className}` : "field";
  const label = itemFieldDisplayName(field);
  return `
    <div class="${cls}">
      <label>${label}</label>
      <div class="color-input-row color-picker-row">
        <button type="button" class="color-picker-trigger" aria-label="${escapeAttr(label)}"></button>
        <input type="color" class="native-color-input" data-item-field="${field}" data-item-kind="color" value="${escapeColor(value)}" />
        <input type="number" class="native-alpha-input" min="0" max="1" step="0.01" data-item-alpha-field="${field}" data-item-kind="alpha" value="${escapeAlpha(value)}" />
      </div>
    </div>
  `;
}

function levelColorField(kind, index, value) {
  return `
    <div class="field">
      <label>L${index + 1} 颜色</label>
      <div class="color-input-row color-picker-row">
        <button type="button" class="color-picker-trigger" aria-label="L${index + 1} 颜色"></button>
        <input type="color" class="native-color-input" data-level-kind="${kind}" data-level-index="${index}" value="${escapeColor(value)}" />
        <input type="number" class="native-alpha-input" min="0" max="1" step="0.01" data-level-alpha-kind="${kind}" data-level-index="${index}" value="${escapeAlpha(value)}" />
      </div>
    </div>
  `;
}

function levelThresholdField(kind, index, value) {
  return `<div class="field"><label>L${index + 1} 阈值</label><input type="number" data-level-kind="${kind}" data-level-index="${index}" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemThresholdField(index, value) {
  return `<div class="field"><label>L${index + 1} 阈值</label><input type="number" data-item-level-kind="threshold" data-item-level-index="${index}" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemLevelColorField(index, value) {
  return `
    <div class="field">
      <label>L${index + 1} 颜色</label>
      <div class="color-input-row color-picker-row">
        <button type="button" class="color-picker-trigger" aria-label="L${index + 1} 颜色"></button>
        <input type="color" class="native-color-input" data-item-level-kind="color" data-item-level-index="${index}" value="${escapeColor(value)}" />
        <input type="number" class="native-alpha-input" min="0" max="1" step="0.01" data-item-level-alpha-kind="color" data-item-level-index="${index}" value="${escapeAlpha(value)}" />
      </div>
    </div>
  `;
}

function customField(index, field, label, value, type = "text") {
  return `<div class="field"><label>${label}</label><input type="${type}" data-custom-index="${index}" data-custom-field="${field}" value="${escapeAttr(value ?? "")}" /></div>`;
}

function customSelectField(index, field, label, optionsHTML) {
  return `<div class="field"><label>${label}</label><select data-custom-index="${index}" data-custom-field="${field}">${optionsHTML}</select></div>`;
}

function escapeHTML(text) {
  return String(text ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function escapeAttr(text) {
  return escapeHTML(text).replaceAll("`", "");
}

function clamp(value, minValue, maxValue) {
  return Math.max(minValue, Math.min(maxValue, value));
}

function normalizeLayoutPadding(value, width, height) {
  const w = Math.max(10, Math.round(num(width, 10)));
  const h = Math.max(10, Math.round(num(height, 10)));
  const maxPadding = Math.max(0, Math.floor((Math.min(w, h) - 10) / 2));
  return clamp(Math.round(num(value, 0)), 0, maxPadding);
}

function getLayoutPadding(cfg = state.config) {
  if (!cfg) return 0;
  return normalizeLayoutPadding(cfg.layout_padding, cfg.width, cfg.height);
}

function getLayoutBounds(cfg = state.config) {
  if (!cfg) {
    return { left: 0, top: 0, right: 0, bottom: 0, padding: 0 };
  }
  const padding = getLayoutPadding(cfg);
  const width = Math.max(10, Math.round(num(cfg.width, 10)));
  const height = Math.max(10, Math.round(num(cfg.height, 10)));
  const left = padding;
  const top = padding;
  const right = Math.max(left, width - padding);
  const bottom = Math.max(top, height - padding);
  return { left, top, right, bottom, padding };
}

function clampItemRectToBounds(rect, cfg = state.config) {
  const bounds = getLayoutBounds(cfg);
  const maxWidth = Math.max(10, Math.round(bounds.right - bounds.left));
  const maxHeight = Math.max(10, Math.round(bounds.bottom - bounds.top));
  const width = clamp(Math.round(num(rect?.width, 10)), 10, maxWidth);
  const height = clamp(Math.round(num(rect?.height, 10)), 10, maxHeight);
  const maxX = Math.max(bounds.left, Math.round(bounds.right - width));
  const maxY = Math.max(bounds.top, Math.round(bounds.bottom - height));
  const x = clamp(Math.round(num(rect?.x, bounds.left)), bounds.left, maxX);
  const y = clamp(Math.round(num(rect?.y, bounds.top)), bounds.top, maxY);
  return { x, y, width, height };
}

function clampAllItemsToBounds() {
  if (!state.config?.items?.length) return false;
  let changed = false;
  for (const item of state.config.items) {
    const next = clampItemRectToBounds({
      x: num(item.x, 0),
      y: num(item.y, 0),
      width: Math.max(10, num(item.width, 10)),
      height: Math.max(10, num(item.height, 10)),
    });
    if (item.x !== next.x || item.y !== next.y || item.width !== next.width || item.height !== next.height) {
      item.x = next.x;
      item.y = next.y;
      item.width = next.width;
      item.height = next.height;
      changed = true;
    }
  }
  return changed;
}

function parseColorValue(value) {
  const raw = String(value || "").trim();
  if (!raw) return { hex: "#000000", alpha: 1 };

  const rgbaMatch = raw.match(/^rgba?\((.+)\)$/i);
  if (rgbaMatch) {
    const parts = rgbaMatch[1].split(",").map((item) => item.trim());
    if (parts.length === 3 || parts.length === 4) {
      const r = clamp(num(parts[0], 0), 0, 255);
      const g = clamp(num(parts[1], 0), 0, 255);
      const b = clamp(num(parts[2], 0), 0, 255);
      const a = parts.length === 4 ? clamp(num(parts[3], 1), 0, 1) : 1;
      const hex = `#${Math.round(r).toString(16).padStart(2, "0")}${Math.round(g).toString(16).padStart(2, "0")}${Math.round(b).toString(16).padStart(2, "0")}`;
      return { hex, alpha: a };
    }
  }

  const hex = raw.startsWith("#") ? raw.slice(1) : raw;
  if (/^[0-9a-fA-F]{8}$/.test(hex)) {
    return {
      hex: `#${hex.slice(0, 6).toLowerCase()}`,
      alpha: parseInt(hex.slice(6, 8), 16) / 255,
    };
  }
  if (/^[0-9a-fA-F]{6}$/.test(hex)) {
    return { hex: `#${hex.toLowerCase()}`, alpha: 1 };
  }
  if (/^[0-9a-fA-F]{4}$/.test(hex)) {
    const expanded = hex
      .split("")
      .map((ch) => ch + ch)
      .join("")
      .toLowerCase();
    return {
      hex: `#${expanded.slice(0, 6)}`,
      alpha: parseInt(expanded.slice(6, 8), 16) / 255,
    };
  }
  if (/^[0-9a-fA-F]{3}$/.test(hex)) {
    const expanded = hex
      .split("")
      .map((ch) => ch + ch)
      .join("")
      .toLowerCase();
    return { hex: `#${expanded}`, alpha: 1 };
  }
  return { hex: "#000000", alpha: 1 };
}

function colorWithAlpha(hexColor, alphaValue) {
  const parsed = parseColorValue(hexColor);
  const alpha = clamp(num(alphaValue, 1), 0, 1);
  if (alpha >= 0.999) {
    return parsed.hex;
  }
  const alphaHex = Math.round(alpha * 255)
    .toString(16)
    .padStart(2, "0");
  return `${parsed.hex}${alphaHex}`;
}

const PICKER_SWATCHES = [
  "rgba(0, 0, 0, 0)",
  "#ffffff",
  "#000000",
  "#ef4444",
  "#f97316",
  "#eab308",
  "#22c55e",
  "#06b6d4",
  "#3b82f6",
  "#8b5cf6",
  "#ec4899",
  "#475569",
];

function destroyColorPickers() {
  if (!Array.isArray(state.ui.colorPickers)) {
    state.ui.colorPickers = [];
    return;
  }
  for (const picker of state.ui.colorPickers) {
    try {
      picker.destroyAndRemove();
    } catch (_) {
      // ignore
    }
  }
  state.ui.colorPickers = [];
}

function initColorPickers() {
  destroyColorPickers();

  const rows = document.querySelectorAll(".color-picker-row");
  rows.forEach((row) => {
    const trigger = row.querySelector(".color-picker-trigger");
    const colorInput = row.querySelector(".native-color-input");
    const alphaInput = row.querySelector(".native-alpha-input");
    if (!trigger || !colorInput || !alphaInput) return;

    const apply = (hex, alpha, emit) => {
      colorInput.value = hex;
      alphaInput.value = clamp(num(alpha, 1), 0, 1).toFixed(2);
      trigger.style.backgroundColor = colorWithAlpha(hex, alphaInput.value);
      if (emit) {
        applyColorFromPicker(colorInput, alphaInput);
      }
    };

    apply(colorInput.value || "#000000", alphaInput.value || "0", false);

    const picker = Pickr.create({
      el: trigger,
      theme: "nano",
      container: document.body,
      default: colorWithAlpha(colorInput.value || "#000000", alphaInput.value || "0"),
      swatches: PICKER_SWATCHES,
      components: {
        preview: true,
        opacity: true,
        hue: true,
        interaction: {
          hex: true,
          rgba: true,
          input: true,
          clear: true,
          save: false,
        },
      },
    });

    picker
      .on("change", (color) => {
        if (!color) return;
        const [r, g, b, a] = color.toRGBA();
        const hex = `#${Math.round(r).toString(16).padStart(2, "0")}${Math.round(g).toString(16).padStart(2, "0")}${Math.round(b).toString(16).padStart(2, "0")}`;
        apply(hex, a, true);
      })
      .on("clear", () => {
        apply("#000000", 0, true);
      });

    state.ui.colorPickers.push(picker);
  });
}

function escapeColor(value) {
  return parseColorValue(value).hex;
}

function escapeAlpha(value) {
  return parseColorValue(value).alpha.toFixed(2);
}

function getPreviewZoomPercent() {
  const value = clamp(Math.round(num(state.ui.previewZoom, 100)), 25, 400);
  state.ui.previewZoom = value;
  return value;
}

function getSnapSize() {
  const value = clamp(Math.round(num(state.ui.snapSize, 10)), 2, 100);
  state.ui.snapSize = value;
  return value;
}

function snapToGrid(value, offset = 0) {
  if (!state.ui.snapEnabled) return Math.round(value);
  const snapSize = getSnapSize();
  const base = num(offset, 0);
  return Math.round((value - base) / snapSize) * snapSize + base;
}

function calcPreviewFitScale(wrapper, cfg) {
  if (!wrapper || !cfg) return 1;
  const width = Math.max(1, wrapper.clientWidth - 24);
  const height = Math.max(1, wrapper.clientHeight - 24);
  const fit = Math.min(width / Math.max(cfg.width, 1), height / Math.max(cfg.height, 1));
  if (!Number.isFinite(fit) || fit <= 0) {
    return 1;
  }
  return fit;
}

function updatePreviewFitScaleFromDOM(forceRender = false) {
  const wrapper = document.getElementById("preview-wrapper");
  if (!wrapper || !state.config) return;
  state.previewWrapperSize = {
    width: Math.round(num(wrapper.clientWidth, 0)),
    height: Math.round(num(wrapper.clientHeight, 0)),
  };
  state.ui.previewFitScale = calcPreviewFitScale(wrapper, state.config);
  if (forceRender) {
    updatePreviewDOM();
  }
}

function ensurePreviewResizeObserver() {
  const wrapper = document.getElementById("preview-wrapper");
  if (!wrapper) {
    if (state.previewResizeObserver) {
      try {
        state.previewResizeObserver.disconnect();
      } catch (_) {
        // ignore
      }
    }
    state.previewWrapperSize = { width: 0, height: 0 };
    return;
  }

  if (!state.previewResizeObserver && typeof ResizeObserver !== "undefined") {
    state.previewResizeObserver = new ResizeObserver((entries) => {
      const entry = entries?.[0];
      if (!entry || !state.config) return;
      const rect = entry.contentRect || {};
      const width = Math.round(num(rect.width, 0));
      const height = Math.round(num(rect.height, 0));
      if (width === state.previewWrapperSize.width && height === state.previewWrapperSize.height) {
        return;
      }
      state.previewWrapperSize = { width, height };
      updatePreviewFitScaleFromDOM(true);
    });
  }

  if (state.previewResizeObserver) {
    try {
      state.previewResizeObserver.disconnect();
    } catch (_) {
      // ignore
    }
    state.previewResizeObserver.observe(wrapper);
  }
  state.previewWrapperSize = {
    width: Math.round(num(wrapper.clientWidth, 0)),
    height: Math.round(num(wrapper.clientHeight, 0)),
  };
  updatePreviewFitScaleFromDOM(false);
}

function updatePreviewDOM() {
  const canvas = document.getElementById("preview-canvas");
  const wrapper = document.getElementById("preview-wrapper");
  if (!canvas || !wrapper || !state.config) return;

  const cfg = state.config;
  const autoZoom = state.ui.previewZoomAuto !== false;
  const zoom = getPreviewZoomPercent() / 100;
  const snapSize = getSnapSize();
  let scale = zoom;
  if (autoZoom) {
    // Keep scale stable during drag to avoid jitter caused by transient overlay/layout changes.
    if (state.drag && state.previewScale > 0) {
      scale = state.previewScale;
    } else {
      const fit = num(state.ui.previewFitScale, 0);
      if (fit > 0) {
        scale = fit;
      } else {
        scale = calcPreviewFitScale(wrapper, cfg);
        state.ui.previewFitScale = scale;
      }
    }
  }
  scale = Math.max(0.05, scale);
  state.previewScale = scale;
  const displayWidth = Math.max(1, cfg.width * scale);
  const displayHeight = Math.max(1, cfg.height * scale);
  const dpr = Math.max(1, window.devicePixelRatio || 1);
  const cssWidth = `${displayWidth}px`;
  const cssHeight = `${displayHeight}px`;
  if (canvas.style.width !== cssWidth) {
    canvas.style.width = cssWidth;
  }
  if (canvas.style.height !== cssHeight) {
    canvas.style.height = cssHeight;
  }
  const nextWidth = Math.max(1, Math.round(displayWidth * dpr));
  const nextHeight = Math.max(1, Math.round(displayHeight * dpr));
  if (canvas.width !== nextWidth) {
    canvas.width = nextWidth;
  }
  if (canvas.height !== nextHeight) {
    canvas.height = nextHeight;
  }

  const zoomLabel = document.getElementById("preview-zoom-value");
  if (zoomLabel) {
    const zoomText = `${Math.round((autoZoom ? scale : zoom) * 100)}%`;
    zoomLabel.textContent = autoZoom ? `自动 ${zoomText}` : zoomText;
  }
  const snapInput = document.querySelector("[data-ui-field='snapSize']");
  if (snapInput && String(snapInput.value) !== String(snapSize)) {
    snapInput.value = String(snapSize);
  }
  const zoomInput = document.querySelector("[data-ui-field='previewZoom']");
  if (zoomInput && String(zoomInput.value) !== String(state.ui.previewZoom)) {
    zoomInput.value = String(state.ui.previewZoom);
  }
  if (zoomInput) {
    zoomInput.disabled = autoZoom;
  }

  const ctx = canvas.getContext("2d");
  if (!ctx) return;
  ctx.save();
  ctx.setTransform(dpr * scale, 0, 0, dpr * scale, 0, 0);
  ctx.clearRect(0, 0, cfg.width, cfg.height);
  ctx.fillStyle = colorToCanvas(cfg.default_background || "#0b1220", "rgba(11, 18, 32, 1)");
  ctx.fillRect(0, 0, cfg.width, cfg.height);

  if (state.previewImageBitmap) {
    try {
      ctx.drawImage(state.previewImageBitmap, 0, 0, cfg.width, cfg.height);
    } catch (_) {
      // ignore stale bitmap draw failure
    }
  }

  const gridEnabled = !!state.ui.showGrid && !state.ui.purePreview;
  if (gridEnabled && snapSize > 0) {
    const bounds = getLayoutBounds(cfg);
    const left = Math.round(bounds.left);
    const top = Math.round(bounds.top);
    const right = Math.round(bounds.right);
    const bottom = Math.round(bounds.bottom);
    ctx.save();
    ctx.strokeStyle = "rgba(148, 163, 184, 0.2)";
    ctx.lineWidth = Math.max(1 / Math.max(scale, 1), 0.5 / scale);
    ctx.beginPath();
    for (let x = left; x <= right; x += snapSize) {
      ctx.moveTo(x + 0.5, top);
      ctx.lineTo(x + 0.5, bottom);
    }
    if ((right - left) % snapSize !== 0) {
      ctx.moveTo(right + 0.5, top);
      ctx.lineTo(right + 0.5, bottom);
    }
    for (let y = top; y <= bottom; y += snapSize) {
      ctx.moveTo(left, y + 0.5);
      ctx.lineTo(right, y + 0.5);
    }
    if ((bottom - top) % snapSize !== 0) {
      ctx.moveTo(left, bottom + 0.5);
      ctx.lineTo(right, bottom + 0.5);
    }
    ctx.stroke();

    if (bounds.padding > 0) {
      ctx.strokeStyle = "rgba(59, 130, 246, 0.35)";
      ctx.lineWidth = Math.max(1.2 / Math.max(scale, 1), 0.8 / scale);
      ctx.strokeRect(left + 0.5, top + 0.5, Math.max(0, right - left), Math.max(0, bottom - top));
    }
    ctx.restore();
  }

  if (!state.ui.purePreview) {
    for (let i = 0; i < cfg.items.length; i += 1) {
      const rect = getPreviewItemRect(i);
      if (!rect) continue;
      const selected = i === state.selectedItem;

      ctx.save();
      ctx.fillStyle = selected ? "rgba(59, 130, 246, 0.14)" : "rgba(255, 255, 255, 0.05)";
      ctx.fillRect(rect.x, rect.y, rect.width, rect.height);
      ctx.lineWidth = selected ? 2 / scale : 1 / scale;
      ctx.strokeStyle = selected ? "#3b82f6" : "rgba(255, 255, 255, 0.35)";
      if (!selected) {
        ctx.setLineDash([4 / scale, 4 / scale]);
      }
      ctx.strokeRect(rect.x, rect.y, rect.width, rect.height);
      ctx.restore();

      if (selected) {
        const handle = getResizeHandleRect(rect);
        ctx.save();
        ctx.fillStyle = "#09090b";
        ctx.strokeStyle = "#3b82f6";
        ctx.lineWidth = 2 / scale;
        ctx.beginPath();
        ctx.arc(handle.cx, handle.cy, handle.radius, 0, Math.PI * 2);
        ctx.fill();
        ctx.stroke();
        ctx.restore();
      }
    }

    const guides = state.ui.centerGuides;
    if (guides?.showVertical) {
      ctx.save();
      ctx.strokeStyle = "rgba(59, 130, 246, 0.8)";
      ctx.lineWidth = 1 / scale;
      ctx.beginPath();
      ctx.moveTo(num(guides.x), 0);
      ctx.lineTo(num(guides.x), cfg.height);
      ctx.stroke();
      ctx.restore();
    }
    if (guides?.showHorizontal) {
      ctx.save();
      ctx.strokeStyle = "rgba(59, 130, 246, 0.8)";
      ctx.lineWidth = 1 / scale;
      ctx.beginPath();
      ctx.moveTo(0, num(guides.y));
      ctx.lineTo(cfg.width, num(guides.y));
      ctx.stroke();
      ctx.restore();
    }
  }

  ctx.restore();
}

function colorToCanvas(value, fallback = "rgba(0, 0, 0, 1)") {
  const parsed = parseColorValue(value);
  const hex = parsed.hex.replace("#", "");
  if (!/^[0-9a-fA-F]{6}$/.test(hex)) return fallback;
  const r = parseInt(hex.slice(0, 2), 16);
  const g = parseInt(hex.slice(2, 4), 16);
  const b = parseInt(hex.slice(4, 6), 16);
  const a = clamp(num(parsed.alpha, 1), 0, 1);
  return `rgba(${r}, ${g}, ${b}, ${a})`;
}

function getPreviewItemRect(index) {
  const item = state.config?.items?.[index];
  if (!item) return null;
  const draft = state.drag?.index === index && state.drag?.draft ? state.drag.draft : null;
  const source = draft || item;
  return {
    x: num(source.x),
    y: num(source.y),
    width: Math.max(10, num(source.width)),
    height: Math.max(10, num(source.height)),
  };
}

function getResizeHandleRect(rect) {
  const scale = Math.max(state.previewScale, 0.05);
  const size = Math.max(10 / scale, 4);
  const x = rect.x + rect.width - size / 2;
  const y = rect.y + rect.height - size / 2;
  return {
    x,
    y,
    size,
    radius: size / 2,
    cx: x + size / 2,
    cy: y + size / 2,
  };
}

function pointInRect(x, y, rect) {
  return x >= rect.x && y >= rect.y && x <= rect.x + rect.width && y <= rect.y + rect.height;
}

function getPreviewPointerPosition(event, canvas) {
  const rect = canvas.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return null;
  return {
    x: (event.clientX - rect.left) / Math.max(state.previewScale, 0.01),
    y: (event.clientY - rect.top) / Math.max(state.previewScale, 0.01),
  };
}

function findPreviewHit(point) {
  if (!point || !state.config) return null;
  const selectedRect = getPreviewItemRect(state.selectedItem);
  if (selectedRect) {
    const handle = getResizeHandleRect(selectedRect);
    if (pointInRect(point.x, point.y, { x: handle.x, y: handle.y, width: handle.size, height: handle.size })) {
      return { index: state.selectedItem, mode: "resize", rect: selectedRect };
    }
  }

  for (let i = state.config.items.length - 1; i >= 0; i -= 1) {
    const rect = getPreviewItemRect(i);
    if (!rect) continue;
    if (pointInRect(point.x, point.y, rect)) {
      return { index: i, mode: "move", rect };
    }
  }
  return null;
}

function updateCenterGuides(rect) {
  const cfg = state.config;
  const movingIndex = Number(state.drag?.index);
  if (!cfg || !rect || state.drag?.mode !== "move" || !Number.isFinite(movingIndex)) {
    state.ui.centerGuides = null;
    return;
  }

  const width = Math.max(10, num(rect.width));
  const height = Math.max(10, num(rect.height));
  const centerX = num(rect.x) + width / 2;
  const centerY = num(rect.y) + height / 2;
  const threshold = 4;
  let nearestX = null;
  let nearestY = null;
  let nearestXDelta = Infinity;
  let nearestYDelta = Infinity;

  for (let i = 0; i < cfg.items.length; i += 1) {
    if (i === movingIndex) continue;
    const target = cfg.items[i];
    if (!target) continue;

    const targetCenterX = num(target.x) + Math.max(10, num(target.width)) / 2;
    const targetCenterY = num(target.y) + Math.max(10, num(target.height)) / 2;
    const dx = Math.abs(centerX - targetCenterX);
    const dy = Math.abs(centerY - targetCenterY);

    if (dx <= threshold && dx < nearestXDelta) {
      nearestXDelta = dx;
      nearestX = targetCenterX;
    }
    if (dy <= threshold && dy < nearestYDelta) {
      nearestYDelta = dy;
      nearestY = targetCenterY;
    }
  }

  const showVertical = Number.isFinite(nearestX);
  const showHorizontal = Number.isFinite(nearestY);
  if (!showVertical && !showHorizontal) {
    state.ui.centerGuides = null;
    return;
  }

  state.ui.centerGuides = {
    showVertical,
    showHorizontal,
    x: showVertical ? nearestX : 0,
    y: showHorizontal ? nearestY : 0,
  };
}

function attachPreviewDragHandlers() {
  updatePreviewDOM();
  const canvas = document.getElementById("preview-canvas");
  if (!canvas) return;

  canvas.addEventListener("pointerdown", (event) => {
    if (state.ui.purePreview || isEditingProfileReadOnly()) {
      state.ui.centerGuides = null;
      return;
    }

    const point = getPreviewPointerPosition(event, canvas);
    const hit = findPreviewHit(point);
    if (!hit) return;

    if (hit.index !== state.selectedItem) {
      state.selectedItem = hit.index;
      syncElementSelectionUI({ rerenderEditor: true });
      return;
    }

    event.preventDefault();
    canvas.setPointerCapture?.(event.pointerId);
    const rect = hit.rect || getPreviewItemRect(hit.index);
    if (!rect) return;
    state.drag = {
      mode: hit.mode === "resize" ? "resize" : "move",
      index: hit.index,
      startX: event.clientX,
      startY: event.clientY,
      originX: num(rect.x),
      originY: num(rect.y),
      originW: Math.max(10, num(rect.width)),
      originH: Math.max(10, num(rect.height)),
      draft: {
        x: num(rect.x),
        y: num(rect.y),
        width: Math.max(10, num(rect.width)),
        height: Math.max(10, num(rect.height)),
      },
      changed: false,
    };
    if (state.drag.mode === "move") {
      updateCenterGuides(state.drag.draft);
    } else {
      state.ui.centerGuides = null;
    }
    updatePreviewDOM();
    renderSelectedItemValues(state.drag.draft);
  });
}

window.addEventListener("pointermove", (event) => {
  if (state.ui.purePreview || isEditingProfileReadOnly()) {
    state.drag = null;
    state.ui.centerGuides = null;
    return;
  }
  if (!state.drag || !state.config) return;

  const dx = (event.clientX - state.drag.startX) / Math.max(state.previewScale, 0.01);
  const dy = (event.clientY - state.drag.startY) / Math.max(state.previewScale, 0.01);

  if (!state.drag.draft) {
    return;
  }

  if (state.drag.mode === "resize") {
    const bounds = getLayoutBounds(state.config);
    const maxWidth = Math.max(10, Math.round(bounds.right - num(state.drag.draft.x, state.drag.originX)));
    const maxHeight = Math.max(10, Math.round(bounds.bottom - num(state.drag.draft.y, state.drag.originY)));
    state.drag.draft.width = clamp(Math.max(10, snapToGrid(state.drag.originW + dx)), 10, maxWidth);
    state.drag.draft.height = clamp(Math.max(10, snapToGrid(state.drag.originH + dy)), 10, maxHeight);
    state.ui.centerGuides = null;
  } else {
    const bounds = getLayoutBounds(state.config);
    const maxX = Math.max(bounds.left, Math.round(bounds.right - num(state.drag.draft.width, state.drag.originW)));
    const maxY = Math.max(bounds.top, Math.round(bounds.bottom - num(state.drag.draft.height, state.drag.originH)));
    state.drag.draft.x = clamp(
      snapToGrid(state.drag.originX + dx, bounds.left),
      bounds.left,
      maxX,
    );
    state.drag.draft.y = clamp(
      snapToGrid(state.drag.originY + dy, bounds.top),
      bounds.top,
      maxY,
    );
    updateCenterGuides(state.drag.draft);
  }

  state.drag.changed =
    state.drag.draft.x !== state.drag.originX ||
    state.drag.draft.y !== state.drag.originY ||
    state.drag.draft.width !== state.drag.originW ||
    state.drag.draft.height !== state.drag.originH;

  updatePreviewDOM();
  renderSelectedItemValues(state.drag.draft);
});

window.addEventListener("pointerup", () => {
  const drag = state.drag;
  state.drag = null;
  state.ui.centerGuides = null;
  if (drag?.changed && state.config?.items?.[drag.index]) {
    const item = state.config.items[drag.index];
    const clamped = clampItemRectToBounds({
      x: num(drag.draft?.x, num(item.x)),
      y: num(drag.draft?.y, num(item.y)),
      width: Math.max(10, num(drag.draft?.width, num(item.width))),
      height: Math.max(10, num(drag.draft?.height, num(item.height))),
    });
    item.x = clamped.x;
    item.y = clamped.y;
    item.width = clamped.width;
    item.height = clamped.height;
    markDirty();
  }
  renderSelectedItemValues();
  updatePreviewDOM();
  if (state.previewRefreshPending) {
    state.previewRefreshPending = false;
    if (isRuntimeStreamReady()) {
      void runtimeStreamCall("request_runtime", {}, 3000).catch(() => {});
    } else {
      void refreshPreviewImage();
    }
  }
});

function renderSelectedItemValues(overrideRect = null) {
  const item = overrideRect || state.config?.items?.[state.selectedItem];
  if (!item) return;
  const xInput = document.querySelector("[data-item-field='x']");
  const yInput = document.querySelector("[data-item-field='y']");
  const wInput = document.querySelector("[data-item-field='width']");
  const hInput = document.querySelector("[data-item-field='height']");
  if (xInput) xInput.value = item.x;
  if (yInput) yInput.value = item.y;
  if (wInput) wInput.value = item.width;
  if (hInput) hInput.value = item.height;
}

async function decodePreviewBitmap(blob) {
  if (typeof createImageBitmap === "function") {
    return createImageBitmap(blob);
  }
  return new Promise((resolve, reject) => {
    const image = new Image();
    const objectURL = URL.createObjectURL(blob);
    image.onload = () => {
      URL.revokeObjectURL(objectURL);
      resolve(image);
    };
    image.onerror = (err) => {
      URL.revokeObjectURL(objectURL);
      reject(err);
    };
    image.src = objectURL;
  });
}

async function refreshPreviewImage() {
  if (state.drag) {
    state.previewRefreshPending = true;
    return;
  }
  if (state.runtimeStream && state.runtimeStream.readyState === WebSocket.CONNECTING) {
    return;
  }
  if (isRuntimeStreamReady()) {
    try {
      await runtimeStreamCall("request_runtime", {}, 3000);
      return;
    } catch (_) {
      // fallback to HTTP preview fetch below
    }
  }
  try {
    const res = await fetch(`/api/preview?t=${Date.now()}`, { cache: "no-store" });
    if (res.status === 204) {
      return;
    }
    if (!res.ok) {
      const text = await res.text();
      throw new Error(`${res.status} ${text || res.statusText}`);
    }

    const blob = await res.blob();
    const bitmap = await decodePreviewBitmap(blob);
    if (state.previewImageBitmap && typeof state.previewImageBitmap.close === "function") {
      state.previewImageBitmap.close();
    }
    state.previewImageBitmap = bitmap;
    state.previewError = "";

    updatePreviewDOM();
    const errorEl = document.getElementById("preview-render-error");
    if (errorEl) {
      errorEl.textContent = "";
    }
  } catch (error) {
    state.previewError = `预览获取失败: ${error.message}`;
    const errorEl = document.getElementById("preview-render-error");
    if (errorEl) {
      errorEl.textContent = state.previewError;
    }
  }
}

function updateRuntimePanelDOM() {
  const mode = getRuntimeModeText();
  const updated = formatTimestamp(state.snapshot?.updated_at);
  const count = Object.keys(state.snapshot?.values || {}).length;

  const modeEl = document.getElementById("runtime-mode");
  if (modeEl) modeEl.textContent = mode;

  const updatedEl = document.getElementById("runtime-updated");
  if (updatedEl) updatedEl.textContent = updated;

  const countEl = document.getElementById("runtime-count");
  if (countEl) countEl.textContent = String(count);

  const runtimeMonitorEl = document.getElementById("runtime-monitor-stats");
  if (runtimeMonitorEl) runtimeMonitorEl.innerHTML = renderRuntimeMonitorStats();

  const collectorEl = document.getElementById("collector-panel");
  if (collectorEl) collectorEl.innerHTML = renderCollectorRows();

  const list = document.getElementById("runtime-values");
  if (list) list.innerHTML = renderRuntimeValueRows();
}

function schedulePreviewFetch(delay = 0) {
  if (state.previewFetchTimer) {
    window.clearTimeout(state.previewFetchTimer);
    state.previewFetchTimer = null;
  }
  state.previewFetchTimer = window.setTimeout(() => {
    state.previewFetchTimer = null;
    void refreshPreviewImage();
  }, Math.max(0, delay));
}

function applySnapshotUpdate(snapshotRes, options = {}) {
  const fromWS = !!options.fromWS;
  state.snapshot = snapshotRes;
  const labels = {};
  for (const [name, item] of Object.entries(snapshotRes?.values || {})) {
    const label = String(item?.label || "").trim();
    labels[name] = label || name;
  }
  const labelChanged = labelsChanged(labels);
  state.monitorLabels = labels;

  const changed = updateMonitorOptions();
  if (changed || labelChanged) {
    render();
  } else {
    updateRuntimePanelDOM();
    updatePreviewDOM();
  }

  if (!fromWS && !state.runtimeStream) {
    if (state.drag) {
      state.previewRefreshPending = true;
    } else {
      schedulePreviewFetch(0);
    }
  }
}

async function pollRuntime() {
  if (state.polling) return;
  state.polling = true;

  try {
    const snapshotRes = await api("/api/snapshot");
    applySnapshotUpdate(snapshotRes);
  } catch (error) {
    setError(`轮询失败: ${error.message}`);
  } finally {
    state.polling = false;
  }
}

function startRuntimeStream() {
  connectRuntimeStream();
}

function stopRuntimeStream() {
  closeRuntimeStream();
}

function startPolling() {
  if (isRuntimeStreamReady()) {
    return;
  }
  if (state.pollTimer) {
    window.clearInterval(state.pollTimer);
  }
  state.pollTimer = window.setInterval(() => {
    void pollRuntime();
  }, 2000);
}

function stopPolling() {
  if (state.pollTimer) {
    window.clearInterval(state.pollTimer);
    state.pollTimer = null;
  }
}

async function init() {
  try {
    const [metaRes, configRes, profilesRes, collectorsRes] = await Promise.all([
      api("/api/meta"),
      api("/api/config"),
      api("/api/profiles").catch(() => ({ items: [], active: "default" })),
      api("/api/collectors").catch(() => ({ items: [] })),
    ]);

    state.meta = metaRes;
    applyProfilesPayload(profilesRes);
    state.collectors = normalizeCollectors(collectorsRes);
    state.editingProfile = state.meta?.active_profile || state.profiles?.[0]?.name || "default";
    state.config = ensureConfig(configRes.config);
    refreshSelectedItem();
    updateMonitorOptions();
    clearDirty();
    setError("");
  } catch (error) {
    state.error = error.message;
  }

  render();
  startRuntimeStream();
  window.setTimeout(() => {
    if (!isRuntimeStreamReady()) {
      void pollRuntime();
    }
  }, 1200);
}

function updateDefaultLevelArray(kind, index, value) {
  if (!state.config) return;
  if (kind === "default-level-color") {
    state.config.level_colors[index] = value;
  } else if (kind === "default-threshold") {
    state.config.default_thresholds[index] = num(value, 0);
  }
}

function updateItemLevelArray(kind, index, value) {
  const item = state.config?.items?.[state.selectedItem];
  if (!item) return;
  if (kind === "threshold") {
    item.thresholds ??= [25, 50, 75, 100];
    item.thresholds[index] = num(value, 0);
  } else if (kind === "color") {
    item.level_colors ??= ["#22c55e", "#eab308", "#f97316", "#ef4444"];
    item.level_colors[index] = value;
  }
}

function applyColorFromPicker(colorInput, alphaInput) {
  if (!state.config || !colorInput || !alphaInput) return;
  if (isEditingProfileReadOnly()) {
    setError("内置只读配置，请先复制");
    render();
    return;
  }

  const value = colorWithAlpha(colorInput.value || "#000000", alphaInput.value || "1");

  if (colorInput.dataset.field) {
    const field = colorInput.dataset.field;
    state.config[field] = value;
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (colorInput.dataset.itemField) {
    const item = state.config.items[state.selectedItem];
    if (!item) return;
    item[colorInput.dataset.itemField] = value;
    ensureItemName(item, state.selectedItem);
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (colorInput.dataset.levelKind) {
    const levelKind = colorInput.dataset.levelKind;
    const levelIndex = Number(colorInput.dataset.levelIndex);
    if (Number.isFinite(levelIndex)) {
      updateDefaultLevelArray(levelKind, levelIndex, value);
      markDirty();
    }
    return;
  }

  if (colorInput.dataset.itemLevelKind) {
    const itemLevelKind = colorInput.dataset.itemLevelKind;
    const itemLevelIndex = Number(colorInput.dataset.itemLevelIndex);
    if (itemLevelKind === "color" && Number.isFinite(itemLevelIndex)) {
      updateItemLevelArray(itemLevelKind, itemLevelIndex, value);
      markDirty();
    }
  }
}

app.addEventListener("input", (event) => {
  if (!state.config) return;
  const target = event.target;

  if (target.dataset.uiField) {
    const field = target.dataset.uiField;
    if (field === "purePreview" || field === "showGrid" || field === "snapEnabled") {
      state.ui[field] = !!target.checked;
      if (field === "purePreview" && state.ui.purePreview) {
        state.drag = null;
        state.ui.centerGuides = null;
      }
    } else if (field === "autoApply") {
      state.ui.autoApply = !!target.checked;
      if (state.ui.autoApply) {
        scheduleAutoApply(0);
      } else {
        clearAutoApplyTimer();
      }
    } else if (field === "autoApplyDelay") {
      state.ui.autoApplyDelay = normalizeAutoApplyDelay(target.value);
      if (state.ui.autoApply && state.dirty) {
        scheduleAutoApply();
      }
    } else if (field === "previewZoomAuto") {
      state.ui.previewZoomAuto = !!target.checked;
      if (!state.ui.previewZoomAuto) {
        state.ui.previewZoom = 100;
      }
    } else if (field === "previewZoom") {
      state.ui.previewZoom = clamp(Math.round(num(target.value, state.ui.previewZoom || 100)), 25, 400);
    } else if (field === "snapSize") {
      state.ui.snapSize = clamp(Math.round(num(target.value, state.ui.snapSize || 10)), 2, 100);
    } else if (field === "addItemType") {
      state.addItemType = target.value;
    } else if (field === "addItemMonitor") {
      state.addItemMonitor = target.value;
    }
    updatePreviewDOM();
    if (field === "addItemType") {
      render();
    }
    return;
  }

  if (isEditingProfileReadOnly()) {
    setError("内置只读配置，请先复制");
    render();
    return;
  }

  if (target.dataset.collectorName) {
    const collectorName = String(target.dataset.collectorName || "").trim();
    const kind = String(target.dataset.collectorKind || "").trim();
    if (!collectorName || !kind) return;
    const entry = ensureCollectorConfigEntry(state.config, collectorName);
    if (kind === "enabled") {
      entry.enabled = !!target.checked;
    } else if (kind === "option") {
      const optionKey = String(target.dataset.collectorOption || "").trim();
      if (!optionKey) return;
      entry.options[optionKey] = String(target.value ?? "");
    } else {
      return;
    }
    syncLegacyCollectorFields(state.config);
    markDirty();
    return;
  }

  if (target.dataset.levelKind) {
    const levelKind = target.dataset.levelKind;
    const levelIndex = Number(target.dataset.levelIndex);
    if (levelKind === "default-level-color") {
      const alphaInput = document.querySelector(`[data-level-alpha-kind="${levelKind}"][data-level-index="${levelIndex}"]`);
      const colorValue = colorWithAlpha(target.value, alphaInput?.value ?? "1");
      updateDefaultLevelArray(levelKind, levelIndex, colorValue);
    } else {
      updateDefaultLevelArray(levelKind, levelIndex, target.value);
    }
    markDirty();
    return;
  }

  if (target.dataset.levelAlphaKind) {
    const levelKind = target.dataset.levelAlphaKind;
    const levelIndex = Number(target.dataset.levelIndex);
    const colorInput = document.querySelector(`[data-level-kind="${levelKind}"][data-level-index="${levelIndex}"]`);
    const colorValue = colorWithAlpha(colorInput?.value || "#000000", target.value);
    updateDefaultLevelArray(levelKind, levelIndex, colorValue);
    markDirty();
    return;
  }

  if (target.dataset.itemLevelKind) {
    const itemLevelKind = target.dataset.itemLevelKind;
    const itemLevelIndex = Number(target.dataset.itemLevelIndex);
    if (itemLevelKind === "color") {
      const alphaInput = document.querySelector(`[data-item-level-alpha-kind="${itemLevelKind}"][data-item-level-index="${itemLevelIndex}"]`);
      const colorValue = colorWithAlpha(target.value, alphaInput?.value ?? "1");
      updateItemLevelArray(itemLevelKind, itemLevelIndex, colorValue);
    } else {
      updateItemLevelArray(itemLevelKind, itemLevelIndex, target.value);
    }
    markDirty();
    return;
  }

  if (target.dataset.itemLevelAlphaKind) {
    const itemLevelKind = target.dataset.itemLevelAlphaKind;
    const itemLevelIndex = Number(target.dataset.itemLevelIndex);
    const colorInput = document.querySelector(`[data-item-level-kind="${itemLevelKind}"][data-item-level-index="${itemLevelIndex}"]`);
    const colorValue = colorWithAlpha(colorInput?.value || "#000000", target.value);
    updateItemLevelArray(itemLevelKind, itemLevelIndex, colorValue);
    markDirty();
    return;
  }

  if (target.dataset.fieldAlpha) {
    const field = target.dataset.fieldAlpha;
    const colorInput = document.querySelector(`[data-field="${field}"]`);
    state.config[field] = colorWithAlpha(colorInput?.value || "#000000", target.value);
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (target.dataset.field) {
    const field = target.dataset.field;
    if (target.dataset.kind === "number") {
      state.config[field] = num(target.value, 0);
    } else if (target.dataset.kind === "checkbox") {
      state.config[field] = !!target.checked;
    } else if (target.dataset.kind === "multi-select") {
      state.config[field] = Array.from(target.selectedOptions).map((option) => option.value);
      if (field === "output_types") {
        state.config.output_types = normalizeOutputTypes(state.config.output_types);
      }
    } else if (target.dataset.kind === "color") {
      const alphaInput = document.querySelector(`[data-field-alpha="${field}"]`);
      state.config[field] = colorWithAlpha(target.value, alphaInput?.value ?? "1");
    } else {
      state.config[field] = target.value;
    }
    if (field === "width" || field === "height") {
      state.config[field] = Math.max(10, state.config[field]);
      state.config.layout_padding = normalizeLayoutPadding(
        state.config.layout_padding,
        state.config.width,
        state.config.height,
      );
      updatePreviewFitScaleFromDOM(false);
    }
    if (field === "layout_padding") {
      state.config.layout_padding = normalizeLayoutPadding(
        state.config.layout_padding,
        state.config.width,
        state.config.height,
      );
      updatePreviewFitScaleFromDOM(false);
    }
    if (field === "width" || field === "height" || field === "layout_padding") {
      clampAllItemsToBounds();
      renderSelectedItemValues();
    }
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (target.dataset.customIndex !== undefined) {
    const index = Number(target.dataset.customIndex);
    const field = target.dataset.customField;
    const item = state.config.custom_monitors[index];
    if (!item) return;

    const optionalNumFields = new Set(["precision", "min", "max", "scale", "offset"]);
    if (field === "sources" && target.tagName === "SELECT") {
      item.sources = Array.from(target.selectedOptions).map((option) => option.value);
    } else if (optionalNumFields.has(field)) {
      item[field] = maybeNumber(target.value);
    } else {
      item[field] = target.value;
    }

    if (field === "type") {
      if (item.type === "mixed") {
        item.sources ??= [];
        item.aggregate ??= "max";
      }
      if (item.type === "coolercontrol") {
        item.source ??= "";
      }
      if (item.type === "librehardwaremonitor") {
        item.source ??= "";
      }
      markDirty();
      render();
      return;
    }

    markDirty();
    return;
  }

  if (target.dataset.itemAlphaField) {
    const item = state.config.items[state.selectedItem];
    if (!item) return;
    const field = target.dataset.itemAlphaField;
    const colorInput = document.querySelector(`[data-item-field="${field}"][data-item-kind="color"]`);
    item[field] = colorWithAlpha(colorInput?.value || "#000000", target.value);
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (target.dataset.itemAttrField) {
    const item = state.config.items[state.selectedItem];
    if (!item) return;
    const attrs = ensureItemRenderAttrs(item);
    const key = String(target.dataset.itemAttrField || "").trim();
    if (!key) return;
    const kind = target.dataset.itemAttrKind || "text";
    if (kind === "number") {
      if (target.value === "") {
        delete attrs[key];
      } else {
        attrs[key] = Number(target.value);
      }
    } else if (kind === "bool") {
      attrs[key] = target.value === "true";
    } else {
      attrs[key] = target.value;
    }
    markDirty();
    updatePreviewDOM();
    return;
  }

  if (target.dataset.itemField) {
    const item = state.config.items[state.selectedItem];
    if (!item) return;

    const field = target.dataset.itemField;
    const kind = target.dataset.itemKind || "text";

    if (field === "threshold_mode") {
      if (target.value === "default") {
        delete item.thresholds;
      } else {
        item.thresholds = ensureFourNumbers(item.thresholds, state.config.default_thresholds);
      }
      markDirty();
      render();
      return;
    }

    if (field === "level_color_mode") {
      if (target.value === "default") {
        delete item.level_colors;
      } else {
        item.level_colors = ensureFourColors(item.level_colors, state.config.level_colors);
      }
      markDirty();
      render();
      return;
    }

    if (field === "type") {
      const nextType = normalizeItemType(target.value);
      const nextSeed = createItemByType(nextType);
      if (MONITOR_ELEMENT_TYPES.has(nextType) && item.monitor) {
        nextSeed.monitor = item.monitor;
      }
      if (MONITOR_ELEMENT_TYPES.has(nextType) && item.unit) {
        nextSeed.unit = item.unit;
      }
      if (isFullElementType(nextType) && isFullElementType(item.type)) {
        nextSeed.render_attrs_map = {
          ...nextSeed.render_attrs_map,
          ...normalizeRenderAttrsMap(item.render_attrs_map),
        };
      }
      const next = normalizeItem(
        {
          ...nextSeed,
          x: item.x,
          y: item.y,
          width: item.width,
          height: item.height,
          edit_ui_name: item.edit_ui_name,
        },
        state.selectedItem,
        state.config?.history_size || 150,
      );
      state.config.items[state.selectedItem] = next;
      ensureItemName(next, state.selectedItem);
      markDirty();
      render();
      return;
    }

    if (field === "point_size") {
      item.point_size = Math.max(10, num(target.value, state.config?.history_size || 150));
      item.history = item.type === "simple_line_chart";
      ensureItemName(item, state.selectedItem);
      markDirty();
      updatePreviewDOM();
      return;
    }

    if (kind === "nullable-number") {
      if (target.value === "") {
        delete item[field];
      } else {
        item[field] = Number(target.value);
      }
    } else if (kind === "bool") {
      item[field] = target.value === "true";
    } else if (kind === "color") {
      const alphaInput = document.querySelector(`[data-item-alpha-field="${field}"]`);
      item[field] = colorWithAlpha(target.value, alphaInput?.value ?? "1");
    } else {
      item[field] = target.value;
    }

    if (field === "x" || field === "y" || field === "width" || field === "height") {
      const rect = clampItemRectToBounds({
        x: num(item.x, 0),
        y: num(item.y, 0),
        width: Math.max(10, num(item.width, 10)),
        height: Math.max(10, num(item.height, 10)),
      });
      item.x = rect.x;
      item.y = rect.y;
      item.width = rect.width;
      item.height = rect.height;
      renderSelectedItemValues(item);
    }

    ensureItemName(item, state.selectedItem);
    markDirty();
    updatePreviewDOM();
  }
});

app.addEventListener("change", async (event) => {
  const target = event.target;

  if (target.id === "profile-select") {
    const name = String(target.value || "").trim();
    if (!name || name === state.editingProfile) return;
    if (!confirmDiscardUnsavedChanges()) {
      target.value = state.editingProfile;
      return;
    }

    try {
      await flushAutoSave();
      const res = await api("/api/profiles/switch", {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      state.editingProfile = name;
      state.meta = { ...(state.meta || {}), active_profile: res.active || name };
      applyProfilesPayload(res);
      state.config = ensureConfig(res.config || {});
      refreshSelectedItem();
      updateMonitorOptions();
      clearDirty();
      setError("");
      render();
      void syncPreviewConfig();
    } catch (error) {
      setError(`切换编辑配置失败: ${error.message}`);
      render();
    }
    return;
  }

  if (isEditingProfileReadOnly()) {
    setError("内置只读配置，请先复制");
    render();
    return;
  }

  if (target.id === "import-file-input") {
    const file = target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const parsed = JSON.parse(text);
      state.config = ensureConfig(parsed);
      refreshSelectedItem();
      updateMonitorOptions();
      markDirty();
      setError("");
      render();
    } catch (error) {
      setError(`导入失败: ${error.message}`);
    }
    target.value = "";
    return;
  }

  if (target.dataset.customField === "sources") {
    const index = Number(target.dataset.customIndex);
    const item = state.config.custom_monitors[index];
    if (!item) return;
    item.sources = Array.from(target.selectedOptions).map((option) => option.value);
    markDirty();
    return;
  }

  if (target.dataset.field === "output_types" && target.dataset.kind === "multi-select") {
    state.config.output_types = normalizeOutputTypes(Array.from(target.selectedOptions).map((option) => option.value));
    markDirty();
    return;
  }

});

app.addEventListener("click", async (event) => {
  const target = event.target.closest("button");
  if (!target) return;
  const action = target.dataset.action;
  if (!action) return;

  if (action === "switch-tab") {
    const tab = target.dataset.tab;
    if (["basic", "elements", "custom", "collection"].includes(tab)) {
      state.activeTab = tab;
      render();
      if (tab === "collection") {
        void refreshCollectors().then(() => updateRuntimePanelDOM());
      }
    }
    return;
  }

  if (action === "refresh-snapshot") {
    if (isRuntimeStreamReady()) {
      await runtimeStreamCall("request_runtime", {}, 3000).catch(() => {});
    } else {
      await pollRuntime();
    }
    return;
  }

  if (action === "preview-zoom-reset") {
    state.ui.previewZoomAuto = false;
    state.ui.previewZoom = 100;
    updatePreviewDOM();
    const autoToggle = document.querySelector("[data-ui-field='previewZoomAuto']");
    if (autoToggle) {
      autoToggle.checked = false;
    }
    return;
  }

  if (action === "save-profile") {
    if (isEditingProfileReadOnly()) {
      setError("内置只读配置，请先复制");
      render();
      return;
    }
    try {
      await saveAndApplyPreview();
      setError("");
    } catch (error) {
      setError(`保存失败: ${error.message}`);
    }
    render();
    return;
  }

  const readOnlyBlockedActions = new Set([
    "import-json",
    "add-custom",
    "remove-custom",
    "add-item",
    "clone-item",
    "remove-item",
    "move-item-up",
    "move-item-down",
    "rename-profile",
    "delete-profile",
  ]);
  if (isEditingProfileReadOnly() && readOnlyBlockedActions.has(action)) {
    setError("内置只读配置，请先复制");
    return;
  }

  if (action === "export-json") {
    const data = JSON.stringify(state.config, null, 2);
    const blob = new Blob([data], { type: "application/json" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = `ax206monitor-${new Date().toISOString().replaceAll(":", "-")}.json`;
    link.click();
    URL.revokeObjectURL(link.href);
    return;
  }

  if (action === "import-json") {
    const input = document.getElementById("import-file-input");
    if (input) input.click();
    return;
  }

  if (action === "refresh-profiles") {
    try {
      const profilesRes = await api("/api/profiles");
      applyProfilesPayload(profilesRes);
      if (!(state.profiles || []).some((item) => item.name === state.editingProfile)) {
        state.editingProfile = state.meta?.active_profile || state.profiles?.[0]?.name || "default";
      }
      setError("");
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "activate-profile") {
    const name = state.editingProfile || selectedProfileName();
    if (!name) return;
    try {
      await flushAutoSave();
      const res = await api("/api/profiles/switch", {
        method: "POST",
        body: JSON.stringify({ name }),
      });
      applyProfilesPayload(res);
      state.meta = { ...(state.meta || {}), active_profile: res.active || name };
      setError("");
      clearDirty();
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "create-profile") {
    const suggested = `profile_${new Date().toISOString().slice(0, 10).replaceAll("-", "")}`;
    const name = window.prompt("输入新配置名称（a-zA-Z0-9._-）", suggested);
    if (!name) return;
    const profileName = name.trim();
    if (!profileName) return;
    try {
      const res = await api("/api/profiles", {
        method: "POST",
        body: JSON.stringify({
          name: profileName,
          config: state.config,
          switch: false,
        }),
      });
      applyProfilesPayload(res);
      state.editingProfile = profileName;
      state.config.name = profileName;
      clearDirty();
      setError("");
      void syncPreviewConfig();
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "save-profile-as") {
    const base = selectedProfileName();
    const name = window.prompt("复制为配置名称（a-zA-Z0-9._-）", `${base}_copy`);
    if (!name) return;
    const profileName = name.trim();
    if (!profileName) return;

    try {
      const res = await api("/api/profiles", {
        method: "POST",
        body: JSON.stringify({
          name: profileName,
          config: state.config,
          switch: false,
        }),
      });
      applyProfilesPayload(res);
      state.editingProfile = profileName;
      state.config.name = profileName;
      clearDirty();
      setError("");
      void syncPreviewConfig();
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "rename-profile") {
    const oldName = state.editingProfile || selectedProfileName();
    if (!oldName) return;
    const input = window.prompt("输入新的配置名称（a-zA-Z0-9._-）", oldName);
    if (!input) return;
    const newName = input.trim();
    if (!newName || newName === oldName) return;

    try {
      await flushAutoSave();
      const res = await api("/api/profiles/rename", {
        method: "POST",
        body: JSON.stringify({
          old_name: oldName,
          new_name: newName,
        }),
      });
      applyProfilesPayload(res);
      state.meta = { ...(state.meta || {}), active_profile: res.active || state.meta?.active_profile || oldName };
      if (state.editingProfile === oldName) {
        state.editingProfile = newName;
      }
      if (res.config) {
        state.config = ensureConfig(res.config);
        refreshSelectedItem();
        updateMonitorOptions();
      } else {
        state.config.name = newName;
      }
      clearDirty();
      setError("");
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "delete-profile") {
    const name = state.editingProfile || selectedProfileName();
    if (!name) return;
    if (!confirmDiscardUnsavedChanges()) return;
    if (!window.confirm(`确认删除配置 ${name} ?`)) return;

    try {
      await flushAutoSave();
      const res = await api(`/api/profiles/${encodeURIComponent(name)}`, { method: "DELETE" });
      applyProfilesPayload(res);
      state.meta = { ...(state.meta || {}), active_profile: res.active || state.meta?.active_profile || "default" };
      state.editingProfile = state.meta.active_profile || res.items?.[0]?.name || "default";
      const configRes = await api(`/api/profiles/${encodeURIComponent(state.editingProfile)}`).catch(() => api("/api/config"));
      state.config = ensureConfig(configRes.config);
      refreshSelectedItem();
      updateMonitorOptions();
      clearDirty();
      setError("");
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "add-custom") {
    state.config.custom_monitors.push(normalizeCustomMonitor({ type: "file" }));
    markDirty();
    render();
    return;
  }

  if (action === "remove-custom") {
    const index = Number(target.dataset.index);
    state.config.custom_monitors.splice(index, 1);
    markDirty();
    render();
    return;
  }

  if (action === "add-item") {
    const item = createItemByType(state.addItemType);
    const rect = clampItemRectToBounds(item);
    item.x = rect.x;
    item.y = rect.y;
    item.width = rect.width;
    item.height = rect.height;
    state.config.items.push(item);
    state.selectedItem = state.config.items.length - 1;
    ensureItemName(state.config.items[state.selectedItem], state.selectedItem);
    markDirty();
    render();
    return;
  }

  if (action === "clone-item") {
    const item = state.config.items[state.selectedItem];
    if (!item) return;
    const cloned = deepClone(item);
    cloned.x = num(cloned.x) + 10;
    cloned.y = num(cloned.y) + 10;
    const rect = clampItemRectToBounds(cloned);
    cloned.x = rect.x;
    cloned.y = rect.y;
    cloned.width = rect.width;
    cloned.height = rect.height;
    cloned.edit_ui_name = "";
    state.config.items.push(cloned);
    state.selectedItem = state.config.items.length - 1;
    ensureItemName(state.config.items[state.selectedItem], state.selectedItem);
    markDirty();
    render();
    return;
  }

  if (action === "remove-item") {
    if (state.selectedItem >= 0) {
      state.config.items.splice(state.selectedItem, 1);
      if (state.selectedItem >= state.config.items.length) {
        state.selectedItem = state.config.items.length - 1;
      }
      markDirty();
      render();
    }
    return;
  }

  if (action === "move-item-up") {
    const index = state.selectedItem;
    if (index > 0) {
      const tmp = state.config.items[index - 1];
      state.config.items[index - 1] = state.config.items[index];
      state.config.items[index] = tmp;
      state.selectedItem = index - 1;
      markDirty();
      render();
    }
    return;
  }

  if (action === "move-item-down") {
    const index = state.selectedItem;
    if (index >= 0 && index < state.config.items.length - 1) {
      const tmp = state.config.items[index + 1];
      state.config.items[index + 1] = state.config.items[index];
      state.config.items[index] = tmp;
      state.selectedItem = index + 1;
      markDirty();
      render();
    }
    return;
  }

  if (action === "select-item") {
    state.selectedItem = Number(target.dataset.index);
    syncElementSelectionUI({ rerenderEditor: true });
    return;
  }

});

window.addEventListener("keydown", (event) => {
  if (!state.config) return;
  if (isEditingProfileReadOnly()) return;
  if (state.ui.purePreview) return;
  if (event.target && ["INPUT", "TEXTAREA", "SELECT"].includes(event.target.tagName)) return;

  const item = state.config.items[state.selectedItem];
  if (!item) return;

  const snapSize = getSnapSize();
  const step = state.ui.snapEnabled ? (event.shiftKey ? snapSize * 5 : snapSize) : (event.shiftKey ? 10 : 1);
  let moved = false;

  if (event.key === "ArrowLeft") {
    item.x = num(item.x) - step;
    moved = true;
  } else if (event.key === "ArrowRight") {
    item.x = num(item.x) + step;
    moved = true;
  } else if (event.key === "ArrowUp") {
    item.y = num(item.y) - step;
    moved = true;
  } else if (event.key === "ArrowDown") {
    item.y = num(item.y) + step;
    moved = true;
  }

  if (!moved) return;
  event.preventDefault();
  const bounds = getLayoutBounds(state.config);
  if (state.ui.snapEnabled) {
    item.x = snapToGrid(item.x, bounds.left);
    item.y = snapToGrid(item.y, bounds.top);
  }
  const itemWidth = Math.max(10, Math.round(num(item.width, 10)));
  const itemHeight = Math.max(10, Math.round(num(item.height, 10)));
  const maxX = Math.max(bounds.left, Math.round(bounds.right - itemWidth));
  const maxY = Math.max(bounds.top, Math.round(bounds.bottom - itemHeight));
  item.x = clamp(Math.round(num(item.x, bounds.left)), bounds.left, maxX);
  item.y = clamp(Math.round(num(item.y, bounds.top)), bounds.top, maxY);

  markDirty();
  updatePreviewDOM();
  renderSelectedItemValues();
});

window.addEventListener("beforeunload", () => {
  destroyColorPickers();
  stopPolling();
  stopRuntimeStream();
  clearAutoApplyTimer();
  if (state.previewFetchTimer) {
    window.clearTimeout(state.previewFetchTimer);
    state.previewFetchTimer = null;
  }
  if (state.previewImageBitmap && typeof state.previewImageBitmap.close === "function") {
    state.previewImageBitmap.close();
  }
  state.previewImageBitmap = null;
  if (state.previewResizeObserver) {
    try {
      state.previewResizeObserver.disconnect();
    } catch (_) {
      // ignore
    }
    state.previewResizeObserver = null;
  }
});

window.addEventListener("resize", () => {
  updatePreviewFitScaleFromDOM(true);
});

void init();

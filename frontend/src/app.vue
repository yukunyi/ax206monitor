<script setup>
import { darkTheme } from "naive-ui";
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import TopBar from "./components/top_bar.vue";
import BasicTab from "./components/basic_tab.vue";
import ElementsTab from "./components/elements_tab.vue";
import RuntimeTab from "./components/runtime_tab.vue";

const state = reactive({
  loading: true,
  error: "",
  meta: null,
  config: null,
  profiles: [],
  collectors: [],
  snapshot: null,
  activeTab: "elements",
  editingProfile: "",
  selectedIndex: -1,
  dirty: false,
  saving: false,
  previewUrl: "",
  previewSync: true,
  zoomAuto: true,
  zoom: 100,
});

const runtime = reactive({
  ws: null,
  ready: false,
  reconnectTimer: null,
  reqSeq: 1,
  pending: {},
});

const profileDialog = reactive({
  show: false,
  mode: "create",
  value: "",
  submitting: false,
  error: "",
});

const DEFAULT_OUTPUT_TYPES = ["memimg", "ax206usb"];
const DEFAULT_ITEM_TYPES = [
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  "simple_label",
  "simple_rect",
  "simple_circle",
  "label_text1",
  "label_text2",
  "full_chart",
  "full_progress",
];

const DEFAULT_COLLECTOR_ENABLED = {
  "go_native.cpu": true,
  "go_native.memory": true,
  "go_native.system": true,
  "go_native.disk": true,
  "go_native.network": true,
  "custom.all": true,
  coolercontrol: false,
  librehardwaremonitor: false,
  rtss: false,
};

const MONITOR_REQUIRED_TYPES = new Set([
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  "label_text1",
  "label_text2",
  "full_chart",
  "full_progress",
]);

const ITEM_STYLE_FIELDS = new Set([
  "font_size",
  "color",
  "bg",
  "unit_color",
  "unit_font_size",
  "border_width",
  "border_color",
  "radius",
  "point_size",
  "thresholds",
  "level_colors",
]);

const STYLE_RENDER_ATTR_FIELDS = new Set([
  "content_padding",
  "value_font_size",
  "label_font_size",
  "meta_font_size",
  "title_font_size",
  "header_divider",
  "header_divider_offset",
  "header_divider_color",
  "body_gap",
  "history_points",
  "show_segment_lines",
  "show_grid_lines",
  "grid_lines",
  "fill_area",
  "line_width",
  "show_avg_line",
  "chart_color",
  "chart_area_bg",
  "chart_area_border_color",
  "progress_style",
  "bar_height",
  "track_color",
  "segments",
  "segment_gap",
  "card_radius",
  "gauge_thickness",
  "ring_thickness",
  "main_font_size",
  "ticks",
  "cells",
  "cell_gap",
]);

const ITEM_TYPE_LABELS = {
  simple_value: "基础数值",
  simple_progress: "基础进度条",
  simple_line_chart: "基础折线图",
  simple_label: "基础标签",
  simple_rect: "基础矩形",
  simple_circle: "基础圆形",
  label_text1: "标签数值-左右",
  label_text2: "标签数值-强调",
  full_chart: "复杂图表",
  full_progress: "复杂进度条",
};

const DEFAULT_TYPE_RENDER_ATTRS = {
  simple_line_chart: {
    history_points: 150,
  },
  label_text1: {
    content_padding: 3,
  },
  label_text2: {
    content_padding: 5,
  },
  full_chart: {
    content_padding: 1,
    body_gap: 4,
    title_font_size: 14,
    value_font_size: 16,
    header_divider: true,
    header_divider_offset: 3,
    header_divider_color: "#94a3b840",
    history_points: 150,
    show_segment_lines: true,
    show_grid_lines: true,
    grid_lines: 4,
    fill_area: true,
    line_width: 2,
    show_avg_line: true,
    chart_color: "#38bdf8",
    chart_area_bg: "",
    chart_area_border_color: "",
  },
  full_progress: {
    content_padding: 1,
    body_gap: 0,
    title_font_size: 14,
    value_font_size: 16,
    header_divider: true,
    header_divider_offset: 3,
    header_divider_color: "#94a3b840",
    progress_style: "gradient",
    bar_height: 0,
    track_color: "#1f2937",
    segments: 12,
    segment_gap: 2,
  },
};

const ALIAS_LABELS = {
  "alias.cpu.usage": "CPU usage",
  "alias.cpu.temp": "CPU temperature",
  "alias.cpu.freq": "CPU frequency",
  "alias.cpu.max_freq": "CPU max frequency",
  "alias.cpu.power": "CPU power",
  "alias.memory.usage": "Memory usage",
  "alias.memory.used": "Memory used",
  "alias.gpu.fps": "GPU FPS",
  "alias.gpu.usage": "GPU usage",
  "alias.gpu.power": "GPU power",
  "alias.gpu.vram": "GPU VRAM usage",
  "alias.gpu.temp": "GPU temperature",
  "alias.gpu.fan": "GPU fan speed",
  "alias.gpu.freq": "GPU frequency",
  "alias.gpu.max_freq": "GPU max frequency",
  "alias.net.upload": "Network upload",
  "alias.net.download": "Network download",
  "alias.net.ip": "IP address",
  "alias.net.interface": "Network interface",
  "alias.system.time": "System time",
  "alias.system.hostname": "Host name",
  "alias.system.load": "System load",
  "alias.system.resolution": "Display resolution",
  "alias.system.refresh_rate": "Display refresh rate",
  "alias.disk.temp": "Disk temperature",
  "alias.fan.cpu": "CPU fan speed",
  "alias.fan.gpu": "GPU fan speed",
  "alias.fan.system": "System fan speed",
};

const PROFILE_NAME_RE = /^[A-Za-z0-9._-]+$/;

const theme_overrides = {
  common: {
    primaryColor: "#3b82f6",
    primaryColorHover: "#60a5fa",
    primaryColorPressed: "#2563eb",
    primaryColorSuppl: "#3b82f6",
    bodyColor: "#0b0f17",
    cardColor: "#111827",
    popoverColor: "#111827",
    modalColor: "#111827",
    tableColor: "#0f172a",
    actionColor: "#1f2937",
    borderColor: "#334155",
    dividerColor: "#334155",
    textColorBase: "#e5e7eb",
    textColor1: "#f8fafc",
    textColor2: "#e2e8f0",
    textColor3: "#94a3b8",
    placeholderColor: "#94a3b8",
    inputColor: "#0f172a",
    inputColorDisabled: "#1f2937",
    closeIconColor: "#94a3b8",
    closeColorHover: "#334155",
  },
  Card: {
    color: "#111827",
    textColor: "#f1f5f9",
    titleTextColor: "#f8fafc",
    borderColor: "#334155",
  },
  Input: {
    color: "#0f172a",
    colorFocus: "#0f172a",
    colorDisabled: "#1f2937",
    textColor: "#f8fafc",
    border: "#334155",
    borderHover: "#60a5fa",
    borderFocus: "#3b82f6",
    placeholderColor: "#94a3b8",
  },
  Button: {
    textColorPrimary: "#f8fafc",
    textColorHoverPrimary: "#f8fafc",
    textColorPressedPrimary: "#e2e8f0",
    textColorFocusPrimary: "#f8fafc",
    borderPrimary: "#2563eb",
    borderHoverPrimary: "#3b82f6",
    borderPressedPrimary: "#1d4ed8",
    colorPrimary: "#2563eb",
    colorHoverPrimary: "#3b82f6",
    colorPressedPrimary: "#1d4ed8",
  },
  Select: {
    peers: {
      InternalSelection: {
        color: "#0f172a",
        textColor: "#f8fafc",
        border: "#334155",
        borderHover: "#60a5fa",
        borderFocus: "#3b82f6",
      },
    },
  },
  DataTable: {
    thColor: "#111827",
    tdColor: "#0f172a",
    borderColor: "#334155",
    thTextColor: "#cbd5e1",
    tdTextColor: "#f8fafc",
    borderRadius: 6,
  },
  Table: {
    tdColor: "#0f172a",
    thColor: "#111827",
    borderColor: "#334155",
  },
};

const readonlyProfile = computed(() => {
  const current = state.profiles.find((item) => item.name === state.editingProfile);
  return !!current?.readonly;
});

const activeProfile = computed(() => String(state.meta?.active_profile || ""));
const monitorCatalog = ref([]);
const monitorCatalogSet = new Set();
const monitorLabelMap = reactive({});

const visibleTabs = computed(() => {
  if (readonlyProfile.value) {
    return [
      { key: "elements", label: "屏幕元素" },
      { key: "collection", label: "采集运行态" },
    ];
  }
  return [
    { key: "basic", label: "基础配置" },
    { key: "elements", label: "屏幕元素" },
    { key: "collection", label: "采集运行态" },
  ];
});

const monitorOptions = computed(() =>
  monitorCatalog.value.map((name) => {
    const label = String(monitorLabelMap[name] || monitorAliasLabel(name) || "").trim();
    if (!label || label === name) return { label: name, value: name };
    return { label: `${label} (${name})`, value: name };
  }),
);

const currentItem = computed(() => {
  if (!state.config?.items?.length) return null;
  if (state.selectedIndex < 0 || state.selectedIndex >= state.config.items.length) return null;
  return state.config.items[state.selectedIndex];
});

function deepClone(obj) {
  return JSON.parse(JSON.stringify(obj));
}

function normalizeThresholds(raw) {
  const source = Array.isArray(raw) ? raw : [];
  const fallback = [25, 50, 75, 100];
  const result = [];
  for (let i = 0; i < 4; i += 1) {
    const n = Number(source[i]);
    result.push(Number.isFinite(n) ? n : fallback[i]);
  }
  return result;
}

function normalizeLevelColors(raw) {
  const source = Array.isArray(raw) ? raw : [];
  const fallback = ["#22c55e", "#eab308", "#f97316", "#ef4444"];
  const result = [];
  for (let i = 0; i < 4; i += 1) {
    const text = String(source[i] || "").trim();
    result.push(text || fallback[i]);
  }
  return result;
}

function defaultTypeStyle(type, config) {
  const kind = String(type || "").trim();
  const isShape = kind === "simple_rect" || kind === "simple_circle";
  const isFull = kind.startsWith("full_");
  const isHistory = kind === "simple_line_chart";
  return {
    font_size: 0,
    small_font_size: 0,
    medium_font_size: 0,
    large_font_size: 0,
    color: String(config.default_color || "#f8fafc"),
    bg: isShape ? "#33415566" : isFull ? "#111827c8" : "",
    unit_color: String(config.default_color || "#f8fafc"),
    unit_font_size: 0,
    point_size: isHistory ? Math.max(10, Number(config.default_history_points || 150)) : 0,
    border_color: "#475569",
    border_width: 0,
    radius: 0,
    render_attrs_map: {},
  };
}

function normalizeTypeDefaults(raw, config) {
  const source = raw && typeof raw === "object" ? raw : {};
  const result = {};
  DEFAULT_ITEM_TYPES.forEach((type) => {
    const base = defaultTypeStyle(type, config);
    const input = source[type] && typeof source[type] === "object" ? source[type] : {};
    const attrsInput =
      input.render_attrs_map && typeof input.render_attrs_map === "object" ? input.render_attrs_map : {};
    const attrsBase = {
      ...(DEFAULT_TYPE_RENDER_ATTRS[type] || {}),
    };
    if (type === "full_chart") {
      attrsBase.title_font_size = Number(config.default_label_font_size || attrsBase.title_font_size || 14);
      attrsBase.value_font_size = Number(config.default_value_font_size || attrsBase.value_font_size || 16);
      attrsBase.history_points = Math.max(10, Number(config.default_history_points || attrsBase.history_points || 150));
    }
    if (type === "simple_line_chart") {
      const rawHistory =
        attrsInput.history_points ??
        input.point_size ??
        config.default_history_points ??
        attrsBase.history_points ??
        150;
      attrsBase.history_points = Math.max(10, Number(rawHistory || 0));
    }
    if (type === "full_progress") {
      attrsBase.title_font_size = Number(config.default_label_font_size || attrsBase.title_font_size || 14);
      attrsBase.value_font_size = Number(config.default_value_font_size || attrsBase.value_font_size || 16);
    }
    const smallFontSize = Number(input.small_font_size ?? input.unit_font_size ?? base.small_font_size ?? 0);
    const mediumFontSize = Number(input.medium_font_size ?? input.font_size ?? base.medium_font_size ?? 0);
    const largeFontSize = Number(input.large_font_size ?? input.font_size ?? base.large_font_size ?? 0);
    result[type] = {
      font_size: 0,
      small_font_size: Math.max(0, Number.isFinite(smallFontSize) ? smallFontSize : 0),
      medium_font_size: Math.max(0, Number.isFinite(mediumFontSize) ? mediumFontSize : 0),
      large_font_size: Math.max(0, Number.isFinite(largeFontSize) ? largeFontSize : 0),
      color: String(input.color ?? base.color ?? ""),
      bg: String(input.bg ?? base.bg ?? ""),
      unit_color: String(input.unit_color ?? base.unit_color ?? ""),
      unit_font_size: 0,
      point_size: Math.max(0, Number(input.point_size ?? base.point_size ?? 0)),
      border_color: String(input.border_color ?? base.border_color ?? ""),
      border_width: Math.max(0, Number(input.border_width ?? base.border_width ?? 0)),
      radius: Math.max(0, Number(input.radius ?? base.radius ?? 0)),
      render_attrs_map: { ...attrsBase, ...attrsInput },
    };
  });
  return result;
}

function normalizeMonitorName(raw) {
  const name = String(raw || "").trim();
  if (!name || name === "-") return "";
  return name;
}

function monitorAliasLabel(raw) {
  const name = String(raw || "").trim();
  if (!name) return "";
  if (ALIAS_LABELS[name]) return ALIAS_LABELS[name];
  if (!name.startsWith("alias.")) return "";
  const text = name
    .slice(6)
    .split(".")
    .filter(Boolean)
    .map((part) => {
      if (part === "cpu" || part === "gpu" || part === "ip" || part === "fps" || part === "vram") {
        return part.toUpperCase();
      }
      return part.charAt(0).toUpperCase() + part.slice(1);
    })
    .join(" ");
  return text || "Alias";
}

function mergeMonitorNames(names) {
  if (!Array.isArray(names) || names.length === 0) return;
  let changed = false;
  names.forEach((raw) => {
    const name = normalizeMonitorName(raw);
    if (!name || monitorCatalogSet.has(name)) return;
    monitorCatalogSet.add(name);
    const aliasLabel = monitorAliasLabel(name);
    if (aliasLabel && !monitorLabelMap[name]) {
      monitorLabelMap[name] = aliasLabel;
    }
    changed = true;
  });
  if (!changed) return;
  monitorCatalog.value = [...monitorCatalogSet].sort();
}

function mergeSnapshotMonitors(snapshot) {
  if (!snapshot || typeof snapshot !== "object") return;
  mergeMonitorNames(snapshot.monitors || []);
  mergeMonitorNames(Object.keys(snapshot.values || {}));
  Object.entries(snapshot.values || {}).forEach(([name, item]) => {
    const monitor = normalizeMonitorName(name);
    if (!monitor) return;
    const label = String(item?.label || monitorAliasLabel(monitor) || "").trim();
    if (!label) return;
    monitorLabelMap[monitor] = label;
  });
}

function mergeConfigMonitors(config) {
  if (!config || typeof config !== "object") return;
  mergeMonitorNames((config.items || []).map((item) => item?.monitor));
  mergeMonitorNames((config.custom_monitors || []).map((item) => item?.name));
}

function ensureCollectorEntry(config, collectorName) {
  if (!config.collector_config || typeof config.collector_config !== "object") {
    config.collector_config = {};
  }
  if (!config.collector_config[collectorName]) {
    config.collector_config[collectorName] = {
      enabled: !!DEFAULT_COLLECTOR_ENABLED[collectorName],
      options: {},
    };
  }
  if (!config.collector_config[collectorName].options) {
    config.collector_config[collectorName].options = {};
  }
}

function normalizeConfig(cfg) {
  const config = deepClone(cfg || {});
  config.name = String(config.name || "web");
  config.width = Math.max(10, Number(config.width || 480));
  config.height = Math.max(10, Number(config.height || 320));
  config.layout_padding = Math.max(0, Number(config.layout_padding || 0));
  config.refresh_interval = Math.max(100, Number(config.refresh_interval || 1000));
  config.collect_warn_ms = Math.max(10, Number(config.collect_warn_ms || 100));
  config.render_wait_max_ms = Math.max(0, Number(config.render_wait_max_ms || 300));
  config.history_size = Math.max(10, Number(config.history_size || 180));
  config.default_history_points = Math.max(10, Number(config.default_history_points || 150));
  config.monitor_auto_tune = config.monitor_auto_tune !== false;
  config.monitor_auto_tune_interval_sec = Math.max(1, Number(config.monitor_auto_tune_interval_sec || 5));
  config.monitor_auto_tune_slow_rate = Math.max(1, Number(config.monitor_auto_tune_slow_rate || 1.2));
  config.monitor_auto_tune_stable_runs = Math.max(1, Number(config.monitor_auto_tune_stable_runs || 3));
  config.monitor_auto_tune_max_scale = Math.max(1, Number(config.monitor_auto_tune_max_scale || 8));
  config.default_font = String(config.default_font || "");
  config.default_color = String(config.default_color || "#f8fafc");
  config.default_background = String(config.default_background || "#0b1220");
  config.default_thresholds = normalizeThresholds(config.default_thresholds);
  config.level_colors = normalizeLevelColors(config.level_colors);
  config.allow_custom_style = config.allow_custom_style === true;
  config.default_font_size = Number(config.default_font_size || 14);
  config.default_value_font_size = Number(config.default_value_font_size || 16);
  config.default_label_font_size = Number(config.default_label_font_size || 14);
  config.default_unit_font_size = Number(config.default_unit_font_size || 12);
  config.font_families = Array.isArray(config.font_families) ? config.font_families : [];
  config.output_types = Array.isArray(config.output_types) ? config.output_types : ["memimg"];
  config.type_defaults = normalizeTypeDefaults(config.type_defaults, config);
  config.items = Array.isArray(config.items) ? config.items : [];
  config.items = config.items.map((item) => {
    const next = { ...(item || {}) };
    next.custom_style = config.allow_custom_style ? next.custom_style === true : false;
    return next;
  });
  config.custom_monitors = Array.isArray(config.custom_monitors) ? config.custom_monitors : [];
  config.collector_config = config.collector_config || {};
  Object.keys(DEFAULT_COLLECTOR_ENABLED).forEach((name) => ensureCollectorEntry(config, name));
  return config;
}

function monitorToDefaultName(raw) {
  const monitor = String(raw || "").trim();
  if (!monitor) return "";
  return monitor;
}

function defaultTypeName(type) {
  const key = String(type || "").trim();
  return ITEM_TYPE_LABELS[key] || key || "element";
}

function createDefaultItem(type = "simple_value", monitor = "") {
  const selectedMonitor = String(monitor || "").trim();
  const defaultMonitor = selectedMonitor || String(monitorOptions.value[0]?.value || "");
  const displayName = monitorToDefaultName(defaultMonitor) || defaultTypeName(type);
  return {
    type,
    edit_ui_name: displayName,
    custom_style: false,
    monitor: MONITOR_REQUIRED_TYPES.has(type) ? defaultMonitor : "",
    x: 10,
    y: 10,
    width: 140,
    height: 36,
    font_size: 0,
    color: "",
    bg: "",
    unit: "auto",
    unit_color: "",
    unit_font_size: 0,
    border_width: 0,
    radius: 0,
    point_size: 0,
  };
}

function setDirty() {
  state.dirty = true;
  schedulePreviewSync();
}

function setError(err) {
  state.error = err ? String(err) : "";
}

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const payload = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(payload?.error || `HTTP ${res.status}`);
  }
  return payload;
}

function wsURL() {
  const proto = window.location.protocol === "https:" ? "wss" : "ws";
  return `${proto}://${window.location.host}/api/ws`;
}

function clearPending(reason = "ws closed") {
  Object.keys(runtime.pending).forEach((id) => {
    const task = runtime.pending[id];
    if (task?.timer) window.clearTimeout(task.timer);
    if (task?.reject) task.reject(new Error(reason));
  });
  runtime.pending = {};
}

function connectWS() {
  if (runtime.ws && (runtime.ws.readyState === WebSocket.OPEN || runtime.ws.readyState === WebSocket.CONNECTING)) {
    return;
  }
  if (runtime.reconnectTimer) {
    window.clearTimeout(runtime.reconnectTimer);
    runtime.reconnectTimer = null;
  }

  const ws = new WebSocket(wsURL());
  runtime.ws = ws;

  ws.onopen = () => {
    runtime.ready = true;
    requestRuntime().catch(() => {});
  };

  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data);
      if (message.type === "runtime") {
        if (message.snapshot && state.activeTab === "collection") {
          state.snapshot = message.snapshot;
        } else if (message.snapshot && !state.snapshot) {
          state.snapshot = message.snapshot;
        }
        if (message.preview_png) {
          state.previewUrl = `data:image/png;base64,${message.preview_png}`;
        }
        return;
      }
      if (message.type === "response") {
        const pending = runtime.pending[message.id];
        if (!pending) return;
        if (pending.timer) window.clearTimeout(pending.timer);
        delete runtime.pending[message.id];
        if (message.ok) pending.resolve(message.result);
        else pending.reject(new Error(message.error || "request failed"));
      }
    } catch (_) {
      // ignore malformed ws messages
    }
  };

  ws.onclose = () => {
    runtime.ready = false;
    clearPending("ws closed");
    runtime.reconnectTimer = window.setTimeout(connectWS, 1000);
  };

  ws.onerror = () => {
    runtime.ready = false;
  };
}

function runtimeCall(type, payload = {}, timeout = 6000) {
  return new Promise((resolve, reject) => {
    if (!runtime.ready || !runtime.ws || runtime.ws.readyState !== WebSocket.OPEN) {
      reject(new Error("websocket not connected"));
      return;
    }
    const id = `${Date.now()}_${runtime.reqSeq++}`;
    const timer = window.setTimeout(() => {
      delete runtime.pending[id];
      reject(new Error(`runtime call timeout: ${type}`));
    }, timeout);
    runtime.pending[id] = { resolve, reject, timer };
    runtime.ws.send(JSON.stringify({ type, id, ...payload }));
  });
}

async function requestRuntime() {
  if (!runtime.ready) return;
  await runtimeCall("request_runtime", {}, 4000);
}

async function refreshMonitorCatalog() {
  try {
    if (runtime.ready) {
      await requestRuntime();
      mergeSnapshotMonitors(state.snapshot);
    }
  } catch (_) {
    // ignore websocket request failures and fallback to HTTP snapshot
  }

  if (monitorCatalog.value.length > 0) return;

  try {
    const snapshot = await api("/api/snapshot");
    if (snapshot && typeof snapshot === "object") {
      state.snapshot = snapshot;
      mergeSnapshotMonitors(snapshot);
    }
  } catch (_) {
    // ignore
  }

  if (monitorCatalog.value.length > 0) return;

  try {
    const meta = await api("/api/meta");
    mergeMonitorNames(meta?.monitors || []);
  } catch (_) {
    // ignore
  }
}

let previewSyncTimer = null;

function schedulePreviewSync() {
  if (!state.previewSync || readonlyProfile.value || !state.config) return;
  if (previewSyncTimer) {
    window.clearTimeout(previewSyncTimer);
    previewSyncTimer = null;
  }
  previewSyncTimer = window.setTimeout(() => {
    previewSyncTimer = null;
    syncPreview(true).catch(() => {});
  }, 220);
}

function setPreviewSync(value) {
  state.previewSync = !!value;
  if (state.previewSync) {
    syncPreview(true).catch(() => {});
  }
}

async function syncPreview(silent = false) {
  if (!state.config) return;
  try {
    await runtimeCall("preview_config", { config: deepClone(state.config) }, 6000);
    await requestRuntime();
    if (!silent) setError("");
  } catch (err) {
    if (!silent) setError(`预览同步失败: ${err.message}`);
  }
}

async function loadInitial() {
  state.loading = true;
  try {
    const [metaRes, configRes, profilesRes, collectorsRes, snapshotRes] = await Promise.all([
      api("/api/meta"),
      api("/api/config"),
      api("/api/profiles").catch(() => ({ active: "", items: [] })),
      api("/api/collectors").catch(() => ({ items: [] })),
      api("/api/snapshot").catch(() => null),
    ]);

    state.meta = metaRes;
    state.config = normalizeConfig(configRes.config);
    state.profiles = profilesRes.items || [];
    state.collectors = collectorsRes.items || [];
    if (snapshotRes && typeof snapshotRes === "object") {
      state.snapshot = snapshotRes;
    }
    mergeMonitorNames(metaRes?.monitors || []);
    mergeSnapshotMonitors(snapshotRes);
    mergeConfigMonitors(state.config);
    state.editingProfile =
      profilesRes.active || metaRes.active_profile || state.profiles[0]?.name || "default";
    state.selectedIndex = state.config.items.length > 0 ? 0 : -1;
    state.dirty = false;
    if (monitorCatalog.value.length === 0) {
      await refreshMonitorCatalog();
    }
    setError("");
  } catch (err) {
    setError(err.message);
  } finally {
    state.loading = false;
  }
}

function patchByPath(target, path, value) {
  if (!target) return;
  const parts = Array.isArray(path) ? path : [path];
  let cur = target;
  for (let i = 0; i < parts.length - 1; i += 1) {
    const key = parts[i];
    if (!cur[key] || typeof cur[key] !== "object") cur[key] = {};
    cur = cur[key];
  }
  cur[parts[parts.length - 1]] = value;
  setDirty();
}

function onBasicChange({ path, value }) {
  if (!state.config || readonlyProfile.value) return;
  patchByPath(state.config, path, value);
  if ((Array.isArray(path) ? path.join(".") : String(path)) === "allow_custom_style" && !value) {
    state.config.items = (state.config.items || []).map((item) => ({
      ...item,
      custom_style: false,
    }));
  }
}

function onItemFieldChange({ field, value }) {
  if (!state.config || readonlyProfile.value || !currentItem.value) return;
  if (ITEM_STYLE_FIELDS.has(field) && !(state.config.allow_custom_style && currentItem.value.custom_style === true)) {
    return;
  }
  if (field === "custom_style" && !state.config.allow_custom_style) {
    currentItem.value.custom_style = false;
    setDirty();
    return;
  }
  currentItem.value[field] = value;
  if (
    field === "type" &&
    MONITOR_REQUIRED_TYPES.has(String(value || "")) &&
    !String(currentItem.value.monitor || "").trim()
  ) {
    currentItem.value.monitor = String(monitorOptions.value[0]?.value || "");
  }
  if (field === "monitor") mergeMonitorNames([value]);
  setDirty();
}

function onItemPatch({ index, patch }) {
  if (!state.config || readonlyProfile.value || !state.config.items[index]) return;
  const item = state.config.items[index];
  if (!patch || typeof patch !== "object") return;
  const nextPatch = { ...patch };
  if (!(state.config.allow_custom_style && item.custom_style === true)) {
    ITEM_STYLE_FIELDS.forEach((key) => {
      if (key in nextPatch) delete nextPatch[key];
    });
    const attrs = nextPatch.render_attrs_map;
    if (attrs && typeof attrs === "object") {
      const filtered = {};
      Object.entries(attrs).forEach(([key, val]) => {
        if (!STYLE_RENDER_ATTR_FIELDS.has(String(key))) {
          filtered[key] = val;
        }
      });
      if (Object.keys(filtered).length > 0) {
        nextPatch.render_attrs_map = filtered;
      } else {
        delete nextPatch.render_attrs_map;
      }
    }
  }
  Object.assign(item, nextPatch);
  setDirty();
}

function addItem(payload = "simple_value") {
  if (!state.config || readonlyProfile.value) return;
  const inputType =
    typeof payload === "string" ? payload : String(payload?.type || "simple_value");
  const inputMonitor =
    typeof payload === "string" ? "" : String(payload?.monitor || "");
  state.config.items.push(createDefaultItem(String(inputType || "simple_value"), inputMonitor));
  mergeConfigMonitors(state.config);
  state.selectedIndex = state.config.items.length - 1;
  setDirty();
}

function cloneItem() {
  if (!state.config || readonlyProfile.value || !currentItem.value) return;
  const item = deepClone(currentItem.value);
  item.x = Number(item.x || 0) + 8;
  item.y = Number(item.y || 0) + 8;
  state.config.items.push(item);
  state.selectedIndex = state.config.items.length - 1;
  setDirty();
}

function removeItem() {
  if (!state.config || readonlyProfile.value || state.selectedIndex < 0) return;
  state.config.items.splice(state.selectedIndex, 1);
  if (state.selectedIndex >= state.config.items.length) {
    state.selectedIndex = state.config.items.length - 1;
  }
  setDirty();
}

function moveItem(step) {
  if (!state.config || readonlyProfile.value || state.selectedIndex < 0) return;
  const from = state.selectedIndex;
  const to = from + step;
  if (to < 0 || to >= state.config.items.length) return;
  const tmp = state.config.items[to];
  state.config.items[to] = state.config.items[from];
  state.config.items[from] = tmp;
  state.selectedIndex = to;
  setDirty();
}

function addCustom() {
  if (!state.config || readonlyProfile.value) return;
  state.config.custom_monitors.push({
    name: "",
    label: "",
    type: "file",
    unit: "",
    path: "",
    source: "",
    sources: [],
    aggregate: "max",
  });
  setDirty();
}

function removeCustom(index) {
  if (!state.config || readonlyProfile.value) return;
  state.config.custom_monitors.splice(index, 1);
  setDirty();
}

function changeCustom({ index, field, value }) {
  if (!state.config || readonlyProfile.value || !state.config.custom_monitors[index]) return;
  state.config.custom_monitors[index][field] = value;
  if (field === "name") mergeMonitorNames([value]);
  setDirty();
}

async function switchProfile() {
  if (!state.editingProfile) return;
  if (state.dirty) {
    setError("当前配置有未保存改动，请先保存应用");
    return;
  }
  try {
    const res = await api("/api/profiles/switch", {
      method: "POST",
      body: JSON.stringify({ name: state.editingProfile }),
    });
    state.meta = { ...(state.meta || {}), active_profile: res.active || state.editingProfile };
    state.config = normalizeConfig(res.config);
    mergeConfigMonitors(state.config);
    state.selectedIndex = state.config.items.length > 0 ? 0 : -1;
    state.dirty = false;
    await refreshMonitorCatalog();
    await syncPreview();
    setError("");
  } catch (err) {
    setError(err.message);
  }
}

async function saveConfig() {
  if (!state.config || readonlyProfile.value) return;
  const missingMonitorIndex = state.config.items.findIndex(
    (item) =>
      MONITOR_REQUIRED_TYPES.has(String(item.type || "")) &&
      !String(item.monitor || "").trim(),
  );
  if (missingMonitorIndex >= 0) {
    setError(`第 ${missingMonitorIndex + 1} 个元素缺少监控项，请先选择监控项后再保存`);
    return;
  }
  state.saving = true;
  try {
    const payload = { config: deepClone(state.config) };
    const res = await api("/api/config", {
      method: "PUT",
      body: JSON.stringify(payload),
    });
    if (res?.meta?.active_profile) {
      state.meta = { ...(state.meta || {}), active_profile: res.meta.active_profile };
    }
    state.dirty = false;
    await syncPreview();
    setError("");
  } catch (err) {
    setError(err.message);
  } finally {
    state.saving = false;
  }
}

async function refreshProfiles() {
  try {
    const res = await api("/api/profiles");
    state.profiles = res.items || [];
    state.meta = { ...(state.meta || {}), active_profile: res.active || state.meta?.active_profile };
    if (!state.editingProfile) state.editingProfile = res.active || state.profiles[0]?.name || "default";
  } catch (err) {
    setError(err.message);
  }
}

async function createProfile() {
  if (!state.config || profileDialog.submitting) return;
  const name = String(profileDialog.value || "").trim();
  if (!name) {
    profileDialog.error = "请输入配置名称";
    return;
  }
  if (!PROFILE_NAME_RE.test(name)) {
    profileDialog.error = "名称仅允许 a-zA-Z0-9._-";
    return;
  }
  profileDialog.error = "";
  profileDialog.submitting = true;
  try {
    const res = await api("/api/profiles", {
      method: "POST",
      body: JSON.stringify({
        name,
        config: deepClone(state.config),
        switch: false,
      }),
    });
    state.profiles = res.items || state.profiles;
    state.editingProfile = name;
    profileDialog.show = false;
    setError("");
  } catch (err) {
    profileDialog.error = err.message;
  } finally {
    profileDialog.submitting = false;
  }
}

async function renameProfile() {
  if (readonlyProfile.value || profileDialog.submitting) return;
  const oldName = state.editingProfile;
  if (!oldName) return;
  const newName = String(profileDialog.value || "").trim();
  if (!newName) {
    profileDialog.error = "请输入新的配置名称";
    return;
  }
  if (!PROFILE_NAME_RE.test(newName)) {
    profileDialog.error = "名称仅允许 a-zA-Z0-9._-";
    return;
  }
  if (!newName || newName === oldName) return;
  profileDialog.error = "";
  profileDialog.submitting = true;
  try {
    const res = await api("/api/profiles/rename", {
      method: "POST",
      body: JSON.stringify({ old_name: oldName, new_name: newName }),
    });
    state.profiles = res.items || state.profiles;
    if (res.active) state.meta = { ...(state.meta || {}), active_profile: res.active };
    state.editingProfile = newName;
    if (res.config) {
      state.config = normalizeConfig(res.config);
      mergeConfigMonitors(state.config);
      await refreshMonitorCatalog();
    }
    state.dirty = false;
    profileDialog.show = false;
    setError("");
  } catch (err) {
    profileDialog.error = err.message;
  } finally {
    profileDialog.submitting = false;
  }
}

async function deleteProfile() {
  if (readonlyProfile.value) return;
  const name = state.editingProfile;
  if (!name) return;
  try {
    const res = await api(`/api/profiles/${encodeURIComponent(name)}`, { method: "DELETE" });
    state.profiles = res.items || [];
    state.meta = { ...(state.meta || {}), active_profile: res.active || "" };
    state.editingProfile = state.meta.active_profile || state.profiles[0]?.name || "default";
    const loaded = await api(`/api/profiles/${encodeURIComponent(state.editingProfile)}`).catch(() =>
      api("/api/config"),
    );
    state.config = normalizeConfig(loaded.config);
    mergeConfigMonitors(state.config);
    state.selectedIndex = state.config.items.length > 0 ? 0 : -1;
    state.dirty = false;
    await refreshMonitorCatalog();
    await syncPreview();
    setError("");
  } catch (err) {
    setError(err.message);
  }
}

function openCreateProfile() {
  if (!state.config) return;
  profileDialog.mode = "create";
  profileDialog.value = "new_profile";
  profileDialog.error = "";
  profileDialog.submitting = false;
  profileDialog.show = true;
}

function openRenameProfile() {
  if (!state.editingProfile || readonlyProfile.value) return;
  profileDialog.mode = "rename";
  profileDialog.value = state.editingProfile;
  profileDialog.error = "";
  profileDialog.submitting = false;
  profileDialog.show = true;
}

async function submitProfileDialog() {
  if (profileDialog.mode === "rename") {
    await renameProfile();
    return;
  }
  await createProfile();
}

function closeProfileDialog() {
  if (profileDialog.submitting) return;
  profileDialog.show = false;
  profileDialog.error = "";
}

function setEditingProfile(name) {
  state.editingProfile = String(name || "").trim();
}

function switchTab(tab) {
  state.activeTab = tab;
  if (tab === "collection") requestRuntime().catch(() => {});
  if (tab === "elements" || tab === "basic") {
    refreshMonitorCatalog().catch(() => {});
  }
}

watch(readonlyProfile, (readonly) => {
  if (readonly && state.activeTab === "basic") {
    state.activeTab = "elements";
  }
});

watch(
  () => state.activeTab,
  (tab) => {
    if (tab === "collection") requestRuntime().catch(() => {});
  },
);

onMounted(async () => {
  await loadInitial();
  connectWS();
  await requestRuntime().catch(() => {});
  if (monitorCatalog.value.length === 0) {
    await refreshMonitorCatalog().catch(() => {});
  }
});

onBeforeUnmount(() => {
  if (previewSyncTimer) {
    window.clearTimeout(previewSyncTimer);
    previewSyncTimer = null;
  }
  if (runtime.reconnectTimer) window.clearTimeout(runtime.reconnectTimer);
  clearPending("shutdown");
  if (runtime.ws) {
    runtime.ws.close();
    runtime.ws = null;
  }
});
</script>

<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="theme_overrides">
    <main class="app_root">
      <TopBar
        :profiles="state.profiles"
        :active-profile="activeProfile"
        :editing-profile="state.editingProfile"
        :readonly-profile="readonlyProfile"
        :dirty="state.dirty"
        :saving="state.saving"
        @update:editing-profile="setEditingProfile"
        @switch-profile="switchProfile"
        @save="saveConfig"
        @create-profile="openCreateProfile"
        @rename-profile="openRenameProfile"
        @delete-profile="deleteProfile"
      />

      <n-alert v-if="state.error" class="global_alert" type="error" :show-icon="false">
        {{ state.error }}
      </n-alert>

      <section class="app_body">
        <div v-if="state.loading" class="loading_wrap">
          <n-spin size="small" />
          <n-text depth="3">加载中...</n-text>
        </div>

        <template v-else-if="state.config && state.meta">
          <nav class="tab_bar">
            <n-space size="small">
              <n-button
                v-for="tab in visibleTabs"
                :key="tab.key"
                size="small"
                :type="state.activeTab === tab.key ? 'primary' : 'default'"
                :secondary="state.activeTab !== tab.key"
                @click="switchTab(tab.key)"
              >
                {{ tab.label }}
              </n-button>
            </n-space>
          </nav>

          <section class="tab_content">
            <BasicTab
              v-if="state.activeTab === 'basic' && !readonlyProfile"
              :config="state.config"
              :meta="state.meta"
              :collectors="state.collectors"
              :monitor-options="monitorOptions"
              :readonly-profile="readonlyProfile"
              @change="onBasicChange"
              @add-custom="addCustom"
              @remove-custom="removeCustom"
              @change-custom="changeCustom"
              @refresh-monitors="refreshMonitorCatalog"
            />

            <ElementsTab
              v-else-if="state.activeTab === 'elements'"
              :config="state.config"
              :meta="state.meta"
              :selected-index="state.selectedIndex"
              :readonly-profile="readonlyProfile"
              :monitor-options="monitorOptions"
              :preview-url="state.previewUrl"
              :preview-sync="state.previewSync"
              :zoom-auto="state.zoomAuto"
              :zoom="state.zoom"
              @select-item="(i) => (state.selectedIndex = i)"
              @add-item="addItem"
              @refresh-monitors="refreshMonitorCatalog"
              @change-preview-sync="setPreviewSync"
              @clone-item="cloneItem"
              @remove-item="removeItem"
              @move-item-up="moveItem(-1)"
              @move-item-down="moveItem(1)"
              @patch-item="onItemPatch"
              @change-item-field="onItemFieldChange"
              @change-zoom-auto="(v) => (state.zoomAuto = !!v)"
              @change-zoom="(v) => (state.zoom = Number(v || 100))"
            />

            <RuntimeTab v-else :snapshot="state.snapshot" />
          </section>
        </template>
      </section>

      <n-modal
        v-model:show="profileDialog.show"
        preset="card"
        :title="profileDialog.mode === 'rename' ? '重命名配置' : '新建配置'"
        style="width: 420px"
        :closable="!profileDialog.submitting"
        :mask-closable="!profileDialog.submitting"
      >
        <n-form label-placement="top" size="small">
          <n-form-item label="配置名称" :validation-status="profileDialog.error ? 'error' : undefined">
            <n-input
              v-model:value="profileDialog.value"
              placeholder="仅支持 a-zA-Z0-9._-"
              :disabled="profileDialog.submitting"
              @keyup.enter="submitProfileDialog"
            />
          </n-form-item>
          <n-text v-if="profileDialog.error" type="error" style="font-size: 12px">
            {{ profileDialog.error }}
          </n-text>
        </n-form>
        <template #footer>
          <n-space justify="end" size="small">
            <n-button size="small" :disabled="profileDialog.submitting" @click="closeProfileDialog">
              取消
            </n-button>
            <n-button
              size="small"
              type="primary"
              :loading="profileDialog.submitting"
              @click="submitProfileDialog"
            >
              确认
            </n-button>
          </n-space>
        </template>
      </n-modal>
    </main>
  </n-config-provider>
</template>

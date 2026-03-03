import "./style.css";
import Pickr from "@simonwep/pickr";
import "@simonwep/pickr/dist/themes/nano.min.css";

const app = document.querySelector("#app");

const ELEMENT_TYPES = ["value", "progress", "line_chart", "label", "rect", "circle"];
const MONITOR_ELEMENT_TYPES = new Set(["value", "progress", "line_chart"]);
const RANGE_ELEMENT_TYPES = new Set(["progress", "line_chart"]);
const LABEL_ELEMENT_TYPES = new Set(["label"]);
const SHAPE_ELEMENT_TYPES = new Set(["rect", "circle"]);

const state = {
  config: null,
  meta: null,
  profiles: [],
  editingProfile: "default",
  snapshot: null,
  monitorOptions: [],
  monitorLabels: {},
  selectedItem: -1,
  error: "",
  dirty: false,
  drag: null,
  previewScale: 1,
  previewImageURL: "",
  previewError: "",
  previewSyncTimer: null,
  previewSyncing: false,
  previewSyncPending: false,
  saveTimer: null,
  saving: false,
  savePending: false,
  savePromise: null,
  polling: false,
  pollTimer: null,
  activeTab: "basic",
  addItemType: "value",
  addItemMonitor: "",
  ui: {
    purePreview: false,
    elementListScrollTop: 0,
    centerGuides: null,
    colorPickers: [],
  },
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
  const t = String(type || "").trim();
  return ELEMENT_TYPES.includes(t) ? t : "value";
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
  const result = {
    type: normalizeItemType(item?.type),
    edit_ui_name: String(item?.edit_ui_name || "").trim(),
    monitor: String(item?.monitor || ""),
    unit: String(item?.unit || "auto"),
    unit_color: String(item?.unit_color || ""),
    x: num(item?.x, 10),
    y: num(item?.y, 10),
    width: Math.max(10, num(item?.width, 120)),
    height: Math.max(10, num(item?.height, 40)),
    text: String(item?.text || (String(item?.type || "") === "label" ? "Label" : "")),
    color: String(item?.color || ""),
    bg: String(item?.bg || ""),
    border_color: String(item?.border_color || ""),
    border_width: Math.max(0, num(item?.border_width, 0)),
    radius: Math.max(0, num(item?.radius, 0)),
    history: !!item?.history,
    point_size: Math.max(10, num(item?.point_size, defaultPointSize)),
  };

  if (item?.font_size !== undefined && item?.font_size !== null && item?.font_size !== "") {
    result.font_size = Math.max(1, num(item.font_size, 16));
  }
  if (item?.unit_font_size !== undefined && item?.unit_font_size !== null && item?.unit_font_size !== "") {
    result.unit_font_size = Math.max(1, num(item.unit_font_size, 12));
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
  if (result.type === "line_chart") {
    result.history = true;
    result.point_size = Math.max(10, num(result.point_size, defaultPointSize));
  } else {
    result.history = false;
    delete result.point_size;
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

function ensureConfig(cfg) {
  const config = cfg || {};
  config.name = String(config.name || "web");
  config.width = Math.max(10, num(config.width, 480));
  config.height = Math.max(10, num(config.height, 320));
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
  config.network_interface = String(config.network_interface || "auto");
  config.libre_hardware_monitor_url = String(config.libre_hardware_monitor_url ?? "");
  config.coolercontrol_url = String(config.coolercontrol_url ?? "");
  config.coolercontrol_username = String(config.coolercontrol_username || "");
  config.coolercontrol_password = String(config.coolercontrol_password || "");

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
  const base = {
    type: itemType,
    edit_ui_name: "",
    monitor: MONITOR_ELEMENT_TYPES.has(itemType) ? monitor : "",
    unit: MONITOR_ELEMENT_TYPES.has(itemType) ? "auto" : "",
    unit_color: "",
    x: 10,
    y: 10,
    width: itemType === "label" ? 140 : 120,
    height: itemType === "label" ? 34 : 40,
    text: itemType === "label" ? "Label" : "",
    color: "",
    bg: "",
    border_color: "",
    border_width: 0,
    radius: itemType === "rect" ? 8 : 0,
    history: itemType === "line_chart",
    point_size: itemType === "line_chart" ? defaultPointSize : undefined,
  };
  if (itemType === "circle") {
    base.width = 60;
    base.height = 60;
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
  if (label && label !== key) {
    return `${label} (${key})`;
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

function markDirty() {
  if (isEditingProfileReadOnly()) {
    setError("内置只读配置，请先复制");
    render();
    return;
  }
  state.dirty = true;
  schedulePreviewSync();
  scheduleAutoSave();
}

function clearDirty() {
  state.dirty = false;
}

function schedulePreviewSync(immediate = false) {
  if (!state.config) return;
  if (state.previewSyncTimer) {
    window.clearTimeout(state.previewSyncTimer);
  }
  state.previewSyncTimer = window.setTimeout(() => {
    state.previewSyncTimer = null;
    void syncPreviewConfig();
  }, immediate ? 0 : 140);
}

function scheduleAutoSave(immediate = false) {
  if (isEditingProfileReadOnly()) return;
  if (!state.config) return;
  if (state.saveTimer) {
    window.clearTimeout(state.saveTimer);
  }
  state.saveTimer = window.setTimeout(() => {
    state.saveTimer = null;
    void saveEditingProfile();
  }, immediate ? 0 : 300);
}

async function saveEditingProfile() {
  if (isEditingProfileReadOnly()) return;
  if (!state.config) return;
  const profileName = state.editingProfile || state.meta?.active_profile || "default";
  if (!profileName) return;

  if (state.saving) {
    state.savePending = true;
    return state.savePromise;
  }

  state.saving = true;
  const payloadConfig = ensureConfig(deepClone(state.config));
  state.savePromise = (async () => {
    try {
      const response = await api(`/api/profiles/${encodeURIComponent(profileName)}`, {
        method: "PUT",
        body: JSON.stringify({ config: payloadConfig }),
      });
      applyProfilesPayload(response);
      state.config = ensureConfig(payloadConfig);
      clearDirty();
      setError("");
    } catch (error) {
      state.dirty = true;
      setError(`自动保存失败: ${error.message}`);
    } finally {
      state.saving = false;
      state.savePromise = null;
      if (state.savePending) {
        state.savePending = false;
        scheduleAutoSave(true);
      }
    }
  })();
  return state.savePromise;
}

async function flushAutoSave() {
  if (state.saveTimer) {
    window.clearTimeout(state.saveTimer);
    state.saveTimer = null;
  }
  if (state.dirty) {
    await saveEditingProfile();
  } else if (state.saving && state.savePromise) {
    await state.savePromise;
  }
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
    await api("/api/preview/config", {
      method: "POST",
      body: JSON.stringify({ config: cfg }),
    });
    await refreshPreviewImage();
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
      schedulePreviewSync(true);
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

  app.innerHTML = `
    ${renderTopBar()}
    <div id="global-error" class="global-error">${escapeHTML(state.error || "")}</div>
    <main class="main-content">
      <div class="tab-nav">
        <button data-action="switch-tab" data-tab="basic" class="${state.activeTab === "basic" ? "active" : ""}">基础配置</button>
        <button data-action="switch-tab" data-tab="elements" class="${state.activeTab === "elements" ? "active" : ""}">屏幕元素</button>
        <button data-action="switch-tab" data-tab="custom" class="${state.activeTab === "custom" ? "active" : ""}">自定义监控项</button>
        <button data-action="switch-tab" data-tab="collection" class="${state.activeTab === "collection" ? "active" : ""}">采集信息</button>
      </div>
      <div class="tab-content">
        ${isEditingProfileReadOnly() ? `<div class="global-error">当前为内置只读配置，不能直接修改。请使用顶部“复制”创建可编辑配置。</div>` : ""}
        ${renderActiveTab(cfg)}
      </div>
    </main>
  `;

  attachPreviewDragHandlers();
  updateRuntimePanelDOM();
  initColorPickers();

  const nextList = document.getElementById("elements-list");
  if (nextList) {
    nextList.scrollTop = Math.max(0, num(state.ui.elementListScrollTop, 0));
  }
}

function renderTopBar() {
  const editing = state.editingProfile || state.meta?.active_profile || "default";
  const editingInfo = getProfileInfo(editing);
  const options = (state.profiles || [])
    .map((item) => {
      const suffix = item.readonly ? " [内置]" : "";
      return `<option value="${escapeAttr(item.name)}" ${item.name === editing ? "selected" : ""}>${escapeHTML(item.name + suffix)}</option>`;
    })
    .join("");

  return `
    <div class="topbar">
      <div class="topbar-row">
        <label>配置文件
          <select id="profile-select">${options}</select>
        </label>
        <button data-action="activate-profile">设为激活</button>
        <button data-action="create-profile">新建</button>
        <button data-action="save-profile-as">复制</button>
        <button data-action="delete-profile" ${editingInfo?.readonly ? "disabled" : ""}>删除</button>
        <button data-action="refresh-profiles">刷新</button>
        <button data-action="export-json">导出 JSON</button>
        <button data-action="import-json">导入 JSON</button>
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

function renderBasicTab(cfg) {
  const outputOptions = (state.meta?.output_types || ["memimg", "ax206usb"])
    .map((value) => `<option value="${escapeAttr(value)}" ${cfg.output_types.includes(value) ? "selected" : ""}>${escapeHTML(value)}</option>`)
    .join("");
  const networkOptions = (state.meta?.network_interfaces || ["auto"])
    .map((value) => `<option value="${escapeAttr(value)}" ${cfg.network_interface === value ? "selected" : ""}>${escapeHTML(value)}</option>`)
    .join("");
  const fontOptions = collectFontOptions(cfg)
    .map((name) => `<option value="${escapeAttr(name)}" ${cfg.default_font === name ? "selected" : ""}>${escapeHTML(name)}</option>`)
    .join("");

  return `
    <div class="layout-single">
      <div class="panel">
        <h3>基础参数</h3>
        <div class="grid">
          ${fieldText("name", "名称", cfg.name)}
          ${fieldNumber("width", "宽度", cfg.width)}
          ${fieldNumber("height", "高度", cfg.height)}
          ${fieldSelectMultiple("output_types", "输出类型(多选)", outputOptions, 2)}
          ${fieldNumber("refresh_interval", "刷新间隔(ms)", cfg.refresh_interval)}
          ${fieldNumber("history_size", "折线历史长度", cfg.history_size)}
          ${fieldSelect("network_interface", "网络接口", networkOptions)}
          ${fieldText("libre_hardware_monitor_url", "Libre URL", cfg.libre_hardware_monitor_url)}
          ${fieldText("coolercontrol_url", "CoolerControl URL", cfg.coolercontrol_url)}
          ${fieldText("coolercontrol_username", "CoolerControl 用户名", cfg.coolercontrol_username)}
          ${fieldText("coolercontrol_password", "CoolerControl 密码", cfg.coolercontrol_password)}
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
    </div>
  `;
}

function renderElementsTab(cfg) {
  const selected = cfg.items[state.selectedItem];
  const typeOptions = (state.meta?.item_types || ELEMENT_TYPES)
    .map((itemType) => `<option value="${escapeAttr(itemType)}" ${state.addItemType === itemType ? "selected" : ""}>${escapeHTML(itemType)}</option>`)
    .join("");

  return `
    <div class="elements-layout">
      <div class="elements-left">
        <div class="panel preview-panel">${renderPreview(cfg)}</div>
        <div class="panel list-panel">${renderElementList(cfg, selected, typeOptions)}</div>
      </div>
      <div class="panel editor-panel">${renderElementEditor(selected)}</div>
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
      <label>监控项
        <select data-ui-field="addItemMonitor" ${state.monitorOptions.length === 0 ? "disabled" : ""}>
          ${addMonitorOptions || '<option value="">(none)</option>'}
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
    return `<h3>元素编辑</h3><div class="notice">请选择左侧列表中的屏幕元素</div>`;
  }

  const typeOptions = (state.meta?.item_types || ELEMENT_TYPES)
    .map((itemType) => `<option value="${escapeAttr(itemType)}" ${item.type === itemType ? "selected" : ""}>${escapeHTML(itemType)}</option>`)
    .join("");

  const monitorOptions = ["<option value=\"\">(none)</option>"]
    .concat(
      state.monitorOptions.map(
        (name) => `<option value=\"${escapeAttr(name)}\" ${item.monitor === name ? "selected" : ""}>${escapeHTML(monitorDisplayName(name))}</option>`,
      ),
    )
    .join("");

  return `
    <h3>元素编辑</h3>
    <div class="editor-block">
      <h4>基础信息</h4>
      <div class="grid base-row">
        ${itemTextField("edit_ui_name", item.edit_ui_name)}
        <div class="field"><label>type</label><select data-item-field="type">${typeOptions}</select></div>
        ${MONITOR_ELEMENT_TYPES.has(item.type) ? `<div class="field"><label>monitor</label><select data-item-field="monitor">${monitorOptions}</select></div>` : ""}
      </div>
    </div>
    <div class="editor-block">
      <h4>布局</h4>
      <div class="xywh-row">
        ${itemNumberField("x", item.x)}
        ${itemNumberField("y", item.y)}
        ${itemNumberField("width", item.width)}
        ${itemNumberField("height", item.height)}
      </div>
    </div>

    ${LABEL_ELEMENT_TYPES.has(item.type) ? renderLabelEditorBlock(item) : ""}
    ${MONITOR_ELEMENT_TYPES.has(item.type) ? renderMonitorEditorBlock(item) : ""}
    ${SHAPE_ELEMENT_TYPES.has(item.type) ? renderShapeEditorBlock(item) : ""}

  `;
}

function renderLabelEditorBlock(item) {
  return `
    <div class="editor-block">
      <h4>标签属性</h4>
      <div class="grid monitor-main-row">
        ${itemTextField("text", item.text)}
        ${itemNumberField("font_size", item.font_size)}
      </div>
      <div class="grid label-appearance-row">
        ${itemColorField("color", item.color)}
        ${itemColorField("bg", item.bg)}
        ${itemColorField("border_color", item.border_color)}
        ${itemNumberField("border_width", item.border_width)}
        ${itemNumberField("radius", item.radius)}
      </div>
    </div>
  `;
}

function renderMonitorEditorBlock(item) {
  const thresholdMode = Array.isArray(item.thresholds) && item.thresholds.length === 4 ? "custom" : "default";
  const colorMode = Array.isArray(item.level_colors) && item.level_colors.length === 4 ? "custom" : "default";
  const isProgressLike = item.type === "progress" || item.type === "line_chart";
  const isChart = item.type === "line_chart";
  const isProgress = item.type === "progress";
  const typeLabel = isChart ? "line_chart" : (isProgress ? "progress" : "value");
  const pointSizeField = isChart ? itemNumberField("point_size", item.point_size ?? state.config?.history_size ?? 150) : "";

  return `
    <div class="editor-block">
      <h4>监控属性</h4>
      <div class="editor-group-title">通用</div>
      <div class="grid monitor-main-row">
        ${itemNumberField("font_size", item.font_size)}
        ${itemTextField("unit", item.unit || "auto")}
        ${itemNumberField("unit_font_size", item.unit_font_size)}
      </div>
      <div class="grid monitor-unit-row">
        ${itemColorField("color", item.color)}
        ${itemColorField("unit_color", item.unit_color)}
      </div>

      ${isProgressLike ? `
        <div class="editor-group-title">${escapeHTML(typeLabel)} 专属</div>
        <div class="grid monitor-range-row">
          ${itemNumberField("min_value", item.min_value)}
          ${itemNumberField("max_value", item.max_value)}
          ${isProgress ? itemNumberField("max", item.max) : pointSizeField}
        </div>
      ` : ""}

      <div class="editor-group-title">外观</div>
      <div class="grid appearance-row">
        ${itemColorField("bg", item.bg)}
        ${itemColorField("border_color", item.border_color)}
        ${itemNumberField("border_width", item.border_width)}
        ${itemNumberField("radius", item.radius)}
      </div>
      <div class="editor-group-title">等级</div>
      <div class="grid mode-row">
        <div class="field"><label>阈值模式</label>
          <select data-item-field="threshold_mode">
            <option value="default" ${thresholdMode === "default" ? "selected" : ""}>使用默认阈值</option>
            <option value="custom" ${thresholdMode === "custom" ? "selected" : ""}>自定义阈值</option>
          </select>
        </div>
        <div class="field"><label>颜色模式</label>
          <select data-item-field="level_color_mode">
            <option value="default" ${colorMode === "default" ? "selected" : ""}>使用默认等级色</option>
            <option value="custom" ${colorMode === "custom" ? "selected" : ""}>自定义等级色</option>
          </select>
        </div>
      </div>

      ${thresholdMode === "custom" ? `
        <div class="grid-4">
          ${itemThresholdField(0, item.thresholds?.[0])}
          ${itemThresholdField(1, item.thresholds?.[1])}
          ${itemThresholdField(2, item.thresholds?.[2])}
          ${itemThresholdField(3, item.thresholds?.[3])}
        </div>
      ` : ""}

      ${colorMode === "custom" ? `
        <div class="grid-4">
          ${itemLevelColorField(0, item.level_colors?.[0])}
          ${itemLevelColorField(1, item.level_colors?.[1])}
          ${itemLevelColorField(2, item.level_colors?.[2])}
          ${itemLevelColorField(3, item.level_colors?.[3])}
        </div>
      ` : ""}
    </div>
  `;
}

function renderShapeEditorBlock(item) {
  return `
    <div class="editor-block">
      <h4>图形属性</h4>
      <div class="grid shape-row">
        ${itemColorField("bg", item.bg)}
        ${itemColorField("border_color", item.border_color)}
        ${itemNumberField("border_width", item.border_width)}
        ${itemNumberField("radius", item.radius)}
      </div>
    </div>
  `;
}

function renderPreview(cfg) {
  const selected = cfg.items[state.selectedItem];
  const { purePreview } = state.ui;
  return `
    <h3>预览</h3>
    <div class="inline-actions">
      <label><input type="checkbox" data-ui-field="purePreview" ${purePreview ? "checked" : ""} /> 纯预览模式</label>
    </div>
    <div class="preview-wrapper" id="preview-wrapper">
      <div class="preview-stage" id="preview-stage" style="width:${cfg.width}px;height:${cfg.height}px"></div>
    </div>
    <div id="preview-render-error" class="error">${escapeHTML(state.previewError || "")}</div>
    <div class="notice">Go 渲染图实时预览；仅选中元素可拖拽/缩放。当前选中：${selected ? `${state.selectedItem + 1}` : "无"}</div>
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
        <h3>自定义监控项</h3>
        <div class="inline-actions">
          <button data-action="add-custom">添加自定义监控项</button>
        </div>
        ${content || '<div class="notice">暂无自定义监控项</div>'}
      </div>
    </div>
  `;
}

function renderLibreCustom(index, monitor) {
  const options = (state.monitorOptions || [])
    .filter((name) => String(name).startsWith("libre_"))
    .map((name) => ({ name, label: monitorLabel(name) || name }));
  const selected = String(monitor.source || "");
  const sensorOptions = ["<option value=\"\">(none)</option>"]
    .concat(
      options.map((item) => {
        const labelParts = [item.label || item.name || "unnamed"];
        if (item.unit) labelParts.push(item.unit);
        const label = labelParts.join(" | ");
        return `<option value=\"${escapeAttr(item.name)}\" ${selected === item.name ? "selected" : ""}>${escapeHTML(label)}</option>`;
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
    .map((name) => ({ name, label: monitorLabel(name) || name }));
  const selected = String(monitor.source || "");
  const sourceOptions = ["<option value=\"\">(none)</option>"]
    .concat(
      options.map((item) => {
        const labelParts = [item.label || item.name || "unnamed"];
        if (item.unit) labelParts.push(item.unit);
        const label = labelParts.join(" | ");
        return `<option value=\"${escapeAttr(item.name)}\" ${selected === item.name ? "selected" : ""}>${escapeHTML(label)}</option>`;
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
          <h3>采集信息</h3>
          <div class="runtime-meta">
            <span>模式: <strong id="runtime-mode">${escapeHTML(getRuntimeModeText())}</strong></span>
            <span>更新时间: <strong id="runtime-updated">${escapeHTML(formatTimestamp(state.snapshot?.updated_at))}</strong></span>
            <span>监控项: <strong id="runtime-count">${Object.keys(state.snapshot?.values || {}).length}</strong></span>
          </div>
          <div class="inline-actions">
            <button data-action="refresh-snapshot">立即刷新</button>
          </div>
          <div class="notice">页面活跃时每 2 秒抓全量监控，30 秒无请求自动回到按配置采集。</div>
        </div>
        <div class="runtime-monitor-stats" id="runtime-monitor-stats">${renderRuntimeMonitorStats()}</div>
        <div class="collection-list" id="runtime-values">${renderRuntimeValueRows()}</div>
      </div>
    </div>
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
      const title = label ? `${label} (${name})` : name;
      return `<div class="runtime-row${cls}"><span>${escapeHTML(title)}</span><span>${escapeHTML(item.text || "-")}</span></div>`;
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

function fieldText(field, label, value = "") {
  return `<div class="field"><label>${label}</label><input data-field="${field}" value="${escapeAttr(value ?? "")}" /></div>`;
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

function itemTextField(field, value) {
  return `<div class="field"><label>${field}</label><input data-item-field="${field}" data-item-kind="text" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemNumberField(field, value) {
  return `<div class="field"><label>${field}</label><input type="number" data-item-field="${field}" data-item-kind="nullable-number" value="${escapeAttr(value ?? "")}" /></div>`;
}

function itemColorField(field, value) {
  return `
    <div class="field">
      <label>${field}</label>
      <div class="color-input-row color-picker-row">
        <button type="button" class="color-picker-trigger" aria-label="${escapeAttr(field)}"></button>
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

function parseColorValue(value) {
  const raw = String(value || "").trim();
  if (!raw) return { hex: "#000000", alpha: 0 };

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
  return { hex: "#000000", alpha: 0 };
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

function updatePreviewDOM() {
  const stage = document.getElementById("preview-stage");
  const wrapper = document.getElementById("preview-wrapper");
  if (!stage || !wrapper || !state.config) return;

  const cfg = state.config;
  const maxWidth = wrapper.clientWidth - 20;
  const scale = Math.min(1, maxWidth / Math.max(cfg.width, 1));
  state.previewScale = scale;
  stage.style.transform = `scale(${scale})`;
  stage.classList.toggle("pure-preview", !!state.ui.purePreview);
  stage.style.backgroundImage = "none";

  const previewImage = `<img id="preview-image" class="preview-image" alt="Go Render Preview" src="${escapeAttr(state.previewImageURL || "")}" draggable="false" />`;
  const guides = state.ui.centerGuides;
  const guideHTML = guides
    ? `${guides.showVertical ? `<div class="preview-guide vertical" style="left:${num(guides.x)}px"></div>` : ""}${
      guides.showHorizontal ? `<div class="preview-guide horizontal" style="top:${num(guides.y)}px"></div>` : ""
    }`
    : "";
  stage.innerHTML =
    previewImage +
    guideHTML +
    cfg.items
      .map((item, index) => {
        const selected = index === state.selectedItem ? "selected" : "";
        return `<div class="preview-item ${selected}" data-preview-index="${index}" style="left:${num(item.x)}px;top:${num(item.y)}px;width:${Math.max(10, num(item.width))}px;height:${Math.max(10, num(item.height))}px;">
          ${selected ? `<div class="resize-handle" data-preview-resize="${index}"></div>` : ""}
        </div>`;
      })
      .join("");
}

function updateCenterGuides(item) {
  const cfg = state.config;
  const movingIndex = Number(state.drag?.index);
  if (!cfg || !item || state.drag?.mode !== "move" || !Number.isFinite(movingIndex)) {
    state.ui.centerGuides = null;
    return;
  }

  const width = Math.max(10, num(item.width));
  const height = Math.max(10, num(item.height));
  const centerX = num(item.x) + width / 2;
  const centerY = num(item.y) + height / 2;
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
  const stage = document.getElementById("preview-stage");
  if (!stage) return;

  stage.addEventListener("pointerdown", (event) => {
    if (state.ui.purePreview || isEditingProfileReadOnly()) {
      state.ui.centerGuides = null;
      return;
    }

    const resizeHandle = event.target.closest(".resize-handle");
    if (resizeHandle) {
      const index = Number(resizeHandle.dataset.previewResize);
      const item = state.config.items[index];
      if (!item) return;
      state.selectedItem = index;
      state.drag = {
        mode: "resize",
        index,
        startX: event.clientX,
        startY: event.clientY,
        originW: num(item.width),
        originH: num(item.height),
      };
      render();
      return;
    }

    const target = event.target.closest(".preview-item");
    if (!target) return;
    const index = Number(target.dataset.previewIndex);
    if (!Number.isFinite(index)) return;

    if (index !== state.selectedItem) {
      state.selectedItem = index;
      render();
      return;
    }

    state.selectedItem = index;
    const item = state.config.items[index];
    state.drag = {
      mode: "move",
      index,
      startX: event.clientX,
      startY: event.clientY,
      originX: num(item.x),
      originY: num(item.y),
    };
    updateCenterGuides(item);
    render();
  });
}

window.addEventListener("pointermove", (event) => {
  if (state.ui.purePreview || isEditingProfileReadOnly()) {
    state.drag = null;
    state.ui.centerGuides = null;
    return;
  }
  if (!state.drag || !state.config) return;
  const item = state.config.items[state.drag.index];
  if (!item) return;

  const dx = (event.clientX - state.drag.startX) / Math.max(state.previewScale, 0.01);
  const dy = (event.clientY - state.drag.startY) / Math.max(state.previewScale, 0.01);

  if (state.drag.mode === "resize") {
    item.width = Math.max(10, Math.round(state.drag.originW + dx));
    item.height = Math.max(10, Math.round(state.drag.originH + dy));
    state.ui.centerGuides = null;
  } else {
    item.x = Math.round(state.drag.originX + dx);
    item.y = Math.round(state.drag.originY + dy);
    updateCenterGuides(item);
  }

  markDirty();
  updatePreviewDOM();
  renderSelectedItemValues();
});

window.addEventListener("pointerup", () => {
  state.drag = null;
  state.ui.centerGuides = null;
  updatePreviewDOM();
});

function renderSelectedItemValues() {
  const item = state.config?.items?.[state.selectedItem];
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

async function refreshPreviewImage() {
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
    const objectURL = URL.createObjectURL(blob);
    if (state.previewImageURL) {
      URL.revokeObjectURL(state.previewImageURL);
    }
    state.previewImageURL = objectURL;
    state.previewError = "";

    const img = document.getElementById("preview-image");
    if (img) {
      img.src = objectURL;
    }
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

  const list = document.getElementById("runtime-values");
  if (list) list.innerHTML = renderRuntimeValueRows();
}

async function pollRuntime() {
  if (state.polling) return;
  state.polling = true;

  try {
    const snapshotRes = await api("/api/snapshot");

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

    await refreshPreviewImage();
  } catch (error) {
    setError(`轮询失败: ${error.message}`);
  } finally {
    state.polling = false;
  }
}

function startPolling() {
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
    const [metaRes, configRes, profilesRes] = await Promise.all([
      api("/api/meta"),
      api("/api/config"),
      api("/api/profiles").catch(() => ({ items: [], active: "default" })),
    ]);

    state.meta = metaRes;
    applyProfilesPayload(profilesRes);
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
  startPolling();
  await pollRuntime();
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
    if (field === "purePreview") {
      state.ui[field] = !!target.checked;
      if (field === "purePreview" && state.ui.purePreview) {
        state.drag = null;
        state.ui.centerGuides = null;
      }
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
      const next = normalizeItem(
        {
          ...createItemByType(target.value),
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
      item.history = item.type === "line_chart";
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
      schedulePreviewSync(true);
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
    }
    return;
  }

  if (action === "refresh-snapshot") {
    await pollRuntime();
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
    try {
      const res = await api("/api/profiles", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          config: state.config,
          switch: false,
        }),
      });
      applyProfilesPayload(res);
      state.editingProfile = name.trim();
      clearDirty();
      setError("");
      schedulePreviewSync(true);
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

    try {
      const res = await api("/api/profiles", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          config: state.config,
          switch: false,
        }),
      });
      applyProfilesPayload(res);
      state.editingProfile = name.trim();
      clearDirty();
      setError("");
      schedulePreviewSync(true);
    } catch (error) {
      setError(error.message);
    }
    render();
    return;
  }

  if (action === "delete-profile") {
    const name = state.editingProfile || selectedProfileName();
    if (!name) return;
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
    render();
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

  const step = event.shiftKey ? 10 : 1;
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

  markDirty();
  updatePreviewDOM();
  renderSelectedItemValues();
});

window.addEventListener("beforeunload", () => {
  destroyColorPickers();
  stopPolling();
  if (state.previewSyncTimer) {
    window.clearTimeout(state.previewSyncTimer);
    state.previewSyncTimer = null;
  }
  if (state.previewImageURL) {
    URL.revokeObjectURL(state.previewImageURL);
    state.previewImageURL = "";
  }
});

void init();

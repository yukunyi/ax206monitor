import { normalizeStyleKeys } from "./style_keys";
import { normalizeItemTypes } from "./item_types";
import { normalizeOutputs } from "./output_configs";

function defaultCreateItemId() {
  return `itm_${Date.now()}`;
}

export function buildStyleKeySet(styleKeys) {
  const set = new Set();
  normalizeStyleKeys(styleKeys).forEach((meta) => {
    if (meta.key) set.add(meta.key);
  });
  return set;
}

export function normalizeStyleMap(raw, styleKeySet) {
  const source = raw && typeof raw === "object" ? raw : {};
  const result = {};
  Object.entries(source).forEach(([key, value]) => {
    const name = String(key || "").trim();
    if (!name || !styleKeySet.has(name)) return;
    result[name] = value;
  });
  return result;
}

export function normalizeRenderAttrs(raw, styleKeySet) {
  const source = raw && typeof raw === "object" ? raw : {};
  const result = {};
  Object.entries(source).forEach(([key, value]) => {
    const name = String(key || "").trim();
    if (!name || styleKeySet.has(name)) return;
    result[name] = value;
  });
  return result;
}

export function normalizeTypeDefaults(raw, styleKeySet, itemTypesRaw = []) {
  const source = raw && typeof raw === "object" ? raw : {};
  const result = {};
  normalizeItemTypes(itemTypesRaw).forEach((type) => {
    const input = source[type] && typeof source[type] === "object" ? source[type] : {};
    result[type] = {
      render_attrs_map: normalizeRenderAttrs(input.render_attrs_map, styleKeySet),
      style: normalizeStyleMap(input.style, styleKeySet),
    };
  });
  Object.keys(source).forEach((type) => {
    if (result[type]) return;
    const input = source[type] && typeof source[type] === "object" ? source[type] : {};
    result[type] = {
      render_attrs_map: normalizeRenderAttrs(input.render_attrs_map, styleKeySet),
      style: normalizeStyleMap(input.style, styleKeySet),
    };
  });
  return result;
}

export function ensureCollectorEntry(config, collectorName, defaultCollectorEnabled) {
  if (!config.collector_config || typeof config.collector_config !== "object") {
    config.collector_config = {};
  }
  if (!config.collector_config[collectorName]) {
    config.collector_config[collectorName] = {
      enabled: !!defaultCollectorEnabled[collectorName],
      options: {},
    };
  }
  if (!config.collector_config[collectorName].options) {
    config.collector_config[collectorName].options = {};
  }
}

export function normalizeConfigModel(
  cfg,
  {
    styleKeysRaw = [],
    itemTypesRaw = [],
    createItemId = defaultCreateItemId,
    defaultCollectorEnabled = {},
  } = {},
) {
  const styleKeySet = buildStyleKeySet(styleKeysRaw);
  const config = JSON.parse(JSON.stringify(cfg || {}));
  config.name = String(config.name || "web");
  config.width = Math.max(10, Number(config.width || 480));
  config.height = Math.max(10, Number(config.height || 320));
  config.layout_padding = Math.max(0, Number(config.layout_padding || 0));
  config.refresh_interval = Math.max(100, Number(config.refresh_interval || 1000));
  config.collect_warn_ms = Math.max(10, Number(config.collect_warn_ms || 100));
  config.render_wait_max_ms = Math.max(0, Number(config.render_wait_max_ms || 300));
  config.history_size = Math.max(10, Number(config.history_size || 180));
  config.default_history_points = Math.max(10, Number(config.default_history_points || 150));
  config.default_font = String(config.default_font || "");
  config.style_base = normalizeStyleMap(config.style_base, styleKeySet);
  config.allow_custom_style = config.allow_custom_style === true;
  config.font_families = Array.isArray(config.font_families) ? config.font_families : [];
  config.outputs = normalizeOutputs(config.outputs, config.output_types);
  config.output_types = [...new Set(config.outputs.map((item) => item.type))];
  config.pause_collect_on_lock = config.pause_collect_on_lock === true;
  config.type_defaults = normalizeTypeDefaults(config.type_defaults, styleKeySet, itemTypesRaw);
  config.items = Array.isArray(config.items) ? config.items : [];
  const itemIdSet = new Set();
  config.items = config.items.map((item) => {
    const next = { ...(item || {}) };
    next.id = String(next.id || "").trim() || createItemId();
    if (itemIdSet.has(next.id)) {
      next.id = createItemId();
    }
    itemIdSet.add(next.id);
    next.custom_style = config.allow_custom_style ? next.custom_style === true : false;
    next.style = normalizeStyleMap(next.style, styleKeySet);
    next.render_attrs_map = normalizeRenderAttrs(next.render_attrs_map, styleKeySet);
    return next;
  });
  config.custom_monitors = Array.isArray(config.custom_monitors) ? config.custom_monitors : [];
  config.collector_config = config.collector_config || {};
  Object.keys(defaultCollectorEnabled).forEach((name) =>
    ensureCollectorEntry(config, name, defaultCollectorEnabled),
  );
  return config;
}

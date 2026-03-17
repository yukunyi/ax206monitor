export const STYLE_SCOPE_BASE = "base";
export const STYLE_SCOPE_TYPE = "type";
export const STYLE_SCOPE_ITEM = "item";

export const DEFAULT_STYLE_KEYS = [
  { key: "font_family", label: "字体", kind: "select", scopes: [STYLE_SCOPE_BASE] },
  { key: "text_font_size", label: "文本字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "unit_font_size", label: "单位字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "value_font_size", label: "值字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "color", label: "文字色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "bg", label: "背景色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "unit_color", label: "单位色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "border_width", label: "边框宽度", kind: "float", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "border_color", label: "边框颜色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "radius", label: "圆角", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "history_points", label: "历史点数", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["simple_line_chart", "full_chart"] },
  { key: "content_padding_x", label: "左右边距", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["label_text", "full_chart", "full_table", "full_progress_h", "full_progress_v", "full_gauge"] },
  { key: "content_padding_y", label: "上下边距", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["label_text", "full_chart", "full_table", "full_progress_h", "full_progress_v", "full_gauge"] },
  { key: "body_gap", label: "标题间距", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "header_height", label: "标题栏高度", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "header_divider", label: "标题分隔线", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "header_divider_width", label: "分隔线宽", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "header_divider_offset", label: "分隔线偏移", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "header_divider_color", label: "分隔线颜色", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h"] },
  { key: "show_segment_lines", label: "分段线", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "show_grid_lines", label: "网格线", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "grid_lines", label: "网格线数量", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "enable_threshold_colors", label: "阈值分段颜色", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["simple_line_chart", "full_chart"] },
  { key: "line_width", label: "线宽", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["simple_line_chart", "simple_line", "full_chart"] },
  {
    key: "line_orientation",
    label: "线方向",
    kind: "select",
    scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM],
    types: ["simple_line"],
    options: [
      { label: "横向", value: "horizontal" },
      { label: "竖向", value: "vertical" },
    ],
  },
  { key: "show_avg_line", label: "均线", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "chart_color", label: "折线颜色", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "chart_area_bg", label: "图表区背景", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  { key: "chart_area_border_color", label: "图表区边框", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart"] },
  {
    key: "progress_style",
    label: "进度样式",
    kind: "select",
    scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM],
    types: ["full_progress_h", "full_progress_v"],
    options: [
      { label: "gradient", value: "gradient" },
      { label: "solid", value: "solid" },
      { label: "segmented", value: "segmented" },
      { label: "stripes", value: "stripes" },
    ],
  },
  { key: "bar_height", label: "条高度", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress_h", "full_progress_v"] },
  { key: "bar_radius", label: "条圆角", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress_h", "full_progress_v"] },
  { key: "track_color", label: "轨道颜色", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress_h", "full_progress_v", "full_gauge"] },
  { key: "segments", label: "分段数量", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress_h", "full_progress_v"] },
  { key: "segment_gap", label: "分段间隔", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress_h", "full_progress_v"] },
  { key: "card_radius", label: "外框圆角", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_table", "full_progress_h", "full_progress_v", "full_gauge"] },
  { key: "table_row_gap", label: "行间距", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_row_radius", label: "行圆角", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_row_bg", label: "行背景", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_row_alt_bg", label: "交替行背景", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_column_gap", label: "列间距", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_label_width_ratio", label: "标签列比例", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "table_show_units", label: "显示单位", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_table"] },
  { key: "gauge_thickness", label: "仪表盘厚度", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_gauge"] },
  { key: "gauge_gap_degrees", label: "底部缺口角度", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_gauge"] },
  { key: "gauge_text_gap", label: "文字间距", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_gauge"] },
];

export function normalizeStyleKeys(raw) {
  const list = Array.isArray(raw) && raw.length > 0 ? raw : DEFAULT_STYLE_KEYS;
  return list
    .map((item) => ({
      key: String(item?.key || "").trim(),
      label: String(item?.label || item?.key || "").trim(),
      kind: String(item?.kind || "text").trim(),
      scopes: Array.isArray(item?.scopes) ? item.scopes.map((s) => String(s || "").trim()).filter(Boolean) : [],
      types: Array.isArray(item?.types) ? item.types.map((t) => String(t || "").trim()).filter(Boolean) : [],
      options: Array.isArray(item?.options)
        ? item.options
            .map((opt) => ({
              label: String(opt?.label || opt?.value || "").trim(),
              value: String(opt?.value || "").trim(),
            }))
            .filter((opt) => !!opt.value)
        : [],
      default: cloneStyleValue(item?.default),
      defaults: normalizeStyleDefaults(item?.defaults),
    }))
    .filter((item) => !!item.key);
}

function cloneStyleValue(value) {
  if (Array.isArray(value)) return value.map((item) => cloneStyleValue(item));
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, cloneStyleValue(item)]),
    );
  }
  return value;
}

function normalizeStyleDefaults(raw) {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return {};
  const out = {};
  Object.entries(raw).forEach(([key, value]) => {
    const name = String(key || "").trim();
    if (!name) return;
    out[name] = cloneStyleValue(value);
  });
  return out;
}

export function styleDefaultValue(styleKeys, key, itemType = "") {
  const normalizedKey = String(key || "").trim();
  if (!normalizedKey) return "";
  const meta = styleMetaMap(styleKeys)[normalizedKey];
  if (!meta) return "";

  const type = String(itemType || "").trim();
  if (type && meta.defaults && Object.prototype.hasOwnProperty.call(meta.defaults, type)) {
    return cloneStyleValue(meta.defaults[type]);
  }
  if (meta.default !== undefined) {
    return cloneStyleValue(meta.default);
  }
  return "";
}

export function supportsScope(meta, scope) {
  return Array.isArray(meta?.scopes) && meta.scopes.includes(scope);
}

export function supportsType(meta, itemType) {
  if (!Array.isArray(meta?.types) || meta.types.length === 0) return true;
  return meta.types.includes(String(itemType || "").trim());
}

export function styleMetaMap(styleKeys) {
  const map = {};
  normalizeStyleKeys(styleKeys).forEach((meta) => {
    map[meta.key] = meta;
  });
  return map;
}

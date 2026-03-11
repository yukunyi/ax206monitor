export const STYLE_SCOPE_BASE = "base";
export const STYLE_SCOPE_TYPE = "type";
export const STYLE_SCOPE_ITEM = "item";

export const DEFAULT_STYLE_KEYS = [
  { key: "font_family", label: "字体", kind: "select", scopes: [STYLE_SCOPE_BASE] },
  { key: "small_font_size", label: "小字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "medium_font_size", label: "中字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "large_font_size", label: "大字号", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "color", label: "文字色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "bg", label: "背景色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "unit_color", label: "单位色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "border_width", label: "边框宽度", kind: "float", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "border_color", label: "边框颜色", kind: "color", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "radius", label: "圆角", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM] },
  { key: "thresholds", label: "阈值(%)", kind: "float4", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_ITEM] },
  { key: "level_colors", label: "等级颜色", kind: "color4", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_ITEM] },
  { key: "history_points", label: "历史点数", kind: "int", scopes: [STYLE_SCOPE_BASE, STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["simple_line_chart", "full_chart"] },
  { key: "content_padding", label: "内边距", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["label_text", "full_chart", "full_progress", "full_gauge"] },
  { key: "body_gap", label: "标题间距", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
  { key: "value_font_size", label: "值字号", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress", "full_gauge"] },
  { key: "label_font_size", label: "标签字号", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_gauge"] },
  { key: "title_font_size", label: "标题字号", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
  { key: "header_divider", label: "标题分隔线", kind: "bool", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
  { key: "header_divider_width", label: "分隔线宽", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
  { key: "header_divider_offset", label: "分隔线偏移", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
  { key: "header_divider_color", label: "分隔线颜色", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress"] },
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
    types: ["full_progress"],
    options: [
      { label: "gradient", value: "gradient" },
      { label: "solid", value: "solid" },
      { label: "segmented", value: "segmented" },
      { label: "stripes", value: "stripes" },
    ],
  },
  { key: "bar_height", label: "条高度", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress"] },
  { key: "bar_radius", label: "条圆角", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress"] },
  { key: "track_color", label: "轨道颜色", kind: "color", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress", "full_gauge"] },
  { key: "segments", label: "分段数量", kind: "int", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress"] },
  { key: "segment_gap", label: "分段间隔", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_progress"] },
  { key: "card_radius", label: "外框圆角", kind: "float", scopes: [STYLE_SCOPE_TYPE, STYLE_SCOPE_ITEM], types: ["full_chart", "full_progress", "full_gauge"] },
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
    }))
    .filter((item) => !!item.key);
}

export function styleDefaultValue(key, itemType = "") {
  const type = String(itemType || "").trim();
  switch (String(key || "").trim()) {
    case "font_family":
      return "";
    case "small_font_size":
      return 14;
    case "medium_font_size":
      return 16;
    case "large_font_size":
      return 18;
    case "color":
      return "#f8fafc";
    case "bg":
      return "";
    case "unit_color":
      return "#f8fafc";
    case "border_width":
      return 0;
    case "border_color":
      return "#475569";
    case "radius":
      return 0;
    case "thresholds":
      return [25, 50, 75, 100];
    case "level_colors":
      return ["#22c55e", "#eab308", "#f97316", "#ef4444"];
    case "history_points":
      return 150;
    case "content_padding":
      return type.startsWith("full_") ? 1 : 3;
    case "body_gap":
      return type === "full_chart" ? 4 : 0;
    case "value_font_size":
    case "label_font_size":
    case "title_font_size":
      return 0;
    case "header_divider":
      return true;
    case "header_divider_width":
      return 1;
    case "header_divider_offset":
      return 3;
    case "header_divider_color":
      return "#94a3b840";
    case "show_segment_lines":
    case "show_grid_lines":
    case "show_avg_line":
      return true;
    case "grid_lines":
      return 4;
    case "enable_threshold_colors":
      return false;
    case "line_width":
      return 1;
    case "line_orientation":
      return "horizontal";
    case "chart_color":
      return "#38bdf8";
    case "chart_area_bg":
    case "chart_area_border_color":
      return "";
    case "progress_style":
      return "gradient";
    case "bar_height":
    case "bar_radius":
      return 0;
    case "track_color":
      return "#1f2937";
    case "segments":
      return 12;
    case "segment_gap":
      return 2;
    case "card_radius":
      return 0;
    case "gauge_thickness":
      return 10;
    case "gauge_gap_degrees":
      return 76;
    case "gauge_text_gap":
      return 4;
    default:
      return "";
  }
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

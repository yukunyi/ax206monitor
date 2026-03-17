export const DEFAULT_ITEM_TYPES = [
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  "simple_line",
  "simple_label",
  "simple_rect",
  "simple_circle",
  "label_text",
  "full_chart",
  "full_table",
  "full_progress_h",
  "full_progress_v",
  "full_gauge",
];

export const ITEM_TYPE_LABELS = {
  simple_value: "基础数值",
  simple_progress: "基础进度条",
  simple_line_chart: "基础折线图",
  simple_line: "基础线条",
  simple_label: "基础标签",
  simple_rect: "基础矩形",
  simple_circle: "基础圆形",
  label_text: "标签数值",
  full_chart: "复杂图表",
  full_table: "复杂表格",
  full_progress_h: "复杂进度条(横向)",
  full_progress_v: "复杂进度条(竖向)",
  full_gauge: "复杂仪表盘",
};

const MONITOR_REQUIRED_TYPE_SET = new Set([
  "simple_value",
  "simple_progress",
  "simple_line_chart",
  "label_text",
  "full_chart",
  "full_progress_h",
  "full_progress_v",
  "full_gauge",
]);

export function getItemTypeLabel(type) {
  const key = String(type || "").trim();
  if (!key) return "";
  return ITEM_TYPE_LABELS[key] || key;
}

export function isMonitorRequiredType(type) {
  return MONITOR_REQUIRED_TYPE_SET.has(String(type || "").trim());
}

export function normalizeItemTypes(metaTypes) {
  if (!Array.isArray(metaTypes) || metaTypes.length <= 0) {
    return [...DEFAULT_ITEM_TYPES];
  }
  const set = new Set();
  metaTypes.forEach((item) => {
    const value = typeof item === "string" ? item : String(item?.value || "");
    const key = String(value || "").trim();
    if (key) set.add(key);
  });
  if (set.size <= 0) return [...DEFAULT_ITEM_TYPES];
  return [...set];
}

export function buildItemTypeOptions(metaTypes) {
  const source = Array.isArray(metaTypes) && metaTypes.length > 0
    ? metaTypes
    : DEFAULT_ITEM_TYPES;
  const seen = new Set();
  const options = [];
  source.forEach((item) => {
    if (typeof item === "string") {
      const value = String(item || "").trim();
      if (!value || seen.has(value)) return;
      seen.add(value);
      options.push({ label: getItemTypeLabel(value), value });
      return;
    }
    const value = String(item?.value || "").trim();
    if (!value || seen.has(value)) return;
    seen.add(value);
    const label = String(item?.label || getItemTypeLabel(value)).trim() || value;
    options.push({ label, value });
  });
  return options;
}

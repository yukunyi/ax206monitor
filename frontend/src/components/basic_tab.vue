<script setup>
import { computed } from "vue";
import PureColorInput from "./pure_color_input.vue";

const props = defineProps({
  config: { type: Object, required: true },
  meta: { type: Object, required: true },
  collectors: { type: Array, default: () => [] },
  monitorOptions: { type: Array, default: () => [] },
  readonlyProfile: { type: Boolean, default: false },
});

const emit = defineEmits([
  "change",
  "add-custom",
  "remove-custom",
  "change-custom",
  "refresh-monitors",
]);

function onField(path, value) {
  emit("change", { path, value });
}

const platform = computed(() => String(props.meta?.platform || "").toLowerCase());

function collectorSupportedOnPlatform(name) {
  const collector = String(name || "").trim().toLowerCase();
  if (collector === "rtss") return platform.value === "windows";
  if (collector === "coolercontrol") return platform.value === "linux";
  if (collector === "librehardwaremonitor") return platform.value === "windows";
  return true;
}

const collectorNames = computed(() => {
  const set = new Set();
  (props.meta.collectors || []).forEach((name) => set.add(String(name)));
  (props.collectors || []).forEach((item) => set.add(String(item.name || "")));
  Object.keys(props.config.collector_config || {}).forEach((name) => set.add(String(name)));
  return [...set]
    .filter((name) => !!name && collectorSupportedOnPlatform(name))
    .sort();
});

const fontOptions = computed(() =>
  (props.meta.font_families || []).map((font) => ({ label: font, value: font })),
);

const monitorSelectOptions = computed(() => props.monitorOptions || []);
const outputTypeOptions = computed(() => {
  const set = new Set();
  (props.meta.output_types || ["memimg", "ax206usb"]).forEach((item) => set.add(String(item || "")));
  (props.config.output_types || []).forEach((item) => set.add(String(item || "")));
  return [...set]
    .filter(Boolean)
    .map((item) => ({ label: item, value: item }));
});

const customTypeOptions = computed(() => {
  const options = [
    { label: "file", value: "file" },
    { label: "mixed", value: "mixed" },
    { label: "coolercontrol", value: "coolercontrol" },
    { label: "librehardwaremonitor", value: "librehardwaremonitor" },
    { label: "rtss", value: "rtss" },
  ];
  return options.filter((item) => collectorSupportedOnPlatform(item.value));
});

const aggregateOptions = [
  { label: "max", value: "max" },
  { label: "min", value: "min" },
  { label: "avg", value: "avg" },
];

const ITEM_TYPE_LABELS = {
  simple_value: "基础数值",
  simple_progress: "基础进度条",
  simple_line_chart: "基础折线图",
  simple_label: "基础标签",
  simple_rect: "基础矩形",
  simple_circle: "基础圆形",
  label_text: "标签数值",
  full_chart: "复杂图表",
  full_progress: "复杂进度条",
};

const typeDefaultRows = computed(() => {
  const order = Array.isArray(props.meta?.item_types) && props.meta.item_types.length > 0
    ? props.meta.item_types.map((item) => (typeof item === "string" ? item : String(item?.value || "")))
    : Object.keys(ITEM_TYPE_LABELS);
  const unique = new Set(order.filter(Boolean));
  Object.keys(props.config.type_defaults || {}).forEach((type) => unique.add(type));
  return [...unique];
});

const progressStyleOptions = [
  { label: "gradient", value: "gradient" },
  { label: "solid", value: "solid" },
  { label: "segmented", value: "segmented" },
  { label: "stripes", value: "stripes" },
];

function collectorEntry(name) {
  if (!props.config.collector_config) return { enabled: false, options: {} };
  return props.config.collector_config[name] || { enabled: false, options: {} };
}

function collectorUrl(name) {
  return collectorEntry(name).options?.url || "";
}

function collectorHasUrl(name) {
  return name === "coolercontrol" || name === "librehardwaremonitor";
}

function collectorHasAuth(name) {
  return name === "coolercontrol" || name === "librehardwaremonitor";
}

function collectorAuthUserLabel(name) {
  if (name === "coolercontrol") return "Username";
  return "User";
}

function collectorOption(name, key) {
  return String(collectorEntry(name).options?.[key] || "");
}

function thresholdValue(index) {
  const list = Array.isArray(props.config.default_thresholds) ? props.config.default_thresholds : [];
  return Number(list[index] ?? [25, 50, 75, 100][index]);
}

function levelColorValue(index) {
  const list = Array.isArray(props.config.level_colors) ? props.config.level_colors : [];
  return String(list[index] || ["#22c55e", "#eab308", "#f97316", "#ef4444"][index]);
}

function typeLabel(type) {
  const key = String(type || "");
  return ITEM_TYPE_LABELS[key] || key;
}

function typeDefaultEntry(type) {
  const table = props.config.type_defaults || {};
  const entry = table[type];
  if (!entry || typeof entry !== "object") return {};
  return entry;
}

function typeDefaultNumber(type, key, fallback = 0) {
  const entry = typeDefaultEntry(type);
  const value = Number(entry[key]);
  return Number.isFinite(value) ? value : fallback;
}

function typeDefaultColor(type, key, fallback = "#f8fafc") {
  const entry = typeDefaultEntry(type);
  const value = String(entry[key] || "").trim();
  return value || fallback;
}

function typeDefaultAttr(type, key, fallback) {
  const attrs = typeDefaultEntry(type).render_attrs_map || {};
  const value = attrs[key];
  if (value === undefined || value === null) return fallback;
  return value;
}
</script>

<template>
  <section class="layout_single basic_tab">
    <div class="basic_inner">
      <n-grid cols="1 s:2" responsive="screen" :x-gap="8" :y-gap="6">
        <n-grid-item>
          <n-card title="画布配置" size="small">
            <n-form label-placement="left" :label-width="112" size="small">
              <n-grid cols="1 s:2" responsive="screen" :x-gap="8" :y-gap="2">
                <n-form-item-gi label="宽度">
                  <n-input-number
                    :value="config.width"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('width', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="高度">
                  <n-input-number
                    :value="config.height"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('height', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="内边距">
                  <n-input-number
                    :value="config.layout_padding"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('layout_padding', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="刷新间隔(ms)">
                  <n-input-number
                    :value="config.refresh_interval"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('refresh_interval', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="采集告警阈值(ms)">
                  <n-input-number
                    :value="config.collect_warn_ms"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('collect_warn_ms', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="绘制等待上限(ms)">
                  <n-input-number
                    :value="config.render_wait_max_ms"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('render_wait_max_ms', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="允许元素样式定制">
                  <n-switch
                    :value="config.allow_custom_style === true"
                    :disabled="readonlyProfile"
                    size="small"
                    @update:value="(v) => onField('allow_custom_style', !!v)"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="输出类型" :span="2">
                  <n-checkbox-group
                    :value="config.output_types || []"
                    :disabled="readonlyProfile"
                    @update:value="(v) => onField('output_types', Array.isArray(v) ? v : [])"
                  >
                    <n-space size="small" :wrap="true">
                      <n-checkbox
                        v-for="item in outputTypeOptions"
                        :key="item.value"
                        :value="item.value"
                        :label="item.label"
                      />
                    </n-space>
                  </n-checkbox-group>
                </n-form-item-gi>
              </n-grid>
            </n-form>
          </n-card>
        </n-grid-item>

        <n-grid-item>
          <n-card title="字体与颜色" size="small">
            <n-form label-placement="left" :label-width="112" size="small">
              <n-grid cols="1 s:2" responsive="screen" :x-gap="8" :y-gap="2">
                <n-form-item-gi label="默认字体" :span="2">
                  <n-select
                    :value="config.default_font"
                    :options="fontOptions"
                    filterable
                    :disabled="readonlyProfile"
                    @update:value="(v) => onField('default_font', String(v || ''))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="默认字号(px)">
                  <n-input-number
                    :value="config.default_font_size"
                    :disabled="readonlyProfile"
                    :show-button="false"
                    @update:value="(v) => onField('default_font_size', Number(v || 0))"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="字体大小(px)" :span="2">
                  <n-space size="small" :wrap="false" class="basic_inline_row">
                    <n-input-number
                      :value="config.default_unit_font_size"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      placeholder="小"
                      @update:value="(v) => onField('default_unit_font_size', Number(v || 0))"
                    />
                    <n-input-number
                      :value="config.default_label_font_size"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      placeholder="中"
                      @update:value="(v) => onField('default_label_font_size', Number(v || 0))"
                    />
                    <n-input-number
                      :value="config.default_value_font_size"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      placeholder="大"
                      @update:value="(v) => onField('default_value_font_size', Number(v || 0))"
                    />
                  </n-space>
                </n-form-item-gi>
                <n-form-item-gi label="默认颜色" :span="2">
                  <n-space size="small" :wrap="false" class="basic_inline_row">
                    <pure-color-input
                      :value="String(config.default_color || '#f8fafc')"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField('default_color', String(v || ''))"
                    />
                    <pure-color-input
                      :value="String(config.default_background || '#0b1220')"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField('default_background', String(v || ''))"
                    />
                  </n-space>
                </n-form-item-gi>
                <n-form-item-gi label="默认阈值" :span="2">
                  <n-space size="small" :wrap="false" class="basic_inline_row">
                    <n-input-number
                      :value="thresholdValue(0)"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['default_thresholds', 0], Number(v || 0))"
                    />
                    <n-input-number
                      :value="thresholdValue(1)"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['default_thresholds', 1], Number(v || 0))"
                    />
                    <n-input-number
                      :value="thresholdValue(2)"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['default_thresholds', 2], Number(v || 0))"
                    />
                    <n-input-number
                      :value="thresholdValue(3)"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['default_thresholds', 3], Number(v || 0))"
                    />
                  </n-space>
                </n-form-item-gi>
                <n-form-item-gi label="等级颜色" :span="2">
                  <n-space size="small" :wrap="false" class="basic_inline_row">
                    <pure-color-input
                      :value="levelColorValue(0)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['level_colors', 0], String(v || ''))"
                    />
                    <pure-color-input
                      :value="levelColorValue(1)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['level_colors', 1], String(v || ''))"
                    />
                    <pure-color-input
                      :value="levelColorValue(2)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['level_colors', 2], String(v || ''))"
                    />
                    <pure-color-input
                      :value="levelColorValue(3)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['level_colors', 3], String(v || ''))"
                    />
                  </n-space>
                </n-form-item-gi>
              </n-grid>
            </n-form>
          </n-card>
        </n-grid-item>
      </n-grid>

      <n-card title="按类型默认参数" size="small" style="margin-top: 8px">
        <n-table size="small" striped>
          <thead>
            <tr>
              <th style="width: 160px">类型</th>
              <th style="width: 90px">小</th>
              <th style="width: 90px">中</th>
              <th style="width: 90px">大</th>
              <th style="width: 90px">边框宽度</th>
              <th style="width: 80px">圆角</th>
              <th style="width: 64px">前景</th>
              <th style="width: 64px">背景</th>
              <th style="width: 64px">边框</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="type in typeDefaultRows" :key="type">
              <td>{{ typeLabel(type) }}</td>
              <td>
                <n-input-number
                  :value="typeDefaultNumber(type, 'small_font_size', 0)"
                  :disabled="readonlyProfile"
                  :show-button="false"
                  @update:value="(v) => onField(['type_defaults', type, 'small_font_size'], Number(v || 0))"
                />
              </td>
              <td>
                <n-input-number
                  :value="typeDefaultNumber(type, 'medium_font_size', 0)"
                  :disabled="readonlyProfile"
                  :show-button="false"
                  @update:value="(v) => onField(['type_defaults', type, 'medium_font_size'], Number(v || 0))"
                />
              </td>
              <td>
                <n-input-number
                  :value="typeDefaultNumber(type, 'large_font_size', 0)"
                  :disabled="readonlyProfile"
                  :show-button="false"
                  @update:value="(v) => onField(['type_defaults', type, 'large_font_size'], Number(v || 0))"
                />
              </td>
              <td>
                <n-input-number
                  :value="typeDefaultNumber(type, 'border_width', 0)"
                  :disabled="readonlyProfile"
                  :show-button="false"
                  @update:value="(v) => onField(['type_defaults', type, 'border_width'], Number(v || 0))"
                />
              </td>
              <td>
                <n-input-number
                  :value="typeDefaultNumber(type, 'radius', 0)"
                  :disabled="readonlyProfile"
                  :show-button="false"
                  @update:value="(v) => onField(['type_defaults', type, 'radius'], Number(v || 0))"
                />
              </td>
              <td>
                <pure-color-input
                  :value="typeDefaultColor(type, 'color', '#f8fafc')"
                  :disabled="readonlyProfile"
                  @update:value="(v) => onField(['type_defaults', type, 'color'], String(v || ''))"
                />
              </td>
              <td>
                <pure-color-input
                  :value="typeDefaultColor(type, 'bg', '#0b1220')"
                  :disabled="readonlyProfile"
                  @update:value="(v) => onField(['type_defaults', type, 'bg'], String(v || ''))"
                />
              </td>
              <td>
                <pure-color-input
                  :value="typeDefaultColor(type, 'border_color', '#475569')"
                  :disabled="readonlyProfile"
                  @update:value="(v) => onField(['type_defaults', type, 'border_color'], String(v || ''))"
                />
              </td>
            </tr>
          </tbody>
        </n-table>

        <n-divider style="margin: 8px 0" />
        <n-grid cols="1 s:2" responsive="screen" :x-gap="8" :y-gap="8">
          <n-grid-item>
            <n-space vertical size="small">
              <n-card size="small" embedded title="基础折线图默认">
                <n-form size="small" label-placement="left" :label-width="84">
                  <n-grid cols="1" :x-gap="6" :y-gap="2">
                    <n-form-item-gi label="历史数据点数">
                      <n-input-number
                        :value="Number(typeDefaultAttr('simple_line_chart', 'history_points', config.default_history_points || 150))"
                        :disabled="readonlyProfile"
                        :show-button="false"
                        @update:value="(v) => onField(['type_defaults', 'simple_line_chart', 'render_attrs_map', 'history_points'], Number(v || 0))"
                      />
                    </n-form-item-gi>
                  </n-grid>
                </n-form>
              </n-card>

              <n-card size="small" embedded title="复杂进度默认">
                <n-form size="small" label-placement="left" :label-width="84">
                  <n-grid cols="1 s:2" responsive="screen" :x-gap="6" :y-gap="2">
                    <n-form-item-gi label="样式">
                      <n-select
                        :value="String(typeDefaultAttr('full_progress', 'progress_style', 'gradient'))"
                        :options="progressStyleOptions"
                        :disabled="readonlyProfile"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'progress_style'], String(v || 'gradient'))"
                      />
                    </n-form-item-gi>
                    <n-form-item-gi label="分段数">
                      <n-input-number
                        :value="Number(typeDefaultAttr('full_progress', 'segments', 12))"
                        :disabled="readonlyProfile"
                        :show-button="false"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'segments'], Number(v || 0))"
                      />
                    </n-form-item-gi>
                    <n-form-item-gi label="标题间距">
                      <n-input-number
                        :value="Number(typeDefaultAttr('full_progress', 'body_gap', 0))"
                        :disabled="readonlyProfile"
                        :show-button="false"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'body_gap'], Number(v || 0))"
                      />
                    </n-form-item-gi>
                    <n-form-item-gi label="条高(0=铺满)">
                      <n-input-number
                        :value="Number(typeDefaultAttr('full_progress', 'bar_height', 0))"
                        :disabled="readonlyProfile"
                        :show-button="false"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'bar_height'], Number(v || 0))"
                      />
                    </n-form-item-gi>
                    <n-form-item-gi label="条圆角">
                      <n-input-number
                        :value="Number(typeDefaultAttr('full_progress', 'bar_radius', 0))"
                        :disabled="readonlyProfile"
                        :show-button="false"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'bar_radius'], Number(v || 0))"
                      />
                    </n-form-item-gi>
                    <n-form-item-gi label="轨道颜色" :span="2">
                      <pure-color-input
                        :value="String(typeDefaultAttr('full_progress', 'track_color', '#1f2937'))"
                        :disabled="readonlyProfile"
                        @update:value="(v) => onField(['type_defaults', 'full_progress', 'render_attrs_map', 'track_color'], String(v || ''))"
                      />
                    </n-form-item-gi>
                  </n-grid>
                </n-form>
              </n-card>
            </n-space>
          </n-grid-item>

          <n-grid-item>
            <n-card size="small" embedded title="复杂图表默认">
              <n-form size="small" label-placement="left" :label-width="84">
                <n-grid cols="1 s:2" responsive="screen" :x-gap="6" :y-gap="2">
                  <n-form-item-gi label="历史数据点数">
                    <n-input-number
                      :value="Number(typeDefaultAttr('full_chart', 'history_points', config.default_history_points || 150))"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'history_points'], Number(v || 0))"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="线宽">
                    <n-input-number
                      :value="Number(typeDefaultAttr('full_chart', 'line_width', 2))"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'line_width'], Number(v || 0))"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="标题间距">
                    <n-input-number
                      :value="Number(typeDefaultAttr('full_chart', 'body_gap', 4))"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'body_gap'], Number(v || 0))"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="标题分割线">
                    <n-switch
                      :value="!!typeDefaultAttr('full_chart', 'header_divider', true)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'header_divider'], !!v)"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="分段线开关">
                    <n-switch
                      :value="!!typeDefaultAttr('full_chart', 'show_segment_lines', typeDefaultAttr('full_chart', 'show_grid_lines', true))"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'show_segment_lines'], !!v)"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="填充区域">
                    <n-switch
                      :value="!!typeDefaultAttr('full_chart', 'fill_area', true)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'fill_area'], !!v)"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="显示均线">
                    <n-switch
                      :value="!!typeDefaultAttr('full_chart', 'show_avg_line', true)"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'show_avg_line'], !!v)"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="线条颜色">
                    <pure-color-input
                      :value="String(typeDefaultAttr('full_chart', 'chart_color', '#38bdf8'))"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'chart_color'], String(v || ''))"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="图表区背景色">
                    <pure-color-input
                      :value="String(typeDefaultAttr('full_chart', 'chart_area_bg', ''))"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'chart_area_bg'], String(v || ''))"
                    />
                  </n-form-item-gi>
                  <n-form-item-gi label="图表区边框色" :span="2">
                    <pure-color-input
                      :value="String(typeDefaultAttr('full_chart', 'chart_area_border_color', ''))"
                      :disabled="readonlyProfile"
                      @update:value="(v) => onField(['type_defaults', 'full_chart', 'render_attrs_map', 'chart_area_border_color'], String(v || ''))"
                    />
                  </n-form-item-gi>
                </n-grid>
              </n-form>
            </n-card>
          </n-grid-item>
        </n-grid>
      </n-card>

      <n-card title="采集器开关" size="small" style="margin-top: 8px">
        <n-table class="collector_table" size="small" striped>
          <thead>
            <tr>
              <th style="width: 30%">采集器</th>
              <th style="width: 12%">启用</th>
              <th>URL/参数</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="name in collectorNames" :key="name">
              <td>{{ name }}</td>
              <td>
                <n-switch
                  :value="!!collectorEntry(name).enabled"
                  :disabled="readonlyProfile"
                  size="small"
                  @update:value="(v) => onField(['collector_config', name, 'enabled'], !!v)"
                />
              </td>
              <td>
                <template v-if="collectorHasUrl(name)">
                  <n-space vertical size="small" style="width: 100%">
                    <n-input
                      :value="collectorUrl(name)"
                      :disabled="readonlyProfile"
                      size="small"
                      :placeholder="name === 'coolercontrol' ? 'http://127.0.0.1:11987' : 'http://127.0.0.1:8085'"
                      @update:value="(v) => onField(['collector_config', name, 'options', 'url'], String(v || ''))"
                    />
                    <n-space v-if="collectorHasAuth(name)" size="small" :wrap="false">
                      <n-input
                        :value="collectorOption(name, 'username')"
                        :disabled="readonlyProfile"
                        size="small"
                        :placeholder="collectorAuthUserLabel(name)"
                        @update:value="(v) => onField(['collector_config', name, 'options', 'username'], String(v || ''))"
                      />
                      <n-input
                        type="password"
                        show-password-on="click"
                        :value="collectorOption(name, 'password')"
                        :disabled="readonlyProfile"
                        size="small"
                        placeholder="Password"
                        @update:value="(v) => onField(['collector_config', name, 'options', 'password'], String(v || ''))"
                      />
                    </n-space>
                  </n-space>
                </template>
                <template v-else>-</template>
              </td>
            </tr>
          </tbody>
        </n-table>
      </n-card>

      <n-card size="small" style="margin-top: 8px">
        <template #header>
          <n-space justify="space-between" align="center">
            <n-text>自定义采集项</n-text>
            <n-space size="small">
              <n-button size="small" tertiary @click="emit('refresh-monitors')">刷新监控项</n-button>
              <n-button size="small" type="primary" :disabled="readonlyProfile" @click="emit('add-custom')">
                新增
              </n-button>
            </n-space>
          </n-space>
        </template>

        <n-alert type="info" :show-icon="false" style="margin-bottom: 8px">
          支持
          {{
            customTypeOptions.map((item) => item.value).join(" / ")
          }}
        </n-alert>

        <n-space vertical size="small">
          <n-card
            v-for="(item, idx) in config.custom_monitors || []"
            :key="idx"
            size="small"
            embedded
          >
            <template #header>
              <n-space justify="space-between" align="center">
                <n-text>{{ item.name || `custom_${idx + 1}` }}</n-text>
                <n-button
                  size="tiny"
                  type="error"
                  tertiary
                  :disabled="readonlyProfile"
                  @click="emit('remove-custom', idx)"
                >
                  删除
                </n-button>
              </n-space>
            </template>

            <n-form label-placement="left" :label-width="64" size="small" class="custom_monitor_form">
              <n-grid cols="1 s:2 m:4" responsive="screen" :x-gap="8" :y-gap="2">
                <n-form-item-gi label="Name">
                  <n-input
                    :value="item.name || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'name', value: String(v || '') })"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="Label">
                  <n-input
                    :value="item.label || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'label', value: String(v || '') })"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="Type">
                  <n-select
                    :value="item.type || 'file'"
                    :disabled="readonlyProfile"
                    :options="customTypeOptions"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'type', value: String(v || 'file') })"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="Unit">
                  <n-input
                    :value="item.unit || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'unit', value: String(v || '') })"
                  />
                </n-form-item-gi>

                <n-form-item-gi v-if="item.type === 'file'" label="Path" :span="4">
                  <n-input
                    :value="item.path || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'path', value: String(v || '') })"
                  />
                </n-form-item-gi>

                <n-form-item-gi v-if="item.type !== 'file'" label="Source" :span="4">
                  <n-select
                    :value="item.source || ''"
                    :disabled="readonlyProfile"
                    :options="monitorSelectOptions"
                    filterable
                    clearable
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'source', value: String(v || '') })"
                  />
                </n-form-item-gi>

                <n-form-item-gi v-if="item.type === 'mixed'" label="Sources" :span="4">
                  <n-select
                    multiple
                    filterable
                    :value="item.sources || []"
                    :disabled="readonlyProfile"
                    :options="monitorSelectOptions"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'sources', value: Array.isArray(v) ? v : [] })"
                  />
                </n-form-item-gi>

                <n-form-item-gi v-if="item.type === 'mixed'" label="Aggregate">
                  <n-select
                    :value="item.aggregate || 'max'"
                    :disabled="readonlyProfile"
                    :options="aggregateOptions"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'aggregate', value: String(v || 'max') })"
                  />
                </n-form-item-gi>
              </n-grid>
            </n-form>
          </n-card>
        </n-space>
      </n-card>
    </div>
  </section>
</template>

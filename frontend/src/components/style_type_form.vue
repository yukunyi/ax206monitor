<script setup>
import { computed } from "vue";
import PureColorInput from "./pure_color_input.vue";

const props = defineProps({
  type: { type: String, required: true },
  attrs: { type: Object, default: () => ({}) },
  disabled: { type: Boolean, default: false },
  labelWidth: { type: Number, default: 84 },
  defaultHistoryPoints: { type: Number, default: 150 },
});

const emit = defineEmits(["update-attr"]);

const progressStyleOptions = [
  { label: "gradient", value: "gradient" },
  { label: "solid", value: "solid" },
  { label: "segmented", value: "segmented" },
  { label: "stripes", value: "stripes" },
];
const lineOrientationOptions = [
  { label: "横向", value: "horizontal" },
  { label: "竖向", value: "vertical" },
];

function attrNum(key, fallback = 0) {
  const raw = props.attrs?.[key];
  const value = Number(raw);
  return Number.isFinite(value) ? value : fallback;
}

function attrBool(key, fallback = false) {
  const raw = props.attrs?.[key];
  if (raw === undefined || raw === null) return fallback;
  if (typeof raw === "boolean") return raw;
  if (typeof raw === "number") return raw !== 0;
  const text = String(raw).trim().toLowerCase();
  if (text === "1" || text === "true" || text === "yes" || text === "on") return true;
  if (text === "0" || text === "false" || text === "no" || text === "off") return false;
  return fallback;
}

function attrText(key, fallback = "") {
  const text = String(props.attrs?.[key] ?? "").trim();
  return text || fallback;
}

function attrColor(key, fallback = "#f8fafc") {
  return attrText(key, fallback);
}

function onAttrNumber(key, value, min = 0) {
  emit("update-attr", { key, value: Math.max(min, Number(value || 0)) });
}

function onAttrBool(key, value) {
  emit("update-attr", { key, value: !!value });
}

function onAttrText(key, value) {
  emit("update-attr", { key, value: String(value || "") });
}

const isSimpleLineChart = computed(() => String(props.type || "") === "simple_line_chart");
const isSimpleLine = computed(() => String(props.type || "") === "simple_line");
const isFullChart = computed(() => String(props.type || "") === "full_chart");
const isFullProgress = computed(() => String(props.type || "") === "full_progress");
const isFullGauge = computed(() => String(props.type || "") === "full_gauge");
const hasForm = computed(
  () =>
    isSimpleLineChart.value ||
    isSimpleLine.value ||
    isFullChart.value ||
    isFullProgress.value ||
    isFullGauge.value,
);
</script>

<template>
  <n-form v-if="hasForm" class="compact_style_form" label-placement="left" size="small" :label-width="labelWidth">
    <n-grid cols="2" :x-gap="4" :y-gap="1">
      <template v-if="isSimpleLineChart">
        <n-form-item-gi label="历史数据点数">
          <n-input-number
            :value="attrNum('history_points', defaultHistoryPoints)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('history_points', v, 10)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="阈值线色">
          <n-switch
            :value="attrBool('enable_threshold_colors', false)"
            :disabled="disabled"
            @update:value="(v) => onAttrBool('enable_threshold_colors', v)"
          />
        </n-form-item-gi>
      </template>

      <template v-if="isSimpleLine">
        <n-form-item-gi label="方向">
          <n-select
            :value="attrText('line_orientation', 'horizontal')"
            :options="lineOrientationOptions"
            :disabled="disabled"
            @update:value="(v) => onAttrText('line_orientation', v || 'horizontal')"
          />
        </n-form-item-gi>
        <n-form-item-gi label="线宽">
          <n-input-number
            :value="attrNum('line_width', 1)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('line_width', v, 1)"
          />
        </n-form-item-gi>
      </template>

      <template v-if="isFullChart || isFullProgress">
        <n-form-item-gi label="内边距">
          <n-input-number
            :value="attrNum('content_padding', 1)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('content_padding', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="标题间距">
          <n-input-number
            :value="attrNum('body_gap', isFullChart ? 4 : 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('body_gap', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="标题字号">
          <n-input-number
            :value="attrNum('title_font_size', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('title_font_size', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="数值字号">
          <n-input-number
            :value="attrNum('value_font_size', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('value_font_size', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="头部分隔线">
          <n-switch
            :value="attrBool('header_divider', true)"
            :disabled="disabled"
            @update:value="(v) => onAttrBool('header_divider', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分隔线宽度">
          <n-input-number
            :value="attrNum('header_divider_width', 1)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('header_divider_width', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分隔偏移">
          <n-input-number
            :value="attrNum('header_divider_offset', 3)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('header_divider_offset', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分隔线色">
          <pure-color-input
            :value="attrColor('header_divider_color', '#94a3b840')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('header_divider_color', v)"
          />
        </n-form-item-gi>
      </template>

      <template v-if="isFullChart">
        <n-form-item-gi label="历史数据点数">
          <n-input-number
            :value="attrNum('history_points', defaultHistoryPoints)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('history_points', v, 10)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="网格线数">
          <n-input-number
            :value="attrNum('grid_lines', 4)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('grid_lines', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分段线开关">
          <n-switch
            :value="attrBool('show_segment_lines', attrBool('show_grid_lines', true))"
            :disabled="disabled"
            @update:value="(v) => onAttrBool('show_segment_lines', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="显示均线">
          <n-switch
            :value="attrBool('show_avg_line', true)"
            :disabled="disabled"
            @update:value="(v) => onAttrBool('show_avg_line', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="阈值线色">
          <n-switch
            :value="attrBool('enable_threshold_colors', false)"
            :disabled="disabled"
            @update:value="(v) => onAttrBool('enable_threshold_colors', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="线宽">
          <n-input-number
            :value="attrNum('line_width', 2)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('line_width', v, 1)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="图线颜色">
          <pure-color-input
            :value="attrColor('chart_color', '#38bdf8')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('chart_color', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="图表区背景色">
          <pure-color-input
            :value="attrColor('chart_area_bg', '')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('chart_area_bg', v)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="图表区边框色">
          <pure-color-input
            :value="attrColor('chart_area_border_color', '')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('chart_area_border_color', v)"
          />
        </n-form-item-gi>
      </template>

      <template v-if="isFullProgress">
        <n-form-item-gi label="进度样式">
          <n-select
            :value="attrText('progress_style', 'gradient')"
            :options="progressStyleOptions"
            :disabled="disabled"
            @update:value="(v) => onAttrText('progress_style', v || 'gradient')"
          />
        </n-form-item-gi>
        <n-form-item-gi label="条高(0=铺满)">
          <n-input-number
            :value="attrNum('bar_height', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('bar_height', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="条圆角">
          <n-input-number
            :value="attrNum('bar_radius', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('bar_radius', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分段数量">
          <n-input-number
            :value="attrNum('segments', 12)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('segments', v, 4)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="分段间隔">
          <n-input-number
            :value="attrNum('segment_gap', 2)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('segment_gap', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="轨道颜色">
          <pure-color-input
            :value="attrColor('track_color', '#1f2937')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('track_color', v)"
          />
        </n-form-item-gi>
      </template>

      <template v-if="isFullGauge">
        <n-form-item-gi label="内边距">
          <n-input-number
            :value="attrNum('content_padding', 1)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('content_padding', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="值字号">
          <n-input-number
            :value="attrNum('value_font_size', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('value_font_size', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="标签字号">
          <n-input-number
            :value="attrNum('label_font_size', 0)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('label_font_size', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="圆环宽度">
          <n-input-number
            :value="attrNum('gauge_thickness', 10)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('gauge_thickness', v, 2)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="底部缺口角度">
          <n-input-number
            :value="attrNum('gauge_gap_degrees', 76)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('gauge_gap_degrees', Math.min(260, Math.max(20, Number(v || 0))), 20)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="文字行间距">
          <n-input-number
            :value="attrNum('gauge_text_gap', 4)"
            :disabled="disabled"
            :show-button="false"
            @update:value="(v) => onAttrNumber('gauge_text_gap', v, 0)"
          />
        </n-form-item-gi>
        <n-form-item-gi label="轨道颜色">
          <pure-color-input
            :value="attrColor('track_color', '#1f2937')"
            :disabled="disabled"
            @update:value="(v) => onAttrText('track_color', v)"
          />
        </n-form-item-gi>
      </template>
    </n-grid>
  </n-form>
</template>

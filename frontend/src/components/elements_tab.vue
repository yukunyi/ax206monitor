<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import PureColorInput from "./pure_color_input.vue";

const props = defineProps({
  config: { type: Object, required: true },
  meta: { type: Object, default: () => ({}) },
  selectedIndex: { type: Number, default: -1 },
  readonlyProfile: { type: Boolean, default: false },
  monitorOptions: { type: Array, default: () => [] },
  previewUrl: { type: String, default: "" },
  previewSync: { type: Boolean, default: true },
  zoomAuto: { type: Boolean, default: true },
  zoom: { type: Number, default: 100 },
});

const emit = defineEmits([
  "select-item",
  "add-item",
  "refresh-monitors",
  "change-preview-sync",
  "clone-item",
  "remove-item",
  "move-item-up",
  "move-item-down",
  "patch-item",
  "change-item-field",
  "change-zoom-auto",
  "change-zoom",
  "fit-scale",
]);

const wrapperRef = ref(null);
const fitScale = ref(1);
const AUTO_FIT_MARGIN = 20;
const SNAP_CENTER_THRESHOLD = 6;
const addType = ref("simple_value");
const addMonitor = ref("");

let resizeObserver = null;
let drag = null;

const selectedItem = computed(() => props.config.items?.[props.selectedIndex] || null);
const globalAllowCustomStyle = computed(() => props.config?.allow_custom_style === true);
const selectedCustomStyleEnabled = computed(
  () => globalAllowCustomStyle.value && selectedItem.value?.custom_style === true,
);

const previewScale = computed(() => {
  if (props.zoomAuto) return fitScale.value;
  return Math.max(0.1, Number(props.zoom || 100) / 100);
});

const canvasStyle = computed(() => ({
  width: `${props.config.width * previewScale.value}px`,
  height: `${props.config.height * previewScale.value}px`,
}));

const TYPE_LABELS = {
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

function typeLabel(type) {
  const key = String(type || "");
  return TYPE_LABELS[key] || key;
}

const itemTypeOptions = computed(() => {
  const fallback = [
    "simple_value",
    "simple_progress",
    "simple_line_chart",
    "simple_label",
    "simple_rect",
    "simple_circle",
    "label_text",
    "full_chart",
    "full_progress",
  ];
  const list = Array.isArray(props.meta?.item_types) && props.meta.item_types.length > 0 ? props.meta.item_types : fallback;
  return list
    .map((item) => {
      if (typeof item === "string") {
        return { label: typeLabel(item), value: item };
      }
      const value = String(item?.value || "");
      if (!value) return null;
      const label = String(item?.label || typeLabel(value));
      return { label, value };
    })
    .filter(Boolean);
});

const monitorSelectOptions = computed(() => props.monitorOptions || []);

const selectedMonitorRequired = computed(() => {
  const type = String(selectedItem.value?.type || "");
  return (
    type === "simple_value" ||
    type === "simple_progress" ||
    type === "simple_line_chart" ||
    type === "label_text" ||
    type === "full_chart" ||
    type === "full_progress"
  );
});

const addMonitorRequired = computed(() => {
  const type = String(addType.value || "");
  return (
    type === "simple_value" ||
    type === "simple_progress" ||
    type === "simple_line_chart" ||
    type === "label_text" ||
    type === "full_chart" ||
    type === "full_progress"
  );
});

const FULL_PROGRESS_STYLE_OPTIONS = [
  { label: "渐变", value: "gradient" },
  { label: "纯色", value: "solid" },
  { label: "分段", value: "segmented" },
  { label: "条纹", value: "stripes" },
];

const selectedType = computed(() => String(selectedItem.value?.type || ""));
const selectedIsLabelText = computed(() => selectedType.value === "label_text");
const selectedIsSimpleLabel = computed(() => selectedType.value === "simple_label");
const selectedIsSimpleLine = computed(() => selectedType.value === "simple_line_chart");
const selectedIsFullChart = computed(() => selectedType.value === "full_chart");
const selectedIsFullProgress = computed(() => selectedType.value === "full_progress");
const selectedHasTitle = computed(
  () => selectedType.value === "full_chart" || selectedType.value === "full_progress",
);

function clamp(v, min, max) {
  return Math.max(min, Math.min(max, v));
}

function toNumber(v, fallback = 0) {
  const n = Number(v);
  if (Number.isFinite(n)) return n;
  return fallback;
}

function toOptionalNumber(v) {
  if (v === null || v === undefined || v === "") return null;
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return n;
}

function colorValue(raw, fallback = "#f8fafc") {
  const value = String(raw || "").trim();
  return value || fallback;
}

function normalizeText(raw) {
  return String(raw || "").trim();
}

function monitorDisplayLabel(monitorName) {
  const name = normalizeText(monitorName);
  if (!name) return "";
  const option = (monitorSelectOptions.value || []).find((item) => String(item?.value || "") === name);
  const rawLabel = normalizeText(option?.label || "");
  if (!rawLabel) return "";
  const suffix = ` (${name})`;
  if (rawLabel.endsWith(suffix)) {
    return normalizeText(rawLabel.slice(0, rawLabel.length - suffix.length));
  }
  if (rawLabel === name) return "";
  return rawLabel;
}

function fallbackItemName(item) {
  if (!item || typeof item !== "object") return "";
  const monitorLabel = monitorDisplayLabel(item.monitor);
  if (monitorLabel) return monitorLabel;
  const rawLabel =
    normalizeText(item.render_attrs_map?.label) ||
    normalizeText(item.renderAttrsMap?.label) ||
    normalizeText(item.label);
  if (rawLabel) return rawLabel;
  const text = normalizeText(item.text);
  if (text) return text;
  return normalizeText(item.type) || "item";
}

function displayItemName(item) {
  const manual = normalizeText(item?.edit_ui_name);
  if (manual) return manual;
  return fallbackItemName(item);
}

function applyCenterSnap(x, y, width, height, currentIndex) {
  const items = Array.isArray(props.config?.items) ? props.config.items : [];
  if (items.length <= 1) return { x, y };

  const centerX = x + width / 2;
  const centerY = y + height / 2;

  let bestShiftX = null;
  let bestShiftY = null;
  let bestAbsX = Number.POSITIVE_INFINITY;
  let bestAbsY = Number.POSITIVE_INFINITY;

  items.forEach((item, index) => {
    if (index === currentIndex || !item) return;
    const otherX = toNumber(item.x, 0);
    const otherY = toNumber(item.y, 0);
    const otherWidth = Math.max(1, toNumber(item.width, 10));
    const otherHeight = Math.max(1, toNumber(item.height, 10));
    const otherCenterX = otherX + otherWidth / 2;
    const otherCenterY = otherY + otherHeight / 2;
    const shiftX = otherCenterX - centerX;
    const shiftY = otherCenterY - centerY;

    const absShiftX = Math.abs(shiftX);
    if (absShiftX <= SNAP_CENTER_THRESHOLD && absShiftX < bestAbsX) {
      bestShiftX = shiftX;
      bestAbsX = absShiftX;
    }
    const absShiftY = Math.abs(shiftY);
    if (absShiftY <= SNAP_CENTER_THRESHOLD && absShiftY < bestAbsY) {
      bestShiftY = shiftY;
      bestAbsY = absShiftY;
    }
  });

  const snappedX = bestShiftX === null ? x : x + bestShiftX;
  const snappedY = bestShiftY === null ? y : y + bestShiftY;
  return { x: snappedX, y: snappedY };
}

function updateFitScale() {
  const el = wrapperRef.value;
  if (!el || !props.config?.width || !props.config?.height) return;
  const rect = el.getBoundingClientRect();
  const innerWidth = Math.max(1, rect.width - AUTO_FIT_MARGIN * 2);
  const innerHeight = Math.max(1, rect.height - AUTO_FIT_MARGIN * 2);
  const sx = innerWidth / props.config.width;
  const sy = innerHeight / props.config.height;
  const scale = Math.max(0.1, Math.min(sx, sy));
  fitScale.value = scale;
  emit("fit-scale", scale);
}

function rectStyle(item, index) {
  return {
    left: `${toNumber(item.x, 0) * previewScale.value}px`,
    top: `${toNumber(item.y, 0) * previewScale.value}px`,
    width: `${toNumber(item.width, 10) * previewScale.value}px`,
    height: `${toNumber(item.height, 10) * previewScale.value}px`,
    borderColor: index === props.selectedIndex ? "#2080f0" : "rgba(255,255,255,0.35)",
  };
}

function startDrag(event, index, mode, handle = "") {
  if (props.readonlyProfile) return;
  const item = props.config.items?.[index];
  if (!item) return;
  event.preventDefault();
  event.stopPropagation();
  drag = {
    index,
    mode,
    handle,
    startX: event.clientX,
    startY: event.clientY,
    base: {
      x: toNumber(item.x, 0),
      y: toNumber(item.y, 0),
      width: toNumber(item.width, 10),
      height: toNumber(item.height, 10),
    },
  };
  window.addEventListener("pointermove", onPointerMove);
  window.addEventListener("pointerup", onPointerUp);
}

function onPointerMove(event) {
  if (!drag) return;
  const item = props.config.items?.[drag.index];
  if (!item) return;
  const dx = (event.clientX - drag.startX) / previewScale.value;
  const dy = (event.clientY - drag.startY) / previewScale.value;

  let x = drag.base.x;
  let y = drag.base.y;
  let width = drag.base.width;
  let height = drag.base.height;

  if (drag.mode === "move") {
    x += dx;
    y += dy;
    const snapped = applyCenterSnap(x, y, width, height, drag.index);
    x = snapped.x;
    y = snapped.y;
  } else {
    if (drag.handle.includes("e")) width += dx;
    if (drag.handle.includes("s")) height += dy;
    if (drag.handle.includes("w")) {
      x += dx;
      width -= dx;
    }
    if (drag.handle.includes("n")) {
      y += dy;
      height -= dy;
    }
  }

  width = Math.max(10, width);
  height = Math.max(10, height);
  x = clamp(x, 0, Math.max(0, props.config.width - width));
  y = clamp(y, 0, Math.max(0, props.config.height - height));

  emit("patch-item", {
    index: drag.index,
    patch: {
      x: Math.round(x),
      y: Math.round(y),
      width: Math.round(width),
      height: Math.round(height),
    },
  });
}

function onPointerUp() {
  drag = null;
  window.removeEventListener("pointermove", onPointerMove);
  window.removeEventListener("pointerup", onPointerUp);
}

function nudge(stepX, stepY) {
  if (props.readonlyProfile || props.selectedIndex < 0) return;
  const item = selectedItem.value;
  if (!item) return;
  const width = toNumber(item.width, 10);
  const height = toNumber(item.height, 10);
  const x = clamp(toNumber(item.x, 0) + stepX, 0, Math.max(0, props.config.width - width));
  const y = clamp(toNumber(item.y, 0) + stepY, 0, Math.max(0, props.config.height - height));
  emit("patch-item", {
    index: props.selectedIndex,
    patch: { x: Math.round(x), y: Math.round(y) },
  });
}

function resolveRenderAttrs(item) {
  if (!item || typeof item !== "object") return {};
  const map = item.render_attrs_map;
  if (map && typeof map === "object") return map;
  const legacyMap = item.renderAttrsMap;
  if (legacyMap && typeof legacyMap === "object") return legacyMap;
  return {};
}

function renderAttrRaw(key) {
  const attrs = resolveRenderAttrs(selectedItem.value);
  return attrs[key];
}

function renderAttrString(key, fallback = "") {
  const raw = renderAttrRaw(key);
  if (raw === undefined || raw === null) return fallback;
  return String(raw);
}

function renderAttrNumber(key, fallback = 0) {
  const raw = renderAttrRaw(key);
  if (raw === undefined || raw === null || raw === "") return fallback;
  const n = Number(raw);
  if (!Number.isFinite(n)) return fallback;
  return n;
}

function renderAttrBool(key, fallback = false) {
  const raw = renderAttrRaw(key);
  if (raw === undefined || raw === null) return fallback;
  if (typeof raw === "boolean") return raw;
  if (typeof raw === "number") return raw !== 0;
  const text = String(raw).trim().toLowerCase();
  if (text === "1" || text === "true" || text === "yes" || text === "on") return true;
  if (text === "0" || text === "false" || text === "no" || text === "off") return false;
  return fallback;
}

function renderAttrColor(key, fallback = "#f8fafc") {
  return colorValue(renderAttrString(key, ""), fallback);
}

function updateRenderAttr(key, value) {
  if (props.readonlyProfile || props.selectedIndex < 0) return;
  const item = selectedItem.value;
  if (!item) return;
  const next = { ...resolveRenderAttrs(item) };
  const shouldDelete =
    value === undefined || value === null || (typeof value === "string" && value.trim() === "");
  if (shouldDelete) delete next[key];
  else next[key] = value;
  emit("patch-item", {
    index: props.selectedIndex,
    patch: { render_attrs_map: next },
  });
}

function updateRenderAttrNumber(key, value, fallback = 0, min = null, max = null) {
  let n = toNumber(value, fallback);
  if (min !== null) n = Math.max(min, n);
  if (max !== null) n = Math.min(max, n);
  updateRenderAttr(key, n);
}

function submitAdd() {
  if (addMonitorRequired.value && !String(addMonitor.value || "").trim()) return;
  emit("add-item", {
    type: String(addType.value || "simple_value"),
    monitor: String(addMonitor.value || ""),
  });
  if (addMonitorRequired.value) addMonitor.value = "";
}

onMounted(() => {
  updateFitScale();
  if (typeof ResizeObserver !== "undefined") {
    resizeObserver = new ResizeObserver(() => updateFitScale());
    if (wrapperRef.value) resizeObserver.observe(wrapperRef.value);
  }
  window.addEventListener("resize", updateFitScale);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", updateFitScale);
  if (resizeObserver) {
    resizeObserver.disconnect();
    resizeObserver = null;
  }
  onPointerUp();
});

watch(
  () => [props.config.width, props.config.height],
  () => updateFitScale(),
);
</script>

<template>
  <section class="elements_layout" :class="{ readonly_preview: readonlyProfile }">
    <n-card class="list_panel" size="small" title="屏幕元素">
      <template #header-extra>
        <n-space size="small">
          <n-button size="tiny" :disabled="readonlyProfile || selectedIndex < 0" @click="emit('clone-item')">复制</n-button>
          <n-button
            size="tiny"
            type="error"
            tertiary
            :disabled="readonlyProfile || selectedIndex < 0"
            @click="emit('remove-item')"
          >
            删除
          </n-button>
        </n-space>
      </template>

      <div v-if="!readonlyProfile" class="add_item_form">
        <n-button size="tiny" tertiary style="margin-bottom: 6px" @click="emit('refresh-monitors')">
          刷新监控项
        </n-button>
        <n-form label-placement="top" size="small">
          <n-form-item label="新增类型">
            <n-select v-model:value="addType" :options="itemTypeOptions" />
          </n-form-item>
          <n-form-item label="监控项" :required="addMonitorRequired">
            <n-select
              v-model:value="addMonitor"
              filterable
              clearable
              :disabled="!addMonitorRequired"
              :options="monitorSelectOptions"
              :status="addMonitorRequired && !addMonitor ? 'error' : undefined"
              :placeholder="addMonitorRequired ? '请选择监控项' : '当前类型无需监控项'"
            />
          </n-form-item>
          <n-button
            size="small"
            type="primary"
            block
            :disabled="addMonitorRequired && !addMonitor"
            @click="submitAdd"
          >
            新增
          </n-button>
        </n-form>
      </div>

      <n-scrollbar class="elements_list">
        <n-space vertical :size="2">
          <button
            v-for="(item, idx) in config.items"
            :key="`${idx}_${item.edit_ui_name || item.type}`"
            type="button"
            class="list_item"
            :class="{ active: idx === selectedIndex }"
            @click="emit('select-item', idx)"
          >
            <span class="item_name">{{ idx + 1 }}. {{ displayItemName(item) }}</span>
            <small class="item_type">{{ typeLabel(item.type) }}</small>
          </button>
        </n-space>
      </n-scrollbar>

      <n-space size="small" style="margin-top: 8px">
        <n-button size="tiny" :disabled="readonlyProfile || selectedIndex <= 0" @click="emit('move-item-up')">上移</n-button>
        <n-button
          size="tiny"
          :disabled="readonlyProfile || selectedIndex < 0 || selectedIndex >= config.items.length - 1"
          @click="emit('move-item-down')"
        >
          下移
        </n-button>
      </n-space>
    </n-card>

    <n-card class="preview_panel" size="small" title="预览">
      <template #header-extra>
        <n-space align="center" size="small">
          <n-checkbox
            :checked="previewSync"
            @update:checked="(v) => emit('change-preview-sync', !!v)"
          >
            同步预览
          </n-checkbox>
          <n-switch
            size="small"
            :value="zoomAuto"
            @update:value="(v) => emit('change-zoom-auto', !!v)"
          >
            <template #checked>自动缩放</template>
            <template #unchecked>自动缩放</template>
          </n-switch>
          <n-slider
            style="width: 150px"
            size="small"
            :min="25"
            :max="400"
            :step="5"
            :value="zoom"
            :disabled="zoomAuto"
            @update:value="(v) => emit('change-zoom', toNumber(v, 100))"
          />
          <n-text depth="3">{{ zoom }}%</n-text>
        </n-space>
      </template>

      <div ref="wrapperRef" class="preview_wrapper">
        <div class="preview_canvas" :style="canvasStyle">
          <img v-if="previewUrl" :src="previewUrl" alt="preview" class="preview_image" />
          <div
            v-for="(item, idx) in config.items"
            :key="`${idx}_${item.type}`"
            class="item_rect"
            :class="{ selected: idx === selectedIndex }"
            :style="rectStyle(item, idx)"
            @click.stop="emit('select-item', idx)"
            @pointerdown="startDrag($event, idx, 'move')"
          >
            <span class="item_tag">{{ idx + 1 }}</span>
            <template v-if="idx === selectedIndex && !readonlyProfile">
              <span class="handle nw" @pointerdown="startDrag($event, idx, 'resize', 'nw')" />
              <span class="handle ne" @pointerdown="startDrag($event, idx, 'resize', 'ne')" />
              <span class="handle sw" @pointerdown="startDrag($event, idx, 'resize', 'sw')" />
              <span class="handle se" @pointerdown="startDrag($event, idx, 'resize', 'se')" />
            </template>
          </div>
        </div>
      </div>
    </n-card>

    <n-card v-if="!readonlyProfile" class="editor_panel" size="small" title="元素编辑">
      <template v-if="selectedItem">
        <n-form label-placement="left" size="small" :label-width="84">
          <n-grid cols="2" :x-gap="6" :y-gap="2">
            <n-form-item-gi label="名称" :span="2">
              <n-input
                :value="selectedItem.edit_ui_name || ''"
                @update:value="(v) => emit('change-item-field', { field: 'edit_ui_name', value: String(v || '') })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="类型" :span="2">
              <n-select
                style="width: 100%"
                :value="selectedItem.type"
                :options="itemTypeOptions"
                @update:value="(v) => emit('change-item-field', { field: 'type', value: String(v || '') })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="监控项" :span="2">
              <n-select
                filterable
                :clearable="!selectedMonitorRequired"
                :value="selectedItem.monitor || ''"
                :options="monitorSelectOptions"
                :status="selectedMonitorRequired && !selectedItem.monitor ? 'error' : undefined"
                :placeholder="selectedMonitorRequired ? '请选择监控项' : '可选'"
                @update:value="(v) => emit('change-item-field', { field: 'monitor', value: String(v || '') })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="样式定制" :span="2">
              <n-space align="center" size="small">
                <n-switch
                  :value="!!selectedItem.custom_style"
                  :disabled="!globalAllowCustomStyle"
                  @update:value="(v) => emit('change-item-field', { field: 'custom_style', value: !!v })"
                />
                <n-text v-if="!globalAllowCustomStyle" depth="3">基础配置未开启样式定制</n-text>
              </n-space>
            </n-form-item-gi>
            <n-form-item-gi label="X">
              <n-input-number
                :value="toNumber(selectedItem.x, 0)"
                :show-button="false"
                @update:value="(v) => emit('change-item-field', { field: 'x', value: toNumber(v, 0) })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="Y">
              <n-input-number
                :value="toNumber(selectedItem.y, 0)"
                :show-button="false"
                @update:value="(v) => emit('change-item-field', { field: 'y', value: toNumber(v, 0) })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="宽度">
              <n-input-number
                :value="toNumber(selectedItem.width, 10)"
                :show-button="false"
                @update:value="(v) => emit('change-item-field', { field: 'width', value: toNumber(v, 10) })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="高度">
              <n-input-number
                :value="toNumber(selectedItem.height, 10)"
                :show-button="false"
                @update:value="(v) => emit('change-item-field', { field: 'height', value: toNumber(v, 10) })"
              />
            </n-form-item-gi>
            <n-form-item-gi v-if="selectedHasTitle" label="标题">
              <n-input
                :value="renderAttrString('title', '')"
                @update:value="(v) => updateRenderAttr('title', String(v || ''))"
              />
            </n-form-item-gi>
            <n-form-item-gi v-if="selectedHasTitle" label="单位">
              <n-input
                :value="selectedItem.unit || ''"
                @update:value="(v) => emit('change-item-field', { field: 'unit', value: String(v || '') })"
              />
            </n-form-item-gi>
            <n-form-item-gi v-else label="单位" :span="2">
              <n-input
                :value="selectedItem.unit || ''"
                @update:value="(v) => emit('change-item-field', { field: 'unit', value: String(v || '') })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="最小值">
              <n-input-number
                clearable
                :show-button="false"
                :value="selectedItem.min_value ?? null"
                @update:value="(v) => emit('change-item-field', { field: 'min_value', value: toOptionalNumber(v) })"
              />
            </n-form-item-gi>
            <n-form-item-gi label="最大值">
              <n-input-number
                clearable
                :show-button="false"
                :value="selectedItem.max_value ?? null"
                @update:value="(v) => emit('change-item-field', { field: 'max_value', value: toOptionalNumber(v) })"
              />
            </n-form-item-gi>
            <n-form-item-gi v-if="selectedIsLabelText" label="标签" :span="2">
              <n-input
                :value="renderAttrString('label', '')"
                @update:value="(v) => updateRenderAttr('label', String(v || ''))"
              />
            </n-form-item-gi>
            <n-form-item-gi v-else-if="selectedIsSimpleLabel" label="标签" :span="2">
              <n-input
                :value="selectedItem.text || ''"
                @update:value="(v) => emit('change-item-field', { field: 'text', value: String(v || '') })"
              />
            </n-form-item-gi>
            <template v-if="selectedCustomStyleEnabled">
              <n-form-item-gi v-if="selectedIsSimpleLine" label="历史数据点数">
                <n-input-number
                  :value="renderAttrNumber('history_points', 150)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('history_points', v, 150, 10, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="字号">
                <n-input-number
                  :value="toNumber(selectedItem.font_size, 0)"
                  :show-button="false"
                  @update:value="(v) => emit('change-item-field', { field: 'font_size', value: toNumber(v, 0) })"
                />
              </n-form-item-gi>
              <n-form-item-gi label="前景色">
                <pure-color-input
                  :value="colorValue(selectedItem.color, '#f8fafc')"
                  @update:value="(v) => emit('change-item-field', { field: 'color', value: String(v || '') })"
                />
              </n-form-item-gi>
              <n-form-item-gi label="背景色">
                <pure-color-input
                  :value="colorValue(selectedItem.bg, '#0b1220')"
                  @update:value="(v) => emit('change-item-field', { field: 'bg', value: String(v || '') })"
                />
              </n-form-item-gi>
              <n-form-item-gi label="单位色">
                <pure-color-input
                  :value="colorValue(selectedItem.unit_color, '#f8fafc')"
                  @update:value="(v) => emit('change-item-field', { field: 'unit_color', value: String(v || '') })"
                />
              </n-form-item-gi>
            </template>
          </n-grid>
        </n-form>

        <template v-if="selectedIsFullChart && selectedCustomStyleEnabled">
          <n-divider style="margin: 8px 0" />
          <n-text depth="3" style="display: block; margin-bottom: 6px">复杂图表配置</n-text>
          <n-form label-placement="left" size="small" :label-width="84">
            <n-grid cols="2" :x-gap="6" :y-gap="2">
              <n-form-item-gi label="内边距">
                <n-input-number
                  :value="renderAttrNumber('content_padding', 1)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('content_padding', v, 1, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分隔偏移">
                <n-input-number
                  :value="renderAttrNumber('header_divider_offset', 3)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('header_divider_offset', v, 3, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="标题间距">
                <n-input-number
                  :value="renderAttrNumber('body_gap', 4)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('body_gap', v, 4, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="标题字号">
                <n-input-number
                  :value="renderAttrNumber('title_font_size', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('title_font_size', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="数值字号">
                <n-input-number
                  :value="renderAttrNumber('value_font_size', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('value_font_size', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="头部分隔线">
                <n-switch
                  :value="renderAttrBool('header_divider', true)"
                  @update:value="(v) => updateRenderAttr('header_divider', !!v)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分隔线色">
                <pure-color-input
                  :value="renderAttrColor('header_divider_color', '#94a3b840')"
                  @update:value="(v) => updateRenderAttr('header_divider_color', String(v || ''))"
                />
              </n-form-item-gi>

              <n-form-item-gi label="历史数据点数">
                <n-input-number
                  :value="renderAttrNumber('history_points', 150)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('history_points', v, 150, 10, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="网格线数">
                <n-input-number
                  :value="renderAttrNumber('grid_lines', 4)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('grid_lines', v, 4, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分段线开关">
                <n-switch
                  :value="renderAttrBool('show_segment_lines', renderAttrBool('show_grid_lines', true))"
                  @update:value="(v) => updateRenderAttr('show_segment_lines', !!v)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="填充区域">
                <n-switch
                  :value="renderAttrBool('fill_area', true)"
                  @update:value="(v) => updateRenderAttr('fill_area', !!v)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="显示均线">
                <n-switch
                  :value="renderAttrBool('show_avg_line', true)"
                  @update:value="(v) => updateRenderAttr('show_avg_line', !!v)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="线宽">
                <n-input-number
                  :value="renderAttrNumber('line_width', 2)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('line_width', v, 2, 1, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="图线颜色">
                <pure-color-input
                  :value="renderAttrColor('chart_color', '#20a0f0')"
                  @update:value="(v) => updateRenderAttr('chart_color', String(v || ''))"
                />
              </n-form-item-gi>
              <n-form-item-gi label="图表区背景色">
                <pure-color-input
                  :value="renderAttrColor('chart_area_bg', '')"
                  @update:value="(v) => updateRenderAttr('chart_area_bg', String(v || ''))"
                />
              </n-form-item-gi>
              <n-form-item-gi label="图表区边框色">
                <pure-color-input
                  :value="renderAttrColor('chart_area_border_color', '')"
                  @update:value="(v) => updateRenderAttr('chart_area_border_color', String(v || ''))"
                />
              </n-form-item-gi>
            </n-grid>
          </n-form>
        </template>

        <template v-if="selectedIsFullProgress && selectedCustomStyleEnabled">
          <n-divider style="margin: 8px 0" />
          <n-text depth="3" style="display: block; margin-bottom: 6px">复杂进度配置</n-text>
          <n-form label-placement="left" size="small" :label-width="84">
            <n-grid cols="2" :x-gap="6" :y-gap="2">
              <n-form-item-gi label="内边距">
                <n-input-number
                  :value="renderAttrNumber('content_padding', 1)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('content_padding', v, 1, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分隔偏移">
                <n-input-number
                  :value="renderAttrNumber('header_divider_offset', 3)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('header_divider_offset', v, 3, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="标题间距">
                <n-input-number
                  :value="renderAttrNumber('body_gap', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('body_gap', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="标题字号">
                <n-input-number
                  :value="renderAttrNumber('title_font_size', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('title_font_size', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="数值字号">
                <n-input-number
                  :value="renderAttrNumber('value_font_size', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('value_font_size', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="头部分隔线">
                <n-switch
                  :value="renderAttrBool('header_divider', true)"
                  @update:value="(v) => updateRenderAttr('header_divider', !!v)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分隔线色">
                <pure-color-input
                  :value="renderAttrColor('header_divider_color', '#94a3b840')"
                  @update:value="(v) => updateRenderAttr('header_divider_color', String(v || ''))"
                />
              </n-form-item-gi>

              <n-form-item-gi label="进度样式" :span="2">
                <n-select
                  :value="renderAttrString('progress_style', 'gradient')"
                  :options="FULL_PROGRESS_STYLE_OPTIONS"
                  @update:value="(v) => updateRenderAttr('progress_style', String(v || 'gradient'))"
                />
              </n-form-item-gi>
              <n-form-item-gi label="条高">
                <n-input-number
                  :value="renderAttrNumber('bar_height', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('bar_height', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="条圆角">
                <n-input-number
                  :value="renderAttrNumber('bar_radius', 0)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('bar_radius', v, 0, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分段数量">
                <n-input-number
                  :value="renderAttrNumber('segments', 12)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('segments', v, 12, 4, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="分段间隔">
                <n-input-number
                  :value="renderAttrNumber('segment_gap', 2)"
                  :show-button="false"
                  @update:value="(v) => updateRenderAttrNumber('segment_gap', v, 2, 0, null)"
                />
              </n-form-item-gi>
              <n-form-item-gi label="轨道颜色">
                <pure-color-input
                  :value="renderAttrColor('track_color', '#1f2937')"
                  @update:value="(v) => updateRenderAttr('track_color', String(v || ''))"
                />
              </n-form-item-gi>
            </n-grid>
          </n-form>
        </template>

        <n-divider style="margin: 8px 0" />
        <n-text depth="3" style="display: block; margin-bottom: 6px">快速调整</n-text>
        <div class="nudge_cross">
          <div class="nudge_y_axis nudge_vertical">
            <n-button size="tiny" @click="nudge(0, -5)">5</n-button>
            <n-button size="tiny" @click="nudge(0, -1)">1</n-button>
          </div>
          <div class="nudge_mid">
            <div class="nudge_left">
              <n-button size="tiny" @click="nudge(-5, 0)">5</n-button>
              <n-button size="tiny" @click="nudge(-1, 0)">1</n-button>
            </div>
            <div class="nudge_center">移动区域</div>
            <div class="nudge_right">
              <n-button size="tiny" @click="nudge(1, 0)">1</n-button>
              <n-button size="tiny" @click="nudge(5, 0)">5</n-button>
            </div>
          </div>
          <div class="nudge_y_axis nudge_vertical">
            <n-button size="tiny" @click="nudge(0, 1)">1</n-button>
            <n-button size="tiny" @click="nudge(0, 5)">5</n-button>
          </div>
        </div>
      </template>
      <n-empty v-else size="small" description="请选择一个元素" />
    </n-card>
  </section>
</template>

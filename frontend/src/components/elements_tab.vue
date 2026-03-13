<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import StyleManagerForm from "./style_manager_form.vue";
import { patchObjectKey } from "../composables/object_patch";
import { buildItemTypeOptions, getItemTypeLabel, isMonitorRequiredType } from "../item_types";

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

const itemTypeOptions = computed(() => buildItemTypeOptions(props.meta?.item_types));

const monitorSelectOptions = computed(() => props.monitorOptions || []);

const selectedMonitorRequired = computed(() => {
  const type = String(selectedItem.value?.type || "");
  return isMonitorRequiredType(type);
});

const addMonitorRequired = computed(() => {
  const type = String(addType.value || "");
  return isMonitorRequiredType(type);
});

const selectedType = computed(() => String(selectedItem.value?.type || ""));
const selectedIsLabelText = computed(() => selectedType.value === "label_text");
const selectedIsSimpleLabel = computed(() => selectedType.value === "simple_label");
const selectedHasTitle = computed(
  () =>
    selectedType.value === "full_chart" ||
    selectedType.value === "full_progress" ||
    selectedType.value === "full_gauge",
);
const selectedSupportsFormat = computed(() => {
  const monitor = normalizeText(selectedItem.value?.monitor);
  if (!monitor) return false;
  return (
    monitor === "alias.system.time" ||
    monitor === "go_native.system.current_time" ||
    monitor === "alias.system.display" ||
    monitor === "go_native.system.display" ||
    monitor === "alias.system.resolution" ||
    monitor === "go_native.system.resolution" ||
    monitor === "alias.system.refresh_rate" ||
    monitor === "go_native.system.refresh_rate"
  );
});
const selectedFormatPlaceholder = computed(() => {
  const monitor = normalizeText(selectedItem.value?.monitor);
  if (monitor === "alias.system.time" || monitor === "go_native.system.current_time") {
    return "时间格式，例如 15:04:05 或 %H:%M:%S";
  }
  if (monitor === "alias.system.resolution" || monitor === "go_native.system.resolution") {
    return "例如 {resolution} 或 {width}x{height}";
  }
  if (monitor === "alias.system.refresh_rate" || monitor === "go_native.system.refresh_rate") {
    return "例如 {refresh_rate}";
  }
  if (monitor === "alias.system.display" || monitor === "go_native.system.display") {
    return "例如 {resolution}@{refresh_rate}";
  }
  return "";
});

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
    normalizeText(item.render_attrs_map?.label);
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
    zIndex: index === props.selectedIndex ? 20 : 2,
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
  return {};
}

function resolveItemStyle(item) {
  if (!item || typeof item !== "object") return {};
  const map = item.style;
  if (map && typeof map === "object") return map;
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

function updateRenderAttr(key, value) {
  if (props.readonlyProfile || props.selectedIndex < 0) return;
  const item = selectedItem.value;
  if (!item) return;
  const next = patchObjectKey(resolveRenderAttrs(item), key, value, { removeEmptyString: true });
  emit("patch-item", {
    index: props.selectedIndex,
    patch: { render_attrs_map: next },
  });
}

function updateItemStyle(payload) {
  if (props.readonlyProfile || props.selectedIndex < 0) return;
  const item = selectedItem.value;
  if (!item) return;
  const key = String(payload?.key || "").trim();
  if (!key) return;
  const next = patchObjectKey(resolveItemStyle(item), key, payload.value);
  emit("patch-item", {
    index: props.selectedIndex,
    patch: { style: next },
  });
}

function removeItemStyle(payload) {
  if (props.readonlyProfile || props.selectedIndex < 0) return;
  const item = selectedItem.value;
  if (!item) return;
  const key = String(payload?.key || "").trim();
  if (!key) return;
  const next = patchObjectKey(resolveItemStyle(item), key, undefined);
  emit("patch-item", {
    index: props.selectedIndex,
    patch: { style: next },
  });
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
            :key="item.id || `${idx}_${item.edit_ui_name || item.type}`"
            type="button"
            class="list_item"
            :class="{ active: idx === selectedIndex }"
            @click="emit('select-item', idx)"
          >
            <span class="item_name">{{ idx + 1 }}. {{ displayItemName(item) }}</span>
            <small class="item_type">{{ getItemTypeLabel(item.type) }}</small>
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
            :key="item.id || `${idx}_${item.type}`"
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
            <n-form-item-gi v-if="selectedSupportsFormat" label="格式" :span="2">
              <n-input
                :value="renderAttrString('format', '')"
                :placeholder="selectedFormatPlaceholder"
                @update:value="(v) => updateRenderAttr('format', String(v || ''))"
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
          </n-grid>
        </n-form>

        <template v-if="selectedCustomStyleEnabled">
          <n-divider style="margin: 8px 0 6px" />
          <style-manager-form
            scope="item"
            :item-type="selectedType"
            :model="resolveItemStyle(selectedItem)"
            :style-keys="meta.style_keys || []"
            :label-width="92"
            :cols="1"
            :disabled="readonlyProfile"
            @update-style="updateItemStyle"
            @remove-style="removeItemStyle"
          />
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

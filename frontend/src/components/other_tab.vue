<script setup>
import { computed } from "vue";
import DeferredInput from "./deferred_input.vue";
import DeferredInputNumber from "./deferred_input_number.vue";
import PureColorInput from "./pure_color_input.vue";
import { normalizeThresholdGroups } from "../config_normalizer";
import { buildMergedAutoThresholdGroups } from "../threshold_group_auto";

const props = defineProps({
  config: { type: Object, required: true },
  meta: { type: Object, required: true },
  snapshot: { type: Object, default: null },
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
const monitorSelectOptions = computed(() =>
  (props.monitorOptions || []).map((option) => {
    const value = String(option?.value || "").trim();
    const rawLabel = String(option?.label || value).trim() || value;
    const suffix = value ? ` (${value})` : "";
    const label = suffix && rawLabel.endsWith(suffix) ? rawLabel.slice(0, -suffix.length) : rawLabel;
    return { label: label || value, value };
  }),
);

function collectorSupportedOnPlatform(name) {
  const collector = String(name || "").trim().toLowerCase();
  if (collector === "rtss") return platform.value === "windows";
  if (collector === "coolercontrol") return platform.value === "linux";
  if (collector === "librehardwaremonitor") return platform.value === "windows";
  return true;
}

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

function thresholdGroups() {
  return Array.isArray(props.config.threshold_groups) ? props.config.threshold_groups : [];
}

function updateThresholdGroups(next) {
  onField("threshold_groups", normalizeThresholdGroups(next));
}

function addThresholdGroup() {
  const next = thresholdGroups().map((item) => ({ ...(item || {}) }));
  next.push({
    name: `group_${next.length + 1}`,
    monitors: [],
    ranges: [{ color: "#22c55e" }],
  });
  updateThresholdGroups(next);
}

function autoDetectThresholdGroups() {
  updateThresholdGroups(
    buildMergedAutoThresholdGroups({
      existingGroups: thresholdGroups(),
      config: props.config,
      snapshot: props.snapshot,
      monitorOptions: props.monitorOptions,
    }),
  );
}

function patchThresholdGroup(index, patch) {
  const next = thresholdGroups().map((item) => ({ ...(item || {}) }));
  next[index] = { ...(next[index] || {}), ...(patch || {}) };
  updateThresholdGroups(next);
}

function removeThresholdGroup(index) {
  updateThresholdGroups(thresholdGroups().filter((_, idx) => idx !== index));
}

function addThresholdGroupRange(groupIndex) {
  const next = thresholdGroups().map((item) => ({
    ...(item || {}),
    ranges: Array.isArray(item?.ranges) ? item.ranges.map((range) => ({ ...(range || {}) })) : [],
  }));
  next[groupIndex].ranges.push({ color: "#22c55e" });
  next[groupIndex].ranges = normalizeEditableRanges(next[groupIndex].ranges);
  updateThresholdGroups(next);
}

function patchThresholdGroupRange(groupIndex, rangeIndex, patch) {
  const next = thresholdGroups().map((item) => ({
    ...(item || {}),
    ranges: Array.isArray(item?.ranges) ? item.ranges.map((range) => ({ ...(range || {}) })) : [],
  }));
  const updatedRange = {
    ...(next[groupIndex].ranges[rangeIndex] || {}),
    ...(patch || {}),
  };
  next[groupIndex].ranges[rangeIndex] = updatedRange;
  next[groupIndex].ranges = normalizeEditableRanges(next[groupIndex].ranges);
  updateThresholdGroups(next);
}

function removeThresholdGroupRange(groupIndex, rangeIndex) {
  const next = thresholdGroups().map((item) => ({
    ...(item || {}),
    ranges: Array.isArray(item?.ranges) ? item.ranges.map((range) => ({ ...(range || {}) })) : [],
  }));
  next[groupIndex].ranges = next[groupIndex].ranges.filter((_, idx) => idx !== rangeIndex);
  next[groupIndex].ranges = normalizeEditableRanges(next[groupIndex].ranges);
  updateThresholdGroups(next);
}

function resolveGroupThresholdUnit(group) {
  const monitors = Array.isArray(group?.monitors) ? group.monitors : [];
  let resolved = "";
  for (const monitorName of monitors) {
    const unit = String(props.snapshot?.values?.[monitorName]?.unit || "").trim();
    if (!unit) continue;
    if (!resolved) {
      resolved = unit;
      continue;
    }
    if (resolved !== unit) {
      return "";
    }
  }
  return resolved;
}

function normalizeEditableRanges(ranges) {
  const list = Array.isArray(ranges) ? ranges.map((range) => ({ ...(range || {}) })) : [];
  if (list.length === 0) return list;
  delete list[0].min;
  for (let index = 0; index < list.length; index += 1) {
    const current = list[index];
    if (current.max === undefined || current.max === null || current.max === "") {
      delete current.max;
      if (index + 1 < list.length) {
        delete list[index + 1].min;
      }
      continue;
    }
    const boundary = Number(current.max);
    if (!Number.isFinite(boundary)) {
      delete current.max;
      if (index + 1 < list.length) {
        delete list[index + 1].min;
      }
      continue;
    }
    current.max = boundary;
    if (index + 1 < list.length) {
      list[index + 1].min = boundary;
    }
  }
  return list;
}

function rangeThresholdValue(group, rangeIndex) {
  const ranges = Array.isArray(group?.ranges) ? group.ranges : [];
  if (rangeIndex < 0 || rangeIndex >= ranges.length) {
    return null;
  }
  const value = ranges[rangeIndex]?.max;
  return value === undefined ? null : value;
}

function patchThresholdGroupThreshold(groupIndex, rangeIndex, rawValue) {
  const next = thresholdGroups().map((item) => ({
    ...(item || {}),
    ranges: Array.isArray(item?.ranges) ? item.ranges.map((range) => ({ ...(range || {}) })) : [],
  }));
  const ranges = next[groupIndex].ranges;
  if (rangeIndex < 0 || rangeIndex >= ranges.length) {
    return;
  }
  if (rawValue === "" || rawValue === null || rawValue === undefined) {
    delete ranges[rangeIndex].max;
  } else {
    const boundary = Number(rawValue);
    if (Number.isFinite(boundary)) {
      ranges[rangeIndex].max = boundary;
    } else {
      delete ranges[rangeIndex].max;
    }
  }
  next[groupIndex].ranges = normalizeEditableRanges(ranges);
  updateThresholdGroups(next);
}
</script>

<template>
  <section class="layout_single basic_tab">
    <div class="basic_inner">
      <n-card title="阈值组" size="small">
        <template #header-extra>
          <n-space size="small">
            <n-button size="small" tertiary :disabled="readonlyProfile" @click="autoDetectThresholdGroups">
              自动识别阈值组
            </n-button>
            <n-button size="small" type="primary" :disabled="readonlyProfile" @click="addThresholdGroup">
              新增阈值组
            </n-button>
          </n-space>
        </template>

        <n-space vertical size="small">
          <n-card
            v-for="(group, groupIndex) in thresholdGroups()"
            :key="group.name || groupIndex"
            size="small"
            embedded
          >
            <template #header>
              <div class="threshold_group_card_header">
                <n-ellipsis class="threshold_group_card_title">
                  {{ group.name || `group_${groupIndex + 1}` }}
                </n-ellipsis>
                <n-button
                  size="tiny"
                  tertiary
                  type="error"
                  :disabled="readonlyProfile"
                  @click="removeThresholdGroup(groupIndex)"
                >
                  删除
                </n-button>
              </div>
            </template>

            <div class="threshold_group_editor">
              <div class="threshold_group_meta">
                <div class="threshold_group_meta_form">
                  <div class="threshold_group_meta_field">
                    <div class="threshold_group_field_label">名称</div>
                    <DeferredInput
                      class="threshold_group_text_input"
                      :value="group.name || ''"
                      :disabled="readonlyProfile"
                      @update:value="(v) => patchThresholdGroup(groupIndex, { name: String(v || '') })"
                    />
                  </div>
                  <div class="threshold_group_meta_field">
                    <div class="threshold_group_field_label">指标</div>
                    <n-select
                      class="threshold_group_monitor_select"
                      multiple
                      filterable
                      clearable
                      :value="group.monitors || []"
                      :disabled="readonlyProfile"
                      :options="monitorSelectOptions"
                      @update:value="(v) => patchThresholdGroup(groupIndex, { monitors: Array.isArray(v) ? v : [] })"
                    />
                  </div>
                </div>
              </div>

              <div class="threshold_group_ranges">
                <div class="threshold_group_ranges_head">
                  <div class="threshold_group_ranges_title">
                    <n-text depth="3">区间</n-text>
                    <n-text depth="3">{{ (group.ranges || []).length }} 条</n-text>
                  </div>
                  <n-button size="tiny" tertiary :disabled="readonlyProfile" @click="addThresholdGroupRange(groupIndex)">
                    新增区间
                  </n-button>
                </div>
                <div class="threshold_group_range_list">
                  <div class="threshold_group_range_header">
                    <span>
                      阈值<span v-if="resolveGroupThresholdUnit(group)"> ({{ resolveGroupThresholdUnit(group) }})</span>
                    </span>
                    <span>颜色</span>
                    <span>操作</span>
                  </div>
                  <div
                    v-for="(range, rangeIndex) in group.ranges || []"
                    :key="`${group.name || groupIndex}-${rangeIndex}`"
                    class="threshold_group_range_row"
                  >
                    <DeferredInputNumber
                      :value="rangeThresholdValue(group, rangeIndex)"
                      :disabled="readonlyProfile"
                      :show-button="false"
                      placeholder="输入当前分界点"
                      @update:value="(v) => patchThresholdGroupThreshold(groupIndex, rangeIndex, v)"
                    />
                    <PureColorInput
                      :value="range.color || ''"
                      :disabled="readonlyProfile"
                      @update:value="(v) => patchThresholdGroupRange(groupIndex, rangeIndex, { color: String(v || '') })"
                    />
                    <n-button
                      size="small"
                      tertiary
                      type="error"
                      :disabled="readonlyProfile"
                      @click="removeThresholdGroupRange(groupIndex, rangeIndex)"
                    >
                      删除
                    </n-button>
                  </div>
                </div>
              </div>
            </div>
          </n-card>
        </n-space>
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
          支持 {{ customTypeOptions.map((item) => item.value).join(" / ") }}
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
                  <DeferredInput
                    :value="item.name || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'name', value: String(v || '') })"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="Label">
                  <DeferredInput
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
                  <DeferredInput
                    :value="item.unit || ''"
                    :disabled="readonlyProfile"
                    @update:value="(v) => emit('change-custom', { index: idx, field: 'unit', value: String(v || '') })"
                  />
                </n-form-item-gi>

                <n-form-item-gi v-if="item.type === 'file'" label="Path" :span="4">
                  <DeferredInput
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

<style scoped>
.threshold_group_card_header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;
}

.threshold_group_card_title {
  min-width: 0;
  flex: 1 1 auto;
}

.threshold_group_editor {
  display: flex;
  gap: 12px;
  align-items: start;
}

.threshold_group_meta,
.threshold_group_ranges {
  min-width: 0;
}

.threshold_group_meta {
  flex: 1 1 auto;
  width: 0;
}

.threshold_group_ranges {
  flex: 0 0 400px;
}

.threshold_group_meta_form,
.threshold_group_meta_field {
  width: 100%;
}

.threshold_group_meta_form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.threshold_group_meta_field {
  min-width: 0;
}

.threshold_group_field_label {
  margin-bottom: 4px;
  color: var(--n-text-color-2);
  font-size: 12px;
  line-height: 1.4;
}

.threshold_group_text_input,
.threshold_group_monitor_select {
  width: 100%;
  min-width: 0;
}

.threshold_group_monitor_select :deep(.n-base-selection.n-base-selection--multiple),
.threshold_group_monitor_select :deep(.n-base-selection) {
  display: flex !important;
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.threshold_group_monitor_select :deep(.n-base-selection-label),
.threshold_group_monitor_select :deep(.n-base-selection-tags) {
  width: 100% !important;
  min-width: 0 !important;
  max-width: none !important;
}

.threshold_group_monitor_select :deep(.n-base-selection-tags) {
  display: flex !important;
  flex-wrap: wrap !important;
  align-items: flex-start !important;
  gap: 4px !important;
}

.threshold_group_monitor_select :deep(.n-tag) {
  max-width: 100% !important;
}

.threshold_group_monitor_select :deep(.n-tag__content) {
  white-space: normal !important;
  word-break: break-word;
  line-height: 1.35;
}

.threshold_group_ranges_title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.threshold_group_ranges_head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.threshold_group_range_list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.threshold_group_range_header,
.threshold_group_range_row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 108px 64px;
  gap: 8px;
  align-items: center;
}

.threshold_group_range_header {
  padding: 0 2px;
  color: var(--n-text-color-3);
  font-size: 12px;
  line-height: 1.4;
}

@media (max-width: 900px) {
  .threshold_group_editor {
    flex-direction: column;
  }

  .threshold_group_ranges {
    flex: 1 1 auto;
    width: 100%;
  }
}

@media (max-width: 640px) {
  .threshold_group_range_header,
  .threshold_group_range_row {
    grid-template-columns: minmax(0, 1fr) 96px 64px;
  }
}
</style>

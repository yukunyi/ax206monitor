<script setup>
import { computed } from "vue";
import DeferredInput from "./deferred_input.vue";

const props = defineProps({
  config: { type: Object, required: true },
  readonlyProfile: { type: Boolean, default: false },
  monitorOptions: { type: Array, default: () => [] },
});

const emit = defineEmits(["add-custom", "remove-custom", "change-custom", "refresh-monitors"]);

const monitorSelectOptions = computed(() => props.monitorOptions || []);

const customTypeOptions = [
  { label: "file", value: "file" },
  { label: "mixed", value: "mixed" },
  { label: "coolercontrol", value: "coolercontrol" },
  { label: "librehardwaremonitor", value: "librehardwaremonitor" },
  { label: "rtss", value: "rtss" },
];

const aggregateOptions = [
  { label: "max", value: "max" },
  { label: "min", value: "min" },
  { label: "avg", value: "avg" },
];
</script>

<template>
  <section class="layout_single">
    <n-card size="small">
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
        支持 file / mixed / coolercontrol / librehardwaremonitor / rtss
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

          <n-form label-placement="top" size="small">
            <n-grid cols="2" :x-gap="8" :y-gap="6">
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

              <n-form-item-gi v-if="item.type === 'file'" label="Path" :span="2">
                <DeferredInput
                  :value="item.path || ''"
                  :disabled="readonlyProfile"
                  @update:value="(v) => emit('change-custom', { index: idx, field: 'path', value: String(v || '') })"
                />
              </n-form-item-gi>

              <n-form-item-gi v-if="item.type !== 'file'" label="Source" :span="2">
                <n-select
                  :value="item.source || ''"
                  :disabled="readonlyProfile"
                  :options="monitorSelectOptions"
                  filterable
                  clearable
                  @update:value="(v) => emit('change-custom', { index: idx, field: 'source', value: String(v || '') })"
                />
              </n-form-item-gi>

              <n-form-item-gi v-if="item.type === 'mixed'" label="Sources" :span="2">
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
  </section>
</template>

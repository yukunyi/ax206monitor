<script setup>
import { computed } from "vue";
import {
  buildOutputTypeOptions,
  createDefaultOutputEntry,
  isAX206Type,
  isHttpPushType,
  OUTPUT_FORMAT_OPTIONS,
} from "../output_configs";

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

const monitorSelectOptions = computed(() => props.monitorOptions || []);
const outputTypeOptions = computed(() =>
  buildOutputTypeOptions(props.meta?.output_types, props.config?.outputs),
);
const outputFormatOptions = OUTPUT_FORMAT_OPTIONS;

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

function collectorFieldDisabled(name) {
  return props.readonlyProfile || !collectorEntry(name).enabled;
}

function outputEntries() {
  return Array.isArray(props.config.outputs) ? props.config.outputs : [];
}

function updateOutputs(next) {
  onField("outputs", Array.isArray(next) ? next : []);
}

function outputEntryByType(type) {
  return outputEntries().find((item) => String(item?.type || "").trim().toLowerCase() === String(type || "").trim().toLowerCase()) || null;
}

function outputEnabled(type) {
  return !!outputEntryByType(type);
}

function outputFieldDisabled(type) {
  return props.readonlyProfile || !outputEnabled(type);
}

function setOutputEnabled(type, enabled) {
  const next = outputEntries().map((item) => ({ ...(item || {}) }));
  const normalized = String(type || "").trim().toLowerCase();
  const index = next.findIndex((item) => String(item?.type || "").trim().toLowerCase() === normalized);
  if (enabled) {
    if (index >= 0) return;
    next.push(createDefaultOutputEntry(normalized));
  } else if (index >= 0) {
    next.splice(index, 1);
  }
  updateOutputs(next);
}

function patchOutputByType(type, patch) {
  const next = outputEntries().map((item) => ({ ...(item || {}) }));
  const normalized = String(type || "").trim().toLowerCase();
  const index = next.findIndex((item) => String(item?.type || "").trim().toLowerCase() === normalized);
  if (index >= 0) {
    next[index] = { ...(next[index] || {}), ...(patch || {}) };
  } else {
    next.push({ ...createDefaultOutputEntry(normalized), ...(patch || {}) });
  }
  updateOutputs(next);
}

function outputTypeLabel(type) {
  if (isAX206Type(type)) return "AX206 USB";
  if (isHttpPushType(type)) return "HTTP Push";
  return String(type || "");
}

</script>

<template>
  <section class="layout_single basic_tab">
    <div class="basic_inner">
      <n-grid cols="1" :x-gap="8" :y-gap="6">
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
                <n-form-item-gi label="锁屏暂停采集">
                  <n-switch
                    :value="config.pause_collect_on_lock === true"
                    :disabled="readonlyProfile"
                    size="small"
                    @update:value="(v) => onField('pause_collect_on_lock', !!v)"
                  />
                </n-form-item-gi>
                <n-form-item-gi label="输出配置" :span="2">
                  <n-table class="collector_table output_table" size="small" striped>
                    <thead>
                      <tr>
                        <th style="width: 30%">输出类型</th>
                        <th style="width: 12%">启用</th>
                        <th>参数</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr v-for="option in outputTypeOptions" :key="option.value">
                        <td>{{ outputTypeLabel(option.value) }}</td>
                        <td>
                          <n-switch
                            :value="outputEnabled(option.value)"
                            :disabled="readonlyProfile"
                            size="small"
                            @update:value="(v) => setOutputEnabled(option.value, !!v)"
                          />
                        </td>
                        <td>
                          <template v-if="isAX206Type(option.value)">
                            <n-space size="small" :wrap="false" align="center">
                              <n-text depth="3">重连间隔</n-text>
                              <n-input-number
                                style="width: 180px"
                                :value="Number(outputEntryByType(option.value)?.reconnect_ms || 3000)"
                                :disabled="outputFieldDisabled(option.value)"
                                :show-button="false"
                                @update:value="(v) => patchOutputByType(option.value, { reconnect_ms: Number(v || 3000) })"
                              />
                            </n-space>
                          </template>
                          <template v-else-if="isHttpPushType(option.value)">
                            <n-space vertical size="small" style="width: 100%">
                              <n-space size="small" :wrap="false" align="center">
                                <n-text depth="3" style="width: 52px">地址</n-text>
                                <n-input
                                  :value="outputEntryByType(option.value)?.url || ''"
                                  :disabled="outputFieldDisabled(option.value)"
                                  size="small"
                                  placeholder="http://127.0.0.1:18090/push"
                                  @update:value="(v) => patchOutputByType(option.value, { url: String(v || '') })"
                                />
                              </n-space>
                              <n-space size="small" :wrap="false">
                                <n-space size="small" :wrap="false" align="center">
                                  <n-text depth="3">格式</n-text>
                                  <n-select
                                    style="width: 120px"
                                    :value="outputEntryByType(option.value)?.format || 'jpeg'"
                                    :disabled="outputFieldDisabled(option.value)"
                                    :options="outputFormatOptions"
                                    @update:value="(v) => patchOutputByType(option.value, { format: String(v || 'jpeg') })"
                                  />
                                </n-space>
                                <n-space size="small" :wrap="false" align="center">
                                  <n-text depth="3">质量</n-text>
                                  <n-input-number
                                    style="width: 120px"
                                    :value="Number(outputEntryByType(option.value)?.quality || 80)"
                                    :disabled="outputFieldDisabled(option.value)"
                                    :show-button="false"
                                    @update:value="(v) => patchOutputByType(option.value, { quality: Number(v || 80) })"
                                  />
                                </n-space>
                              </n-space>
                            </n-space>
                          </template>
                          <template v-else>-</template>
                        </td>
                      </tr>
                    </tbody>
                  </n-table>
                </n-form-item-gi>
              </n-grid>
            </n-form>
          </n-card>
        </n-grid-item>
      </n-grid>

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
                      :disabled="collectorFieldDisabled(name)"
                      size="small"
                      :placeholder="name === 'coolercontrol' ? 'http://127.0.0.1:11987' : 'http://127.0.0.1:8085'"
                      @update:value="(v) => onField(['collector_config', name, 'options', 'url'], String(v || ''))"
                    />
                    <n-space v-if="collectorHasAuth(name)" size="small" :wrap="false">
                      <n-input
                        :value="collectorOption(name, 'username')"
                        :disabled="collectorFieldDisabled(name)"
                        size="small"
                        :placeholder="collectorAuthUserLabel(name)"
                        @update:value="(v) => onField(['collector_config', name, 'options', 'username'], String(v || ''))"
                      />
                      <n-input
                        type="password"
                        show-password-on="click"
                        :value="collectorOption(name, 'password')"
                        :disabled="collectorFieldDisabled(name)"
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

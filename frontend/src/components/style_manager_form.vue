<script setup>
import { computed, ref } from "vue";
import PureColorInput from "./pure_color_input.vue";
import {
  normalizeStyleKeys,
  styleDefaultValue,
  styleMetaMap,
  supportsScope,
  supportsType,
} from "../style_keys";

const props = defineProps({
  scope: { type: String, required: true },
  itemType: { type: String, default: "" },
  model: { type: Object, default: () => ({}) },
  styleKeys: { type: Array, default: () => [] },
  showAllKeys: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  labelWidth: { type: Number, default: 92 },
  cols: { type: [Number, String], default: 2 },
});

const emit = defineEmits(["update-style", "remove-style"]);

const addKey = ref("");

const metas = computed(() => normalizeStyleKeys(props.styleKeys));
const metaIndex = computed(() => styleMetaMap(metas.value));

function allowMeta(meta) {
  if (!meta) return false;
  if (props.showAllKeys) return true;
  return supportsScope(meta, props.scope) && supportsType(meta, props.itemType);
}

const modelKeys = computed(() =>
  Object.keys(props.model || {})
    .map((key) => String(key || "").trim())
    .filter(Boolean),
);

const filteredModelKeys = computed(() => {
  return modelKeys.value.filter((key) => {
    const meta = metaIndex.value[key];
    if (!meta) return true;
    return allowMeta(meta);
  });
});

const orderedModelKeys = computed(() => {
  const orderMap = {};
  metas.value.forEach((meta, idx) => {
    orderMap[meta.key] = idx;
  });
  return [...filteredModelKeys.value].sort((a, b) => {
    const ai = orderMap[a];
    const bi = orderMap[b];
    if (Number.isFinite(ai) && Number.isFinite(bi)) return ai - bi;
    if (Number.isFinite(ai)) return -1;
    if (Number.isFinite(bi)) return 1;
    return a.localeCompare(b);
  });
});

const addOptions = computed(() => {
  const used = new Set(filteredModelKeys.value);
  return metas.value
    .filter((meta) => allowMeta(meta))
    .filter((meta) => !used.has(meta.key))
    .map((meta) => ({ label: `${meta.label} (${meta.key})`, value: meta.key }));
});

function metaFor(key) {
  return metaIndex.value[String(key || "").trim()] || null;
}

function fieldKind(key) {
  return metaFor(key)?.kind || "text";
}

function fieldLabel(key) {
  const meta = metaFor(key);
  if (!meta) return String(key || "");
  return `${meta.label}`;
}

function numberValue(key, fallback = 0) {
  const raw = props.model?.[key];
  const n = Number(raw);
  return Number.isFinite(n) ? n : fallback;
}

function boolValue(key, fallback = false) {
  const raw = props.model?.[key];
  if (raw === undefined || raw === null) return fallback;
  if (typeof raw === "boolean") return raw;
  if (typeof raw === "number") return raw !== 0;
  const text = String(raw).trim().toLowerCase();
  return text === "1" || text === "true" || text === "yes" || text === "on";
}

function textValue(key, fallback = "") {
  const raw = props.model?.[key];
  if (raw === undefined || raw === null) return fallback;
  return String(raw);
}

function colorValue(key, fallback = "") {
  const text = String(props.model?.[key] || "").trim();
  return text || fallback;
}

function numberArray4(key, fallback) {
  const out = Array.isArray(fallback) ? [...fallback] : [0, 0, 0, 0];
  const raw = props.model?.[key];
  if (!Array.isArray(raw)) return out;
  for (let i = 0; i < 4; i += 1) {
    const n = Number(raw[i]);
    if (Number.isFinite(n)) out[i] = n;
  }
  return out;
}

function colorArray4(key, fallback) {
  const out = Array.isArray(fallback) ? [...fallback] : ["", "", "", ""];
  const raw = props.model?.[key];
  if (!Array.isArray(raw)) return out;
  for (let i = 0; i < 4; i += 1) {
    const text = String(raw[i] || "").trim();
    if (text) out[i] = text;
  }
  return out;
}

function patchArrayValue(key, index, nextValue) {
  const kind = fieldKind(key);
  if (kind === "float4") {
    const prev = numberArray4(key, [25, 50, 75, 100]);
    prev[index] = Number(nextValue || 0);
    emit("update-style", { key, value: prev });
    return;
  }
  if (kind === "color4") {
    const prev = colorArray4(key, ["#22c55e", "#eab308", "#f97316", "#ef4444"]);
    prev[index] = String(nextValue || "");
    emit("update-style", { key, value: prev });
  }
}

function updateField(key, value) {
  emit("update-style", { key: String(key || ""), value });
}

function removeField(key) {
  emit("remove-style", { key: String(key || "") });
}

function onAddSelected() {
  const key = String(addKey.value || "").trim();
  if (!key) return;
  let defaultValue = styleDefaultValue(key, props.itemType);
  if (key === "font_family") {
    const options = metaFor(key)?.options || [];
    const first = options.length > 0 ? String(options[0]?.value || "").trim() : "";
    if (first) defaultValue = first;
  }
  updateField(key, defaultValue);
  addKey.value = "";
}
</script>

<template>
  <section class="style_manager_form">
    <div class="style_manager_add">
      <n-select
        class="style_manager_add_select"
        v-model:value="addKey"
        size="small"
        filterable
        clearable
        placeholder="选择样式 key 后添加"
        :options="addOptions"
        :disabled="disabled"
      />
      <n-button size="small" type="primary" :disabled="disabled || !addKey" @click="onAddSelected">
        添加
      </n-button>
    </div>

    <n-empty v-if="orderedModelKeys.length === 0" size="small" description="暂无样式覆盖" />

    <n-form
      v-else
      class="compact_style_form"
      label-placement="left"
      size="small"
      :label-width="labelWidth"
    >
      <n-grid :cols="cols" :x-gap="6" :y-gap="1">
        <n-form-item-gi v-for="key in orderedModelKeys" :key="key" :label="fieldLabel(key)">
          <n-space size="small" :wrap="false" align="center" class="style_field_row">
            <template v-if="fieldKind(key) === 'int' || fieldKind(key) === 'float'">
              <n-input-number
                :value="numberValue(key, 0)"
                :show-button="false"
                :disabled="disabled"
                @update:value="(v) => updateField(key, Number(v || 0))"
              />
            </template>
            <template v-else-if="fieldKind(key) === 'bool'">
              <n-switch
                :value="boolValue(key, false)"
                :disabled="disabled"
                @update:value="(v) => updateField(key, !!v)"
              />
            </template>
            <template v-else-if="fieldKind(key) === 'color'">
              <pure-color-input
                :value="colorValue(key, '')"
                :disabled="disabled"
                @update:value="(v) => updateField(key, String(v || ''))"
              />
            </template>
            <template v-else-if="fieldKind(key) === 'select'">
              <n-select
                :value="textValue(key, '')"
                :disabled="disabled"
                :options="metaFor(key)?.options || []"
                filterable
                clearable
                @update:value="(v) => updateField(key, String(v || ''))"
              />
            </template>
            <template v-else-if="fieldKind(key) === 'float4'">
              <n-space size="4" :wrap="false" class="array_inputs">
                <n-input-number
                  v-for="(item, idx) in numberArray4(key, [25, 50, 75, 100])"
                  :key="idx"
                  :value="item"
                  :show-button="false"
                  :disabled="disabled"
                  @update:value="(v) => patchArrayValue(key, idx, v)"
                />
              </n-space>
            </template>
            <template v-else-if="fieldKind(key) === 'color4'">
              <n-space size="4" :wrap="false" class="array_inputs">
                <pure-color-input
                  v-for="(item, idx) in colorArray4(key, ['#22c55e', '#eab308', '#f97316', '#ef4444'])"
                  :key="idx"
                  :value="item"
                  :disabled="disabled"
                  @update:value="(v) => patchArrayValue(key, idx, v)"
                />
              </n-space>
            </template>
            <template v-else>
              <n-input
                :value="textValue(key, '')"
                :disabled="disabled"
                @update:value="(v) => updateField(key, String(v || ''))"
              />
            </template>
            <n-button size="tiny" tertiary type="error" :disabled="disabled" @click="removeField(key)">
              删除
            </n-button>
          </n-space>
        </n-form-item-gi>
      </n-grid>
    </n-form>
  </section>
</template>

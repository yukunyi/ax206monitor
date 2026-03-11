<script setup>
import { computed, ref } from "vue";
import StyleManagerForm from "./style_manager_form.vue";
import { normalizeStyleKeys, styleDefaultValue } from "../style_keys";

const props = defineProps({
  config: { type: Object, required: true },
  meta: { type: Object, required: true },
  readonlyProfile: { type: Boolean, default: false },
});

const emit = defineEmits(["change"]);

function onField(path, value) {
  emit("change", { path, value });
}

const ITEM_TYPE_LABELS = {
  simple_value: "基础数值",
  simple_progress: "基础进度条",
  simple_line_chart: "基础折线图",
  simple_line: "基础线条",
  simple_label: "基础标签",
  simple_rect: "基础矩形",
  simple_circle: "基础圆形",
  label_text: "标签数值",
  full_chart: "复杂图表",
  full_progress: "复杂进度条",
  full_gauge: "复杂仪表盘",
};

const typeDefaultRows = computed(() => {
  const order = Array.isArray(props.meta?.item_types) && props.meta.item_types.length > 0
    ? props.meta.item_types.map((item) => (typeof item === "string" ? item : String(item?.value || "")))
    : Object.keys(ITEM_TYPE_LABELS);
  const unique = new Set(order.filter(Boolean));
  Object.keys(props.config.type_defaults || {}).forEach((type) => unique.add(type));
  return [...unique];
});

function isComplexType(type) {
  return String(type || "").startsWith("full_");
}

const typeDefaultGroups = computed(() => {
  const simple = [];
  const complex = [];
  typeDefaultRows.value.forEach((type) => {
    if (isComplexType(type)) {
      complex.push(type);
      return;
    }
    simple.push(type);
  });
  return [
    { key: "simple", types: simple },
    { key: "complex", types: complex },
  ].filter((group) => group.types.length > 0);
});

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

function onTypeStyleChange(type, payload) {
  onField(["type_defaults", type, "style", payload.key], payload.value);
}

function onTypeStyleRemove(type, payload) {
  onField(["type_defaults", type, "style", payload.key], undefined);
}

function onBaseStyleChange({ key, value }) {
  onField(["style_base", key], value);
}

function onBaseStyleRemove({ key }) {
  onField(["style_base", key], undefined);
}

const showGlobalDefaults = ref(false);
const selectedGlobalType = ref("__all__");

const globalTypeOptions = computed(() => {
  const fromMeta = Array.isArray(props.meta?.item_types) ? props.meta.item_types : [];
  const candidates = fromMeta.length > 0
    ? fromMeta.map((item) => (typeof item === "string" ? item : String(item?.value || "")))
    : Object.keys(ITEM_TYPE_LABELS);
  const set = new Set(candidates.map((item) => String(item || "").trim()).filter(Boolean));
  Object.keys(props.config.type_defaults || {}).forEach((type) => set.add(String(type || "").trim()));
  const options = [{ label: "全部类型", value: "__all__" }];
  [...set].sort().forEach((type) => {
    options.push({
      label: `${typeLabel(type)} (${type})`,
      value: type,
    });
  });
  return options;
});

const globalDefaultRows = computed(() => {
  const metas = normalizeStyleKeys(props.meta?.style_keys || []);
  return metas.map((meta) => {
    const scopes = Array.isArray(meta.scopes) ? meta.scopes : [];
    const types = Array.isArray(meta.types) ? meta.types : [];
    const scopeText = scopes
      .map((scope) => {
        if (scope === "base") return "基础";
        if (scope === "type") return "类型默认";
        if (scope === "item") return "元素覆盖";
        return scope;
      })
      .join(" / ");
    const typeText = types.length > 0 ? types.join(", ") : "全部类型";
    let defaultRaw = styleDefaultValue(meta.key, "");
    if (meta.key === "font_family") {
      const options = Array.isArray(meta.options) ? meta.options : [];
      const first = options.length > 0 ? String(options[0]?.value || "").trim() : "";
      if (first) defaultRaw = first;
    }
    const defaultText = formatDefaultValue(defaultRaw);
    return {
      key: meta.key,
      label: meta.label,
      kind: String(meta.kind || ""),
      types,
      scopeText,
      typeText,
      defaultRaw,
      defaultText,
    };
  });
});

const filteredGlobalDefaultRows = computed(() => {
  const selected = String(selectedGlobalType.value || "__all__").trim();
  if (selected === "__all__") return globalDefaultRows.value;
  return globalDefaultRows.value.filter((row) => {
    const types = Array.isArray(row.types) ? row.types : [];
    return types.length === 0 || types.includes(selected);
  });
});

function formatDefaultValue(value) {
  if (Array.isArray(value)) return value.join(", ");
  if (typeof value === "boolean") return value ? "true" : "false";
  return String(value ?? "");
}

function rowColorValues(row) {
  const kind = String(row?.kind || "");
  if (kind === "color4") {
    return Array.isArray(row?.defaultRaw)
      ? row.defaultRaw.map((item) => String(item || "").trim()).filter(Boolean)
      : [];
  }
  if (kind === "color") {
    const value = String(row?.defaultRaw || "").trim();
    return value ? [value] : [];
  }
  return [];
}

function isColorRow(row) {
  const kind = String(row?.kind || "");
  return kind === "color" || kind === "color4";
}
</script>

<template>
  <section class="layout_single type_defaults_tab">
    <div class="basic_inner">
      <n-card class="default_style_card" title="默认样式" size="small">
        <template #header-extra>
          <n-button size="tiny" tertiary @click="showGlobalDefaults = true">全局默认参数</n-button>
        </template>
        <style-manager-form
          scope="base"
          :item-type="''"
          :show-all-keys="true"
          :model="config.style_base || {}"
          :style-keys="meta.style_keys || []"
          :label-width="91"
          :disabled="readonlyProfile"
          :cols="2"
          @update-style="onBaseStyleChange"
          @remove-style="onBaseStyleRemove"
        />
      </n-card>

      <div class="type_default_groups">
        <section
          v-for="group in typeDefaultGroups"
          :key="group.key"
          class="type_default_group"
        >
          <div class="type_default_flat">
            <section
              v-for="type in group.types"
              :key="type"
              class="type_default_type"
            >
              <header class="type_default_type_title">
                <span class="type_default_type_name">{{ typeLabel(type) }}</span>
                <span class="type_default_type_key">{{ type }}</span>
              </header>

              <section class="type_default_section">
                <style-manager-form
                  scope="type"
                  :item-type="type"
                  :model="typeDefaultEntry(type).style || {}"
                  :style-keys="meta.style_keys || []"
                  :label-width="91"
                  :disabled="readonlyProfile"
                  :cols="2"
                  @update-style="(payload) => onTypeStyleChange(type, payload)"
                  @remove-style="(payload) => onTypeStyleRemove(type, payload)"
                />
              </section>
            </section>
          </div>
        </section>
      </div>

      <n-modal v-model:show="showGlobalDefaults" class="global_defaults_modal" preset="card" style="width: 1400px; max-width: 98vw" title="全局默认参数">
        <n-space align="center" size="small" style="margin-bottom: 8px">
          <n-text depth="3">按类型过滤</n-text>
          <n-select
            style="width: 320px"
            v-model:value="selectedGlobalType"
            :options="globalTypeOptions"
            filterable
          />
        </n-space>
        <div class="global_defaults_table_wrap">
          <n-table size="small" striped>
            <thead>
              <tr>
                <th style="width: 140px">参数</th>
                <th style="width: 170px">Key</th>
                <th style="width: 160px">作用范围</th>
                <th style="width: 220px">适用类型</th>
                <th style="width: 260px">默认值</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in filteredGlobalDefaultRows" :key="row.key">
                <td>{{ row.label }}</td>
                <td><code>{{ row.key }}</code></td>
                <td>{{ row.scopeText }}</td>
                <td>{{ row.typeText }}</td>
                <td class="global_defaults_value">
                  <div class="global_defaults_value_inner">
                    <div v-if="isColorRow(row)" class="global_defaults_swatches">
                      <span
                        v-for="(color, idx) in rowColorValues(row)"
                        :key="`${row.key}_${idx}`"
                        class="global_defaults_swatch"
                        :title="color"
                        :style="{ backgroundColor: color }"
                      />
                    </div>
                    <span>{{ row.defaultText }}</span>
                  </div>
                </td>
              </tr>
            </tbody>
          </n-table>
        </div>
      </n-modal>
    </div>
  </section>
</template>

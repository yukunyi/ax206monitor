<script setup>
import { computed } from "vue";
import StyleBaseForm from "./style_base_form.vue";
import StyleTypeForm from "./style_type_form.vue";

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

function typeHasTypeStyle(type) {
  const key = String(type || "");
  return (
    key === "simple_line_chart" ||
    key === "simple_line" ||
    key === "full_chart" ||
    key === "full_progress" ||
    key === "full_gauge"
  );
}
</script>

<template>
  <section class="layout_single type_defaults_tab">
    <div class="basic_inner">
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
                <header class="type_default_section_title">基础样式</header>
                <style-base-form
                  :model="typeDefaultEntry(type)"
                  :cols="3"
                  :label-width="87"
                  :disabled="readonlyProfile"
                  @update-field="({ field, value }) => onField(['type_defaults', type, field], value)"
                />
              </section>

              <div
                v-if="typeHasTypeStyle(type)"
                class="type_default_divider"
              />

              <section
                v-if="typeHasTypeStyle(type)"
                class="type_default_section"
              >
                <header class="type_default_section_title">类型样式</header>
                <style-type-form
                  :type="type"
                  :attrs="typeDefaultEntry(type).render_attrs_map || {}"
                  :default-history-points="Number(config.default_history_points || 150)"
                  :label-width="91"
                  :disabled="readonlyProfile"
                  @update-attr="({ key, value }) => onField(['type_defaults', type, 'render_attrs_map', key], value)"
                />
              </section>
            </section>
          </div>
        </section>
      </div>
    </div>
  </section>
</template>

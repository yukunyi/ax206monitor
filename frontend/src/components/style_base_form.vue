<script setup>
import PureColorInput from "./pure_color_input.vue";

const props = defineProps({
  model: { type: Object, default: () => ({}) },
  disabled: { type: Boolean, default: false },
  labelWidth: { type: Number, default: 84 },
  cols: { type: [Number, String], default: 2 },
});

const emit = defineEmits(["update-field"]);

function num(key, fallback = 0) {
  const raw = props.model?.[key];
  const value = Number(raw);
  return Number.isFinite(value) ? value : fallback;
}

function color(key, fallback = "") {
  const text = String(props.model?.[key] || "").trim();
  return text || fallback;
}

function onNumber(key, value) {
  emit("update-field", { field: key, value: Math.max(0, Number(value || 0)) });
}

function onColor(key, value) {
  emit("update-field", { field: key, value: String(value || "") });
}
</script>

<template>
  <n-form class="compact_style_form" label-placement="left" size="small" :label-width="labelWidth">
    <n-grid :cols="cols" :x-gap="4" :y-gap="1">
      <n-form-item-gi label="小字号">
        <n-input-number
          :value="num('small_font_size', 0)"
          :disabled="disabled"
          :show-button="false"
          @update:value="(v) => onNumber('small_font_size', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="中字号">
        <n-input-number
          :value="num('medium_font_size', 0)"
          :disabled="disabled"
          :show-button="false"
          @update:value="(v) => onNumber('medium_font_size', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="大字号">
        <n-input-number
          :value="num('large_font_size', 0)"
          :disabled="disabled"
          :show-button="false"
          @update:value="(v) => onNumber('large_font_size', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="边框宽度">
        <n-input-number
          :value="num('border_width', 0)"
          :disabled="disabled"
          :show-button="false"
          @update:value="(v) => onNumber('border_width', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="圆角">
        <n-input-number
          :value="num('radius', 0)"
          :disabled="disabled"
          :show-button="false"
          @update:value="(v) => onNumber('radius', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="前景色">
        <pure-color-input
          :value="color('color', '#f8fafc')"
          :disabled="disabled"
          @update:value="(v) => onColor('color', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="背景色">
        <pure-color-input
          :value="color('bg', '#0b1220')"
          :disabled="disabled"
          @update:value="(v) => onColor('bg', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="边框色">
        <pure-color-input
          :value="color('border_color', '#475569')"
          :disabled="disabled"
          @update:value="(v) => onColor('border_color', v)"
        />
      </n-form-item-gi>
      <n-form-item-gi label="单位色">
        <pure-color-input
          :value="color('unit_color', '#f8fafc')"
          :disabled="disabled"
          @update:value="(v) => onColor('unit_color', v)"
        />
      </n-form-item-gi>
    </n-grid>
  </n-form>
</template>

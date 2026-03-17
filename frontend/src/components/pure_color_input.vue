<script setup>
import { computed, ref, watch } from "vue";

const props = defineProps({
  value: { type: String, default: "" },
  disabled: { type: Boolean, default: false },
});

const emit = defineEmits(["update:value"]);

const draft = ref(String(props.value || ""));

const PRESET_COLORS = [
  "rgba(0,0,0,0)",
  "#f8fafc",
  "#ffffff",
  "#e2e8f0",
  "#cbd5e1",
  "#94a3b8",
  "#000000",
  "#0f172a",
  "#1e293b",
  "#334155",
  "#64748b",
  "#ef4444",
  "#dc2626",
  "#f97316",
  "#ea580c",
  "#f59e0b",
  "#eab308",
  "#84cc16",
  "#22c55e",
  "#16a34a",
  "#10b981",
  "#14b8a6",
  "#06b6d4",
  "#0ea5e9",
  "#3b82f6",
  "#2563eb",
  "#6366f1",
  "#8b5cf6",
  "#7c3aed",
  "#a855f7",
  "#ec4899",
  "#d946ef",
  "#f43f5e",
  "rgba(255,255,255,0.85)",
  "rgba(255,255,255,0.4)",
  "rgba(255,255,255,0.15)",
  "rgba(0,0,0,0.6)",
  "rgba(0,0,0,0.35)",
];

watch(
  () => props.value,
  (value) => {
    draft.value = String(value || "");
  },
);

const previewColor = computed(() => {
  const normalized = normalizeColorValue(props.value);
  return normalized || "#f8fafc";
});

const nativeColorValue = computed(() => {
  const rgba = parseToRGBA(props.value);
  if (!rgba) return "#f8fafc";
  return toHex6(rgba.r, rgba.g, rgba.b);
});

function toHex6(r, g, b) {
  const toHex = (v) => Math.max(0, Math.min(255, Math.round(v))).toString(16).padStart(2, "0");
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function parseToRGBA(raw) {
  const input = String(raw || "").trim();
  if (!input) return null;

  const hexMatch = input.match(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/);
  if (hexMatch) {
    const hex = hexMatch[1];
    if (hex.length === 3 || hex.length === 4) {
      const alpha = hex.length === 4 ? parseInt(hex[3] + hex[3], 16) / 255 : 1;
      return {
        r: parseInt(hex[0] + hex[0], 16),
        g: parseInt(hex[1] + hex[1], 16),
        b: parseInt(hex[2] + hex[2], 16),
        a: clamp(alpha, 0, 1),
      };
    }
    const alpha = hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1;
    return {
      r: parseInt(hex.slice(0, 2), 16),
      g: parseInt(hex.slice(2, 4), 16),
      b: parseInt(hex.slice(4, 6), 16),
      a: clamp(alpha, 0, 1),
    };
  }

  const rgbaMatch = input.match(/^rgba?\((.+)\)$/i);
  if (!rgbaMatch) return null;
  const parts = rgbaMatch[1].split(",").map((item) => item.trim());
  if (parts.length < 3) return null;
  const r = Number(parts[0]);
  const g = Number(parts[1]);
  const b = Number(parts[2]);
  if (!Number.isFinite(r) || !Number.isFinite(g) || !Number.isFinite(b)) return null;
  const a = parts.length >= 4 ? Number(parts[3]) : 1;
  if (!Number.isFinite(a)) return null;
  return {
    r: clamp(r, 0, 255),
    g: clamp(g, 0, 255),
    b: clamp(b, 0, 255),
    a: clamp(a, 0, 1),
  };
}

function isTransparentColor(raw) {
  const rgba = parseToRGBA(raw);
  if (!rgba) return false;
  return rgba.a <= 0.001;
}

function normalizeColorValue(raw) {
  const input = String(raw || "").trim();
  if (!input) return "";

  const hexMatch = input.match(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/);
  if (hexMatch) {
    return `#${hexMatch[1].toLowerCase()}`;
  }

  const rgbaMatch = input.match(/^rgba?\((.+)\)$/i);
  if (!rgbaMatch) return "";
  const parts = rgbaMatch[1].split(",").map((item) => item.trim());
  if (parts.length < 3) return "";

  const r = Number(parts[0]);
  const g = Number(parts[1]);
  const b = Number(parts[2]);
  if (!Number.isFinite(r) || !Number.isFinite(g) || !Number.isFinite(b)) return "";

  const rr = Math.round(clamp(r, 0, 255));
  const gg = Math.round(clamp(g, 0, 255));
  const bb = Math.round(clamp(b, 0, 255));

  if (parts.length < 4) {
    return `rgba(${rr},${gg},${bb},1)`;
  }
  const a = Number(parts[3]);
  if (!Number.isFinite(a)) return "";
  const aa = clamp(a, 0, 1);
  return `rgba(${rr},${gg},${bb},${aa})`;
}

function commit(value) {
  const normalized = normalizeColorValue(value);
  if (!normalized) return;
  draft.value = normalized;
  emit("update:value", normalized);
}

function onPreset(color) {
  if (props.disabled) return;
  commit(color);
}

function onApplyDraft() {
  if (props.disabled) return;
  commit(draft.value);
}

function onDraftInput(value) {
  draft.value = String(value || "");
}

function onNativeColorInput(event) {
  if (props.disabled) return;
  const value = String(event?.target?.value || "").trim();
  if (!value) return;
  commit(value);
}

</script>

<template>
  <div class="pure_color_input" :class="{ disabled }">
    <n-popover
      trigger="click"
      placement="bottom-start"
      :disabled="disabled"
      to="body"
    >
      <template #trigger>
        <button
          type="button"
          class="pure_color_trigger"
          :class="{ transparent: isTransparentColor(previewColor) }"
          :disabled="disabled"
          :style="{ backgroundColor: previewColor }"
        />
      </template>
      <div class="pure_color_panel" @click.stop>
        <div class="pure_color_presets">
          <button
            v-for="color in PRESET_COLORS"
            :key="color"
            type="button"
            class="pure_color_preset"
            :class="{ transparent: isTransparentColor(color) }"
            :style="{ backgroundColor: color }"
            :title="color"
            @click="onPreset(color)"
          />
        </div>
        <div class="pure_color_tools">
          <input
            class="pure_native_picker"
            type="color"
            :disabled="disabled"
            :value="nativeColorValue"
            @input="onNativeColorInput"
          />
          <n-input
            class="pure_color_text"
            size="small"
            :disabled="disabled"
            :value="draft"
            placeholder="rgba(59,130,246,0.8) / #3b82f6"
            @update:value="onDraftInput"
            @blur="onApplyDraft"
            @keydown.enter.prevent="onApplyDraft"
          />
          <n-button size="small" :disabled="disabled" @click="onApplyDraft">应用</n-button>
        </div>
        <n-text depth="3" style="font-size: 11px">
          支持 rgba(...) / rgb(...) / #RGB / #RRGGBB / #RRGGBBAA
        </n-text>
      </div>
    </n-popover>
  </div>
</template>

<style scoped>
.pure_color_input {
  position: relative;
  display: inline-flex;
  align-items: center;
}

.pure_color_trigger {
  width: 26px;
  height: 26px;
  border-radius: 6px;
  border: 1px solid #64748b;
  cursor: pointer;
  padding: 0;
  background-image:
    linear-gradient(45deg, rgba(148, 163, 184, 0.38) 25%, transparent 25%),
    linear-gradient(-45deg, rgba(148, 163, 184, 0.38) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, rgba(148, 163, 184, 0.38) 75%),
    linear-gradient(-45deg, transparent 75%, rgba(148, 163, 184, 0.38) 75%);
  background-size: 8px 8px;
  background-position: 0 0, 0 4px, 4px -4px, -4px 0;
}

.pure_color_trigger.transparent::after,
.pure_color_preset.transparent::after {
  content: "";
  display: block;
  width: 100%;
  height: 100%;
  background: linear-gradient(135deg, transparent 45%, rgba(239, 68, 68, 0.9) 45%, rgba(239, 68, 68, 0.9) 55%, transparent 55%);
}

.pure_color_input.disabled .pure_color_trigger {
  opacity: 0.55;
  cursor: not-allowed;
}

.pure_color_panel {
  position: relative;
  z-index: 100000;
  min-width: 260px;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid #334155;
  background: #111827;
  box-shadow: 0 10px 28px rgba(0, 0, 0, 0.45);
}

.pure_color_presets {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 6px;
  margin-bottom: 8px;
}

.pure_color_preset {
  position: relative;
  width: 22px;
  height: 22px;
  border-radius: 5px;
  border: 1px solid #64748b;
  cursor: pointer;
  padding: 0;
  background-image:
    linear-gradient(45deg, rgba(148, 163, 184, 0.38) 25%, transparent 25%),
    linear-gradient(-45deg, rgba(148, 163, 184, 0.38) 25%, transparent 25%),
    linear-gradient(45deg, transparent 75%, rgba(148, 163, 184, 0.38) 75%),
    linear-gradient(-45deg, transparent 75%, rgba(148, 163, 184, 0.38) 75%);
  background-size: 8px 8px;
  background-position: 0 0, 0 4px, 4px -4px, -4px 0;
}

.pure_color_tools {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 6px;
}

.pure_native_picker {
  width: 30px;
  height: 26px;
  border: 1px solid #64748b;
  border-radius: 6px;
  background: #0f172a;
  padding: 0;
}

.pure_color_text {
  flex: 1;
  min-width: 0;
}
</style>

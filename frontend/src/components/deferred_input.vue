<script setup>
import { computed, ref, useAttrs, watch } from "vue";

defineOptions({ inheritAttrs: false });

const props = defineProps({
  value: { type: [String, Number], default: "" },
});

const emit = defineEmits(["update:value"]);
const attrs = useAttrs();
const draft = ref(String(props.value ?? ""));
const inputType = computed(() => String(attrs.type || "").trim().toLowerCase());

watch(
  () => props.value,
  (next) => {
    const normalized = String(next ?? "");
    if (normalized !== draft.value) {
      draft.value = normalized;
    }
  },
);

function commit() {
  const normalized = String(draft.value ?? "");
  if (normalized !== String(props.value ?? "")) {
    emit("update:value", normalized);
  }
}

function handleUpdateValue(value) {
  draft.value = String(value ?? "");
}

function handleKeydown(event) {
  if (inputType.value === "textarea") return;
  if (event.key === "Enter") {
    commit();
  }
}
</script>

<template>
  <n-input
    v-bind="attrs"
    :value="draft"
    @update:value="handleUpdateValue"
    @blur="commit"
    @keydown="handleKeydown"
  />
</template>


<script setup>
import { ref, useAttrs, watch } from "vue";

defineOptions({ inheritAttrs: false });

const props = defineProps({
  value: { type: Number, default: null },
});

const emit = defineEmits(["update:value"]);
const attrs = useAttrs();
const draft = ref(props.value ?? null);

watch(
  () => props.value,
  (next) => {
    if (next !== draft.value) {
      draft.value = next ?? null;
    }
  },
);

function commit() {
  if (draft.value !== props.value) {
    emit("update:value", draft.value);
  }
}

function handleUpdateValue(value) {
  draft.value = value ?? null;
}

function handleKeydown(event) {
  if (event.key === "Enter") {
    commit();
  }
}
</script>

<template>
  <n-input-number
    v-bind="attrs"
    :value="draft"
    @update:value="handleUpdateValue"
    @blur="commit"
    @keydown="handleKeydown"
  />
</template>


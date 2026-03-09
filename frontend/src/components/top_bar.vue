<script setup>
import { computed } from "vue";

const props = defineProps({
  profiles: { type: Array, default: () => [] },
  activeProfile: { type: String, default: "" },
  editingProfile: { type: String, default: "" },
  readonlyProfile: { type: Boolean, default: false },
  dirty: { type: Boolean, default: false },
  saving: { type: Boolean, default: false },
});

const emit = defineEmits([
  "update:editing-profile",
  "switch-profile",
  "save",
  "create-profile",
  "rename-profile",
  "delete-profile",
  "import-config",
  "export-config",
]);

const profileOptions = computed(() =>
  (props.profiles || []).map((item) => ({
    label: `${item.name}${item.readonly ? " (builtin)" : ""}`,
    value: item.name,
  })),
);

const canSetDefault = computed(
  () => !!props.editingProfile && props.editingProfile !== props.activeProfile,
);

const saveState = computed(() => {
  if (props.saving) return { type: "info", text: "保存中" };
  if (props.dirty) return { type: "warning", text: "未保存" };
  return { type: "success", text: "已同步" };
});

function onProfileUpdate(value) {
  emit("update:editing-profile", String(value || ""));
}
</script>

<template>
  <header class="top_bar">
    <n-space justify="space-between" align="center" :wrap="true" size="small">
      <n-space align="center" :wrap="true" size="small">
        <n-text depth="3">配置</n-text>
        <n-select
          :value="editingProfile"
          :options="profileOptions"
          size="small"
          style="width: 240px"
          @update:value="onProfileUpdate"
        />
        <n-button size="small" @click="emit('create-profile')">新建</n-button>
        <n-button size="small" :disabled="readonlyProfile" @click="emit('rename-profile')">重命名</n-button>
        <n-popconfirm @positive-click="emit('delete-profile')">
          <template #trigger>
            <n-button size="small" :disabled="readonlyProfile" type="error" tertiary>删除</n-button>
          </template>
          删除当前配置？
        </n-popconfirm>
        <n-button
          v-if="canSetDefault"
          type="primary"
          size="small"
          @click="emit('switch-profile')"
        >
          设为默认
        </n-button>
        <n-tag v-if="readonlyProfile" size="small" type="warning">内置只读</n-tag>
      </n-space>

      <n-space align="center" :wrap="true" size="small">
        <n-button size="small" @click="emit('import-config')">导入</n-button>
        <n-button size="small" @click="emit('export-config')">导出</n-button>
        <n-button
          type="primary"
          size="small"
          :disabled="readonlyProfile || !dirty || saving"
          @click="emit('save')"
        >
          保存应用
        </n-button>
        <n-tag size="small" :type="saveState.type">{{ saveState.text }}</n-tag>
      </n-space>
    </n-space>
  </header>
</template>

<script setup>
import { computed } from "vue";

const props = defineProps({
  profiles: { type: Array, default: () => [] },
  activeProfile: { type: String, default: "" },
  editingProfile: { type: String, default: "" },
  loadedProfile: { type: String, default: "" },
  readonlyProfile: { type: Boolean, default: false },
  dirty: { type: Boolean, default: false },
  saving: { type: Boolean, default: false },
  profileLoading: { type: Boolean, default: false },
  canUndo: { type: Boolean, default: false },
});

const emit = defineEmits([
  "update:editing-profile",
  "switch-profile",
  "save",
  "undo",
  "restore",
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
  () => !!props.editingProfile && !props.profileLoading && props.editingProfile === props.loadedProfile && props.editingProfile !== props.activeProfile,
);

const profileReady = computed(
  () => !props.profileLoading && (!!props.editingProfile ? props.editingProfile === props.loadedProfile : true),
);

const saveState = computed(() => {
  if (props.profileLoading) return { type: "info", text: "切换中" };
  if (!profileReady.value) return { type: "warning", text: "未加载" };
  if (props.saving) return { type: "info", text: "保存中" };
  if (props.dirty) return { type: "warning", text: "未保存" };
  return { type: "success", text: "已保存" };
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
          :disabled="profileLoading || saving"
          @update:value="onProfileUpdate"
        />
        <n-button size="small" :disabled="profileLoading || saving" @click="emit('create-profile')">新建</n-button>
        <n-button size="small" :disabled="readonlyProfile || profileLoading || saving" @click="emit('rename-profile')">重命名</n-button>
        <n-popconfirm @positive-click="emit('delete-profile')">
          <template #trigger>
            <n-button size="small" :disabled="readonlyProfile || profileLoading || saving" type="error" tertiary>删除</n-button>
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
        <n-button size="small" :disabled="profileLoading || saving" @click="emit('import-config')">导入</n-button>
        <n-button size="small" :disabled="profileLoading || saving" @click="emit('export-config')">导出</n-button>
        <n-button
          size="small"
          :disabled="readonlyProfile || !canUndo || saving || profileLoading || !profileReady"
          @click="emit('undo')"
        >
          撤销
        </n-button>
        <n-button
          v-if="dirty"
          size="small"
          :disabled="readonlyProfile || saving || profileLoading || !profileReady"
          @click="emit('restore')"
        >
          恢复
        </n-button>
        <n-button
          type="primary"
          size="small"
          :disabled="readonlyProfile || !dirty || saving || profileLoading || !profileReady"
          @click="emit('save')"
        >
          保存
        </n-button>
        <n-tag size="small" :type="saveState.type">{{ saveState.text }}</n-tag>
      </n-space>
    </n-space>
  </header>
</template>

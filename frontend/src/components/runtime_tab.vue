<script setup>
import { computed, h, onBeforeUnmount, onMounted, ref } from "vue";

const props = defineProps({
  snapshot: { type: Object, default: null },
});

const filter = ref("");
const tableWrapRef = ref(null);
const tableMaxHeight = ref(320);
let tableResizeObserver = null;

const runtimeRows = computed(() => {
  const values = props.snapshot?.values || {};
  const list = Object.entries(values).map(([name, item]) => ({
    name,
    available: !!item?.available,
    label: item?.label || "",
    display: String(item?.text ?? "-"),
  }));
  list.sort((a, b) => a.name.localeCompare(b.name));
  const keyword = filter.value.trim().toLowerCase();
  if (!keyword) return list;
  return list.filter(
    (row) => row.name.toLowerCase().includes(keyword) || row.label.toLowerCase().includes(keyword),
  );
});

const systemRows = computed(() => {
  const values = props.snapshot?.values || {};
  const list = Object.entries(values)
    .filter(([name]) => String(name || "").startsWith("go_native.system"))
    .map(([name, item]) => ({
      name,
      available: !!item?.available,
      display: String(item?.text ?? "-"),
    }));
  list.sort((a, b) => a.name.localeCompare(b.name));
  return list;
});

const columns = computed(() => [
  { title: "name", key: "name", minWidth: 240, ellipsis: { tooltip: true } },
  { title: "label", key: "label", minWidth: 160, ellipsis: { tooltip: true } },
  {
    title: "available",
    key: "available",
    width: 100,
    render: (row) =>
      h(
        "span",
        { class: row.available ? "runtime_avail_on" : "runtime_avail_off" },
        row.available ? "yes" : "no",
      ),
  },
  { title: "value", key: "display", minWidth: 140, ellipsis: { tooltip: true } },
]);

function updateTableHeight() {
  const el = tableWrapRef.value;
  if (!el) return;
  tableMaxHeight.value = Math.max(220, Math.floor(el.clientHeight));
}

onMounted(() => {
  updateTableHeight();
  if (typeof ResizeObserver !== "undefined") {
    tableResizeObserver = new ResizeObserver(() => updateTableHeight());
    if (tableWrapRef.value) tableResizeObserver.observe(tableWrapRef.value);
  }
  window.addEventListener("resize", updateTableHeight);
});

onBeforeUnmount(() => {
  if (tableResizeObserver) {
    tableResizeObserver.disconnect();
    tableResizeObserver = null;
  }
  window.removeEventListener("resize", updateTableHeight);
});
</script>

<template>
  <section class="runtime_layout">
    <n-card class="runtime_meta" title="系统指标" size="small">
      <div class="runtime_sys_list">
        <div v-for="row in systemRows" :key="row.name" class="runtime_sys_row">
          <span class="runtime_sys_name">{{ row.name }}</span>
          <strong :class="row.available ? 'runtime_sys_value' : 'runtime_sys_value runtime_avail_off'">
            {{ row.display }}
          </strong>
        </div>
        <n-empty v-if="systemRows.length === 0" size="small" description="无系统指标" />
      </div>
    </n-card>

    <n-card class="runtime_values" title="监控项明细" size="small">
      <n-input v-model:value="filter" size="small" clearable placeholder="搜索监控项" style="margin-bottom: 8px" />
      <div ref="tableWrapRef" class="runtime_table_holder">
        <n-data-table
          size="small"
          :columns="columns"
          :data="runtimeRows"
          :pagination="false"
          :max-height="tableMaxHeight"
          :style="{ height: `${tableMaxHeight}px` }"
        />
      </div>
    </n-card>
  </section>
</template>

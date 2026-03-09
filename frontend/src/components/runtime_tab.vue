<script setup>
import { computed, h, onBeforeUnmount, onMounted, ref } from "vue";

const props = defineProps({
  snapshot: { type: Object, default: null },
});

const filter = ref("");
const tableWrapRef = ref(null);
const tableMaxHeight = ref(320);
let tableResizeObserver = null;

const runtimeStats = computed(() => props.snapshot?.monitor_runtime || null);

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

function fmtTime(raw) {
  if (!raw) return "-";
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return String(raw);
  return d.toLocaleString();
}

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
      <n-grid cols="1 s:2 m:3" responsive="screen" :x-gap="8" :y-gap="8">
        <n-gi><div class="runtime_stat"><span>模式</span><strong>{{ snapshot?.mode || '-' }}</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>更新时间</span><strong>{{ fmtTime(snapshot?.updated_at) }}</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>监控项数</span><strong>{{ Object.keys(snapshot?.values || {}).length }}</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>collect.max</span><strong>{{ runtimeStats?.collect_max_ms || 0 }}ms</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>collect.avg</span><strong>{{ runtimeStats?.collect_avg_ms || 0 }}ms</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>render.max</span><strong>{{ runtimeStats?.render_max_ms || 0 }}ms</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>render.avg</span><strong>{{ runtimeStats?.render_avg_ms || 0 }}ms</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>output.max</span><strong>{{ runtimeStats?.output_max_ms || 0 }}ms</strong></div></n-gi>
        <n-gi><div class="runtime_stat"><span>output.avg</span><strong>{{ runtimeStats?.output_avg_ms || 0 }}ms</strong></div></n-gi>
      </n-grid>
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

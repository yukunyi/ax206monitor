import { normalizeThresholdGroups } from "./config_normalizer";
import { normalizeMonitorName } from "./monitor_aliases";
import { inferUnitRangeProfile, normalizeRangeUnitToken } from "./range_profiles";

const AUTO_RANGE_COLORS = ["#22c55e", "#eab308", "#f97316", "#ef4444"];
const THROUGHPUT_MAX_BYTES_PER_SEC = 100 * 1024 * 1024;
const RPM_GROUP_BASE = Object.freeze({
  family: "rpm",
  min: 0,
  max: 2500,
});
const FREQUENCY_MAX_HZ = 5 * 1000 * 1000 * 1000;

function normalizeText(value) {
  return String(value || "").trim();
}

function normalizeSearchText(value) {
  return normalizeText(value).toLowerCase();
}

function parseFiniteNumber(value) {
  const number = Number(value);
  return Number.isFinite(number) ? number : null;
}

function formatNumberToken(value) {
  if (!Number.isFinite(value)) return "0";
  if (Number.isInteger(value)) return String(value);
  return String(value).replace(/[^\d.-]+/g, "_").replace(/\./g, "_");
}

function isPercentUnit(unit) {
  const normalized = normalizeRangeUnitToken(unit);
  return normalized === "%" || normalized === "percent" || normalized === "percentage" || normalized === "pct";
}

function isTemperatureCandidate(unit, name, label) {
  const normalizedUnit = normalizeRangeUnitToken(unit);
  if (normalizedUnit === "°c" || normalizedUnit === "℃" || normalizedUnit === "celsius" || normalizedUnit === "c") {
    return true;
  }
  if (normalizedUnit) {
    return false;
  }
  const text = `${normalizeSearchText(name)} ${normalizeSearchText(label)}`;
  return /\btemp\b|\btemperature\b|温度/.test(text);
}

function isPercentCandidate(unit, name, label) {
  const normalizedUnit = normalizeRangeUnitToken(unit);
  if (isPercentUnit(normalizedUnit)) return true;
  if (normalizedUnit) return false;
  const text = `${normalizeSearchText(name)} ${normalizeSearchText(label)}`;
  return /\bpercent\b|\bpercentage\b|\busage\b|\butil(?:ization)?\b|\bload\b|\bprogress\b|\bduty\b/.test(text);
}

function throughputUnitScale(unit) {
  switch (normalizeRangeUnitToken(unit)) {
    case "b/s":
      return 1;
    case "kb/s":
    case "kib/s":
      return 1024;
    case "mb/s":
    case "mib/s":
      return 1024 * 1024;
    case "gb/s":
    case "gib/s":
      return 1024 * 1024 * 1024;
    case "tb/s":
    case "tib/s":
      return 1024 * 1024 * 1024 * 1024;
    default:
      return 0;
  }
}

function frequencyUnitScale(unit) {
  switch (normalizeRangeUnitToken(unit)) {
    case "hz":
      return 1;
    case "khz":
      return 1000;
    case "mhz":
      return 1000 * 1000;
    case "ghz":
      return 1000 * 1000 * 1000;
    case "thz":
      return 1000 * 1000 * 1000 * 1000;
    default:
      return 0;
  }
}

function buildNamedRangeSpec(family, min, max) {
  return {
    name: `${family}_${formatNumberToken(min)}_${formatNumberToken(max)}`,
    min,
    max,
  };
}

function inferThroughputSpec(unit) {
  const scale = throughputUnitScale(unit);
  if (!scale) return null;
  return buildNamedRangeSpec("throughput", 0, THROUGHPUT_MAX_BYTES_PER_SEC / scale);
}

function inferRPMSpec(unit, name, label) {
  const normalizedUnit = normalizeRangeUnitToken(unit);
  if (normalizedUnit === "rpm") {
    return buildNamedRangeSpec(RPM_GROUP_BASE.family, RPM_GROUP_BASE.min, RPM_GROUP_BASE.max);
  }
  if (normalizedUnit) {
    return null;
  }
  const text = `${normalizeSearchText(name)} ${normalizeSearchText(label)}`;
  if (!/\brpm\b/.test(text)) {
    return null;
  }
  return buildNamedRangeSpec(RPM_GROUP_BASE.family, RPM_GROUP_BASE.min, RPM_GROUP_BASE.max);
}

function inferFrequencySpec(unit) {
  const scale = frequencyUnitScale(unit);
  if (!scale) return null;
  return buildNamedRangeSpec("frequency", 0, FREQUENCY_MAX_HZ / scale);
}

function createStandardRanges(min, max) {
  const start = Number(min);
  const end = Number(max);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return [];
  }
  const step = (end - start) / 4;
  return AUTO_RANGE_COLORS.map((color, index) => {
    const entry = { color };
    if (index > 0) {
      entry.min = start + step * index;
    }
    entry.max = start + step * (index + 1);
    return entry;
  });
}

function collectMonitorMeta({ config, snapshot, monitorOptions }) {
  const candidates = new Map();

  function ensureCandidate(rawName) {
    const name = normalizeMonitorName(rawName);
    if (!name) return null;
    const existing = candidates.get(name);
    if (existing) return existing;
    const created = { name, label: "", unit: "", min: null, max: null };
    candidates.set(name, created);
    return created;
  }

  Object.entries(snapshot?.values || {}).forEach(([rawName, value]) => {
    const candidate = ensureCandidate(rawName);
    if (!candidate) return;
    if (!candidate.label) candidate.label = normalizeText(value?.label);
    if (!candidate.unit) candidate.unit = normalizeText(value?.unit);
  });

  (config?.items || []).forEach((item) => {
    ensureCandidate(item?.monitor);
  });

  (config?.custom_monitors || []).forEach((item) => {
    const candidate = ensureCandidate(item?.name);
    if (!candidate) return;
    if (!candidate.label) candidate.label = normalizeText(item?.label);
    if (!candidate.unit) candidate.unit = normalizeText(item?.unit);
    if (candidate.min === null) candidate.min = parseFiniteNumber(item?.min);
    if (candidate.max === null) candidate.max = parseFiniteNumber(item?.max);
  });

  (monitorOptions || []).forEach((option) => {
    const candidate = ensureCandidate(option?.value);
    if (!candidate) return;
    if (!candidate.label) candidate.label = normalizeText(option?.label);
  });

  return [...candidates.values()];
}

function inferThresholdSpec(candidate) {
  if (!candidate || !candidate.name) return null;
  const { name, label, unit, min, max } = candidate;
  const unitProfile = inferUnitRangeProfile(unit);
  if (unitProfile) {
    return unitProfile;
  }
  if (isTemperatureCandidate(unit, name, label)) {
    return inferUnitRangeProfile("°C");
  }
  if (isPercentCandidate(unit, name, label)) {
    return inferUnitRangeProfile("%");
  }
  const throughputSpec = inferThroughputSpec(unit);
  if (throughputSpec) {
    return throughputSpec;
  }
  const rpmSpec = inferRPMSpec(unit, name, label);
  if (rpmSpec) {
    return rpmSpec;
  }
  const frequencySpec = inferFrequencySpec(unit);
  if (frequencySpec) {
    return frequencySpec;
  }
  if (Number.isFinite(min) && Number.isFinite(max) && max > min) {
    return buildNamedRangeSpec("range", min, max);
  }
  return null;
}

function buildAutoGroups({ config, snapshot, monitorOptions }) {
  const grouped = new Map();
  collectMonitorMeta({ config, snapshot, monitorOptions }).forEach((candidate) => {
    const spec = inferThresholdSpec(candidate);
    if (!spec) return;
    const entry = grouped.get(spec.name) || {
      name: spec.name,
      monitors: [],
      ranges: createStandardRanges(spec.min, spec.max),
    };
    entry.monitors.push(candidate.name);
    grouped.set(spec.name, entry);
  });
  return normalizeThresholdGroups([...grouped.values()]);
}

export function buildMergedAutoThresholdGroups({ existingGroups = [], config, snapshot, monitorOptions = [] }) {
  const merged = new Map();
  normalizeThresholdGroups(existingGroups).forEach((group) => {
    merged.set(group.name, {
      name: group.name,
      monitors: Array.isArray(group.monitors) ? [...group.monitors] : [],
      ranges: Array.isArray(group.ranges) ? group.ranges.map((item) => ({ ...(item || {}) })) : [],
    });
  });
  buildAutoGroups({ config, snapshot, monitorOptions }).forEach((group) => {
    const previous = merged.get(group.name);
    if (!previous) {
      merged.set(group.name, group);
      return;
    }
    previous.monitors = [...new Set([...(previous.monitors || []), ...(group.monitors || [])])];
    previous.ranges = group.ranges;
    merged.set(group.name, previous);
  });
  return normalizeThresholdGroups([...merged.values()]);
}

export function applyAutoThresholdGroupsToConfig(config, context = {}) {
  const next = {
    ...(config || {}),
  };
  next.threshold_groups = buildMergedAutoThresholdGroups({
    existingGroups: next.threshold_groups,
    config: next,
    snapshot: context.snapshot,
    monitorOptions: context.monitorOptions,
  });
  return next;
}

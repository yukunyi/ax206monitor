function normalizeText(value) {
  return String(value || "").trim();
}

export function normalizeRangeUnitToken(unit) {
  return normalizeText(unit).toLowerCase().replace(/\s+/g, "");
}

export function inferUnitRangeProfile(unit) {
  const normalized = normalizeRangeUnitToken(unit);
  if (normalized === "°c" || normalized === "℃" || normalized === "celsius") {
    return {
      name: "temperature_30_110",
      min: 30,
      max: 110,
    };
  }
  if (normalized === "%" || normalized === "percent" || normalized === "percentage" || normalized === "pct") {
    return {
      name: "percent_0_100",
      min: 0,
      max: 100,
    };
  }
  return null;
}

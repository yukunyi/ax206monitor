function shouldRemoveValue(value, removeEmptyString = false) {
  if (value === undefined || value === null) return true;
  if (!removeEmptyString) return false;
  return typeof value === "string" && value.trim() === "";
}

export function patchObjectKey(source, key, value, options = {}) {
  const name = String(key || "").trim();
  if (!name) return { ...(source || {}) };
  const next = { ...(source || {}) };
  const removeEmptyString = options.removeEmptyString === true;
  if (shouldRemoveValue(value, removeEmptyString)) {
    delete next[name];
    return next;
  }
  next[name] = value;
  return next;
}


export function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value === false || value === null || value === undefined) continue;
    if (key === "class") node.className = value;
    else if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value === true ? "" : value);
  }
  append(node, children);
  return node;
}

function append(node, children) {
  for (const child of children.flat(Infinity)) {
    if (child === null || child === undefined || child === false) continue;
    node.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
}

// panel is a bordered frame with small squares on its corners.
export function panel(attrs, ...children) {
  return el(
    "div",
    { ...attrs, class: `panel ${attrs.class || ""}`.trim() },
    ["tl", "tr", "bl", "br"].map((corner) => el("span", { class: `tick ${corner}`, "aria-hidden": "true" })),
    children,
  );
}

export function state(status) {
  return el("span", { class: `state ${status}` }, "● ", status);
}

const KIND_LABELS = { host: "host", vm: "vm", container: "container", appliance: "appliance" };

export function kindLabel(kind) {
  return KIND_LABELS[kind] || kind;
}

export function pageTitle(text, ...after) {
  return el("div", { class: "page-head" }, el("h1", { class: "title" }, text), after);
}

export function emptyState(title, text) {
  return el("div", { class: "empty" }, el("div", { class: "empty-title" }, title), el("p", {}, text));
}

export function searchBox(placeholder, value, oninput) {
  return el(
    "label",
    { class: "search" },
    el("span", { class: "search-key", "aria-hidden": "true" }, "/"),
    el("input", { type: "search", placeholder, value, "data-search": true, "aria-label": placeholder, oninput }),
  );
}

export function chips(options, value, onpick, label) {
  return el(
    "div",
    { class: "chips", role: "group", "aria-label": label },
    options.map(([v, text]) =>
      el("button", { type: "button", class: v === value ? "chip on" : "chip", "aria-pressed": String(v === value), onclick: () => onpick(v) }, text),
    ),
  );
}

export function bar(fraction, width) {
  const filled = Math.round(Math.max(0, Math.min(1, fraction)) * width);
  return "█".repeat(filled) + "░".repeat(width - filled);
}

export function relativeTime(date) {
  const seconds = Math.max(0, Math.round((Date.now() - date.getTime()) / 1000));
  if (seconds < 5) return "just now";
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  return `${Math.round(minutes / 60)}h ago`;
}

export function byName(a, b) {
  return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" });
}

export function matches(query, ...fields) {
  const q = query.trim().toLowerCase();
  return !q || fields.some((f) => String(f || "").toLowerCase().includes(q));
}

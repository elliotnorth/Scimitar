const SVG_NS = "http://www.w3.org/2000/svg";

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

export function searchIcon() {
  const svg = document.createElementNS(SVG_NS, "svg");
  svg.setAttribute("width", "15");
  svg.setAttribute("height", "15");
  svg.setAttribute("viewBox", "0 0 24 24");
  svg.setAttribute("fill", "none");
  svg.setAttribute("stroke", "currentColor");
  svg.setAttribute("stroke-width", "2");
  const circle = document.createElementNS(SVG_NS, "circle");
  circle.setAttribute("cx", "11");
  circle.setAttribute("cy", "11");
  circle.setAttribute("r", "7");
  const line = document.createElementNS(SVG_NS, "path");
  line.setAttribute("d", "M20 20l-3.5-3.5");
  line.setAttribute("stroke-linecap", "round");
  svg.append(circle, line);
  return svg;
}

const STATUS_LABELS = { up: "Up", down: "Down", unknown: "Unknown" };

export function statusBadge(status) {
  return el("span", { class: "status" }, el("span", { class: `dot ${status}` }), STATUS_LABELS[status] || status);
}

export function statusPill(status, label) {
  return el("span", { class: "pill", title: STATUS_LABELS[status] }, el("span", { class: `dot ${status}` }), label);
}

const KIND_LABELS = { host: "Host", vm: "Virtual machine", container: "Container", appliance: "Appliance" };

export function kindLabel(kind) {
  return KIND_LABELS[kind] || kind;
}

export function emptyState(title, text) {
  return el("div", { class: "empty" }, el("h3", {}, title), el("p", {}, text));
}

export function dataTable(columns, rows) {
  return el(
    "div",
    { class: "table-wrap" },
    el(
      "table",
      {},
      el("thead", {}, el("tr", {}, columns.map((c) => el("th", { class: c.class }, c.label)))),
      el("tbody", {}, rows),
    ),
  );
}

export function searchBox(placeholder, value, oninput) {
  return el(
    "label",
    { class: "search" },
    searchIcon(),
    el("input", { type: "search", placeholder, value, "data-search": true, "aria-label": placeholder, oninput }),
    el("kbd", {}, "/"),
  );
}

export function select(options, value, onchange, label) {
  return el(
    "select",
    { "aria-label": label, onchange },
    options.map(([v, text]) => el("option", { value: v, selected: v === value }, text)),
  );
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

"use strict";

const REFRESH_MS = 10000;
const SOURCE_LABELS = { name: "Name", tag: "Tag", notes: "Notes" };

const state = {
  snapshot: null,
  catalog: [],
  openDetails: new Map(),
};

function el(tag, attrs = {}, ...children) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (key === "class") node.className = value;
    else if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value);
  }
  for (const child of children.flat()) {
    if (child !== null && child !== undefined) {
      node.append(child instanceof Node ? child : document.createTextNode(String(child)));
    }
  }
  return node;
}

function dot(status) {
  return el("span", { class: `dot ${status}`, title: status });
}

async function getJSON(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error((await res.json().catch(() => ({}))).error || res.statusText);
  return res.json();
}

async function decide(key, catalogID) {
  const res = await fetch("api/review", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ key, catalog_id: catalogID }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    alert(`Could not save: ${body.error || res.statusText}`);
  }
  await refresh();
}

function catalogName(id) {
  return state.catalog.find((c) => c.id === id)?.name || id;
}

function renderSummary(s) {
  const counts = { up: 0, down: 0, unknown: 0 };
  for (const d of s.devices) counts[d.status]++;
  const pills = [
    el("span", { class: "pill" }, dot("up"), `${counts.up} up`),
    el("span", { class: "pill" }, dot("down"), `${counts.down} down`),
    el("span", { class: "pill" }, dot("unknown"), `${counts.unknown} unknown`),
  ];
  if (s.review.length) pills.push(el("span", { class: "pill" }, `${s.review.length} to review`));
  document.getElementById("summary").replaceChildren(...pills);
  document.getElementById("updated").textContent = s.updated_at
    ? `Updated ${new Date(s.updated_at).toLocaleTimeString()}`
    : "Waiting for first discovery";
}

function renderErrors(s) {
  const items = Object.entries(s.errors).map(([provider, message]) =>
    el("div", { class: "error" }, el("strong", {}, provider), `: ${message}`),
  );
  document.getElementById("errors").replaceChildren(...items);
}

function renderReview(s, devicesById) {
  const section = document.getElementById("review-section");
  section.hidden = s.review.length === 0;
  document.getElementById("review-count").textContent = s.review.length || "";
  const items = s.review.map((item) => {
    const select = el(
      "select",
      { "aria-label": "Other service" },
      el("option", { value: "" }, "Other service…"),
      state.catalog.map((c) => el("option", { value: c.id }, c.name)),
    );
    return el(
      "div",
      { class: "card review-item" },
      el("div", {}, el("strong", {}, devicesById.get(item.device_id)?.name || item.device_id)),
      el("div", { class: "source" }, SOURCE_LABELS[item.source] || item.source),
      el("div", { class: "value" }, item.value),
      el(
        "div",
        { class: "actions" },
        item.candidate_ids.map((id) =>
          el("button", { class: "primary", onclick: () => decide(item.key, id) }, `It's ${catalogName(id)}`),
        ),
        select,
        el("button", { onclick: () => select.value && decide(item.key, select.value) }, "Confirm"),
        el("button", { onclick: () => decide(item.key, "") }, "Not a service"),
      ),
    );
  });
  document.getElementById("review").replaceChildren(...items);
}

function renderServices(s, devicesById) {
  document.getElementById("service-count").textContent = s.services.length || "";
  const tiles = s.services.map((svc) =>
    el(
      "div",
      { class: "card service" },
      dot(svc.status),
      el("div", {}, el("div", {}, svc.name), el("div", { class: "where" }, devicesById.get(svc.device_id)?.name || "")),
    ),
  );
  document
    .getElementById("services")
    .replaceChildren(...(tiles.length ? tiles : [el("div", { class: "empty" }, "No services detected yet.")]));
}

function detailsBlock(deviceID) {
  const entry = state.openDetails.get(deviceID);
  if (!entry) return null;
  if (entry.error) return el("div", { class: "details" }, el("dd", {}, entry.error));
  if (!entry.details) return el("div", { class: "details" }, el("dd", {}, "Loading…"));
  return el(
    "dl",
    { class: "details" },
    entry.details.flatMap((d) => [el("dt", {}, d.label), el("dd", {}, d.value)]),
  );
}

async function toggleDetails(deviceID) {
  if (state.openDetails.has(deviceID)) {
    state.openDetails.delete(deviceID);
    render();
    return;
  }
  state.openDetails.set(deviceID, {});
  render();
  try {
    const details = await getJSON(`api/details?id=${encodeURIComponent(deviceID)}`);
    if (state.openDetails.has(deviceID)) state.openDetails.set(deviceID, { details });
  } catch (err) {
    if (state.openDetails.has(deviceID)) state.openDetails.set(deviceID, { error: err.message });
  }
  render();
}

function deviceRow(device, servicesByDevice, extraClass) {
  const chips = (servicesByDevice.get(device.id) || []).map((svc) =>
    el("span", { class: "pill" }, dot(svc.status), svc.name),
  );
  return [
    el(
      "div",
      { class: `row ${extraClass}`, onclick: () => toggleDetails(device.id) },
      dot(device.status),
      el("span", { class: "name" }, device.name),
      el("span", { class: "kind" }, device.kind),
      el("span", { class: "chips" }, chips),
    ),
    detailsBlock(device.id),
  ];
}

function renderDevices(s) {
  document.getElementById("device-count").textContent = s.devices.length || "";
  const servicesByDevice = new Map();
  for (const svc of s.services) {
    if (!servicesByDevice.has(svc.device_id)) servicesByDevice.set(svc.device_id, []);
    servicesByDevice.get(svc.device_id).push(svc);
  }
  const children = new Map();
  const parented = new Set();
  for (const link of s.links) {
    if (link.kind !== "runs-on") continue;
    if (!children.has(link.to)) children.set(link.to, []);
    children.get(link.to).push(link.from);
    parented.add(link.from);
  }
  const byId = new Map(s.devices.map((d) => [d.id, d]));
  const roots = s.devices.filter((d) => !parented.has(d.id));
  const cards = roots.map((root) =>
    el(
      "div",
      { class: "card host" },
      deviceRow(root, servicesByDevice, "host-row"),
      (children.get(root.id) || [])
        .map((id) => byId.get(id))
        .filter(Boolean)
        .map((guest) => deviceRow(guest, servicesByDevice, "guest")),
    ),
  );
  document
    .getElementById("devices")
    .replaceChildren(...(cards.length ? cards : [el("div", { class: "empty" }, "No devices discovered yet.")]));
}

function render() {
  const s = state.snapshot;
  if (!s) return;
  const devicesById = new Map(s.devices.map((d) => [d.id, d]));
  renderSummary(s);
  renderErrors(s);
  renderReview(s, devicesById);
  renderServices(s, devicesById);
  renderDevices(s);
}

async function refresh() {
  try {
    state.snapshot = await getJSON("api/snapshot");
    render();
  } catch (err) {
    document.getElementById("errors").replaceChildren(el("div", { class: "error" }, `Scimitar is unreachable: ${err.message}`));
  }
}

async function start() {
  state.catalog = await getJSON("api/catalog").catch(() => []);
  await refresh();
  setInterval(refresh, REFRESH_MS);
}

start();

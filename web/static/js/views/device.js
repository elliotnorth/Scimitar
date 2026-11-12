import { getJSON, LoginRequired } from "../api.js";
import { byName, el, emptyState, kindLabel, panel, state } from "../dom.js";
import { deviceHref, store } from "../store.js";

export function deviceView(id) {
  const head = el("div");
  const body = el("div", { class: "stack" });
  const root = el("div", { class: "wrap page" }, head, body);
  let details = { loading: true };

  getJSON(`api/details?id=${encodeURIComponent(id)}`)
    .then((d) => (details = { items: d }))
    .catch((err) => {
      if (!(err instanceof LoginRequired)) details = { error: err.message };
    })
    .finally(update);

  function update() {
    const device = store.index.devicesById.get(id);
    if (!device) {
      head.replaceChildren();
      body.replaceChildren(emptyState("Device not found", "It may have been removed from its source."), el("a", { class: "lnk", href: "#/devices" }, "← back to devices"));
      return;
    }
    head.replaceChildren(
      el("a", { class: "lnk crumb", href: "#/devices" }, "← devices"),
      el("div", { class: "page-head" }, el("h1", { class: "title" }, device.name), state(device.status)),
      el("div", { class: "mono dim small" }, `${kindLabel(device.kind)} · ${device.provider}`),
    );

    const { servicesByDevice, parentOf, childrenOf, devicesById } = store.index;
    const sections = [section("details", detailsBody(device))];
    const services = servicesByDevice.get(device.id) || [];
    sections.push(section("apps", services.length ? panel({ class: "rows" }, services.map((s) => el("div", { class: "row" }, el("span", { class: "row-main" }, s.name), state(s.status)))) : el("p", { class: "dim" }, "No apps found on this device.")));
    const parent = devicesById.get(parentOf.get(device.id));
    if (parent) sections.push(section("runs on", panel({ class: "rows" }, linkRow(parent))));
    const guests = (childrenOf.get(device.id) || []).map((g) => devicesById.get(g)).filter(Boolean).sort(byName);
    if (guests.length) sections.push(section(`guests · ${guests.length}`, panel({ class: "rows" }, guests.map(linkRow))));
    body.replaceChildren(...sections);
  }

  function detailsBody(device) {
    if (details.loading) return el("p", { class: "dim" }, "loading…");
    if (details.error) return el("p", { class: "dim" }, `Could not load details: ${details.error}`);
    const rows = [...details.items.map((d) => [d.label, d.value]), ["scimitar id", device.id]];
    return panel({ class: "pad" }, el("dl", { class: "kv" }, rows.map(([k, v]) => [el("dt", { class: "cap" }, k.toLowerCase()), el("dd", { class: k === "scimitar id" ? "mono small" : "" }, v)])));
  }

  return { root, update };
}

function section(title, content) {
  return el("section", {}, el("div", { class: "cap section-cap" }, title), content);
}

function linkRow(device) {
  return el("a", { class: "row", href: deviceHref(device.id) }, el("span", { class: "row-main" }, device.name, " ", el("span", { class: "mono dim small" }, kindLabel(device.kind))), state(device.status));
}

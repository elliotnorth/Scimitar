import { getJSON, LoginRequired } from "../api.js";
import { byName, el, emptyState, kindLabel, statusBadge } from "../dom.js";
import { deviceHref, store } from "../store.js";

export function deviceView(id) {
  const head = el("div");
  const body = el("div");
  const root = el("section", {}, head, body);
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
      body.replaceChildren(
        emptyState("Device not found", "It may have been removed from its source."),
        el("p", { class: "empty" }, el("a", { href: "#/devices", class: "btn" }, "Back to devices")),
      );
      return;
    }
    head.replaceChildren(
      el(
        "div",
        { class: "page-head" },
        el(
          "div",
          {},
          el("div", { class: "breadcrumb" }, el("a", { href: "#/devices" }, "Devices"), "/", el("span", {}, device.name)),
          el("h1", {}, device.name),
          el("p", {}, `${kindLabel(device.kind)} from ${device.provider}`),
        ),
        statusBadge(device.status),
      ),
    );
    body.replaceChildren(el("div", { class: "grid-2" }, el("div", {}, detailsCard(device)), el("div", {}, sideCards(device))));
  }

  function detailsCard(device) {
    let content;
    if (details.loading) content = el("p", { class: "muted" }, "Loading details…");
    else if (details.error) content = el("p", { class: "muted" }, `Could not load details: ${details.error}`);
    else {
      const rows = [["Name", device.name], ["Kind", kindLabel(device.kind)], ["Source", device.provider], ...details.items.map((d) => [d.label, d.value]), ["Scimitar ID", device.id]];
      content = el("dl", { class: "kv" }, rows.map(([k, v]) => [el("dt", {}, k), el("dd", { class: k === "Scimitar ID" ? "mono" : "" }, v)]));
    }
    return el("div", { class: "card" }, el("div", { class: "card-head" }, el("h2", {}, "Details")), el("div", { class: "card-body" }, content));
  }

  function sideCards(device) {
    const { servicesByDevice, parentOf, childrenOf, devicesById } = store.index;
    const services = servicesByDevice.get(device.id) || [];
    const cards = [
      listCard(
        "Services",
        services.map((s) => el("div", { class: "list-row" }, el("span", {}, s.name), statusBadge(s.status))),
        "No services found on this device.",
      ),
    ];
    const parent = devicesById.get(parentOf.get(device.id));
    if (parent) cards.push(listCard("Runs on", [deviceRow(parent)]));
    const guests = (childrenOf.get(device.id) || []).map((gid) => devicesById.get(gid)).filter(Boolean).sort(byName);
    if (guests.length) cards.push(listCard(`Guests (${guests.length})`, guests.map(deviceRow)));
    return cards;
  }

  return { root, update };
}

function listCard(title, rows, emptyText) {
  return el(
    "div",
    { class: "card" },
    el("div", { class: "card-head" }, el("h2", {}, title)),
    rows.length ? rows : el("div", { class: "card-body muted" }, emptyText),
  );
}

function deviceRow(device) {
  return el("a", { class: "list-row", href: deviceHref(device.id) }, el("span", {}, device.name), statusBadge(device.status));
}

import { byName, chips, el, emptyState, kindLabel, matches, pageTitle, panel, searchBox, state } from "../dom.js";
import { deviceHref, store } from "../store.js";

const filters = { query: "", status: "" };
const STATUS_OPTIONS = [["", "all"], ["up", "up"], ["down", "down"], ["unknown", "unknown"]];

export function devicesView() {
  const results = el("div", { class: "stack" });
  const statusChips = el("div");
  const root = el(
    "div",
    { class: "wrap page" },
    pageTitle("Devices"),
    el(
      "div",
      { class: "toolbar" },
      searchBox("search devices and apps", filters.query, (e) => {
        filters.query = e.target.value;
        update();
      }),
      statusChips,
    ),
    results,
  );

  function shown(device) {
    const services = store.index.servicesByDevice.get(device.id) || [];
    return (!filters.status || device.status === filters.status) && matches(filters.query, device.name, device.kind, device.provider, ...services.map((s) => s.name));
  }

  function update() {
    statusChips.replaceChildren(
      chips(STATUS_OPTIONS, filters.status, (v) => {
        filters.status = v;
        update();
      }, "Filter by status"),
    );
    const { devices } = store.snapshot;
    if (!devices.length) {
      results.replaceChildren(emptyState("No devices yet", "They show up here after the first discovery. Check settings if this takes more than a minute."));
      return;
    }
    const { childrenOf, parentOf, devicesById } = store.index;
    const groups = [];
    const hosts = devices.filter((d) => childrenOf.has(d.id) || d.kind === "host").sort(byName);
    for (const host of hosts) {
      const guests = (childrenOf.get(host.id) || []).map((id) => devicesById.get(id)).filter(Boolean).sort(byName);
      const visible = guests.filter(shown);
      if (!shown(host) && !visible.length) continue;
      groups.push(
        panel(
          { class: "rows" },
          deviceRow(host, "host-row", `${guests.length} ${guests.length === 1 ? "guest" : "guests"} · ${host.provider}`),
          visible.map((g) => deviceRow(g, "", kindLabel(g.kind))),
        ),
      );
    }
    const loose = devices.filter((d) => !parentOf.has(d.id) && !childrenOf.has(d.id) && d.kind !== "host" && shown(d)).sort(byName);
    if (loose.length) groups.push(panel({ class: "rows" }, loose.map((d) => deviceRow(d, "", `${kindLabel(d.kind)} · ${d.provider}`))));
    results.replaceChildren(...(groups.length ? groups : [emptyState("Nothing matches", "Try a different search or filter.")]));
  }

  return { root, update };
}

function deviceRow(device, extraClass, detail) {
  const services = store.index.servicesByDevice.get(device.id) || [];
  return el(
    "a",
    { class: `row ${extraClass}`.trim(), href: deviceHref(device.id) },
    el("span", { class: "row-main" }, el("span", { class: "row-name" }, device.name), " ", el("span", { class: "mono dim small" }, detail)),
    el("span", { class: "mono dim small hide-sm" }, services.map((s) => s.name).join(" · ")),
    state(device.status),
  );
}

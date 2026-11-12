import { byName, dataTable, el, emptyState, kindLabel, matches, searchBox, select, statusBadge, statusPill } from "../dom.js";
import { deviceHref, parentName, store } from "../store.js";

const filters = { query: "", status: "", kind: "" };

export function devicesView() {
  const results = el("div");
  const root = el(
    "section",
    {},
    el(
      "div",
      { class: "page-head" },
      el("div", {}, el("h1", {}, "Devices"), el("p", {}, "Hosts, virtual machines and containers found by your sources.")),
    ),
    el(
      "div",
      { class: "toolbar" },
      searchBox("Search by name, host or service", filters.query, (e) => {
        filters.query = e.target.value;
        update();
      }),
      select(
        [["", "All statuses"], ["up", "Up"], ["down", "Down"], ["unknown", "Unknown"]],
        filters.status,
        (e) => {
          filters.status = e.target.value;
          update();
        },
        "Filter by status",
      ),
      select(
        [["", "All kinds"], ["host", "Hosts"], ["vm", "Virtual machines"], ["container", "Containers"]],
        filters.kind,
        (e) => {
          filters.kind = e.target.value;
          update();
        },
        "Filter by kind",
      ),
    ),
    results,
  );

  function update() {
    const { devices } = store.snapshot;
    if (!devices.length) {
      results.replaceChildren(emptyState("No devices yet", "Scimitar shows devices here after the first discovery. Check Settings if this takes more than a minute."));
      return;
    }
    const services = store.index.servicesByDevice;
    const shown = devices
      .filter((d) => !filters.status || d.status === filters.status)
      .filter((d) => !filters.kind || d.kind === filters.kind)
      .filter((d) => matches(filters.query, d.name, d.kind, d.provider, parentName(d.id), ...(services.get(d.id) || []).map((s) => s.name)))
      .sort(byName);
    if (!shown.length) {
      results.replaceChildren(emptyState("No matching devices", "Try a different search or filter."));
      return;
    }
    const rows = shown.map((d) =>
      el(
        "tr",
        { class: "clickable", onclick: () => (location.hash = deviceHref(d.id)) },
        el("td", {}, el("div", { class: "cell-title" }, d.name), el("div", { class: "cell-sub" }, `${kindLabel(d.kind)} · ${d.provider}`)),
        el("td", { class: "hide-mobile" }, parentName(d.id) || el("span", { class: "muted" }, "—")),
        el("td", { class: "hide-mobile" }, el("div", { class: "pills" }, (services.get(d.id) || []).map((s) => statusPill(s.status, s.name)))),
        el("td", {}, statusBadge(d.status)),
      ),
    );
    results.replaceChildren(
      dataTable(
        [{ label: "Device" }, { label: "Runs on", class: "hide-mobile" }, { label: "Services", class: "hide-mobile" }, { label: "Status" }],
        rows,
      ),
      el("div", { class: "table-foot" }, `${shown.length} of ${devices.length} devices`),
    );
  }

  return { root, update };
}

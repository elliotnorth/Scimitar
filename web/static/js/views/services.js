import { byName, dataTable, el, emptyState, matches, searchBox, select, statusBadge } from "../dom.js";
import { deviceHref, deviceName, parentName, store } from "../store.js";

const filters = { query: "", status: "" };

export function servicesView() {
  const results = el("div");
  const root = el(
    "section",
    {},
    el(
      "div",
      { class: "page-head" },
      el("div", {}, el("h1", {}, "Services"), el("p", {}, "Apps Scimitar recognized on your devices. A service shows as down when its device is down.")),
    ),
    el(
      "div",
      { class: "toolbar" },
      searchBox("Search services or devices", filters.query, (e) => {
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
    ),
    results,
  );

  function update() {
    const { services } = store.snapshot;
    if (!services.length) {
      results.replaceChildren(emptyState("No services yet", "Name or tag a VM after the app it runs, like immich or nextcloud, and Scimitar will find it."));
      return;
    }
    const shown = services
      .filter((s) => !filters.status || s.status === filters.status)
      .filter((s) => matches(filters.query, s.name, deviceName(s.device_id), parentName(s.device_id)))
      .sort(byName);
    if (!shown.length) {
      results.replaceChildren(emptyState("No matching services", "Try a different search or filter."));
      return;
    }
    const rows = shown.map((s) =>
      el(
        "tr",
        { class: "clickable", onclick: () => (location.hash = deviceHref(s.device_id)) },
        el("td", {}, el("div", { class: "cell-title" }, s.name), el("div", { class: "cell-sub" }, `on ${deviceName(s.device_id)}`)),
        el("td", { class: "hide-mobile" }, parentName(s.device_id) || el("span", { class: "muted" }, "—")),
        el("td", {}, statusBadge(s.status)),
      ),
    );
    results.replaceChildren(
      dataTable([{ label: "Service" }, { label: "Host", class: "hide-mobile" }, { label: "Status" }], rows),
      el("div", { class: "table-foot" }, `${shown.length} of ${services.length} services`),
    );
  }

  return { root, update };
}

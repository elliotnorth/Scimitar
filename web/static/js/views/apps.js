import { byName, chips, el, emptyState, matches, openLink, pageTitle, panel, searchBox, state } from "../dom.js";
import { deviceHref, deviceName, parentName, store } from "../store.js";

const filters = { query: "", status: "" };
const STATUS_OPTIONS = [["", "all"], ["up", "up"], ["down", "down"], ["unknown", "unknown"]];

export function appsView() {
  const results = el("div");
  const statusChips = el("div");
  const root = el(
    "div",
    { class: "wrap page" },
    pageTitle("Apps", el("p", { class: "dim" }, "Scimitar checks each app at its address. An app with no address stays unknown until you set one on its device page.")),
    el(
      "div",
      { class: "toolbar" },
      searchBox("search apps", filters.query, (e) => {
        filters.query = e.target.value;
        update();
      }),
      statusChips,
    ),
    results,
  );

  function update() {
    statusChips.replaceChildren(
      chips(STATUS_OPTIONS, filters.status, (v) => {
        filters.status = v;
        update();
      }, "Filter by status"),
    );
    const { services } = store.snapshot;
    if (!services.length) {
      results.replaceChildren(emptyState("No apps yet", "Name or tag a VM after the app it runs, like immich or nextcloud, and Scimitar will find it."));
      return;
    }
    const shown = services
      .filter((s) => !filters.status || s.status === filters.status)
      .filter((s) => matches(filters.query, s.name, deviceName(s.device_id), parentName(s.device_id)))
      .sort(byName);
    if (!shown.length) {
      results.replaceChildren(emptyState("Nothing matches", "Try a different search or filter."));
      return;
    }
    results.replaceChildren(
      panel(
        { class: "rows" },
        shown.map((s) =>
          el(
            "div",
            { class: "row three" },
            el("span", { class: "row-main" }, el("a", { class: "row-name", href: deviceHref(s.device_id) }, s.name), " ", el("span", { class: "mono dim small" }, `on ${deviceName(s.device_id)}`)),
            openLink(s.url) || el("a", { class: "lnk mono small", href: deviceHref(s.device_id) }, s.url ? "" : "set address"),
            state(s.status),
          ),
        ),
      ),
      el("div", { class: "foot-note" }, `${shown.length} of ${services.length} apps`),
    );
  }

  return { root, update };
}

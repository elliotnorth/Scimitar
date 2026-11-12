import { LoginRequired, postJSON } from "../api.js";
import { el, emptyState, pageTitle, panel } from "../dom.js";
import { catalogName, deviceHref, deviceName, store } from "../store.js";
import { toast } from "../toast.js";

const SOURCE_LABELS = { name: "its name", tag: "a tag", notes: "its notes" };

export function reviewView(refresh) {
  const list = el("div");
  const root = el(
    "div",
    { class: "wrap page" },
    pageTitle("Review", el("p", { class: "dim" }, "Scimitar found these but is not sure what they are. Your answers are remembered.")),
    list,
  );

  async function decide(item, catalogID) {
    const device = deviceName(item.device_id);
    try {
      await postJSON("api/review", { key: item.key, catalog_id: catalogID });
      toast(catalogID ? `${device} runs ${catalogName(catalogID)}` : `${device}: not an app`);
    } catch (err) {
      if (!(err instanceof LoginRequired)) toast(`Could not save: ${err.message}`, "error");
    }
    await refresh();
  }

  function itemRow(item) {
    const other = el(
      "select",
      { "aria-label": "Something else", onchange: (e) => e.target.value && decide(item, e.target.value) },
      el("option", { value: "" }, "something else…"),
      store.catalog.filter((c) => !item.candidate_ids.includes(c.id)).map((c) => el("option", { value: c.id }, c.name)),
    );
    return el(
      "div",
      { class: "review-item" },
      el(
        "div",
        { class: "review-main" },
        el("a", { class: "row-name", href: deviceHref(item.device_id) }, deviceName(item.device_id)),
        el("div", { class: "cap" }, `found in ${SOURCE_LABELS[item.source] || item.source}`),
        el("pre", { class: "review-value" }, item.value),
      ),
      el(
        "div",
        { class: "review-actions" },
        item.candidate_ids.map((id, i) => el("button", { class: i === 0 ? "btn go small" : "btn small", type: "button", onclick: () => decide(item, id) }, `it's ${catalogName(id)}`)),
        other,
        el("button", { class: "btn small", type: "button", onclick: () => decide(item, "") }, "not an app"),
      ),
    );
  }

  function update() {
    const { review } = store.snapshot;
    list.replaceChildren(review.length ? panel({ class: "rows" }, review.map(itemRow)) : emptyState("All caught up", "Nothing needs review right now."));
  }

  return { root, update };
}

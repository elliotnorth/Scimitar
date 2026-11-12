import { postJSON, LoginRequired } from "../api.js";
import { el, emptyState } from "../dom.js";
import { catalogName, deviceHref, deviceName, store } from "../store.js";
import { toast } from "../toast.js";

const SOURCE_LABELS = { name: "its name", tag: "a tag", notes: "its notes" };

export function reviewView(refresh) {
  const list = el("div");
  const root = el(
    "section",
    {},
    el(
      "div",
      { class: "page-head" },
      el(
        "div",
        {},
        el("h1", {}, "Review"),
        el("p", {}, "Scimitar found these but is not sure what they are. Your answers are remembered."),
      ),
    ),
    list,
  );

  async function decide(item, catalogID) {
    const device = deviceName(item.device_id);
    try {
      await postJSON("api/review", { key: item.key, catalog_id: catalogID });
      toast(catalogID ? `${device} runs ${catalogName(catalogID)}` : `${device}: marked as not a service`);
    } catch (err) {
      if (!(err instanceof LoginRequired)) toast(`Could not save: ${err.message}`, "error");
    }
    await refresh();
  }

  function itemRow(item) {
    const other = el(
      "select",
      {
        "aria-label": "Something else",
        onchange: (e) => e.target.value && decide(item, e.target.value),
      },
      el("option", { value: "" }, "Something else…"),
      store.catalog.filter((c) => !item.candidate_ids.includes(c.id)).map((c) => el("option", { value: c.id }, c.name)),
    );
    return el(
      "div",
      { class: "review-item" },
      el(
        "div",
        {},
        el("a", { class: "cell-title", href: deviceHref(item.device_id) }, deviceName(item.device_id)),
        el("div", { class: "cell-sub" }, `Found in ${SOURCE_LABELS[item.source] || item.source}:`),
        el("div", { class: "review-value" }, item.value),
      ),
      el(
        "div",
        { class: "review-actions" },
        item.candidate_ids.map((id, i) =>
          el("button", { class: i === 0 ? "btn btn-primary btn-small" : "btn btn-small", type: "button", onclick: () => decide(item, id) }, `It's ${catalogName(id)}`),
        ),
        other,
        el("button", { class: "btn btn-small", type: "button", onclick: () => decide(item, "") }, "Not a service"),
      ),
    );
  }

  function update() {
    const { review } = store.snapshot;
    if (!review.length) {
      list.replaceChildren(emptyState("All caught up", "Nothing needs review right now."));
      return;
    }
    list.replaceChildren(el("div", { class: "card" }, review.map(itemRow)), el("div", { class: "table-foot" }, `${review.length} to review`));
  }

  return { root, update };
}

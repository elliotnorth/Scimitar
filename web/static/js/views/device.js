import { getJSON, LoginRequired, postJSON, sendJSON } from "../api.js";
import { byName, el, emptyState, kindLabel, openLink, panel, state, textInput } from "../dom.js";
import { deviceHref, store } from "../store.js";
import { toast } from "../toast.js";

export function deviceView(id, refresh) {
  const head = el("div");
  const body = el("div", { class: "stack" });
  const root = el("div", { class: "wrap page" }, head, body);
  let details = { loading: true };
  // The app whose address is being typed, so a background refresh does not wipe the box.
  let editing = "";
  let confirmingRemove = false;

  getJSON(`api/details?id=${encodeURIComponent(id)}`)
    .then((d) => (details = { items: d }))
    .catch((err) => {
      if (!(err instanceof LoginRequired)) details = { error: err.message };
    })
    .finally(update);

  async function saveAddress(service, url) {
    try {
      await postJSON("api/apps", { id: service.id, url });
      toast(url ? `${service.name} is checked at its new address` : `${service.name} is back on its automatic address`);
      editing = "";
    } catch (err) {
      if (!(err instanceof LoginRequired)) toast(err.message, "error");
      return;
    }
    await refresh();
    update();
  }

  async function remove(device) {
    try {
      await sendJSON("DELETE", `api/devices?id=${encodeURIComponent(device.id)}`);
      toast(`${device.name} was removed`);
      location.hash = "#/devices";
    } catch (err) {
      if (!(err instanceof LoginRequired)) toast(`Could not remove: ${err.message}`, "error");
    }
    await refresh();
  }

  function appRow(service) {
    if (editing !== service.id) {
      return el(
        "div",
        { class: "row three" },
        el("span", { class: "row-main" }, el("span", { class: "row-name" }, service.name), " ", el("span", { class: "mono dim small" }, service.url || "no address to check")),
        el(
          "span",
          { class: "row-actions" },
          openLink(service.url),
          el("button", { class: "lnk mono small", type: "button", onclick: () => { editing = service.id; update(); } }, service.url ? "change" : "set address"),
        ),
        state(service.status),
      );
    }
    const input = textInput({ value: service.custom_url ? service.url : "", placeholder: service.url || "http://192.168.1.5:8080", "aria-label": `Address of ${service.name}` });
    const form = el(
      "form",
      { class: "row editing", onsubmit: (e) => { e.preventDefault(); saveAddress(service, input.value.trim()); } },
      el("span", { class: "row-name" }, service.name),
      input,
      el(
        "span",
        { class: "row-actions" },
        el("button", { class: "btn go small", type: "submit" }, "save"),
        service.custom_url ? el("button", { class: "btn small", type: "button", onclick: () => saveAddress(service, "") }, "automatic") : null,
        el("button", { class: "btn small", type: "button", onclick: () => { editing = ""; update(); } }, "cancel"),
      ),
    );
    setTimeout(() => input.focus(), 0);
    return form;
  }

  function update() {
    const device = store.index.devicesById.get(id);
    if (!device) {
      head.replaceChildren();
      body.replaceChildren(emptyState("Device not found", "It may have been removed from its source."), el("a", { class: "lnk", href: "#/devices" }, "← back to devices"));
      return;
    }
    const where = [kindLabel(device.kind), device.custom ? "added by hand" : device.provider, device.address].filter(Boolean);
    head.replaceChildren(
      el("a", { class: "lnk crumb", href: "#/devices" }, "← devices"),
      el("div", { class: "page-head" }, el("h1", { class: "title" }, device.name), state(device.status)),
      el("div", { class: "mono dim small" }, where.join(" · ")),
    );
    if (editing && body.contains(document.activeElement)) return;

    const { servicesByDevice, parentOf, childrenOf, devicesById, connectedTo } = store.index;
    const sections = [section("details", detailsBody(device))];
    const services = servicesByDevice.get(device.id) || [];
    sections.push(section("apps", services.length ? panel({ class: "rows" }, services.map(appRow)) : el("p", { class: "dim" }, "No apps found on this device.")));
    const parent = devicesById.get(parentOf.get(device.id));
    if (parent) sections.push(section("runs on", panel({ class: "rows" }, linkRow(parent))));
    const neighbors = (connectedTo.get(device.id) || []).map((n) => devicesById.get(n)).filter(Boolean).sort(byName);
    if (neighbors.length) sections.push(section("connected to", panel({ class: "rows" }, neighbors.map(linkRow))));
    const guests = (childrenOf.get(device.id) || []).map((g) => devicesById.get(g)).filter(Boolean).sort(byName);
    if (guests.length) sections.push(section(`guests · ${guests.length}`, panel({ class: "rows" }, guests.map(linkRow))));
    if (device.custom) {
      sections.push(
        el(
          "section",
          { class: "row-actions" },
          confirmingRemove
            ? [
                el("button", { class: "btn danger", type: "button", onclick: () => remove(device) }, `yes, remove ${device.name}`),
                el("button", { class: "btn", type: "button", onclick: () => { confirmingRemove = false; update(); } }, "keep it"),
              ]
            : el("button", { class: "btn", type: "button", onclick: () => { confirmingRemove = true; update(); } }, "remove device"),
        ),
      );
    }
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

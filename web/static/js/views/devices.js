import { LoginRequired, postJSON } from "../api.js";
import { byName, chips, el, emptyState, kindLabel, matches, pageTitle, panel, searchBox, state, textInput } from "../dom.js";
import { deviceHref, store } from "../store.js";
import { toast } from "../toast.js";

const filters = { query: "", status: "" };
const NETWORK_KINDS = new Set(["switch", "firewall", "router", "access-point", "modem"]);
const STATUS_OPTIONS = [["", "all"], ["up", "up"], ["down", "down"], ["unknown", "unknown"]];

export function devicesView(refresh) {
  const results = el("div", { class: "stack" });
  const statusChips = el("div");
  const addForm = el("div", { hidden: true });
  const addButton = el("button", { class: "btn small", type: "button", "aria-expanded": "false", onclick: () => toggleForm(addForm.hidden) }, "+ add device");
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
      addButton,
    ),
    addForm,
    results,
  );

  function toggleForm(open) {
    addForm.hidden = !open;
    addButton.setAttribute("aria-expanded", String(open));
    if (open && !addForm.firstChild) addForm.append(buildForm());
    if (open) addForm.querySelector("input").focus();
  }

  function buildForm() {
    const name = textInput({ placeholder: "living room NAS", required: true, maxlength: "64" });
    const kind = el("select", { class: "input" }, store.info.device_kinds.map((k) => el("option", { value: k, selected: k === "nas" }, kindLabel(k))));
    const address = textInput({ placeholder: "192.168.1.20 or nas.lan", required: true, inputmode: "url" });
    const port = textInput({ placeholder: "any", inputmode: "numeric", maxlength: "5" });
    const error = el("p", { class: "form-error", role: "alert" });
    const submit = el("button", { class: "btn go", type: "submit" }, "add device");

    async function add(event) {
      event.preventDefault();
      error.textContent = "";
      if (port.value.trim() && !/^\d+$/.test(port.value.trim())) {
        error.textContent = "The port must be a number, or empty.";
        return;
      }
      submit.disabled = true;
      try {
        await postJSON("api/devices", { name: name.value, kind: kind.value, address: address.value, port: Number(port.value.trim() || 0) });
        toast(`${name.value.trim()} was added`);
        name.value = address.value = port.value = "";
        toggleForm(false);
        await refresh();
      } catch (err) {
        if (!(err instanceof LoginRequired)) error.textContent = err.message.replace(/^./, (c) => c.toUpperCase());
      } finally {
        submit.disabled = false;
      }
    }

    return panel(
      { class: "pad" },
      el(
        "form",
        { class: "form", onsubmit: add },
        el("p", { class: "dim form-wide" }, "For anything Scimitar cannot read by itself: a NAS, printer, access point, camera, UPS, game console and so on. It counts as up when it answers on the network."),
        field("name", name),
        field("what is it", kind),
        field("address", address),
        field("port that must be open (optional)", port),
        el("div", { class: "form-wide row-actions" }, submit, el("button", { class: "btn", type: "button", onclick: () => toggleForm(false) }, "cancel")),
        el("div", { class: "form-wide" }, error),
      ),
    );
  }

  function shown(device) {
    const services = store.index.servicesByDevice.get(device.id) || [];
    return (!filters.status || device.status === filters.status) && matches(filters.query, device.name, device.kind, kindLabel(device.kind), device.provider, device.address, ...services.map((s) => s.name));
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
      results.replaceChildren(emptyState("No devices yet", "They show up here after the first discovery. Check settings if this takes more than a minute, or add one yourself."));
      return;
    }
    const { childrenOf, parentOf, devicesById } = store.index;
    const groups = [];
    const hosts = devices.filter((d) => childrenOf.has(d.id) || (d.kind === "host" && !d.custom)).sort(byName);
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
    const grouped = new Set(hosts);
    const loose = devices.filter((d) => !parentOf.has(d.id) && !grouped.has(d) && shown(d)).sort(byName);
    const describe = (d) => [kindLabel(d.kind), d.custom ? d.address : d.provider].filter(Boolean).join(" · ");
    const gear = loose.filter((d) => NETWORK_KINDS.has(d.kind));
    const other = loose.filter((d) => !NETWORK_KINDS.has(d.kind));
    if (gear.length) groups.push(el("section", {}, el("div", { class: "cap section-cap" }, "network"), panel({ class: "rows" }, gear.map((d) => deviceRow(d, "", describe(d))))));
    if (other.length) groups.push(el("section", {}, el("div", { class: "cap section-cap" }, "other devices"), panel({ class: "rows" }, other.map((d) => deviceRow(d, "", describe(d))))));
    results.replaceChildren(...(groups.length ? groups : [emptyState("Nothing matches", "Try a different search or filter.")]));
  }

  return { root, update };
}

function field(label, control) {
  return el("label", { class: "field" }, el("span", { class: "cap" }, label), control);
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

import { getJSON, LoginRequired, postJSON, sendJSON } from "../api.js";
import { chips, el, pageTitle, panel, state, textInput } from "../dom.js";
import { store } from "../store.js";
import { currentTheme, randomTheme, setTheme, THEME_STYLES, themes } from "../themes.js";
import { toast } from "../toast.js";

let themeFilter = "all";

const ALERT_TYPES = [
  ["ntfy", "ntfy", "https://ntfy.sh/your-topic"],
  ["discord", "Discord", "https://discord.com/api/webhooks/…"],
  ["slack", "Slack", "https://hooks.slack.com/services/…"],
  ["gotify", "Gotify", "https://gotify.example/message?token=…"],
  ["webhook", "Webhook (JSON)", "https://example.com/hook"],
];

export function settingsView(logout) {
  // Each part is its own box, so a background refresh redraws the sources
  // without wiping what is being typed into the forms.
  const themeBox = el("section");
  const sourcesBox = el("section");
  const alertsBox = el("section");
  const body = el("div", { class: "stack" }, themeBox, sourcesBox, alertsBox, passwordSection(), catalogSection(), aboutSection(), el("section", {}, el("button", { class: "btn", type: "button", onclick: logout }, "log out")));
  const root = el("div", { class: "wrap page" }, pageTitle("Settings"), body);

  // ---- theme
  function pick(name) {
    setTheme(name);
    renderTheme();
  }

  function swatch(name, small) {
    const { colors } = themes.get(name);
    const active = name === currentTheme();
    const button = el(
      "button",
      { type: "button", class: small ? "swatch small" : "swatch", "aria-pressed": String(active), title: name, "data-keep": `theme ${name}`, onclick: () => pick(name) },
      el("span", { class: "dots", "aria-hidden": "true" }, ["accent", "up", "down"].map((key) => {
        const dot = el("span", { class: "dot" });
        dot.style.background = colors[key];
        return dot;
      })),
      el("span", { class: "swatch-name" }, name),
    );
    button.style.background = colors.bg;
    button.style.color = colors.ink;
    button.style.borderColor = active ? colors.ink : colors.line;
    return button;
  }

  function renderTheme() {
    // Keep the theme list scrolled and focused where it was.
    const scrolled = themeBox.querySelector(".theme-grid")?.scrollTop || 0;
    const focused = themeBox.contains(document.activeElement) ? document.activeElement.dataset.keep : "";
    const names = [...themes.keys()];
    const featured = names.filter((n) => themes.get(n).style === "featured");
    const rest = names.filter((n) => themes.get(n).style !== "featured" && (themeFilter === "all" || themes.get(n).style === themeFilter));
    themeBox.replaceChildren(
      el("div", { class: "section-bar" }, el("div", { class: "cap" }, "theme ", el("span", { class: "ink" }, currentTheme())), el("button", { class: "lnk", type: "button", "data-keep": "surprise", onclick: () => pick(randomTheme()) }, "surprise me ↻")),
      el("div", { class: "swatches" }, featured.map((n) => swatch(n, false))),
      el(
        "div",
        { class: "section-bar" },
        el("span", { class: "cap faint" }, `all ${names.length}`),
        chips([["all", "all"], ...THEME_STYLES.map((s) => [s, s])], themeFilter, (v) => {
          themeFilter = v;
          renderTheme();
        }, "Filter themes"),
      ),
      panel({ class: "theme-panel" }, el("div", { class: "theme-grid" }, rest.map((n) => swatch(n, true)))),
    );
    themeBox.querySelector(".theme-grid").scrollTop = scrolled;
    if (focused) [...themeBox.querySelectorAll("[data-keep]")].find((node) => node.dataset.keep === focused)?.focus();
  }

  // ---- sources
  function renderSources() {
    const { providers, errors, devices } = store.snapshot;
    const rows = providers
      .map((name) => ({ name, error: errors[name], count: devices.filter((d) => d.provider === name).length }))
      .filter((p) => p.name !== "custom" || p.count)
      .map((p) =>
        el(
          "div",
          { class: "row" },
          el("span", { class: "row-main" }, el("span", { class: "row-name" }, p.name === "custom" ? "added by hand" : p.name), " ", el("span", { class: "mono dim small", title: p.error }, p.error || `${p.count} ${p.count === 1 ? "device" : "devices"}`)),
          state(p.error ? "down" : "up"),
        ),
      );
    sourcesBox.replaceChildren(
      el("div", { class: "cap section-cap" }, "sources"),
      rows.length ? panel({ class: "rows" }, rows) : el("p", { class: "dim" }, "No sources yet."),
      el("p", { class: "dim small" }, "Add any device by hand on the devices page. On the server, add a Juniper device with ", el("code", {}, "sudo scimitar add-device"), " or change Proxmox with ", el("code", {}, "sudo scimitar install"), "."),
    );
  }

  // ---- alerts
  let alerts = null;
  let alertNote = "";
  let alertResults = [];

  function renderAlerts() {
    if (!alerts) {
      alertsBox.replaceChildren(el("div", { class: "cap section-cap" }, "alerts"), el("p", { class: "dim" }, alertNote || "loading…"));
      return;
    }
    const rows = alerts.channels.map((channel, i) => {
      const placeholder = ALERT_TYPES.find(([type]) => type === channel.type)?.[2] || "";
      const result = alertResults[i];
      return el(
        "div",
        { class: "alert-row" },
        el("select", { class: "input", "aria-label": "Where to send", onchange: (e) => { channel.type = e.target.value; alertResults = []; renderAlerts(); } }, ALERT_TYPES.map(([type, label]) => el("option", { value: type, selected: type === channel.type }, label))),
        textInput({ value: channel.url, placeholder, "aria-label": "Address", inputmode: "url", oninput: (e) => (channel.url = e.target.value) }),
        el("button", { class: "btn small", type: "button", "aria-label": "Remove", onclick: () => { alerts.channels.splice(i, 1); alertResults = []; renderAlerts(); } }, "remove"),
        result ? el("span", { class: `state ${result.error ? "down" : "up"} alert-result` }, result.error ? `● ${result.error}` : "● sent") : null,
      );
    });
    const toggle = (key, label) => {
      const box = el("input", { type: "checkbox", onchange: (e) => (alerts[key] = e.target.checked) });
      box.checked = alerts[key];
      return el("label", { class: "check" }, box, label);
    };
    alertsBox.replaceChildren(
      el("div", { class: "cap section-cap" }, "alerts"),
      panel(
        { class: "pad form-stack" },
        el("p", { class: "dim" }, "Get a message when something goes down and when it comes back. Paste the address your service gives you."),
        rows,
        el("div", {}, el("button", { class: "btn small", type: "button", onclick: () => { alerts.channels.push({ type: "ntfy", url: "" }); alertResults = []; renderAlerts(); alertsBox.querySelector(".alert-row:last-of-type input").focus(); } }, "+ add a place to send alerts")),
        el("div", { class: "checks" }, el("span", { class: "cap" }, "tell me about"), toggle("devices", "devices"), toggle("apps", "apps"), toggle("sources", "sources Scimitar can't reach")),
        el(
          "div",
          { class: "row-actions" },
          el("button", { class: "btn go", type: "button", onclick: () => saveAlerts(false) }, "save"),
          el("button", { class: "btn", type: "button", onclick: () => saveAlerts(true) }, "save and send a test"),
          el("span", { class: "dim small", role: "status" }, alertNote),
        ),
      ),
    );
  }

  async function saveAlerts(test) {
    alertNote = "";
    alertResults = [];
    try {
      const filled = { ...alerts, channels: alerts.channels.filter((c) => c.url.trim()) };
      alerts = await (await sendJSON("PUT", "api/alerts", filled)).json();
      if (!test) {
        toast("Alert settings saved");
      } else if (!alerts.channels.length) {
        alertNote = "Add a place to send alerts first.";
      } else {
        alertResults = await (await postJSON("api/alerts/test")).json();
        alertNote = alertResults.some((r) => r.error) ? "Some did not arrive." : "Test sent. Check your phone or chat.";
      }
    } catch (err) {
      if (err instanceof LoginRequired) return;
      alertNote = err.message.replace(/^./, (c) => c.toUpperCase());
    }
    renderAlerts();
  }

  getJSON("api/alerts")
    .then((settings) => (alerts = settings))
    .catch((err) => {
      if (!(err instanceof LoginRequired)) alertNote = `Could not load alert settings: ${err.message}`;
    })
    .finally(renderAlerts);

  renderTheme();
  renderAlerts();
  return { root, update: renderSources };
}

function passwordSection() {
  const current = textInput({ type: "password", autocomplete: "current-password", required: true });
  const next = textInput({ type: "password", autocomplete: "new-password", required: true, minlength: "8" });
  const again = textInput({ type: "password", autocomplete: "new-password", required: true });
  const error = el("p", { class: "form-error", role: "alert" });
  const submit = el("button", { class: "btn go", type: "submit" }, "change password");

  async function change(event) {
    event.preventDefault();
    error.textContent = "";
    if (next.value !== again.value) {
      error.textContent = "The two new passwords are not the same.";
      return;
    }
    submit.disabled = true;
    try {
      await postJSON("api/password", { current: current.value, new: next.value });
      current.value = next.value = again.value = "";
      toast("Password changed. Other devices have to log in again.");
    } catch (err) {
      if (!(err instanceof LoginRequired)) error.textContent = err.message.replace(/^./, (c) => c.toUpperCase()) + ".";
    } finally {
      submit.disabled = false;
    }
  }

  const field = (label, control) => el("label", { class: "field" }, el("span", { class: "cap" }, label), control);
  return el(
    "section",
    {},
    el("div", { class: "cap section-cap" }, "password"),
    panel(
      { class: "pad" },
      el(
        "form",
        { class: "form", onsubmit: change },
        el("div", { class: "form-wide narrow" }, field("current password", current)),
        field("new password (8 characters or more)", next),
        field("new password again", again),
        el("div", { class: "form-wide row-actions" }, submit, el("span", { class: "dim small" }, "Forgot it? Run ", el("code", {}, "sudo scimitar set-password"), " on the server.")),
        el("div", { class: "form-wide" }, error),
      ),
    ),
  );
}

function catalogSection() {
  return el(
    "section",
    {},
    el("div", { class: "cap section-cap" }, "recognized apps"),
    el("details", { class: "dim" }, el("summary", {}, `${store.catalog.length} apps Scimitar knows by name`), el("p", {}, store.catalog.map((c) => c.name).join(" · "))),
  );
}

function aboutSection() {
  return el(
    "section",
    {},
    el("div", { class: "cap section-cap" }, "about"),
    panel(
      { class: "pad" },
      el(
        "dl",
        { class: "kv" },
        el("dt", { class: "cap" }, "version"),
        el("dd", { class: "mono" }, store.info.version || "unknown"),
        el("dt", { class: "cap" }, "update"),
        el("dd", {}, el("code", {}, "sudo scimitar update")),
        el("dt", { class: "cap" }, "source"),
        el("dd", {}, el("a", { class: "lnk", href: "https://github.com/elliotnorth/Scimitar", target: "_blank", rel: "noopener noreferrer" }, "github.com/elliotnorth/Scimitar ↗")),
      ),
    ),
  );
}

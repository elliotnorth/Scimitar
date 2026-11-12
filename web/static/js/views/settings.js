import { chips, el, pageTitle, panel, state } from "../dom.js";
import { store } from "../store.js";
import { currentTheme, randomTheme, setTheme, THEME_STYLES, themes } from "../themes.js";

let themeFilter = "all";

export function settingsView(logout) {
  const body = el("div", { class: "stack" });
  const root = el("div", { class: "wrap page" }, pageTitle("Settings"), body);

  function pick(name) {
    setTheme(name);
    update();
  }

  function swatch(name, small) {
    const { colors } = themes.get(name);
    const active = name === currentTheme();
    const button = el(
      "button",
      { type: "button", class: small ? "swatch small" : "swatch", "aria-pressed": String(active), title: name, onclick: () => pick(name) },
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

  function themeSection() {
    const names = [...themes.keys()];
    const featured = names.filter((n) => themes.get(n).style === "featured");
    const rest = names.filter((n) => themes.get(n).style !== "featured" && (themeFilter === "all" || themes.get(n).style === themeFilter));
    return el(
      "section",
      {},
      el("div", { class: "section-bar" }, el("div", { class: "cap" }, "theme ", el("span", { class: "ink" }, currentTheme())), el("button", { class: "lnk", type: "button", onclick: () => pick(randomTheme()) }, "surprise me ↻")),
      el("div", { class: "swatches" }, featured.map((n) => swatch(n, false))),
      el(
        "div",
        { class: "section-bar" },
        el("span", { class: "cap faint" }, `all ${names.length}`),
        chips([["all", "all"], ...THEME_STYLES.map((s) => [s, s])], themeFilter, (v) => {
          themeFilter = v;
          update();
        }, "Filter themes"),
      ),
      panel({ class: "theme-panel" }, el("div", { class: "theme-grid" }, rest.map((n) => swatch(n, true)))),
    );
  }

  function update() {
    const { providers, errors, devices } = store.snapshot;
    body.replaceChildren(
      themeSection(),
      el(
        "section",
        {},
        el("div", { class: "cap section-cap" }, "sources"),
        panel(
          { class: "rows" },
          providers.map((name) => {
            const error = errors[name];
            const count = devices.filter((d) => d.provider === name).length;
            return el("div", { class: "row" }, el("span", { class: "row-main" }, el("span", { class: "row-name" }, name), " ", el("span", { class: "mono dim small" }, error || `${count} devices`)), state(error ? "down" : "up"));
          }),
        ),
        el("p", { class: "dim small" }, "Change sources with ", el("code", {}, "sudo scimitar install"), " on the server."),
      ),
      el(
        "section",
        {},
        el("div", { class: "cap section-cap" }, "recognized apps"),
        el("p", { class: "dim" }, store.catalog.map((c) => c.name).join(" · ")),
      ),
      el(
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
      ),
      el("section", {}, el("button", { class: "btn", type: "button", onclick: logout }, "log out")),
    );
  }

  return { root, update };
}

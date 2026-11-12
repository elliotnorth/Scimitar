import { el } from "../dom.js";
import { store } from "../store.js";

export function settingsView(logout) {
  const body = el("div", { class: "stack" });
  const root = el(
    "section",
    {},
    el("div", { class: "page-head" }, el("div", {}, el("h1", {}, "Settings"), el("p", {}, "Sources, recognized apps and this installation."))),
    body,
  );

  function update() {
    const { providers, errors, devices } = store.snapshot;
    const sources = providers.map((name) => {
      const error = errors[name];
      const count = devices.filter((d) => d.provider === name).length;
      return el(
        "div",
        { class: "list-row" },
        el(
          "div",
          {},
          el("div", { class: "cell-title" }, name),
          el("div", { class: "cell-sub" }, error ? error : `${count} devices`),
        ),
        el("span", { class: "status" }, el("span", { class: `dot ${error ? "down" : "up"}` }), error ? "Error" : "Connected"),
      );
    });
    body.replaceChildren(
      card("Sources", sources, el("p", { class: "card-note" }, "Change the source with ", el("code", {}, "sudo scimitar install"), " on the server.")),
      card(
        "Recognized apps",
        el("div", { class: "card-body pills" }, store.catalog.map((c) => el("span", { class: "pill" }, c.name))),
        el("p", { class: "card-note" }, "Name or tag a VM or container after one of these and Scimitar finds it."),
      ),
      card(
        "About",
        el(
          "div",
          { class: "card-body" },
          el(
            "dl",
            { class: "kv" },
            el("dt", {}, "Version"),
            el("dd", { class: "mono" }, store.info.version || "unknown"),
            el("dt", {}, "Update"),
            el("dd", {}, el("code", {}, "sudo scimitar update")),
            el("dt", {}, "License"),
            el("dd", {}, "Apache-2.0"),
            el("dt", {}, "Source"),
            el("dd", {}, el("a", { href: "https://github.com/elliotnorth/Scimitar", target: "_blank", rel: "noopener noreferrer", class: "link" }, "github.com/elliotnorth/Scimitar")),
          ),
        ),
      ),
      card("Session", el("div", { class: "list-row" }, el("span", { class: "muted" }, "You are logged in as admin."), el("button", { class: "btn btn-small", type: "button", onclick: logout }, "Log out"))),
    );
  }

  return { root, update };
}

function card(title, ...content) {
  return el("div", { class: "card" }, el("div", { class: "card-head" }, el("h2", {}, title)), content);
}

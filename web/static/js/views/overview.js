import { bar, el, panel } from "../dom.js";
import { LOGO, startGlyphs } from "../glyphs.js";
import { deviceHref, store } from "../store.js";

const BAR_WIDTH = 18;

export function overviewView() {
  const fieldNode = el("pre", { class: "field-base", "aria-hidden": "true" });
  const bladeNode = el("pre", { class: "field-blade", "aria-hidden": "true" });
  const lead = el("p", { class: "lead" });
  const stats = el("div", { class: "stats" });
  const problems = el("div");
  const problemsSection = el("section", {}, el("h2", { class: "title-2" }, "Problems"), problems);

  const root = el(
    "div",
    {},
    el(
      "section",
      { class: "hero" },
      el("div", { class: "glyphs" }, fieldNode, bladeNode),
      el(
        "div",
        { class: "wrap hero-body" },
        el("pre", { class: "logo", "aria-label": "Scimitar" }, LOGO),
        lead,
        el(
          "div",
          { class: "actions" },
          el("button", { class: "btn go", type: "button", onclick: () => problemsSection.scrollIntoView({ behavior: "smooth" }) }, "see problems"),
          el("a", { class: "lnk", href: "#/apps" }, "open your apps ↗"),
        ),
      ),
    ),
    el("div", { class: "wrap page" }, panel({}, stats), problemsSection),
  );

  const glyphs = startGlyphs(fieldNode, bladeNode);

  function update() {
    const { devices, services, review, providers, errors } = store.snapshot;
    const up = devices.filter((d) => d.status === "up").length;
    const down = devices.filter((d) => d.status === "down");
    const failing = Object.keys(errors);

    lead.replaceChildren(el("b", {}, devices.length ? `${up} of ${devices.length} devices are up.` : "No devices yet."), summary(down, failing, review.length));

    stats.replaceChildren(
      stat("devices", up, devices.length, devices.length ? up / devices.length : 0),
      stat("apps", services.length, null, services.length ? 1 - services.filter((s) => s.status === "down").length / services.length : 0),
      stat("sources", providers.length - failing.length, providers.length, providers.length ? 1 - failing.length / providers.length : 0),
      stat("to review", review.length, null, null),
    );

    const rows = [
      ...down.map((d) => problemRow("down", "down", d.name, `${d.kind} on ${d.provider}`, deviceHref(d.id))),
      ...failing.map((name) => problemRow("down", "error", `can't reach ${name}`, errors[name], "#/settings")),
    ];
    if (review.length) rows.push(problemRow("warn", "new", `${review.length} ${review.length === 1 ? "app" : "apps"} to confirm`, "found in names, tags and notes", "#/review"));
    problems.replaceChildren(panel({ class: "rows" }, rows.length ? rows : el("div", { class: "row calm" }, el("span", { class: "state up" }, "● ok"), "nothing needs you right now.")));
  }

  return { root, update, destroy: glyphs.stop };
}

function summary(down, failing, reviewCount) {
  const parts = [];
  if (down.length === 1) parts.push(`${down[0].name} is down.`);
  else if (down.length > 1) parts.push(`${down.length} devices are down.`);
  if (failing.length) parts.push(`Can't reach ${failing.join(", ")}.`);
  if (!parts.length && reviewCount) parts.push(`${reviewCount} new ${reviewCount === 1 ? "app needs" : "apps need"} a look.`);
  return parts.length ? parts.join(" ") : "Everything looks fine.";
}

function stat(label, value, total, fraction) {
  return el(
    "div",
    { class: "stat" },
    el("div", { class: "cap" }, label),
    el("div", { class: "big" }, String(value), total === null ? null : el("span", { class: "faint" }, `/${total}`)),
    fraction === null ? el("div", { class: "bar" }, " ") : el("div", { class: "bar up" }, bar(fraction, BAR_WIDTH)),
  );
}

function problemRow(tone, tag, title, detail, href) {
  return el(
    "a",
    { class: "row problem", href },
    el("span", { class: `state ${tone}` }, `● ${tag}`),
    el("span", { class: "row-main" }, title, " ", el("span", { class: "dim" }, detail)),
    el("span", { class: "dim" }, "view →"),
  );
}

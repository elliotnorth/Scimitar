import { getJSON, LoginRequired, login, postJSON, whenLoginRequired } from "./api.js";
import { el, relativeTime } from "./dom.js";
import { setSnapshot, store } from "./store.js";
import { applyTheme, currentTheme } from "./themes.js";
import { appsView } from "./views/apps.js";
import { deviceView } from "./views/device.js";
import { devicesView } from "./views/devices.js";
import { overviewView } from "./views/overview.js";
import { reviewView } from "./views/review.js";
import { settingsView } from "./views/settings.js";

const REFRESH_MS = 10000;
const PAGES = ["overview", "apps", "devices", "review", "settings"];
const ALIASES = { services: "apps" };

const $ = (id) => document.getElementById(id);
let current = null;
let currentRoute = "";
let lastError = "";

applyTheme(currentTheme());

function routeFor(hash) {
  const [, section = "overview", ...rest] = hash.replace(/^#/, "").split("/");
  let id = rest.join("/");
  try {
    id = decodeURIComponent(id);
  } catch {
    // A hand-edited link with a stray % is used as it is.
  }
  return { section: ALIASES[section] || section || "overview", id };
}

function createView({ section, id }) {
  switch (section) {
    case "overview":
      return overviewView();
    case "apps":
      return appsView();
    case "devices":
      return id ? deviceView(id, refresh) : devicesView(refresh);
    case "review":
      return reviewView(refresh);
    case "settings":
      return settingsView(logout);
  }
  location.hash = "#/overview";
  return null;
}

function currentHash() {
  return location.hash || "#/overview";
}

function mount() {
  if (!store.snapshot || currentHash() === currentRoute) return;
  const route = routeFor(currentHash());
  const view = createView(route);
  if (!view) return;
  current?.destroy?.();
  currentRoute = currentHash();
  current = view;
  $("view").replaceChildren(view.root);
  for (const tab of document.querySelectorAll("[data-tab]")) {
    const active = tab.dataset.tab === route.section;
    tab.classList.toggle("active", active);
    if (active) tab.setAttribute("aria-current", "page");
    else tab.removeAttribute("aria-current");
  }
  view.update();
  window.scrollTo(0, 0);
}

function renderChrome() {
  const s = store.snapshot;
  const badge = $("review-badge");
  badge.hidden = s.review.length === 0;
  badge.textContent = s.review.length;
  $("version").textContent = store.info.version || "";
  const failing = Object.keys(s.errors);
  const onOverview = routeFor(currentHash()).section === "overview";
  $("banner").replaceChildren(
    ...(onOverview || !failing.length ? [] : [el("a", { class: "banner", href: "#/settings" }, `can't reach ${failing.join(", ")}`, el("span", { class: "dim" }, " · see settings"))]),
  );
  renderLive();
}

function renderLive() {
  const at = store.snapshot?.updated_at;
  const offline = Boolean(lastError);
  $("live").classList.toggle("offline", offline);
  $("live-text").textContent = offline ? "offline" : at ? "live" : "starting";
  $("live").title = offline ? lastError : at ? `updated ${relativeTime(new Date(at))}` : "waiting for first discovery";
}

function showLogin(visible) {
  $("login-screen").hidden = !visible;
  $("shell").hidden = visible;
  if (visible) {
    current?.destroy?.();
    current = null;
    currentRoute = "";
    setTimeout(() => $("password").focus(), 0);
  }
}

async function refresh() {
  try {
    if (!store.catalog.length) store.catalog = await getJSON("api/catalog");
    if (!store.info.version) store.info = await getJSON("api/info");
    const changed = setSnapshot(await getJSON("api/snapshot"));
    lastError = "";
    showLogin(false);
    renderChrome();
    // Redrawing an unchanged page would drop keyboard focus and close open menus every few seconds.
    if (currentHash() !== currentRoute || !current) mount();
    else if (changed) current.update();
  } catch (err) {
    if (err instanceof LoginRequired) return;
    lastError = err.message;
    renderLive();
    $("banner").replaceChildren(el("div", { class: "banner" }, "Scimitar is not answering. ", el("span", { class: "dim" }, err.message)));
  }
}

async function logout() {
  await postJSON("api/logout", {}).catch(() => {});
  store.snapshot = null;
  showLogin(true);
}

$("login").addEventListener("submit", async (event) => {
  event.preventDefault();
  const error = $("login-error");
  const status = await login($("password").value).catch(() => 0);
  if (status === 204) {
    $("password").value = "";
    error.textContent = "";
    await refresh();
    return;
  }
  error.textContent = status === 401 ? "That password is not right." : "Could not reach Scimitar.";
});

function typing() {
  const tag = document.activeElement?.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
}

document.addEventListener("keydown", (event) => {
  if (event.ctrlKey || event.metaKey || event.altKey || $("shell").hidden) return;
  const search = document.querySelector("[data-search]");
  if (event.key === "/" && search && !typing()) {
    event.preventDefault();
    search.focus();
    return;
  }
  if (event.key === "Escape" && document.activeElement === search) {
    search.value = "";
    search.dispatchEvent(new Event("input"));
    search.blur();
    return;
  }
  const page = PAGES[Number(event.key) - 1];
  if (page && !typing()) location.hash = `#/${page}`;
});

document.addEventListener("visibilitychange", () => {
  if (!document.hidden) refresh();
});

window.addEventListener("hashchange", () => {
  mount();
  if (store.snapshot) renderChrome();
});
whenLoginRequired(() => showLogin(true));

refresh();
setInterval(() => !document.hidden && refresh(), REFRESH_MS);
setInterval(renderLive, 1000);

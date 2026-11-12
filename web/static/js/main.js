import { getJSON, LoginRequired, login, postJSON, whenLoginRequired } from "./api.js";
import { el, relativeTime } from "./dom.js";
import { setSnapshot, store } from "./store.js";
import { deviceView } from "./views/device.js";
import { devicesView } from "./views/devices.js";
import { reviewView } from "./views/review.js";
import { servicesView } from "./views/services.js";
import { settingsView } from "./views/settings.js";

const REFRESH_MS = 10000;

const $ = (id) => document.getElementById(id);
let current = null;
let currentRoute = "";

function routeFor(hash) {
  const [, section = "devices", ...rest] = hash.replace(/^#/, "").split("/");
  return { section, id: rest.length ? decodeURIComponent(rest.join("/")) : "" };
}

function createView({ section, id }) {
  switch (section) {
    case "devices":
      return id ? deviceView(id) : devicesView();
    case "services":
      return servicesView();
    case "review":
      return reviewView(refresh);
    case "settings":
      return settingsView(logout);
  }
  location.hash = "#/devices";
  return null;
}

function currentHash() {
  return location.hash || "#/devices";
}

function mount() {
  if (!store.snapshot || currentHash() === currentRoute) return;
  const route = routeFor(currentHash());
  const view = createView(route);
  if (!view) return;
  currentRoute = currentHash();
  current = view;
  $("view").replaceChildren(view.root);
  for (const tab of document.querySelectorAll("[data-tab]")) {
    tab.classList.toggle("active", tab.dataset.tab === route.section);
  }
  view.update();
  window.scrollTo(0, 0);
}

function renderChrome() {
  const s = store.snapshot;
  const badge = $("review-badge");
  badge.hidden = s.review.length === 0;
  badge.textContent = s.review.length;
  const failing = Object.entries(s.errors);
  $("banner").replaceChildren(
    ...failing.map(([name, message]) =>
      el("div", { class: "banner" }, el("strong", {}, `Can't reach ${name}.`), el("span", {}, message)),
    ),
  );
  renderUpdated();
}

function renderUpdated() {
  const at = store.snapshot?.updated_at;
  $("updated").textContent = at ? `Updated ${relativeTime(new Date(at))}` : "Waiting for first discovery…";
}

function showLogin(visible) {
  $("login-screen").hidden = !visible;
  $("shell").hidden = visible;
  if (visible) {
    currentRoute = "";
    setTimeout(() => $("password").focus(), 0);
  }
}

async function refresh() {
  try {
    if (!store.catalog.length) store.catalog = await getJSON("api/catalog");
    if (!store.info.version) store.info = await getJSON("api/info");
    setSnapshot(await getJSON("api/snapshot"));
    showLogin(false);
    renderChrome();
    if (currentHash() === currentRoute && current) current.update();
    else mount();
  } catch (err) {
    if (err instanceof LoginRequired) return;
    $("banner").replaceChildren(el("div", { class: "banner" }, el("strong", {}, "Scimitar is not answering."), el("span", {}, err.message)));
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

$("logout").addEventListener("click", logout);

document.addEventListener("keydown", (event) => {
  const search = document.querySelector("[data-search]");
  if (event.key === "/" && search && document.activeElement?.tagName !== "INPUT") {
    event.preventDefault();
    search.focus();
  }
  if (event.key === "Escape" && document.activeElement === search) {
    search.value = "";
    search.dispatchEvent(new Event("input"));
    search.blur();
  }
});

document.addEventListener("visibilitychange", () => {
  if (!document.hidden) refresh();
});

window.addEventListener("hashchange", mount);
whenLoginRequired(() => showLogin(true));

refresh();
setInterval(() => !document.hidden && refresh(), REFRESH_MS);
setInterval(renderUpdated, 1000);

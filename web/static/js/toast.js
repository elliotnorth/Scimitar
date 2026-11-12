import { el } from "./dom.js";

const VISIBLE_MS = 3500;

export function toast(message, kind = "") {
  const node = el("div", { class: `toast ${kind}` }, message);
  document.getElementById("toasts").append(node);
  setTimeout(() => node.remove(), VISIBLE_MS);
}

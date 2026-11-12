const STORAGE_KEY = "scimitar.theme";

export const DEFAULT_THEME = "rust glow";
export const THEME_STYLES = ["night", "ink", "deep", "glow", "paper"];

const hsl = (h, s, l) => `hsl(${h} ${s}% ${l}%)`;

const DARK_STATUS = { up: hsl(125, 38, 68), down: hsl(6, 78, 68), warn: hsl(40, 72, 66) };

const FEATURED = {
  ivory: { bg: "#08080a", raise: "#101013", hover: "#111114", ink: "#edebe4", dim: "#9b988f", faint: "#5f5d57", line: "#26251f", soft: "#18171a", accent: "#edebe4", up: "#a3c98f", down: "#e8796e", warn: "#dcb56d" },
  dusk: { bg: "#07060b", raise: "#0e0c15", hover: "#100e18", ink: "#ebe6f5", dim: "#9d94b4", faint: "#5f5679", line: "#251e37", soft: "#1a1529", accent: "#a58bf5", up: "#8fd0a8", down: "#f08a9a", warn: "#e3c17c" },
  steel: { bg: "#080a0d", raise: "#0e1217", hover: "#0f1419", ink: "#e3e8ee", dim: "#8e99a6", faint: "#56606c", line: "#1f262e", soft: "#151a20", accent: "#9fc3e6", up: "#8fd1b0", down: "#ef8a83", warn: "#e2bd78" },
  phosphor: { bg: "#050806", raise: "#0b110c", hover: "#0c130d", ink: "#cdeed2", dim: "#7fa688", faint: "#4b6b52", line: "#1a2d1e", soft: "#122016", accent: "#6ee794", up: "#6ee794", down: "#ff7a7a", warn: "#e8cf5a" },
};

const HUES = [
  ["coral", 8], ["ember", 18], ["rust", 28], ["amber", 40], ["gold", 50],
  ["lime", 85], ["moss", 105], ["jade", 150], ["mint", 165], ["teal", 180],
  ["cyan", 190], ["ice", 200], ["sky", 210], ["azure", 220], ["cobalt", 232],
  ["indigo", 248], ["violet", 265], ["plum", 285], ["orchid", 305], ["rose", 335],
];

const GENERATORS = {
  night: (h) => ({ bg: hsl(h, 30, 5), raise: hsl(h, 24, 8), hover: hsl(h, 24, 9), ink: hsl(h, 25, 92), dim: hsl(h, 12, 64), faint: hsl(h, 12, 38), line: hsl(h, 20, 16), soft: hsl(h, 22, 11), accent: hsl(h, 80, 72), ...DARK_STATUS }),
  ink: (h) => ({ bg: hsl(h, 6, 5), raise: hsl(h, 6, 8), hover: hsl(h, 6, 9), ink: hsl(h, 8, 91), dim: hsl(h, 5, 62), faint: hsl(h, 5, 37), line: hsl(h, 6, 15), soft: hsl(h, 6, 10), accent: hsl(h, 70, 68), ...DARK_STATUS }),
  deep: (h) => ({ bg: hsl(h, 45, 8), raise: hsl(h, 40, 11), hover: hsl(h, 40, 12), ink: hsl(h, 40, 94), dim: hsl(h, 22, 68), faint: hsl(h, 20, 42), line: hsl(h, 32, 20), soft: hsl(h, 36, 14), accent: hsl(h, 90, 76), ...DARK_STATUS }),
  glow: (h) => ({ bg: hsl(0, 0, 2), raise: hsl(h, 20, 5), hover: hsl(h, 25, 6), ink: hsl(h, 20, 90), dim: hsl(h, 14, 62), faint: hsl(h, 18, 34), line: hsl(h, 30, 14), soft: hsl(h, 30, 8), accent: hsl(h, 100, 62), up: hsl(140, 80, 58), down: hsl(355, 95, 64), warn: hsl(45, 95, 58) }),
  paper: (h) => ({ bg: hsl(h, 30, 96), raise: hsl(h, 25, 99), hover: hsl(h, 25, 92), ink: hsl(h, 25, 12), dim: hsl(h, 10, 36), faint: hsl(h, 10, 62), line: hsl(h, 15, 84), soft: hsl(h, 20, 90), accent: hsl(h, 65, 38), up: hsl(135, 45, 30), down: hsl(5, 65, 44), warn: hsl(35, 80, 36) }),
};

export const themes = new Map();
for (const [name, colors] of Object.entries(FEATURED)) themes.set(name, { style: "featured", colors });
for (const style of THEME_STYLES) {
  for (const [hue, h] of HUES) themes.set(`${hue} ${style}`, { style, colors: GENERATORS[style](h) });
}

export function currentTheme() {
  let saved = null;
  try {
    saved = localStorage.getItem(STORAGE_KEY);
  } catch {
    // Storage can be blocked; the default theme still applies.
  }
  return themes.has(saved) ? saved : DEFAULT_THEME;
}

export function applyTheme(name) {
  const theme = themes.get(name) || themes.get(DEFAULT_THEME);
  const root = document.documentElement;
  for (const [key, value] of Object.entries(theme.colors)) root.style.setProperty(`--${key}`, value);
  root.style.colorScheme = theme.style === "paper" ? "light" : "dark";
}

export function setTheme(name) {
  try {
    localStorage.setItem(STORAGE_KEY, name);
  } catch {
    // Not remembered across reloads, but still applied now.
  }
  applyTheme(name);
}

export function randomTheme() {
  const names = [...themes.keys()];
  return names[Math.floor(Math.random() * names.length)];
}

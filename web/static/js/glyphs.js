const RAMP = " .·:-=+*#%@";
const COLS = 120;
const ROWS = 17;
const FRAME_MS = 120;

export const LOGO = `███████╗ ██████╗██╗███╗   ███╗██╗████████╗ █████╗ ██████╗
██╔════╝██╔════╝██║████╗ ████║██║╚══██╔══╝██╔══██╗██╔══██╗
███████╗██║     ██║██╔████╔██║██║   ██║   ███████║██████╔╝
╚════██║██║     ██║██║╚██╔╝██║██║   ██║   ██╔══██║██╔══██╗
███████║╚██████╗██║██║ ╚═╝ ██║██║   ██║   ██║  ██║██║  ██║
╚══════╝ ╚═════╝╚═╝╚═╝     ╚═╝╚═╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝`;

// frame draws a slow wave field and a curved blade through it, as two layers of text.
function frame(t) {
  const field = [];
  const blade = [];
  for (let y = 0; y < ROWS; y++) {
    const ny = y / (ROWS - 1);
    let f = "";
    let b = "";
    for (let x = 0; x < COLS; x++) {
      const nx = x / (COLS - 1);
      const wave = 0.5 + 0.5 * Math.sin(nx * 8 + t * 0.55 + Math.sin(ny * 3 + t * 0.3) * 1.6) * Math.cos(ny * 4.5 - t * 0.35 + nx * 2);
      const curve = 0.92 - 0.95 * Math.pow(nx, 1.7) + 0.04 * Math.sin(t * 0.8 + nx * 6);
      const band = Math.exp(-Math.pow(Math.abs(ny - curve) * 7.5, 2)) * (0.35 + 0.65 * nx);
      f += RAMP[Math.floor(Math.min(0.999, wave * 0.42) * RAMP.length)];
      b += band > 0.18 ? RAMP[Math.min(RAMP.length - 1, Math.floor(band * RAMP.length))] : " ";
    }
    field.push(f);
    blade.push(b);
  }
  return { field: field.join("\n"), blade: blade.join("\n") };
}

export function startGlyphs(fieldNode, bladeNode) {
  let t = 0;
  const draw = () => {
    const { field, blade } = frame(t);
    fieldNode.textContent = field;
    bladeNode.textContent = blade;
  };
  draw();
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return { stop() {} };
  const timer = setInterval(() => {
    if (document.hidden) return;
    t += 0.12;
    draw();
  }, FRAME_MS);
  return { stop: () => clearInterval(timer) };
}

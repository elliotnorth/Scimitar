export const store = {
  snapshot: null,
  catalog: [],
  info: { version: "" },
  index: null,
  signature: "",
};

// setSnapshot stores the newest snapshot and reports whether anything other
// than its timestamp changed, so pages are only redrawn when there is news.
export function setSnapshot(snapshot) {
  const signature = JSON.stringify({ ...snapshot, updated_at: null });
  const changed = !store.snapshot || signature !== store.signature;
  store.snapshot = snapshot;
  store.signature = signature;
  if (changed) store.index = buildIndex(snapshot);
  return changed;
}

function buildIndex(s) {
  const devicesById = new Map(s.devices.map((d) => [d.id, d]));
  const servicesByDevice = new Map();
  for (const svc of s.services) {
    if (!servicesByDevice.has(svc.device_id)) servicesByDevice.set(svc.device_id, []);
    servicesByDevice.get(svc.device_id).push(svc);
  }
  const parentOf = new Map();
  const childrenOf = new Map();
  const connectedTo = new Map();
  for (const link of s.links) {
    if (link.kind === "connected") {
      for (const [a, b] of [[link.from, link.to], [link.to, link.from]]) {
        if (!connectedTo.has(a)) connectedTo.set(a, []);
        connectedTo.get(a).push(b);
      }
      continue;
    }
    if (link.kind !== "runs-on") continue;
    parentOf.set(link.from, link.to);
    if (!childrenOf.has(link.to)) childrenOf.set(link.to, []);
    childrenOf.get(link.to).push(link.from);
  }
  return { devicesById, servicesByDevice, parentOf, childrenOf, connectedTo };
}

export function deviceName(id) {
  return store.index?.devicesById.get(id)?.name || id;
}

export function parentName(id) {
  const parent = store.index?.parentOf.get(id);
  return parent ? deviceName(parent) : "";
}

export function catalogName(id) {
  return store.catalog.find((c) => c.id === id)?.name || id;
}

export function deviceHref(id) {
  return `#/devices/${encodeURIComponent(id)}`;
}

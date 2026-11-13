export class LoginRequired extends Error {}

let onLoginRequired = () => {};

export function whenLoginRequired(handler) {
  onLoginRequired = handler;
}

async function checked(res) {
  if (res.status === 401) {
    onLoginRequired();
    throw new LoginRequired();
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error || res.statusText);
  }
  return res;
}

export async function getJSON(url) {
  return (await checked(await fetch(url))).json();
}

// sendJSON makes a request that changes something. Leave body out for requests that carry none.
export async function sendJSON(method, url, body) {
  return checked(
    await fetch(url, {
      method,
      headers: body === undefined ? {} : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
    }),
  );
}

export function postJSON(url, body) {
  return sendJSON("POST", url, body);
}

export async function login(password) {
  const res = await fetch("api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  return res.status;
}

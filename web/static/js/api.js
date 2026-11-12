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

export async function postJSON(url, body) {
  return checked(
    await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  );
}

export async function login(password) {
  const res = await fetch("api/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  return res.status;
}

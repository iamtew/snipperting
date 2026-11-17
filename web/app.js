async function api(path, opt) {
  const r = await fetch(path, opt);
  if (!r.ok) throw new Error(await r.text());
  const ct = r.headers.get("content-type") || "";
  if (ct.includes("json")) return r.json();
  return r.text();
}

async function applyTheme() {
  try {
    const c = await api("/api/config");
    document.documentElement.dataset.theme = c.theme || "system";
    return c;
  } catch (e) {
    return { theme: "system" };
  }
}

function opposite(action) {
  return action === "inject" ? "clipboard" : "inject";
}

function actLetter(action) {
  return action === "inject" ? "I" : "C";
}

async function doAct(path, op, value) {
  return api("/api/snippets/act", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path, op, value: value || "" }),
  });
}

async function openUI(kind, file) {
  return api("/api/ui", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ kind, file: file || "" }),
  });
}

function mdLite(src) {
  const esc = (s) => s.replace(/[&<>]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[c]));
  return esc(src)
    .replace(/^### (.*)$/gm, "<h3>$1</h3>")
    .replace(/^## (.*)$/gm, "<h2>$1</h2>")
    .replace(/^# (.*)$/gm, "<h1>$1</h1>")
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\n/g, "<br>\n");
}

applyTheme();

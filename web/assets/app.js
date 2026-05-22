// region.id playground — vanilla ES module, no framework.
//
// Talks to the same origin's /api/* tree. The UI builds cascading dropdowns
// for endpoints that take an ID, lazy-fetching child lists when a parent is
// picked. JSON syntax highlight is a small hand-rolled regex pass.

const $ = (sel) => document.querySelector(sel);

const ENDPOINTS = [
  { id: "provinces",     path: "/api/provinces.json",                    params: [] },
  { id: "regencies",     path: "/api/regencies/{provinceId}.json",       params: ["provinceId"] },
  { id: "districts",     path: "/api/districts/{regencyId}.json",        params: ["provinceId", "regencyId"] },
  { id: "villages",      path: "/api/villages/{districtId}.json",        params: ["provinceId", "regencyId", "districtId"] },
  { id: "province",      path: "/api/province/{id}.json",                params: ["provinceId"], idAlias: { provinceId: "id" } },
  { id: "regency",       path: "/api/regency/{id}.json",                 params: ["provinceId", "regencyId"], idAlias: { regencyId: "id" } },
  { id: "district",      path: "/api/district/{id}.json",                params: ["provinceId", "regencyId", "districtId"], idAlias: { districtId: "id" } },
  { id: "village",       path: "/api/village/{id}.json",                 params: ["provinceId", "regencyId", "districtId", "villageId"], idAlias: { villageId: "id" } },
];

const state = {
  endpoint: ENDPOINTS[0],
  provinceId: "",
  regencyId: "",
  districtId: "",
  villageId: "",
  // cache: { [path]: payload }
  cache: new Map(),
};

async function fetchJSON(path) {
  if (state.cache.has(path)) return state.cache.get(path);
  const r = await fetch(path);
  if (!r.ok) throw new Error(`HTTP ${r.status}`);
  const data = await r.json();
  state.cache.set(path, data);
  return data;
}

function buildURL(ep) {
  let p = ep.path;
  // {id} aliases — last segment of the params chain becomes the id substitution.
  const alias = ep.idAlias || {};
  for (const key of ep.params) {
    const realKey = alias[key] || key;
    const value = state[key] || "";
    p = p.replace(`{${realKey}}`, value);
  }
  return p;
}

function initEndpointSelector() {
  const sel = $("#endpoint");
  ENDPOINTS.forEach((ep, i) => {
    const opt = document.createElement("option");
    opt.value = i;
    opt.textContent = `GET ${ep.path}`;
    sel.appendChild(opt);
  });
  sel.addEventListener("change", () => {
    state.endpoint = ENDPOINTS[sel.value];
    renderParams();
    updateURL();
  });
}

function renderParams() {
  const row = $("#paramRow");
  row.innerHTML = "";
  for (const key of state.endpoint.params) {
    row.appendChild(makeParamField(key));
  }
  // Populate cascades.
  populateProvinces();
  if (state.provinceId) populateRegencies();
  if (state.regencyId) populateDistricts();
  if (state.districtId) populateVillages();
}

function makeParamField(key) {
  const label = document.createElement("label");
  label.className = "field grow";
  const span = document.createElement("span");
  span.textContent = labelFor(key);
  const sel = document.createElement("select");
  sel.id = `sel-${key}`;
  sel.dataset.key = key;
  sel.innerHTML = `<option value="">— pilih —</option>`;
  sel.addEventListener("change", () => onParamChange(key, sel.value));
  label.append(span, sel);
  return label;
}

function labelFor(key) {
  return ({
    provinceId: "Province",
    regencyId: "Regency",
    districtId: "District",
    villageId: "Village",
  })[key] || key;
}

function onParamChange(key, value) {
  state[key] = value;
  // Reset deeper levels.
  const chain = ["provinceId", "regencyId", "districtId", "villageId"];
  const i = chain.indexOf(key);
  for (let j = i + 1; j < chain.length; j++) state[chain[j]] = "";
  updateURL();
  if (key === "provinceId" && value) populateRegencies();
  if (key === "regencyId" && value) populateDistricts();
  if (key === "districtId" && value) populateVillages();
}

async function populateProvinces() {
  const sel = $("#sel-provinceId");
  if (!sel) return;
  try {
    const data = await fetchJSON("/api/provinces.json");
    fillSelect(sel, data, state.provinceId);
  } catch (e) { /* ignore */ }
}

async function populateRegencies() {
  const sel = $("#sel-regencyId");
  if (!sel || !state.provinceId) return;
  try {
    const data = await fetchJSON(`/api/regencies/${state.provinceId}.json`);
    fillSelect(sel, data, state.regencyId);
  } catch (e) {}
}

async function populateDistricts() {
  const sel = $("#sel-districtId");
  if (!sel || !state.regencyId) return;
  try {
    const data = await fetchJSON(`/api/districts/${state.regencyId}.json`);
    fillSelect(sel, data, state.districtId);
  } catch (e) {}
}

async function populateVillages() {
  const sel = $("#sel-villageId");
  if (!sel || !state.districtId) return;
  try {
    const data = await fetchJSON(`/api/villages/${state.districtId}.json`);
    fillSelect(sel, data, state.villageId);
  } catch (e) {}
}

function fillSelect(sel, list, currentValue) {
  // Sort by name for a nicer browsing experience.
  list = list.slice().sort((a, b) => a.name.localeCompare(b.name, "id"));
  sel.innerHTML = `<option value="">— pilih (${list.length}) —</option>` +
    list.map(x => `<option value="${x.id}"${x.id === currentValue ? " selected" : ""}>${x.name} (${x.id})</option>`).join("");
}

function updateURL() {
  const u = buildURL(state.endpoint);
  $("#url").textContent = u;
}

async function send() {
  const url = buildURL(state.endpoint);
  const t0 = performance.now();
  const meta = $("#meta");
  const out = $("#response");
  out.innerHTML = `<span class="muted">Fetching ${url}…</span>`;
  meta.textContent = "";
  try {
    const r = await fetch(url);
    const text = await r.text();
    const ms = (performance.now() - t0).toFixed(0);
    const bytes = new Blob([text]).size;
    meta.textContent = `${r.status} ${r.statusText} · ${ms} ms · ${formatBytes(bytes)}`;
    let pretty;
    try {
      pretty = JSON.stringify(JSON.parse(text), null, 2);
    } catch {
      pretty = text;
    }
    out.innerHTML = highlight(pretty);
  } catch (e) {
    out.innerHTML = `<span class="muted">Error: ${e.message}</span>`;
  }
}

function formatBytes(n) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}

function highlight(text) {
  // Escape HTML first.
  text = text.replace(/[&<>]/g, ch => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;" }[ch]));
  // Strings (including keys).
  text = text.replace(/"([^"\\]|\\.)*"(\s*:)?/g, (m, _g, colon) => {
    return colon ? `<span class="tok-key">${m}</span>` : `<span class="tok-str">${m}</span>`;
  });
  // Numbers / booleans / null.
  text = text.replace(/\b-?\d+(\.\d+)?\b/g, m => `<span class="tok-num">${m}</span>`);
  text = text.replace(/\b(true|false)\b/g, m => `<span class="tok-bool">${m}</span>`);
  text = text.replace(/\bnull\b/g, m => `<span class="tok-null">${m}</span>`);
  return text;
}

function copyCurl() {
  const u = new URL(buildURL(state.endpoint), location.origin).toString();
  const cmd = `curl ${u}`;
  navigator.clipboard.writeText(cmd).then(
    () => { $("#meta").textContent = "Copied!"; setTimeout(() => $("#meta").textContent = "", 1500); },
    () => { $("#meta").textContent = "Copy failed"; }
  );
}

async function loadMeta() {
  try {
    const m = await fetchJSON("/api/meta.json");
    $("#ver").textContent = `v${m.version}`;
    if (m.counts) {
      $("#stats").innerHTML = [
        ["provinces", m.counts.provinces],
        ["regencies", m.counts.regencies],
        ["districts", m.counts.districts],
        ["villages",  m.counts.villages],
      ].map(([label, n]) =>
        `<span class="stat"><strong>${n.toLocaleString("en-US")}</strong> ${label}</span>`).join("");
    }
  } catch (e) {
    $("#ver").textContent = "static";
  }
}

// Boot.
initEndpointSelector();
renderParams();
updateURL();
loadMeta();
$("#send").addEventListener("click", send);
$("#copyCurl").addEventListener("click", copyCurl);

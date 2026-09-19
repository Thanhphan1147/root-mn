"use strict";

let game = null;

const PHASE = { S: "Setup", B: "Birdsong", D: "Daylight", E: "Evening" };

// Autumn map: clearing positions (percent) and the 18 printed paths.
const POS = {
  C1: [12, 15], C2: [88, 15], C3: [88, 85], C4: [12, 85],
  C5: [50, 7], C6: [93, 48], C7: [56, 83], C8: [28, 95],
  C9: [5, 48], C10: [50, 32], C11: [71, 60], C12: [32, 55],
};
const EDGES = [
  ["C1","C5"],["C1","C9"],["C1","C10"],["C2","C5"],["C2","C6"],["C2","C10"],
  ["C3","C6"],["C3","C7"],["C3","C11"],["C4","C8"],["C4","C9"],["C4","C12"],
  ["C6","C11"],["C7","C8"],["C7","C12"],["C9","C12"],["C10","C12"],["C11","C12"],
];

const LS_KEY = "root-rmn-state-v1";

function persist() {
  if (!window.RootEngine || !game) return;
  try { localStorage.setItem(LS_KEY, window.RootEngine.save()); } catch (e) { /* quota */ }
}

function doAction(id) {
  if (!window.RootEngine) return;
  game = JSON.parse(window.RootEngine.apply(id));
  persist();
  render();
}

function newGame() {
  if (!window.RootEngine) return;
  if (game && !confirm("Start a new game? The current one will be replaced.")) return;
  game = JSON.parse(window.RootEngine.newGame(Math.floor(Math.random() * 1e9)));
  persist();
  render();
}

function autoSetup() {
  if (!window.RootEngine) return;
  game = JSON.parse(window.RootEngine.autoSetup(Math.floor(Math.random() * 1e9)));
  persist();
  render();
}

function render() {
  const g = game;
  document.getElementById("turnbar").innerHTML =
    `<div>round <b>${g.round}</b> · phase <b>${PHASE[g.phase] || g.phase}</b></div>` +
    `<div>current <b>${g.current}</b>${g.winner && g.winner.length ? " · winner " + g.winner.join("+") : ""}</div>`;

  renderPlayers(g);
  renderBoard(g);
  renderActions(g);
  renderLog(g);
}

function renderPlayers(g) {
  const el = document.getElementById("players");
  el.innerHTML = "";
  for (const f of g.order) {
    const p = g.players[f];
    const div = document.createElement("div");
    div.className = "pcard " + f + (f === g.current ? " current" : "");
    let extra = "";
    if (f === "MC") {
      extra = `<div class="row"><span>wood supply</span><span>${p.WoodSupply}</span></div>` +
        `<div class="row"><span>saw/ws/rec</span><span>${p.Sawmills}/${p.Workshops}/${p.Recruiters}</span></div>` +
        `<div class="row"><span>keep</span><span>${p.KeepClearing}</span></div>`;
    } else if (f === "ED") {
      extra = `<div class="row"><span>leader</span><span>${p.Leader}</span></div>` +
        `<div class="row"><span>roosts</span><span>${countRoosts(g, "ED")}</span></div>`;
      const dec = p.Decree || {};
      for (const col of ["RECRUIT", "MOVE", "BATTLE", "BUILD"]) {
        const cards = (dec[col] || []).map(cardLabel).join(" ");
        if (cards) extra += `<div class="row"><span>${col.slice(0, 3)}</span><span class="cards">${cards}</span></div>`;
      }
    } else if (f === "WA") {
      extra = `<div class="row"><span>officers</span><span>${p.Officers}</span></div>` +
        `<div class="row"><span>supporters</span><span class="cards">${(p.Supporters || []).map(cardLabel).join(" ")}</span></div>`;
    } else if (f === "VB") {
      extra = `<div class="row"><span>character</span><span>${p.Character}</span></div>` +
        `<div class="row"><span>at</span><span>${p.Pawn}</span></div>` +
        `<div class="row"><span>items</span><span class="cards">${itemList(p)}</span></div>`;
      const rel = p.Relationships || {};
      const tags = Object.entries(rel).map(([k, v]) => `<span class="tag ${v === "hostile" ? "hostile" : ""}">${k}:${v}</span>`).join("");
      extra += `<div class="tags">${tags}</div>`;
    }
    div.innerHTML =
      `<div class="phead"><span class="f">${f}</span><span class="vp">${p.VP} VP</span></div>` +
      `<div class="pbody">${extra}` +
      `<div class="row"><span>crafted</span><span>${(p.Crafted || []).map(cardLabel).join(" ") || "—"}</span></div>` +
      `</div>`;
    div.append(renderHand(p));
    el.append(div);
  }
}

function renderHand(p) {
  const wrap = document.createElement("div");
  wrap.className = "hand";
  const hand = p.Hand || [];
  if (hand.length === 0) {
    wrap.innerHTML = '<div class="hempty">no cards</div>';
    return wrap;
  }
  for (const id of hand) {
    const info = (game.cards && game.cards[id]) || { name: id, suit: "B", desc: "" };
    const c = document.createElement("div");
    c.className = "hcard suit-" + (info.suit || "B");
    c.innerHTML =
      `<div class="hname">${info.name}<span class="hid">${id}</span></div>` +
      `<div class="hcost">${info.cost ? "craft: " + info.cost : (info.kind === "ambush" ? "battle" : info.kind)}</div>` +
      `<div class="hdesc">${info.desc}</div>`;
    wrap.append(c);
  }
  return wrap;
}

function countRoosts(g, f) {
  let n = 0;
  for (const c of Object.values(g.clearings)) {
    for (const b of (c.Buildings || [])) if (b.Owner === f && b.Type === "roost") n++;
  }
  return n;
}

function itemList(p) {
  const out = [];
  for (const [id, it] of Object.entries(p.Items || {})) {
    let s = it.Type;
    if (it.Zone === "track") s += "↑";
    if (!it.FaceUp) s += "×";
    if (it.Damaged) s += "✗";
    out.push(s);
  }
  return out.join(" ");
}

function cardLabel(id) {
  if (id === "VIZIER") return "Viz";
  if (/^[FRMB]\d\d$/.test(id)) return id;
  return id;
}

function renderBoard(g) {
  const el = document.getElementById("board");
  el.innerHTML = "";

  // Roads (SVG underlay).
  const NS = "http://www.w3.org/2000/svg";
  const svg = document.createElementNS(NS, "svg");
  svg.setAttribute("class", "roads");
  svg.setAttribute("viewBox", "0 0 100 100");
  svg.setAttribute("preserveAspectRatio", "none");
  for (const [a, b] of EDGES) {
    if (!POS[a] || !POS[b]) continue;
    for (const cls of ["casing", "road"]) {
      const ln = document.createElementNS(NS, "line");
      ln.setAttribute("x1", POS[a][0]); ln.setAttribute("y1", POS[a][1]);
      ln.setAttribute("x2", POS[b][0]); ln.setAttribute("y2", POS[b][1]);
      ln.setAttribute("class", cls);
      ln.dataset.c1 = a; ln.dataset.c2 = b;
      svg.append(ln);
    }
  }
  el.append(svg);

  const ids = Object.keys(g.clearings).sort((a, b) => parseInt(a.slice(1)) - parseInt(b.slice(1)));
  const hl = new Set((g.legal || []).map(a => a.clearing || a.to || a.from).filter(Boolean));

  for (const id of ids) {
    const c = g.clearings[id];
    const div = document.createElement("div");
    div.className = "clearing" + (hl.has(id) ? " hl" : "");
    const [x, y] = POS[id] || [50, 50];
    div.style.left = x + "%";
    div.style.top = y + "%";
    div.dataset.clearing = id;
    div.onmouseenter = () => highlightRoads(svg, id, true);
    div.onmouseleave = () => highlightRoads(svg, id, false);

    let chips = "";
    const order = ["MC", "ED", "WA", "VB"];
    for (const f of order) {
      const n = (c.Warriors || {})[f];
      if (n) chips += `<span class="chip ${f}">${f}×${n}</span>`;
    }
    for (const b of (c.Buildings || [])) chips += `<span class="chip ${b.Owner}">${b.Type}</span>`;
    for (const t of (c.Tokens || [])) chips += `<span class="chip ${t.Owner}">${t.Type}</span>`;
    if (c.Sympathy) chips += `<span class="chip WA">sympathy</span>`;
    const wood = c.Wood ? `<span class="wood">wood ${c.Wood}</span>` : "";
    const ruin = c.Ruin ? `<span class="ruin">ruin${c.RuinItem ? " " + c.RuinItem : ""}</span>` : "";
    div.innerHTML =
      `<div class="cid"><span>${id}</span><span class="suit ${c.Suit}">${c.Suit}</span></div>` +
      `<div class="crowd">${chips}${wood}</div>${ruin}`;
    el.append(div);
  }
  document.getElementById("boardfoot").textContent =
    "roads: " + EDGES.map(([a, b]) => a + "–" + b).join("  ");
}

function highlightRoads(svg, id, on) {
  for (const ln of svg.querySelectorAll("line")) {
    if (ln.dataset.c1 === id || ln.dataset.c2 === id) {
      ln.classList.toggle("hot", on);
    }
  }
}

function renderActions(g) {
  const el = document.getElementById("actions");
  el.innerHTML = "";
  const head = document.getElementById("actionhead");
  const pend = document.getElementById("pendhint");
  pend.textContent = g.pending ? g.pending.Kind + " (" + g.pending.Player + ")" : "";
  if (g.setupMode) {
    head.textContent = "Setup · " + (g.setupStage || "");
  } else {
    head.textContent = g.winner && g.winner.length ? "Game over" : "Actions · " + g.current;
  }

  if (g.battle) {
    const b = g.battle;
    const banner = document.createElement("div");
    banner.className = "battlebar";
    const dice = (b.D1 !== undefined && b.D1 !== null) ? ` · dice ${b.D1}-${b.D2}` : "";
    banner.innerHTML = `<b>Battle</b> ${b.Attacker}→${b.Defender} at ${b.Clearing}` +
      `<div class="step">step ${b.Step}/5 · ${b.StepName}${dice}</div>` +
      `<div class="hint">attacker hits ${b.AtkHits || 0} · defender hits ${b.DefHits || 0}` +
      (b.Remaining ? ` · ${b.Remaining} to assign` : "") + `</div>`;
    el.append(banner);
  }

  if (g.winner && g.winner.length) {
    const d = document.createElement("div");
    d.className = "winner";
    d.textContent = "Winner: " + g.winner.join(" + ");
    el.append(d);
    return;
  }
  const acts = g.legal || [];
  if (acts.length === 0) {
    const d = document.createElement("div");
    d.className = "hint";
    d.textContent = "No legal actions.";
    el.append(d);
    return;
  }
  // Group: pending first, then by kind.
  for (const a of acts) {
    const b = document.createElement("button");
    b.className = (a.kind || "").replace(/:/g, "-");
    b.textContent = a.label || a.id;
    b.onclick = () => doAction(a.id);
    el.append(b);
  }
}

function renderLog(g) {
  const el = document.getElementById("log");
  el.innerHTML = "";
  const entries = g.log || [];
  for (const e of entries.slice(-120)) {
    const li = document.createElement("li");
    li.className = (e.kind || "") + (e.kind === "battle" || e.kind === "turmoil" || e.kind === "ambush" ? " battle" : "");
    li.innerHTML = `<span class="seq">${e.seq}</span><span>${e.round}.${e.phase}</span>` +
      `<span class="act ${e.actor}">${e.actor}</span><span>${e.text}</span>`;
    el.append(li);
  }
  el.scrollTop = el.scrollHeight;
}

document.getElementById("newgame").onclick = newGame;
document.getElementById("autosetup").onclick = autoSetup;

async function boot() {
  const status = document.getElementById("actions");
  status.innerHTML = '<div class="hint">loading engine…</div>';
  const go = new Go();
  const resp = await fetch("root.wasm");
  const buf = await resp.arrayBuffer();
  const result = await WebAssembly.instantiate(buf, go.importObject);
  go.run(result.instance);
  for (let i = 0; i < 400 && !window.RootEngine; i++) {
    await new Promise(r => setTimeout(r, 25));
  }
  if (!window.RootEngine) throw new Error("engine failed to start");
  const saved = localStorage.getItem(LS_KEY);
  if (saved) {
    const restored = JSON.parse(window.RootEngine.load(saved));
    if (restored && !restored.error) game = restored;
  }
  if (!game) game = JSON.parse(window.RootEngine.newGame(Math.floor(Math.random() * 1e9)));
  render();
}

boot().catch(err => {
  document.getElementById("actions").innerHTML =
    `<div class="winner">Failed to start: ${err}. Serve this folder over http(s) (e.g. GitHub Pages).</div>`;
});

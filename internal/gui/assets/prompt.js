(() => {
  const $ = (s) => document.querySelector(s);
  const q = $("#spotQ");
  const list = $("#spotList");
  const empty = $("#spotEmpty");
  const note = $("#spotNote");
  const shortcut = $("#spotShortcut");
  if (!q || !list) return;

  let prompts = [];
  let filtered = [];
  let picked = 0;
  let busy = false;
  let info = {};

  setLocale(window.bootPrefs?.lang);

  async function api(path, body) {
    const r = await fetch("/api/" + path, body === undefined ? {} : {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    const data = await r.json().catch(() => ({}));
    if (!r.ok) throw new Error(data.error || "magpie couldn't do that");
    return data;
  }

  function prettyShortcut(s) {
    if (!s) return "";
    const mac = /^Mac/.test(navigator.platform);
    return s.split("+").map((x) => {
      const k = x.toLowerCase();
      if (k === "cmdorctrl") return mac ? "⌘" : "Ctrl";
      if (k === "cmd" || k === "command") return "⌘";
      if (k === "ctrl") return mac ? "⌃" : "Ctrl";
      if (k === "option" || k === "alt" || k === "optionoralt") return mac ? "⌥" : "Alt";
      if (k === "shift") return mac ? "⇧" : "Shift";
      if (k === "space") return "Space";
      return x.length === 1 ? x.toUpperCase() : x;
    }).join(mac ? "" : "+");
  }

  function matchRank(p, words) {
    const name = p.name.toLocaleLowerCase();
    const text = p.text.toLocaleLowerCase();
    let total = 0;
    for (const word of words) {
      const at = name.indexOf(word);
      const inText = text.indexOf(word);
      if (at < 0 && inText < 0) return -1;
      total += at === 0 ? 100 : at > 0 ? 50 : 10;
      if (name.includes(word)) total += 20;
    }
    return total;
  }

  function filter() {
    const words = q.value.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
    filtered = prompts
      .map((p) => ({ p, rank: words.length ? matchRank(p, words) : 0 }))
      .filter((x) => x.rank >= 0)
      .sort((a, b) => b.rank - a.rank || a.p.name.localeCompare(b.p.name))
      .map((x) => x.p);
    picked = Math.min(picked, Math.max(0, filtered.length - 1));
    draw();
  }

  function draw() {
    list.replaceChildren();
    if (!filtered.length) {
      list.hidden = true;
      empty.hidden = false;
      empty.textContent = prompts.length
        ? t("No matching prompts")
        : t("No prompts yet — add one in Library → Prompts.");
      return;
    }
    list.hidden = false;
    empty.hidden = true;
    for (const [i, p] of filtered.entries()) {
      const row = document.createElement("button");
      row.type = "button";
      row.className = "spot-row" + (i === picked ? " on" : "");
      row.setAttribute("role", "option");
      row.setAttribute("aria-selected", String(i === picked));
      row.dataset.i = String(i);

      const who = document.createElement("span");
      who.className = "spot-who";
      const name = document.createElement("span");
      name.className = "spot-name";
      name.textContent = p.name;
      const preview = document.createElement("span");
      preview.className = "spot-preview";
      preview.textContent = p.text;
      who.append(name, preview);

      const enter = document.createElement("span");
      enter.className = "spot-enter";
      enter.textContent = "↵";
      row.append(who, enter);
      row.onclick = () => { picked = i; insert(); };
      row.onmousemove = () => {
        if (picked === i) return;
        picked = i;
        paint();
      };
      list.append(row);
    }
    paint();
  }

  function paint() {
    for (const row of list.children) {
      const on = +row.dataset.i === picked;
      row.classList.toggle("on", on);
      row.setAttribute("aria-selected", String(on));
    }
    const row = list.children[picked];
    if (row) row.scrollIntoView({ block: "nearest" });
  }

  function showNote(text, needsAccess) {
    note.replaceChildren();
    note.hidden = !text;
    if (!text) return;
    note.append(document.createTextNode(t(text)));
    if (needsAccess) {
      const open = document.createElement("button");
      open.type = "button";
      open.textContent = t("Open System Settings");
      open.onclick = () => api("prompts/accessibility", {}).catch((e) => showNote(e.message));
      note.append(open);
    }
  }

  async function load() {
    try {
      info = await api("prompts");
      prompts = info.prompts || [];
      shortcut.textContent = prettyShortcut(info.shortcut);
      if (info.shortcutError) showNote(info.shortcutError);
      else if (!info.insertAvailable) showNote("Automatic insertion is unavailable. Choosing a prompt copies it to the clipboard.", info.needsAccess);
      else if (info.needsAccess) showNote("Grant magpie Accessibility access to insert prompts automatically.", true);
      else showNote("");
      filter();
      q.focus({ preventScroll: true });
      q.select();
    } catch (e) {
      prompts = [];
      filtered = [];
      draw();
      showNote(e.message);
    }
  }

  async function insert() {
    if (busy || !filtered[picked]) return;
    busy = true;
    list.setAttribute("aria-busy", "true");
    try {
      const out = await api("prompts/insert", { id: filtered[picked].id });
      if (out.inserted) return;
      showNote(out.error || "The prompt couldn't be inserted.", out.needsAccess);
    } catch (e) {
      showNote(e.message);
    } finally {
      busy = false;
      list.removeAttribute("aria-busy");
    }
  }

  q.addEventListener("input", () => { picked = 0; filter(); });
  q.addEventListener("keydown", (e) => {
    if (e.key === "ArrowDown" || e.key === "ArrowUp") {
      e.preventDefault();
      if (!filtered.length) return;
      picked = (picked + (e.key === "ArrowDown" ? 1 : -1) + filtered.length) % filtered.length;
      paint();
    } else if (e.key === "Enter") {
      e.preventDefault();
      insert();
    }
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      api("prompts/hide", {}).catch(() => {});
    }
  });
  window.addEventListener("focus", load);
  window.promptSpotShown = load;
  load();
})();

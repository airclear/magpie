// The prompt picker and Library › Prompts. No backend: the JSON API is faked.
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = path.resolve(__dirname, "../assets");
const HOME = "/Users/aimer";
const prompts = [
  { id: "review", name: "Code review", text: "Review this change for bugs and missing tests." },
  { id: "explain", name: "Explain simply", text: "Explain this code in plain language." },
];

async function file(route, url) {
  const name = url.pathname === "/" || url.pathname === "/index.html" ? "index.html" : url.pathname.slice(1);
  const body = await fs.readFile(path.join(assets, name));
  const type = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(name)];
  return route.fulfill({ body, contentType: type });
}

function pickerServer(lang, events, insertResult = { inserted: true }) {
  return async (route) => {
    const url = new URL(route.request().url());
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:${JSON.stringify(lang)},theme:"light",web:false};` });
    if (url.pathname === "/api/prompts") return route.fulfill({ json: { prompts, shortcut: "CmdOrCtrl+Shift+Space", insertAvailable: true } });
    if (url.pathname === "/api/prompts/insert") {
      assert.equal(route.request().method(), "POST");
      events.insert.push(route.request().postDataJSON());
      return route.fulfill({ json: insertResult });
    }
    if (url.pathname === "/api/prompts/hide") {
      assert.equal(route.request().method(), "POST");
      events.hide++;
      return route.fulfill({ json: { ok: true } });
    }
    if (url.pathname === "/api/prompts/accessibility") {
      assert.equal(route.request().method(), "POST");
      events.accessibility++;
      return route.fulfill({ json: { ok: true } });
    }
    if (url.pathname.startsWith("/api/")) return route.fulfill({ json: {} });
    return file(route, url);
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  for (const lang of ["en", "zh", "zh-TW", "ja", "de"]) {
    test(`${engine} ${lang}: the prompt picker searches and inserts`, async (t) => {
      const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
      t.after(() => browser.close());
      const ctx = await browser.newContext({ viewport: { width: 680, height: 430 }, reducedMotion: "reduce" });
      const page = await ctx.newPage();
      page.setDefaultTimeout(5000);
      const errors = [], events = { insert: [], hide: 0, accessibility: 0 };
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", pickerServer(lang, events));
      await page.goto("http://magpie.test/prompt.html");

      const q = page.locator("#spotQ");
      await q.waitFor();
      assert.equal(await q.evaluate((e) => document.activeElement === e), true, "the search field has focus");
      assert.equal(await page.locator(".spot-row").count(), 2);
      await q.fill("review");
      await page.waitForFunction(() => document.querySelectorAll(".spot-row").length === 1);
      assert.match(await page.locator(".spot-row").textContent(), /Code review/);
      await q.press("Enter");
      await page.waitForFunction(() => true);
      assert.deepEqual(events.insert, [{ id: "review" }]);
      assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
      assert.deepEqual(errors, []);
      await ctx.close();
    });
  }

  test(`${engine}: the prompt picker keyboard and clipboard fallback`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const ctx = await browser.newContext({ viewport: { width: 420, height: 320 }, reducedMotion: "reduce" });
    const page = await ctx.newPage();
    page.setDefaultTimeout(5000);
    const errors = [], events = { insert: [], hide: 0, accessibility: 0 };
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", pickerServer("en", events, {
      inserted: false,
      copied: true,
      needsAccess: true,
      error: "Automatic insertion is not available. The prompt is on the clipboard.",
    }));
    await page.goto("http://magpie.test/prompt.html");
    await page.locator("#spotQ").press("ArrowDown");
    assert.equal(await page.locator(".spot-row.on").textContent().then((s) => s.includes("Explain simply")), true);
    await page.locator("#spotQ").press("Enter");
    await page.getByText("Automatic insertion is not available. The prompt is on the clipboard.").waitFor();
    await page.getByRole("button", { name: "Open System Settings" }).click();
    assert.equal(events.accessibility, 1);
    await page.locator("#spotQ").press("Escape");
    assert.equal(events.hide, 1);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.deepEqual(errors, []);
    await ctx.close();
  });

  test(`${engine}: Library owns the prompt list`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const ctx = await browser.newContext({ viewport: { width: 860, height: 700 }, reducedMotion: "reduce" });
    await ctx.addInitScript(() => localStorage.setItem("magpie.libTab", "prompts"));
    const page = await ctx.newPage();
    page.setDefaultTimeout(5000);
    const errors = [];
    let saved = null;
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", async (route) => {
      const url = new URL(route.request().url());
      const json = (v) => route.fulfill({ json: v });
      if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"light",web:false};` });
      if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
      if (url.pathname === "/api/state") return json({ agents: [], profiles: [], settings: { lang: "en", theme: "light" } });
      if (url.pathname === "/api/settings") return json({ lang: "en", theme: "light", promptShortcut: "CmdOrCtrl+Shift+Space" });
      if (url.pathname === "/api/plugins") return json({ plugins: [] });
      if (url.pathname === "/api/library") return json({
        dir: `${HOME}/.magpie/library`, backups: `${HOME}/.magpie/backups`, home: HOME,
        agents: [{ id: "codex", name: "Codex", icon: "", skills: `${HOME}/.codex/skills`, mcp: `${HOME}/.codex/mcp.json` }],
        instructions: { agents: [], sets: [] }, foundServers: [], projects: [], foundSkills: [], skills: [], servers: [], hooks: [],
      });
      if (url.pathname === "/api/prompts") return json({ prompts, shortcut: "CmdOrCtrl+Shift+Space", insertAvailable: true });
      if (url.pathname === "/api/prompts/save") {
        saved = route.request().postDataJSON();
        return json({ prompts: [...prompts, { id: "new", name: saved.name, text: saved.text }], shortcut: "CmdOrCtrl+Shift+Space", insertAvailable: true });
      }
      if (url.pathname === "/api/usage/quotas") return json([]);
      if (url.pathname === "/api/groups") return json({ groups: [] });
      if (url.pathname === "/api/providers") return json({ providers: [], presets: [], excluded: [], gateway: { url: "http://127.0.0.1:3425", running: true, window: true } });
      if (url.pathname.startsWith("/api/")) return json({});
      return file(route, url);
    });
    await page.goto("http://magpie.test/");
    await page.locator('button[data-view="library"]').click();
    const view = page.locator("#view-library");
    await view.locator(".lib-prompt").first().waitFor();
    assert.match(await view.locator(".lib-tabs .opt.on").textContent(), /^Prompts/);
    assert.match(await view.textContent(), /Prompt picker shortcut/);
    await view.getByRole("button", { name: "＋ Add prompt" }).first().click();
    const dialog = page.locator("#modal .dialog");
    await dialog.locator('input').fill("Tests");
    await dialog.locator("textarea").fill("Write a failing test first.");
    await dialog.getByRole("button", { name: "Add" }).click();
    assert.deepEqual(saved, { id: "", name: "Tests", text: "Write a failing test first." });
    await view.getByText("Tests", { exact: true }).waitFor();
    assert.match(await view.locator(".lib-tabs .opt.on").textContent(), /Prompts · 3/);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), true);
    assert.deepEqual(errors, []);
    await ctx.close();
  });
}

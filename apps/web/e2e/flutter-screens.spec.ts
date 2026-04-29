import { test } from "@playwright/test";
import path from "path";

const OUT = path.resolve(__dirname, "../../../docs/demo-screenshots");

test.use({ viewport: { width: 412, height: 900 } });

test("capture flutter login screen", async ({ page }) => {
  test.setTimeout(60_000);
  await page.goto("http://localhost:3002/");
  await page.waitForFunction(
    () => document.querySelectorAll("flt-glass-pane,flutter-view").length > 0,
    null,
    { timeout: 30000 },
  );
  await page.waitForTimeout(2500);
  await page.screenshot({
    path: path.join(OUT, "10-flutter-login.png"),
    fullPage: false,
  });
});

test("capture flutter home with seeded token", async ({ page, request }) => {
  test.setTimeout(60_000);

  // Login HTTP directo al API para obtener un token valido.
  const res = await request.post("http://localhost:8080/auth/login", {
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-Slug": "demo",
    },
    data: { identifier: "admin@demo.ph.localhost", password: "admin123" },
  });
  const body = await res.json();
  const access = body.access_token as string;
  const refresh = body.refresh_token as string;

  // Cargar Flutter, esperar boot, inyectar tokens en localStorage y recargar.
  await page.goto("http://localhost:3002/");
  await page.waitForFunction(
    () => document.querySelectorAll("flt-glass-pane,flutter-view").length > 0,
    null,
    { timeout: 30000 },
  );
  await page.evaluate(
    ({ access, refresh }) => {
      // shared_preferences en web usa localStorage con prefijo 'flutter.'.
      localStorage.setItem("flutter.ph.access_token", `"${access}"`);
      localStorage.setItem("flutter.ph.refresh_token", `"${refresh}"`);
    },
    { access, refresh },
  );
  await page.reload();
  await page.waitForFunction(
    () => document.querySelectorAll("flt-glass-pane,flutter-view").length > 0,
    null,
    { timeout: 30000 },
  );
  // Dejar tiempo para que Home cargue datos del API.
  await page.waitForTimeout(5000);
  await page.screenshot({
    path: path.join(OUT, "11-flutter-home.png"),
    fullPage: false,
  });
});

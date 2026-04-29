import { test } from "@playwright/test";
import path from "path";

const OUT = path.resolve(__dirname, "../../../docs/demo-screenshots");

test.use({ viewport: { width: 1440, height: 900 } });

test("capture web screens", async ({ page }) => {
  test.setTimeout(60_000);

  await page.goto("/login");
  await page.waitForSelector("button", { timeout: 5000 });
  await page.screenshot({ path: path.join(OUT, "01-login.png"), fullPage: true });

  await page.getByLabel("Correo o documento").fill("admin@demo.ph.localhost");
  await page.getByLabel("Contrasena").fill("admin123");
  await page.getByRole("button", { name: /iniciar sesion/i }).click();
  await page.waitForURL(/\/dashboard$/);
  await page.waitForSelector("h1");
  await page.screenshot({ path: path.join(OUT, "02-dashboard.png"), fullPage: true });

  await page.getByRole("link", { name: "Paquetes" }).click();
  await page.waitForURL(/\/dashboard\/packages$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "03-packages.png"), fullPage: true });

  await page.getByRole("link", { name: "Anuncios" }).click();
  await page.waitForURL(/\/dashboard\/announcements$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "04-announcements.png"), fullPage: true });

  await page.getByRole("link", { name: "Control de acceso" }).click();
  await page.waitForURL(/\/dashboard\/access-control$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "05-access-control.png"), fullPage: true });

  await page.getByRole("link", { name: "Unidades" }).click();
  await page.waitForURL(/\/dashboard\/units$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "06-units.png"), fullPage: true });

  await page.getByRole("link", { name: "Usuarios y roles" }).click();
  await page.waitForURL(/\/dashboard\/users$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "07-users-roles.png"), fullPage: true });

  await page.getByRole("link", { name: "Parqueaderos" }).click();
  await page.waitForURL(/\/dashboard\/parking$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "08-parking-placeholder.png"), fullPage: true });

  await page.getByRole("link", { name: "Finanzas" }).click();
  await page.waitForURL(/\/dashboard\/finance$/);
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(OUT, "09-finance-placeholder.png"), fullPage: true });
});

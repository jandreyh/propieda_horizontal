import { test, expect } from "@playwright/test";

test.describe("PH demo smoke", () => {
  test("login admin -> dashboard -> packages", async ({ page }) => {
    // 1. Llegamos al login (middleware redirige /dashboard a /login).
    await page.goto("/login");
    await expect(page).toHaveTitle(/Propiedad Horizontal/i);
    await expect(page.getByText("Plataforma de administracion")).toBeVisible();

    // 2. Submit con credenciales seed.
    await page.getByLabel("Correo o documento").fill("admin@demo.ph.localhost");
    await page.getByLabel("Contrasena").fill("admin123");
    await page.getByRole("button", { name: /iniciar sesion/i }).click();

    // 3. Llegamos al dashboard.
    await page.waitForURL(/\/dashboard$/, { timeout: 10_000 });
    await expect(page.getByRole("heading", { name: "Resumen" })).toBeVisible();

    // El sidebar muestra al usuario.
    await expect(page.getByText("Admin Demo")).toBeVisible();

    // 4. Stat cards presentes (datos del seed).
    await expect(page.getByText("Paquetes en porteria")).toBeVisible();
    await expect(page.getByText("Anuncios vigentes")).toBeVisible();

    // 5. El feed muestra el anuncio sembrado "Bienvenido al sistema".
    await expect(
      page.getByText("Bienvenido al sistema", { exact: true }),
    ).toBeVisible();

    // 6. Navegar a /dashboard/packages y ver el paquete sembrado.
    await page.getByRole("link", { name: "Paquetes" }).click();
    await page.waitForURL(/\/dashboard\/packages$/);
    await expect(
      page.getByRole("heading", { name: "Paquetes" }),
    ).toBeVisible();
    await expect(page.getByText("Admin Demo").first()).toBeVisible();
    await expect(page.getByText("DEMO-0001")).toBeVisible();

    // 7. Anuncios.
    await page.getByRole("link", { name: "Anuncios" }).click();
    await page.waitForURL(/\/dashboard\/announcements$/);
    await expect(
      page.getByText(
        "Este es el conjunto demo. Todo lo que ves esta servido por el backend Go real.",
      ),
    ).toBeVisible();

    // 8. Logout.
    await page.getByRole("button", { name: /cerrar sesion/i }).click();
    await page.waitForURL(/\/login/, { timeout: 10_000 });
  });
});

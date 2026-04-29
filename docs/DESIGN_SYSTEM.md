# Design System — SaaS Propiedad Horizontal

Este documento es la fuente de verdad del sistema de diseño visual de
`apps/web` y `apps/mobile`. El subagent `design-auditor` lo usa como rubrica.
Cualquier desviacion debe justificarse en un ADR de UX.

---

## 1. Stack obligatorio

| Capa | Herramienta | Notas |
|------|-------------|-------|
| Componentes web | shadcn/ui | Radix UI + Tailwind, accesible por default |
| Estilos | Tailwind CSS v4 | Solo tokens del sistema |
| Iconos | lucide-react | Set unico, NO mezclar con otros |
| Motion | framer-motion | Subtle, respetar `prefers-reduced-motion` |
| i18n | next-intl | Castellano colombiano por default, en preparado |
| Componentes mobile | React Native + native primitives | Reservado a Expo SDK 55 |

**Prohibido**: Material UI, Ant Design, Chakra, otros sistemas mezclados con shadcn.

---

## 2. Tokens

### Colores (semanticos)

Definidos en `apps/web/app/globals.css` como variables CSS:

```css
:root {
  /* Brand — azul institucional + acento calido */
  --primary:        220 90% 45%;    /* #1E5BD9 — confianza, autoridad */
  --primary-fg:     0 0% 100%;
  --accent:         32 95% 50%;     /* #F59E0B — calor, acciones positivas */
  --accent-fg:      0 0% 0%;

  /* Neutrales */
  --background:     0 0% 100%;
  --foreground:     220 15% 12%;
  --muted:          220 15% 96%;
  --muted-fg:       220 8% 40%;
  --border:         220 15% 88%;

  /* Estado */
  --success:        145 65% 40%;    /* #2D9E5F */
  --warning:        38 95% 50%;     /* #F5A623 */
  --destructive:    0 75% 50%;      /* #DC2626 */
  --info:           210 90% 50%;    /* #2196F3 */

  /* Superficies por elevacion */
  --surface-0:      var(--background);
  --surface-1:      0 0% 98%;
  --surface-2:      0 0% 96%;

  --radius:         0.5rem;
  --radius-sm:      0.25rem;
  --radius-lg:      1rem;
  --radius-full:    9999px;
}

.dark {
  --background:     220 18% 10%;
  --foreground:     0 0% 96%;
  --muted:          220 12% 18%;
  --muted-fg:       220 8% 65%;
  --border:         220 12% 22%;
  /* primary y accent se mantienen, ajustando luminosidad si necesario */
}
```

**Reglas**:
- NUNCA hardcodear `#hex` en componentes; usar `bg-primary`, `text-foreground`.
- Contraste minimo 4.5:1 texto normal, 3:1 texto >=18pt o componentes interactivos.
- Validar contraste con axe-core.

### Tipografia

```css
--font-sans: 'Inter Variable', system-ui, -apple-system, sans-serif;
--font-mono: 'JetBrains Mono Variable', ui-monospace, monospace;

--fs-xs:   0.75rem;   /* 12px — captions */
--fs-sm:   0.875rem;  /* 14px — secondary */
--fs-base: 1rem;      /* 16px — body */
--fs-lg:   1.125rem;  /* 18px — body emphasis */
--fs-xl:   1.25rem;   /* 20px — h4 */
--fs-2xl:  1.5rem;    /* 24px — h3 */
--fs-3xl:  1.875rem;  /* 30px — h2 */
--fs-4xl:  2.25rem;   /* 36px — h1 */
--fs-5xl:  3rem;      /* 48px — display */

--fw-regular: 400;
--fw-medium:  500;
--fw-semibold: 600;
--fw-bold:    700;

--lh-tight:   1.2;
--lh-normal:  1.5;
--lh-relaxed: 1.7;
```

**Reglas**:
- Body en 16px minimo. Captions 12px solo para metadata no critica.
- Una sola fuente sans en toda la app (Inter). Mono solo para codigo/IDs.
- Line-height 1.5 para body, 1.2 para headings.

### Spacing

Escala base `0.25rem` (4px). Tailwind default.

```
0   1     2     3     4    5    6    8    10   12    16    20    24
0   4px   8px   12px  16px 20px 24px 32px 40px 48px  64px  80px  96px
```

**Reglas**:
- Densidad estandar: gap-4 entre items, p-6 en cards, py-12 entre secciones.
- Densidad compacta (tablas, listados): gap-2, py-3.
- Touch targets minimo 44x44px (2.75rem).

### Breakpoints

```
sm:  640px   /* phone landscape */
md:  768px   /* tablet portrait */
lg:  1024px  /* tablet landscape, laptop pequena */
xl:  1280px  /* desktop estandar */
2xl: 1536px  /* desktop grande */
```

**Mobile-first siempre**. La web debe ser usable en 360px sin scroll horizontal.

### Sombras (elevacion)

```
shadow-sm:   subtle, inputs y botones disabled
shadow:      cards estandar
shadow-md:   modales, popovers
shadow-lg:   sheets, drawers
shadow-xl:   raras (alertas globales)
```

### Motion

```css
--duration-fast:    150ms;  /* hover, focus */
--duration-normal:  250ms;  /* enter/exit */
--duration-slow:    400ms;  /* page transitions */

--ease-out:  cubic-bezier(0.16, 1, 0.3, 1);
--ease-inout: cubic-bezier(0.4, 0, 0.2, 1);
```

**Reglas**:
- Respetar `prefers-reduced-motion: reduce` → reducir o eliminar.
- Transformaciones (translate, scale) preferidas sobre layout properties.
- Sin animaciones >5s sin pausa.

---

## 3. Componentes obligatorios (de shadcn)

Para cada caso de uso UI, usar el componente shadcn correspondiente. NO
inventar custom cuando existe.

| Caso de uso | Componente |
|-------------|------------|
| Boton accion | `Button` con `variant=default\|outline\|ghost\|destructive` |
| Form input | `Input`, `Textarea`, `Select`, `Checkbox`, `RadioGroup`, `Switch` |
| Form validacion | `Form` (react-hook-form + zod adapter) |
| Card | `Card`, `CardHeader`, `CardContent`, `CardFooter` |
| Tabla | `Table` + `tanstack/react-table` |
| Modal | `Dialog` |
| Slide panel | `Sheet` |
| Confirmacion | `AlertDialog` |
| Tooltip | `Tooltip` |
| Popover | `Popover` |
| Dropdown menu | `DropdownMenu` |
| Toast / notif | `Toast` (sonner) |
| Loading | `Skeleton` para placeholders, `Spinner` para inline |
| Tabs | `Tabs` |
| Acordeon | `Accordion` |
| Empty state | Composicion: icono lucide + heading + descripcion + CTA |
| Avatar | `Avatar` |
| Badge | `Badge` con variants |
| Breadcrumb | `Breadcrumb` |
| Date picker | `Calendar` + `Popover` |
| File upload | `Input type=file` styled + drop zone custom minimo |

---

## 4. Voz y lenguaje

- **Tono**: Formal-amigable. Castellano colombiano. Sin tutearte ("usted").
- **Verbos**: Concretos, en infinitivo para botones ("Agregar unidad", no "Agregar nueva unidad").
- **Errores**: Constructivos. Explicar que paso + como arreglarlo. Sin "Error:".
- **Vacios**: Indicar siguiente accion ("Aun no hay anuncios. **Crear el primero**.").
- **Confirmaciones destructivas**: Doble click NO. Modal con texto del recurso a borrar.
- **Numericos**: Formato es-CO. Decimales con coma. Miles con punto.
- **Fechas**: `28 abr 2026`. Hora 24h. Zona America/Bogota.
- **Moneda**: `$ 1.250.000` (peso colombiano, sin decimales).

---

## 5. Patrones de pantalla

### Login
- Centrada, una sola columna en mobile, dos columnas en desktop (form + ilustracion/branding).
- Inputs con label encima (NO floating label).
- "Olvido su contraseña" como link secundario debajo del CTA primario.
- MFA en pantalla siguiente, con timer de validez del codigo.

### Dashboard
- Sidebar fija en desktop (lg+). En mobile: hamburguesa → Sheet.
- Top bar con: search, notificaciones, avatar+menu.
- Grid de cards de KPI en home (anuncios pendientes, paquetes hoy, visitantes hoy).
- "Acciones rapidas" prominentes para el rol (admin: crear anuncio; portero: registrar paquete).

### Listados (unidades, residentes, paquetes)
- Header con titulo, search, filtros (Sheet en mobile, popover en desktop), CTA primario.
- Tabla en desktop. Cards apiladas en mobile.
- Paginacion debajo. Sin scroll infinito (es B2B, no feed social).
- Empty state si vacio.

### Detalle
- Breadcrumb arriba.
- Tabs si hay subsecciones (datos, historial, permisos).
- Acciones primarias arriba derecha. Acciones destructivas en menu (`...`).

### Forms
- Una columna en mobile. Dos columnas en desktop si tiene sentido.
- Validacion en blur + on submit.
- Mensajes de error debajo de cada input, con `aria-describedby`.
- Boton submit deshabilitado hasta que el form sea valido.

---

## 6. Estados de componentes

Cada componente interactivo debe definir 5 estados visibles:

1. **Default**
2. **Hover** — fondo o opacidad cambia
3. **Focus-visible** — anillo azul (`ring-2 ring-primary`)
4. **Disabled** — opacity-50 + cursor-not-allowed
5. **Loading** — Spinner inline + disabled

Forms agregan: **Error** (border-destructive) y **Success** (border-success
opcional).

---

## 7. Responsive checklist

- [ ] 360px: sin scroll horizontal, todos los CTAs accesibles.
- [ ] 768px: layouts de dos columnas comienzan a aparecer.
- [ ] 1280px: layout target para desktop estandar.
- [ ] Touch targets >=44x44px en mobile.
- [ ] Imagenes con `next/image` para sizing automatico.
- [ ] Tablas: en mobile convertir a lista de cards o scroll horizontal con
      sticky first column.

---

## 8. Que NO hacer

- ❌ Usar emojis en UI (solo iconos lucide).
- ❌ Animar todo (motion debe ser intencional).
- ❌ Box shadows custom; usar las del sistema.
- ❌ Border radius arbitrario; solo `--radius*`.
- ❌ Texto en imagenes (excepto logo).
- ❌ Carruseles con autoplay.
- ❌ Modales dentro de modales (re-arquitecturar).
- ❌ "Click aqui" como link text. El texto del link describe el destino.
- ❌ "Error" como mensaje. Decir QUE paso y QUE hacer.

---

## 9. Referencias

- shadcn/ui: https://ui.shadcn.com/
- Radix Primitives: https://www.radix-ui.com/primitives
- WCAG 2.2: https://www.w3.org/WAI/WCAG22/quickref/
- Inclusive Components (Heydon Pickering): patrones a11y reales.
- Refactoring UI (Adam Wathan, Steve Schoger): jerarquia visual.

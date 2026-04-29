# Principios UX — SaaS Propiedad Horizontal

Heuristicas y reglas que cualquier pantalla del producto debe respetar. El
subagent `design-auditor` y `accessibility-auditor` los usan como rubrica.

---

## 1. Heuristicas de Nielsen aplicadas al dominio

### 1.1 Visibilidad del estado del sistema
- Toast de confirmacion en TODA accion exitosa (`Anuncio creado`, `Paquete entregado`).
- Loading state en operaciones >300ms.
- Indicador de "guardando..." en forms autosave.
- Status badge visible en toda entidad con estado (paquete: pendiente/entregado, visita: prearegistrada/ingreso/salida).

### 1.2 Coincidencia con el mundo real
- Lenguaje de propiedad horizontal: `unidad` (no apartamento generico), `residente` (no usuario), `portero` (no security guard), `administracion`, `comite`, `asamblea`.
- Iconografia esperada: 📦 paquete, 🚪 acceso, 📢 anuncio, 🔧 mantenimiento (NO traducciones literales del ingles).

### 1.3 Control y libertad del usuario
- Cancelar SIEMPRE disponible en forms multistep.
- Undo en operaciones reversibles (toast con boton "Deshacer" 10s).
- Confirmacion modal en destructivas con nombre del recurso.

### 1.4 Consistencia y estandares
- Botones primarios siempre a la derecha en desktop, full-width en mobile.
- "Cancelar" a la izquierda (o como link), "Confirmar" a la derecha (boton).
- Iconos en posicion consistente respecto al texto.
- Etiquetas de form siempre encima del input.

### 1.5 Prevencion de errores
- Inputs con tipo correcto (`type="email"`, `inputmode="numeric"`).
- Mascaras para telefono, cedula, placa.
- Validacion en blur (no solo en submit).
- Botones destructivos requieren confirmacion.
- Disabled hasta que el input sea valido (o mostrar inline lo que falta).

### 1.6 Reconocimiento sobre recordar
- Search en listados largos.
- Breadcrumbs en navegacion profunda.
- Filtros que se mantienen en URL (deep-linkable).
- Recientes / favoritos donde aplica.

### 1.7 Flexibilidad y eficiencia
- Atajos de teclado: `/` focus search, `Esc` cierra modales, `Enter` confirma.
- Bulk actions en tablas (seleccionar multiples, accion unica).
- Filtros guardados (futuro).

### 1.8 Diseño estetico y minimalista
- Una sola accion primaria por pantalla.
- Texto enfocado: NO instrucciones obvias.
- Espacio en blanco generoso (`p-6` minimo en cards).

### 1.9 Ayuda al usuario a reconocer, diagnosticar y recuperarse de errores
- Mensaje: que paso + porque + como arreglarlo.
- Errores HTTP en formato RFC 7807 mapeados a mensajes amigables.
- Errores 5xx muestran "Vuelva a intentar" + boton retry, no stack trace.

### 1.10 Ayuda y documentacion
- Tooltips en campos no obvios (uso `Info` icono lucide + Popover).
- Empty states con texto explicativo + link a help (cuando exista docs).

---

## 2. Reglas de accesibilidad WCAG 2.2 AA (resumen operativo)

### 2.1 Perceptible
- Texto alternativo en imagenes informativas.
- Contraste >=4.5:1 (texto normal), >=3:1 (texto >=18pt o UI components).
- Texto puede escalarse a 200% sin perdida.
- No usar solo color para transmitir info (icono + texto).

### 2.2 Operable
- Todo accesible por teclado.
- Focus visible siempre.
- No keyboard trap.
- Tiempos ajustables o sin limite (excepto OTP MFA con timer claro).
- Sin contenido que parpadee >3 veces/seg.

### 2.3 Comprensible
- Idioma declarado (`<html lang="es-CO">`).
- Errores claros y constructivos.
- Forms con labels asociadas (`for`/`id` o `aria-label`).

### 2.4 Robusto
- HTML valido.
- ARIA solo cuando no hay alternativa semantica.
- Funciona con tecnologias asistivas (NVDA, JAWS, VoiceOver).

---

## 3. Multi-tenant UX

### 3.1 Resolucion del tenant
- Subdominio en web: `conjunto-las-palmas.<dominio>.com`.
- Header de marca del tenant arriba (logo + nombre).
- NO mostrar dropdown "select your tenant" — el subdominio implica el tenant.

### 3.2 Personalizacion limitada por tenant
- Permitir custom logo + color primario via `tenant_config` branding.
- NO permitir custom CSS (compromete consistencia y a11y).
- Custom welcome message en login.

### 3.3 Aislamiento visible
- NUNCA mostrar datos de otros tenants (obvio, pero verificar en QA).
- Si admin de tenant intenta acceder a recurso de otro: 404, no 403 (no leak).

---

## 4. Roles y sus pantallas

| Rol | Pantallas principales | Acciones criticas |
|-----|------------------------|-------------------|
| `platform_superadmin` | Lista de tenants, audit logs central, impersonation | Crear tenant, suspender, ver billing |
| `tenant_admin` | Dashboard, unidades, residentes, anuncios, settings, branding | Crear roles, modificar permisos |
| `accountant` | Cargos, pagos, reportes financieros (post-MVP) | Generar facturas, conciliar |
| `guard` (portero) | Visitantes, paquetes, blacklist, registrar entrada/salida | Validar QR, registrar paquete |
| `resident` | Anuncios, paquetes propios, visitantes pre-registrados | Pre-registrar visitante, marcar leido |

Cada rol tiene un dashboard distinto. El menu lateral solo muestra modulos
permitidos por sus scopes.

---

## 5. Patrones de error frecuentes a EVITAR

| ❌ NO hacer | ✅ Hacer |
|------------|---------|
| "Error: Algo salio mal" | "No pudimos guardar. Verifique su conexion y vuelva a intentar." |
| Spinner sin contexto | Skeleton del layout final + texto "Cargando residentes..." |
| Modal con tres botones | Modal con UNA accion primaria + cancelar |
| Tabla sin paginacion ni search | Paginacion server-side desde 50 filas + search |
| Iconos sin label | Icono + tooltip (`aria-label`) |
| "Click aqui" | Texto del link describe el destino |
| Forms sin autosave en multistep | Autosave + indicador visible |
| Confirmacion solo "OK/Cancel" en destructiva | Texto del recurso + boton destructive con texto especifico ("Eliminar unidad #401") |

---

## 6. Performance percibida

- First Contentful Paint <1.5s en 4G.
- Skeleton screens, NO spinners en blanco.
- Optimistic updates donde sea seguro (toggle, marcar leido).
- Pre-cargar datos predecibles (next page de tabla, dashboard widgets).
- Imagenes con `next/image` y placeholders blur.

---

## 7. Que reporta el design-auditor (anclaje)

Cuando el agente audita, valida que cada pagina cumpla:

1. ✅ Una sola jerarquia clara (un h1, max un CTA primario).
2. ✅ Solo tokens del DESIGN_SYSTEM.
3. ✅ Solo componentes shadcn (no custom inventado).
4. ✅ Solo iconos lucide.
5. ✅ axe-core sin violaciones criticas.
6. ✅ Estados visibles (default/hover/focus/disabled/loading/error).
7. ✅ Responsive 360/768/1280 funcional.
8. ✅ Lenguaje del DICCIONARIO_DOMINIO.
9. ✅ Empty states con CTA.
10. ✅ Errores constructivos, accesibles via screen reader.

Si alguna falla: score < 7 en esa dimension, listar issue concreto.

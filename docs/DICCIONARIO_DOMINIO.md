# Diccionario de Dominio — SaaS Propiedad Horizontal

Glosario obligatorio de terminos. Aplica a:
- Codigo (entidades, variables, comentarios).
- UI labels y mensajes (`apps/web`, `apps/mobile`).
- Documentacion y specs.
- Comunicacion entre humanos del equipo.

Si un termino no esta aqui, es porque NO debe usarse o esta pendiente.

---

## 1. Entidades

| Termino interno (codigo) | Label UI (es-CO) | Descripcion |
|--------------------------|------------------|-------------|
| `tenant` | "Conjunto" | Una unidad de cliente del SaaS = un conjunto residencial con su propia DB |
| `unit` | "Unidad" | Apartamento, casa, local, oficina dentro de un tenant. Tiene tipo y numero |
| `tower` | "Torre" | Agrupacion fisica de unidades verticales |
| `block` | "Bloque" | Agrupacion fisica horizontal o conjunto de torres |
| `stage` | "Etapa" | Fase constructiva (Etapa 1, 2, etc.) |
| `person` | "Persona" | Persona fisica registrada en el sistema |
| `resident` | "Residente" | Persona vinculada operativamente a una unidad (propietario, inquilino, autorizado) |
| `owner` | "Propietario" | Tipo de relacion residente-unidad |
| `tenant_occupant` | "Inquilino" | Tipo de relacion residente-unidad (NO confundir con `tenant`) |
| `authorized` | "Autorizado" | Persona que el residente autoriza a habitar/visitar |
| `vehicle` | "Vehiculo" | Carro, moto, bici asignada a una persona |
| `visitor` | "Visitante" | Persona externa que ingresa al conjunto |
| `package` | "Paquete" | Correspondencia recibida en porteria para un residente |
| `announcement` | "Anuncio" | Comunicado del administrador o comite a residentes |
| `audience` | "Audiencia" | Grupo destinatario de un anuncio (todos, una torre, un rol) |
| `acknowledgement` | "Confirmacion de lectura" | Evidencia de que un residente leyo un anuncio |
| `incident` | "Incidente" | Evento de seguridad reportado |
| `pqrs` | "PQRS" | Peticion, queja, reclamo o sugerencia (terminologia legal CO) |
| `assembly` | "Asamblea" | Reunion deliberativa de propietarios |
| `vote` | "Votacion" | Acto eleccionario en asamblea |
| `parking_spot` | "Parqueadero" | Celda de parqueo asignable |
| `reservation` | "Reserva" | Booking de zona comun |
| `common_area` | "Zona comun" | Salon social, BBQ, piscina, etc. |
| `fee` | "Cuota" | Cargo recurrente o extraordinario al propietario |
| `payment` | "Pago" | Transferencia o pago en efectivo registrado |
| `penalty` | "Multa" | Sancion economica por incumplimiento de manual de convivencia |
| `audit_log` | "Auditoria" | Registro append-only de acciones criticas |

---

## 2. Roles

| Termino interno (codigo) | Label UI | Descripcion |
|--------------------------|----------|-------------|
| `platform_superadmin` | "Superadmin Plataforma" | Identidad global central. Soporte y operacion del SaaS |
| `tenant_admin` | "Administrador" | Maximo rol del conjunto. Crea roles, configura modulos |
| `accountant` | "Contador" | Acceso a modulo financiero |
| `guard` | "Portero" | Acceso a porteria (paquetes, visitantes, blacklist) |
| `committee_member` | "Miembro de Comite" | Acceso a anuncios, asambleas |
| `resident` | "Residente" | Rol estandar para habitantes |
| `family_member` | "Familiar" | Sub-rol de residente con scope limitado |

---

## 3. Estados (status enums)

### Paquetes
- `pending`: "Pendiente" — registrado, aun no entregado
- `delivered`: "Entregado"
- `returned`: "Devuelto" — el residente rechaza
- `archived`: "Archivado"

### Visitantes
- `pre_registered`: "Pre-registrado"
- `checked_in`: "Ingreso registrado"
- `checked_out`: "Salida registrada"
- `denied`: "Acceso denegado"
- `expired`: "Pre-registro vencido"

### Anuncios
- `draft`: "Borrador"
- `scheduled`: "Programado"
- `published`: "Publicado"
- `archived`: "Archivado"

### Unidades
- `active`: "Activa"
- `inactive`: "Inactiva"
- `under_construction`: "En construccion"

### Reservas (post-MVP)
- `requested`: "Solicitada"
- `confirmed`: "Confirmada"
- `cancelled`: "Cancelada"
- `no_show`: "No asistio"

---

## 4. Acciones (verbos en UI)

| Verbo interno | UI |
|---------------|-----|
| `create` | "Crear" / "Agregar" |
| `update` | "Editar" / "Actualizar" |
| `delete` | "Eliminar" (soft) |
| `archive` | "Archivar" |
| `restore` | "Restaurar" |
| `publish` | "Publicar" |
| `schedule` | "Programar" |
| `register_entry` | "Registrar entrada" |
| `register_exit` | "Registrar salida" |
| `acknowledge` | "Marcar como leido" |
| `pre_register` | "Pre-registrar" |
| `assign` | "Asignar" |
| `unassign` | "Quitar asignacion" |
| `revoke` | "Revocar" |
| `approve` | "Aprobar" |
| `reject` | "Rechazar" |

---

## 5. Conceptos transversales

| Termino interno | Label UI | Notas |
|-----------------|----------|-------|
| `MFA` | "Verificacion en dos pasos" | Nunca "MFA" en UI |
| `OTP` | "Codigo de verificacion" | Nunca "OTP" |
| `JWT` | (interno) | Nunca aparece en UI |
| `idempotency_key` | (interno) | Header HTTP, no UI |
| `version` (col) | (interno) | Columna de optimistic lock |
| `soft_delete` | "Eliminar" | UI no distingue de hard delete |
| `RBAC` | (interno) | UI dice "Roles y permisos" |
| `scope` | "Alcance" | "Alcance de permisos" |

---

## 6. Formato de datos en UI

### Cedula
- Formato: `1.234.567.890`
- Input mask: separador con punto cada 3 digitos.

### Telefono
- Formato: `+57 300 123 4567`
- Input mask: prefijo +57 fijo (CO).

### Placa de vehiculo
- Formato carro: `ABC123` (3 letras + 3 numeros).
- Formato moto: `ABC12D` (3 letras + 2 numeros + 1 letra).

### Fecha
- Formato corto: `28 abr 2026`
- Formato largo: `28 de abril de 2026`
- Formato con hora: `28 abr 2026, 14:30` (24h)
- Zona: `America/Bogota` (UTC-5)

### Moneda
- Formato: `$ 1.250.000`
- Sin decimales para COP (peso colombiano).
- Decimales con coma si aplica: `1.234.567,89`.

### Numero
- Miles con punto, decimales con coma: `1.234,56`.

---

## 7. Errores comunes a evitar

❌ "Apartamento" → ✅ "Unidad" (puede ser casa, local, oficina).
❌ "Usuario" → ✅ "Residente" / "Persona" segun contexto.
❌ "Login" → ✅ "Iniciar sesion".
❌ "Logout" → ✅ "Cerrar sesion".
❌ "Email" en UI → ✅ "Correo".
❌ "Phone" → ✅ "Telefono".
❌ "Settings" → ✅ "Configuracion".
❌ "Profile" → ✅ "Perfil".
❌ "Search" → ✅ "Buscar".
❌ "Submit" → ✅ "Enviar" / "Guardar" / "Crear" segun contexto.
❌ Tutear ("haz click", "tu cuenta") → ✅ Usted ("haga click", "su cuenta").
❌ "Tenant" en UI → ✅ "Conjunto".
❌ "Multi-tenant" en UI → (no aparece, es interno).

# Specs Frozen — Fases POST-MVP

Cada archivo en este directorio es la spec consolidada de una fase POST-MVP
(8-15) producto del proceso de Discovery (`/descubrir <N>`).

## Estados

- **Borrador**: Discovery termino, falta validacion del usuario.
- **Frozen**: Validada por el usuario. Es la fuente de verdad para `/fase <N>`.
- **Superseded**: Reemplazada por una version posterior. Mantener archivo
  como historico, indicar archivo sucesor.

## Reglas

- Una spec Frozen NO se modifica sin pasar por re-discovery (parcial o total).
- Si un cambio es minimo (typo, ajuste de campo): editar directamente y
  bumpear version en el header (`v1.1`, `v1.2`, etc.).
- Si un cambio cambia decisiones de negocio: re-discovery con `/descubrir <N>`,
  archivar la version vieja, generar nueva.

## Plantilla canonica

```markdown
# Fase <N> — Spec — <Nombre del modulo>

**Estado**: Frozen
**Validado por**: <usuario>
**Fecha de freeze**: <YYYY-MM-DD>
**Version**: 1.0

## 1. Resumen ejecutivo
## 2. Decisiones tomadas
## 3. Supuestos adoptados (no bloqueantes)
## 4. Open Questions
## 5. Modelo de datos propuesto
## 6. Endpoints
## 7. Permisos nuevos a registrar
## 8. Casos extremos (edge cases)
## 9. Operaciones transaccionales / idempotentes
## 10. Configuracion por tenant
## 11. Notificaciones / eventos
## 12. Reportes / metricas
## 13. Riesgos y mitigaciones
## 14. Multi-agente sugerido
## 15. DoD adicional especifico de la fase
```

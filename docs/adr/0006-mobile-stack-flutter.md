# ADR 0006 — Stack movil: Flutter en lugar de Expo

**Estado**: Accepted
**Fecha**: 2026-04-29
**Reemplaza**: la fila "Movil" de la tabla de stack en CLAUDE.md (Expo SDK 55 + RN 0.83)

## Contexto

El MVP scaffolded inicialmente la app movil con Expo SDK 55 + React Native 0.83 + TypeScript (commit `6ba42a1`). Esa eleccion fue razonable cuando el unico target visible era iOS/Android sobre el ecosistema JS, y reusaba conocimiento del stack web Next.js.

Al construir el demo funcional del MVP el usuario pidio explicitamente migrar a Flutter por los siguientes motivos:

1. UI nativa (Skia/CanvasKit) con mejor control fino sobre componentes y animaciones, sin la "RN bridge tax" en interacciones criticas (porteria, scanner QR de paquetes en futuro).
2. Un solo lenguaje + framework para web/Android/iOS sin matrices de dependencias divergentes (Expo vs RN bare, Hermes vs JSC).
3. Tooling AOT nativo (Dart -> AOT) y tree-shaking agresivo de iconos (verificado: 99.5% reduccion en este proyecto), lo que importa cuando se distribuye a residentes con planes de datos limitados.

## Decision

- El target movil oficial pasa a ser **Flutter 3.27+ con Dart 3.6+**.
- El proyecto Expo previo se preserva en `apps/mobile_expo_legacy/` como historia, sin desarrollarse mas. Se eliminara cuando la app Flutter haya cubierto feature parity y haya pasado piloto con un conjunto real.
- En `apps/mobile/` vive el proyecto Flutter. Plataformas habilitadas en este snapshot: `web` (validable sin Android Studio). Android e iOS se habilitan en el momento que haya equipo o CI con Android SDK / Xcode (`flutter create . --platforms=android,ios`).
- Convenciones:
  - Material 3 con `seedColor: 0xFF4F46E5` para alinear con el indigo del web.
  - HTTP via `package:http` con un wrapper `lib/api/client.dart` que injecta `X-Tenant-Slug` (default `demo`) y `Authorization: Bearer <jwt>`.
  - Persistencia de tokens via `shared_preferences` (web -> localStorage; Android/iOS -> KeyStore/Keychain). En produccion se reemplazara por `flutter_secure_storage` cuando se habilite Android/iOS.
  - Internacionalizacion via `package:intl` con locale `es_CO`.

## Consecuencias

### Positivas
- Frontend movil con codigo unificado para web/Android/iOS sin maps de polyfills RN.
- Compilacion AOT mas predecible que Metro/Hermes para perfiles de baja gama (operadores de porteria con telefonos viejos del conjunto).
- Tree-shaking de iconos y assets significativo (medicion en este snapshot: 99.5% en MaterialIcons + CupertinoIcons).
- El equipo gana experiencia en un stack que se puede portar a desktop (Windows/macOS) si en el futuro la administracion del conjunto quiere una app pesada de oficina.

### Negativas
- Quien sabe React/TS (el web esta en Next.js 16) no comparte codigo directo con la movil. La integracion compartida es solo el JSON del API.
- Para compilar Android/iOS hace falta tooling adicional (Android Studio + Xcode). Esto NO se incluye en el devcontainer Linux por tamano (~10 GB extra). Se documenta en el README mobile.
- El bundle de Flutter Web es mas grande que un Next.js equivalente; se acepta porque la app movil web es un canal secundario, no el primario.

## Alternativas consideradas

| Alternativa | Por que se descarto |
|-------------|---------------------|
| Mantener Expo SDK 55 + RN 0.83 | El usuario lo descarto explicitamente; ver Contexto. |
| KMP/Compose Multiplatform | Stack JVM-centrico; el equipo no tiene experiencia y aporta dependencias Gradle pesadas. |
| Capacitor + Next.js | Permitiria reusar el web, pero el rendimiento en interacciones rapidas (scanner QR, captura de fotos) es inferior. |
| Native iOS + Native Android separados | Doble esfuerzo, doble codigo, doble equipo. No es viable para una iniciativa con un solo desarrollador. |

## Implicaciones tecnicas concretas

- Estructura de `apps/mobile/`:

  ```
  apps/mobile/
    pubspec.yaml
    web/                    # entrypoint web (index.html, manifest.json)
    lib/
      main.dart
      api/client.dart       # ApiClient con tenant header + bearer
      screens/
        login_screen.dart
        home_screen.dart    # paquetes, visitas activas, anuncios
    test/
  ```

- Comandos:
  ```bash
  # Validar codigo
  flutter analyze
  # Compilar para web (Chrome)
  flutter build web --release
  # Servir local
  flutter run -d chrome --web-port 3001
  # Habilitar Android (cuando este Android Studio instalado)
  flutter create . --platforms=android
  flutter build apk --release
  ```

- Variables de build:
  ```bash
  flutter run -d chrome \
    --dart-define=API_BASE_URL=http://localhost:8080 \
    --dart-define=TENANT_SLUG=demo
  ```

- CI futuro:
  - `flutter analyze` reemplaza `pnpm lint` en mobile.
  - `flutter test` reemplaza `pnpm test` en mobile.
  - Build de release web va a `apps/mobile/build/web/` (servible como estatico).

## Tareas de seguimiento

- [ ] Habilitar plataformas Android e iOS cuando haya entorno (no en este snapshot).
- [ ] Cambiar `shared_preferences` a `flutter_secure_storage` en mobile cuando se habilite Android/iOS.
- [ ] Implementar pantallas faltantes con feature parity al dashboard web (units, vehicles, blacklist).
- [ ] Decidir si se publica un APK firmado (Play Console) o solo se distribuye el web build.
- [ ] Borrar `apps/mobile_expo_legacy/` cuando la app Flutter cubra feature parity y haya pasado piloto.

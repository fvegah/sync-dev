# Roadmap: SyncDev UX Improvements

**Milestone:** v1.1 - UX Improvements
**Created:** 2026-01-22
**Status:** Planning

## Milestone Goal

Transformar SyncDev de una app funcional pero básica a una aplicación de sincronización profesional con:
- Presencia en la barra de menú de macOS (system tray)
- Visibilidad completa del progreso de sincronización
- Interfaz nativa que se sienta como una app de Apple
- Almacenamiento seguro de secrets en Keychain

## Success Criteria

Al completar este milestone:
1. El usuario puede cerrar la ventana y la app sigue corriendo en la barra de menú
2. El ícono de la barra de menú indica claramente el estado (idle, sincronizando, error)
3. El usuario ve progreso detallado: global, por archivo, velocidad y tiempo restante
4. La UI se siente nativa de macOS (como Finder o System Preferences)
5. Los shared secrets están almacenados de forma segura en Keychain

## Phases

### Phase 1: Keychain Security
**Goal:** Migrar almacenamiento de secrets de texto plano a macOS Keychain

**Requirements:**
- [ ] SEC-01: Almacenar shared secrets en macOS Keychain

**Approach:**
- Usar zalando/go-keyring para acceso a Keychain sin CGo
- Migrar secrets existentes en primer inicio
- Mantener backward compatibility con config.json para otros datos

**Key Files:**
- `internal/secrets/keychain.go` - Nuevo: wrapper de Keychain
- `internal/config/store.go` - Integrar keyring + migration
- `internal/sync/engine.go` - Usar keychain para secrets

**Plans:** 2 plans

Plans:
- [ ] 01-01-PLAN.md — Create keychain manager, tests, and config integration with migration
- [ ] 01-02-PLAN.md — Update sync engine to use keychain for all secret operations

**Risks:**
- Apps sin code signing muestran prompts de Keychain - documentar para desarrollo

**Success Criteria:**
- [ ] Secrets se guardan en Keychain, no en config.json
- [ ] Migración automática de secrets existentes
- [ ] App funciona sin prompts molestos de Keychain

---

### Phase 2: Menu Bar Integration
**Goal:** App vive en la barra de menú de macOS como una app de sincronización profesional

**Requirements:**
- [ ] TRAY-01: App se minimiza a system tray al cerrar ventana
- [ ] TRAY-02: Menú contextual (Sync ahora, Pausar, Abrir, Salir)
- [ ] TRAY-03: Ícono cambia según estado (idle, sincronizando, error)

**Approach:**
- Investigar Wails v3 para system tray nativo vs NSStatusItem via cgo
- Template images para íconos (22pt max height, 16x16pt para circular)
- NSMenu para menú contextual

**Key Files:**
- `systray.go` - Actualmente deshabilitado, reescribir
- `app.go` - Manejar cierre de ventana sin terminar app
- `main.go` - Configurar comportamiento de ventana

**Risks:**
- Wails v2 + fyne.io/systray tiene conflictos conocidos
- Posible necesidad de migrar a Wails v3

**Success Criteria:**
- [ ] Cerrar ventana minimiza a barra de menú
- [ ] Ícono muestra 3 estados: idle (check), syncing (flechas animadas), error (!)
- [ ] Menú tiene: "Sync Now", "Pause/Resume", "Open SyncDev", separador, "Quit"

---

### Phase 3: Progress Display
**Goal:** Usuario tiene visibilidad completa del progreso de sincronización

**Requirements:**
- [ ] PROG-01: Barra de progreso global (% total)
- [ ] PROG-02: Barra de progreso por archivo individual
- [ ] PROG-03: Velocidad (MB/s) y tiempo estimado restante
- [ ] PROG-04: Lista en tiempo real de archivos sincronizándose

**Approach:**
- Throttling de callbacks a 10-20 Hz para evitar freeze de UI
- Exponential smoothing para ETA estable (evitar saltos)
- Limitar lista de archivos a 100 items máximo

**Key Files:**
- `internal/sync/engine.go` - Agregar callbacks de progreso por archivo
- `internal/sync/transfer.go` - Emitir eventos de progreso
- `frontend/src/lib/SyncStatus.svelte` - Rediseñar con barras de progreso
- `frontend/src/stores/app.js` - Nuevo store para progreso

**Risks:**
- Engine actual (1,247 líneas) es monolítico, refactoring puede ser necesario
- File transfers usan base64 (ineficiente), puede afectar precisión de velocidad

**Success Criteria:**
- [ ] Barra de progreso global muestra % completado
- [ ] Barra de progreso por archivo durante transferencia
- [ ] Velocidad en MB/s actualizada cada 1-2 segundos
- [ ] ETA estable (no salta entre valores)
- [ ] Lista muestra archivos activos (max 100)

---

### Phase 4: Native macOS UI
**Goal:** Interfaz que se siente como una app nativa de Apple

**Requirements:**
- [ ] UI-01: Rediseño con estilo macOS nativo
- [ ] UI-02: Componentes siguiendo Apple Human Interface Guidelines

**Approach:**
- Upgrade a Svelte 5 + shadcn-svelte + Tailwind CSS
- SF Symbols para iconografía
- Colores del sistema (--system-blue, etc.)
- Sidebar translúcida estilo Finder
- Modo claro/oscuro siguiendo preferencias del sistema

**Key Files:**
- `frontend/src/App.svelte` - Layout con sidebar
- `frontend/src/lib/*.svelte` - Rediseñar todos los componentes
- `frontend/package.json` - Agregar dependencias UI

**Risks:**
- Svelte 3 → 5 puede requerir cambios significativos
- shadcn-svelte es relativamente nuevo, documentación limitada

**Success Criteria:**
- [ ] Sidebar estilo Finder con navegación
- [ ] Iconos SF Symbols en toda la UI
- [ ] Colores del sistema para estados (azul selección, rojo error, verde éxito)
- [ ] Modo oscuro/claro automático
- [ ] Fuente SF Pro o sistema por defecto

---

## Phase Dependencies

```
Phase 1 (Keychain) ──┐
                     ├──► Phase 3 (Progress) ──► Phase 4 (UI)
Phase 2 (Menu Bar) ──┘
```

- **Phase 1** y **Phase 2** pueden ejecutarse en paralelo (sin dependencias)
- **Phase 3** requiere Phase 2 para mostrar estado en ícono del tray
- **Phase 4** debe ser última para pulir la UI completa

## Requirement Mapping

| Req ID | Requirement | Phase |
|--------|-------------|-------|
| SEC-01 | Keychain storage | Phase 1 |
| TRAY-01 | Minimize to tray | Phase 2 |
| TRAY-02 | Context menu | Phase 2 |
| TRAY-03 | Status icon | Phase 2 |
| PROG-01 | Global progress bar | Phase 3 |
| PROG-02 | Per-file progress | Phase 3 |
| PROG-03 | Speed + ETA | Phase 3 |
| PROG-04 | File list | Phase 3 |
| UI-01 | macOS native style | Phase 4 |
| UI-02 | Apple HIG compliance | Phase 4 |

## Coverage Validation

- **SEC requirements:** 1/1 mapped (100%)
- **TRAY requirements:** 3/3 mapped (100%)
- **PROG requirements:** 4/4 mapped (100%)
- **UI requirements:** 2/2 mapped (100%)

**Total:** 10/10 requirements mapped across 4 phases

---

*Roadmap created: 2026-01-22*

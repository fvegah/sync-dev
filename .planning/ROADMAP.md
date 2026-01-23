# Roadmap: SyncDev UX Improvements

**Milestone:** v1.1 - UX Improvements
**Created:** 2026-01-22
**Status:** In Progress (Phase 4 Not Started)

## Milestone Goal

Transformar SyncDev de una app funcional pero basica a una aplicacion de sincronizacion profesional con:
- Presencia en la barra de menu de macOS (system tray)
- Visibilidad completa del progreso de sincronizacion
- Interfaz nativa que se sienta como una app de Apple
- Almacenamiento seguro de secrets en Keychain

## Success Criteria

Al completar este milestone:
1. El usuario puede cerrar la ventana y la app sigue corriendo en la barra de menu
2. El icono de la barra de menu indica claramente el estado (idle, sincronizando, error)
3. El usuario ve progreso detallado: global, por archivo, velocidad y tiempo restante
4. La UI se siente nativa de macOS (como Finder o System Preferences)
5. Los shared secrets estan almacenados de forma segura en Keychain

## Phases

### Phase 1: Keychain Security ✓
**Status:** Complete (2026-01-22)
**Goal:** Migrar almacenamiento de secrets de texto plano a macOS Keychain

**Requirements:**
- [x] SEC-01: Almacenar shared secrets en macOS Keychain

**Approach:**
- Usar zalando/go-keyring para acceso a Keychain sin CGo
- Migrar secrets existentes en primer inicio
- Mantener backward compatibility con config.json para otros datos

**Key Files:**
- `internal/secrets/keychain.go` - Manager interface y KeychainManager
- `internal/config/store.go` - Migration y GetSecrets()
- `internal/sync/engine.go` - getSecretForPeer() helper

**Plans:** 2/2 complete

Plans:
- [x] 01-01-PLAN.md — Keychain manager, tests, and config integration with migration
- [x] 01-02-PLAN.md — Engine integration with keychain for all secret operations

**Success Criteria:**
- [x] Secrets se guardan en Keychain, no en config.json
- [x] Migracion automatica de secrets existentes
- [x] App funciona sin prompts molestos de Keychain (go-keyring usa /usr/bin/security)

---

### Phase 2: Menu Bar Integration ✓
**Status:** Complete (2026-01-22)
**Goal:** App vive en la barra de menu de macOS como una app de sincronizacion profesional

**Requirements:**
- [x] TRAY-01: App se minimiza a system tray al cerrar ventana
- [x] TRAY-02: Menu contextual (Sync ahora, Pausar, Abrir, Salir)
- [x] TRAY-03: Icono cambia segun estado (idle, sincronizando, error)

**Approach:**
- Migrated to Wails v3 for native system tray support
- Template images for icons (22x22 black + alpha, macOS adapts for dark mode)
- Wails v3 native menu API for context menu

**Key Files:**
- `main.go` - Wails v3 entry with ActivationPolicyAccessory, tray.NewManager()
- `app.go` - Tray state updates, IsPaused/Pause/Resume methods
- `internal/tray/tray.go` - Manager with SetMenu, SetState, AttachWindow
- `internal/tray/icons.go` - Embedded icons via go:embed
- `internal/tray/icons/*.png` - 3 template icons (idle, syncing, error)

**Plans:** 3/3 complete

Plans:
- [x] 02-01-PLAN.md — Wails v3 Migration (window hide-on-close, remove v2)
- [x] 02-02-PLAN.md — System Tray Implementation (tray manager, context menu, icons)
- [x] 02-03-PLAN.md — Dynamic Icon States (status->tray state mapping)

**Success Criteria:**
- [x] Cerrar ventana minimiza a barra de menu
- [x] Icono muestra 3 estados: idle, syncing, error
- [x] Menu tiene: "Sync Now", "Pause/Resume", "Open SyncDev", separador, "Quit"

---

### Phase 3: Progress Display ✓
**Status:** Complete (2026-01-23)
**Goal:** Usuario tiene visibilidad completa del progreso de sincronizacion

**Requirements:**
- [x] PROG-01: Barra de progreso global (% total)
- [x] PROG-02: Barra de progreso por archivo individual
- [x] PROG-03: Velocidad (MB/s) y tiempo estimado restante
- [x] PROG-04: Lista en tiempo real de archivos sincronizandose

**Approach:**
- ProgressAggregator en Go para throttling a ~15 Hz
- Exponential smoothing (alpha=0.1) para ETA estable
- Limitar lista de archivos activos a 10 items

**Key Files:**
- `internal/models/progress.go` - AggregateProgress y FileProgress structs
- `internal/sync/progress.go` - ProgressAggregator implementation
- `internal/sync/engine.go` - Aggregator integration
- `frontend/src/stores/app.js` - Derived stores para speed/ETA
- `frontend/src/lib/SyncStatus.svelte` - Enhanced progress UI

**Plans:** 4/4 complete

Plans:
- [x] 03-01-PLAN.md — Backend progress aggregator with throttling and smoothing
- [x] 03-02-PLAN.md — Engine integration with ProgressAggregator
- [x] 03-03-PLAN.md — Frontend progress UI with speed and ETA display
- [x] 03-04-PLAN.md — Active files list and human verification

**Success Criteria:**
- [x] Barra de progreso global muestra % completado
- [x] Barra de progreso por archivo durante transferencia
- [x] Velocidad en MB/s actualizada cada 1-2 segundos
- [x] ETA estable (no salta entre valores)
- [x] Lista muestra archivos activos (max 10)

---

### Phase 4: Native macOS UI
**Goal:** Interfaz que se siente como una app nativa de Apple

**Requirements:**
- [ ] UI-01: Rediseno con estilo macOS nativo
- [ ] UI-02: Componentes siguiendo Apple Human Interface Guidelines

**Approach:**
- Upgrade a Svelte 5 + shadcn-svelte + Tailwind CSS
- SF Symbols para iconografia
- Colores del sistema (--system-blue, etc.)
- Sidebar translucida estilo Finder
- Modo claro/oscuro siguiendo preferencias del sistema

**Key Files:**
- `frontend/src/App.svelte` - Layout con sidebar
- `frontend/src/lib/*.svelte` - Redisenar todos los componentes
- `frontend/package.json` - Agregar dependencias UI

**Risks:**
- Svelte 3 -> 5 puede requerir cambios significativos
- shadcn-svelte es relativamente nuevo, documentacion limitada

**Success Criteria:**
- [ ] Sidebar estilo Finder con navegacion
- [ ] Iconos SF Symbols en toda la UI
- [ ] Colores del sistema para estados (azul seleccion, rojo error, verde exito)
- [ ] Modo oscuro/claro automatico
- [ ] Fuente SF Pro o sistema por defecto

---

## Phase Dependencies

```
Phase 1 (Keychain) ──┐
                     ├──► Phase 3 (Progress) ──► Phase 4 (UI)
Phase 2 (Menu Bar) ──┘
```

- **Phase 1** y **Phase 2** pueden ejecutarse en paralelo (sin dependencias)
- **Phase 3** requiere Phase 2 para mostrar estado en icono del tray
- **Phase 4** debe ser ultima para pulir la UI completa

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

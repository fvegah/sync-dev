# SyncDev

## What This Is

SyncDev es una aplicación de escritorio para macOS que sincroniza archivos entre computadores en una red local. Usa descubrimiento mDNS para encontrar peers automáticamente y un protocolo TCP personalizado para transferir archivos. La app está construida con Wails (Go backend + Svelte frontend).

## Core Value

**Los usuarios pueden sincronizar carpetas entre sus dispositivos en la misma red sin configuración manual de IPs o servidores.**

## Requirements

### Validated

<!-- Funcionalidad existente que ya está implementada y funcionando -->

- ✓ Descubrimiento automático de peers via mDNS — existing
- ✓ Pairing entre dispositivos con código de 6 dígitos — existing
- ✓ Sincronización bidireccional de archivos — existing
- ✓ Gestión de folder pairs (crear, editar, eliminar) — existing
- ✓ Patrones de exclusión por folder pair — existing
- ✓ Auto-sync con intervalo configurable — existing
- ✓ Protocolo TCP con HMAC para autenticación — existing
- ✓ Persistencia de configuración en JSON — existing

### Active

<!-- Scope actual — lo que vamos a construir en esta iteración -->

- [ ] **TRAY-01**: App se minimiza a system tray al cerrar ventana (sigue corriendo en background)
- [ ] **TRAY-02**: Menú contextual en ícono del tray (Sync ahora, Pausar, Abrir, Salir)
- [ ] **TRAY-03**: Ícono del tray cambia según estado (idle, sincronizando, error)
- [ ] **PROG-01**: Barra de progreso global mostrando % total de sincronización
- [ ] **PROG-02**: Barra de progreso por archivo individual durante transferencia
- [ ] **PROG-03**: Mostrar velocidad de transferencia (MB/s) y tiempo estimado restante
- [ ] **PROG-04**: Lista en tiempo real de archivos sincronizándose
- [ ] **UI-01**: Rediseño de interfaz con estilo macOS nativo (semejante a Finder, System Preferences)
- [ ] **UI-02**: Componentes consistentes siguiendo Human Interface Guidelines de Apple
- [ ] **SEC-01**: Almacenar shared secrets en macOS Keychain en lugar de texto plano

### Out of Scope

<!-- Explícitamente excluido — documentado para evitar scope creep -->

- Soporte WAN (internet) — requiere relay server o NAT traversal, planeado para v2+
- Encriptación de archivos en tránsito — HMAC provee autenticidad pero no confidencialidad
- Rate limiting en pairing — código de 6 dígitos es suficiente para LAN confiable
- Confirmación/backup antes de borrar archivos — complejidad alta para v1
- Soporte Windows/Linux — enfocados en macOS por ahora

## Context

**Codebase existente:**
- Go 1.23 backend con Wails v2.11.0
- Svelte 3.49.0 frontend con Vite
- mDNS discovery via hashicorp/mdns
- System tray actualmente deshabilitado por conflicto fyne.io/systray + Wails
- UI actual es funcional pero básica, sin estilo definido

**Tech debt identificado:**
- `internal/sync/engine.go` es monolítico (1,247 líneas)
- File transfers usan base64 encoding (ineficiente)
- Secrets en texto plano en `~/.syncdev/config.json`

**Documentación del codebase:**
- `.planning/codebase/STACK.md` — tecnologías y dependencias
- `.planning/codebase/ARCHITECTURE.md` — diseño del sistema
- `.planning/codebase/CONCERNS.md` — deuda técnica y problemas

## Constraints

- **Platform**: macOS solamente (Wails + system tray nativo)
- **Stack**: Mantener Go/Wails/Svelte — no migrar frameworks
- **Compatibilidad**: Debe seguir funcionando con peers existentes (protocolo backward compatible)
- **Keychain**: Usar Security.framework de macOS para secrets

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| System tray nativo vs fyne.io/systray | fyne.io tiene conflictos con Wails; usar solución nativa | — Pending |
| UI estilo macOS nativo | Usuario quiere que se sienta como app de Apple | — Pending |
| Solo Keychain (no cross-platform) | Enfocados en macOS; simplifica implementación | — Pending |

---
*Last updated: 2026-01-22 after initialization*

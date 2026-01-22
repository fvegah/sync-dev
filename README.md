# SyncDev

<p align="center">
  <img src="build/appicon.png" alt="SyncDev Logo" width="128" height="128">
</p>

<p align="center">
  <strong>Sincronización bidireccional de carpetas entre equipos Mac en red local</strong>
</p>

<p align="center">
  <a href="#características">Características</a> •
  <a href="#requisitos-del-sistema">Requisitos</a> •
  <a href="#instalación">Instalación</a> •
  <a href="#uso">Uso</a> •
  <a href="#configuración">Configuración</a> •
  <a href="#desarrollo">Desarrollo</a>
</p>

---

## Descripción

SyncDev es una aplicación nativa para macOS que permite sincronizar carpetas de forma bidireccional entre dos o más computadoras Mac conectadas a la misma red local (LAN). A diferencia de soluciones basadas en la nube, SyncDev opera completamente dentro de su red local, garantizando privacidad y velocidades de transferencia óptimas.

## Características

| Característica | Descripción |
|----------------|-------------|
| **Descubrimiento Automático** | Utiliza mDNS/Bonjour para detectar automáticamente otros equipos Mac ejecutando SyncDev en la red local |
| **Emparejamiento Seguro** | Sistema de códigos de 6 dígitos para autorizar la conexión entre dispositivos |
| **Sincronización Bidireccional** | Los cambios realizados en cualquier dispositivo se propagan al otro automáticamente |
| **Resolución de Conflictos** | Estrategia "Last-Write-Wins" (gana la última escritura) para resolver conflictos de edición |
| **Exclusiones Configurables** | Soporte para patrones glob para excluir archivos y carpetas de la sincronización |
| **Transferencia Segura** | Autenticación HMAC con secreto compartido para todas las transferencias |
| **Interfaz Nativa** | Aplicación nativa de macOS con interfaz moderna y soporte para modo oscuro |

## Requisitos del Sistema

### Para Usuarios
- macOS 10.15 (Catalina) o superior
- Conexión a red local (WiFi o Ethernet)
- Permisos de acceso a disco para las carpetas a sincronizar

### Para Desarrollo
- Go 1.21 o superior
- Node.js 18 o superior
- Wails CLI v2.11.0 o superior
- Xcode Command Line Tools

## Instalación

### Opción 1: Desde DMG (Recomendado)

1. Descargue el archivo `SyncDev-x.x.x.dmg` desde la sección de [Releases](https://github.com/fvegah/sync-dev/releases)
2. Abra el archivo DMG descargado
3. Arrastre `SyncDev.app` a la carpeta `Aplicaciones`
4. En el primer inicio, haga clic derecho sobre la aplicación y seleccione "Abrir" para autorizar la ejecución

### Opción 2: Desde Código Fuente

```bash
# 1. Instalar Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# 2. Clonar el repositorio
git clone https://github.com/fvegah/sync-dev.git
cd sync-dev

# 3. Compilar la aplicación
make build

# 4. Instalar en /Applications
make install
```

### Opción 3: Compilación Universal (Intel + Apple Silicon)

```bash
make build-universal
```

## Uso

### Paso 1: Iniciar SyncDev en Ambos Equipos

Ejecute SyncDev en los equipos Mac que desea sincronizar. Ambos deben estar conectados a la misma red local.

### Paso 2: Emparejar Dispositivos

1. En la pestaña **Dispositivos**, haga clic en "Generate Pairing Code" en uno de los equipos
2. En el otro equipo, localice el dispositivo descubierto y haga clic en "Pair"
3. Ingrese el código de 6 dígitos mostrado en el primer equipo
4. Una vez emparejados, verá la etiqueta "Paired" junto al dispositivo

### Paso 3: Configurar Pares de Carpetas

1. Vaya a la pestaña **Carpetas**
2. Haga clic en "Add Folder Pair"
3. Seleccione el dispositivo emparejado
4. Especifique la ruta de la carpeta local
5. Especifique la ruta correspondiente en el dispositivo remoto
6. Haga clic en "Add Folder Pair"

### Paso 4: Sincronizar

La sincronización ocurre automáticamente según el intervalo configurado (por defecto: 5 minutos). También puede:

- Hacer clic en "Sync Now" en la pestaña **Sync** para sincronización inmediata
- Usar el ícono de sincronización en cada par de carpetas individualmente

## Configuración

La configuración se almacena en `~/.syncdev/config.json`. Los ajustes disponibles en la pestaña **Settings** incluyen:

| Opción | Descripción | Valor por Defecto |
|--------|-------------|-------------------|
| Device Name | Nombre visible para otros dispositivos | Nombre del equipo |
| Sync Interval | Intervalo de sincronización automática | 5 minutos |
| Auto Sync | Habilitar/deshabilitar sincronización automática | Habilitado |
| Global Exclusions | Patrones de archivos a excluir | Ver abajo |

### Exclusiones por Defecto

```
.DS_Store
.git
.svn
node_modules
*.tmp
*.swp
*~
.Trash
Thumbs.db
```

## Inicio Automático

Para configurar SyncDev para que inicie automáticamente al iniciar sesión:

```bash
# Habilitar inicio automático
./scripts/setup-autostart.sh install

# Deshabilitar inicio automático
./scripts/setup-autostart.sh uninstall

# Verificar estado
./scripts/setup-autostart.sh status
```

## Arquitectura

```
┌─────────────────────────────────────────────────────────────┐
│                    SyncDev.app (Wails)                       │
│  ┌─────────────────┐    ┌─────────────────────────────────┐ │
│  │   Frontend      │    │      Backend (Go)               │ │
│  │   (Svelte)      │◄──►│  - Sync Engine                  │ │
│  │                 │    │  - mDNS Discovery (Bonjour)     │ │
│  │  - Config UI    │    │  - P2P File Transfer (TCP)      │ │
│  │  - Peer List    │    │  - File Index Manager           │ │
│  │  - Sync Status  │    │  - Config Manager               │ │
│  └─────────────────┘    └─────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Estructura del Proyecto

```
sync-dev/
├── main.go                     # Punto de entrada de la aplicación
├── app.go                      # Bindings Go ↔ Frontend
├── wails.json                  # Configuración de Wails
├── Makefile                    # Comandos de compilación
├── internal/
│   ├── config/
│   │   ├── config.go           # Estructuras de configuración
│   │   └── store.go            # Persistencia JSON
│   ├── models/
│   │   ├── peer.go             # Modelo de dispositivo
│   │   └── file.go             # Modelo de archivo
│   ├── network/
│   │   ├── discovery.go        # Descubrimiento mDNS/Bonjour
│   │   ├── server.go           # Servidor TCP
│   │   ├── client.go           # Cliente TCP
│   │   └── protocol.go         # Protocolo de mensajes
│   └── sync/
│       ├── engine.go           # Motor de sincronización
│       ├── index.go            # Gestión de índices
│       ├── scanner.go          # Escaneo de carpetas
│       └── transfer.go         # Transferencia de archivos
├── frontend/
│   ├── src/
│   │   ├── App.svelte          # Componente principal
│   │   ├── lib/
│   │   │   ├── PeerList.svelte
│   │   │   ├── FolderPairs.svelte
│   │   │   ├── SyncStatus.svelte
│   │   │   └── Settings.svelte
│   │   └── stores/
│   │       └── app.js          # Estado de la aplicación
│   └── package.json
├── scripts/
│   ├── create-dmg.sh           # Creación de DMG
│   ├── setup-autostart.sh      # Configuración de inicio automático
│   └── com.syncdev.agent.plist # LaunchAgent para auto-inicio
└── build/
    └── bin/                    # Aplicación compilada
```

## Desarrollo

### Comandos Disponibles

```bash
# Modo desarrollo con hot reload
make dev

# Compilar para plataforma actual
make build

# Compilar binario universal (Intel + Apple Silicon)
make build-universal

# Crear DMG para distribución
make dmg

# Instalar en /Applications
make install

# Ejecutar la aplicación compilada
make run

# Limpiar artefactos de compilación
make clean
```

### Tecnologías Utilizadas

| Componente | Tecnología |
|------------|------------|
| Backend | Go 1.21+ |
| Frontend | Svelte 3 |
| Framework de Escritorio | Wails v2 |
| Descubrimiento de Red | mDNS (Bonjour) |
| Transferencia de Archivos | TCP con autenticación HMAC |
| Empaquetado | DMG nativo de macOS |

## Solución de Problemas

### Los dispositivos no se descubren mutuamente

1. Verifique que ambos equipos están en la misma red
2. Compruebe que el firewall no está bloqueando el puerto 52525
3. Reinicie SyncDev en ambos equipos
4. Verifique que mDNS/Bonjour está habilitado en el sistema

### La sincronización no funciona

1. Verifique que los dispositivos están emparejados (etiqueta "Paired")
2. Asegúrese de que los pares de carpetas están habilitados (interruptor activado)
3. Confirme que la ruta remota existe en el otro equipo
4. Revise los permisos de acceso a las carpetas

### Problemas de permisos

En macOS, es posible que necesite otorgar a SyncDev:

1. **Acceso completo al disco**: Preferencias del Sistema → Seguridad y Privacidad → Privacidad → Acceso total al disco
2. **Acceso a red**: Acepte el diálogo cuando se le solicite al primer inicio

## Seguridad

- Las comunicaciones entre dispositivos están autenticadas mediante HMAC con un secreto compartido
- El secreto compartido se genera durante el proceso de emparejamiento
- Los datos se transmiten únicamente dentro de la red local
- No se envía información a servidores externos

## Contribuir

Las contribuciones son bienvenidas. Por favor:

1. Fork el repositorio
2. Cree una rama para su característica (`git checkout -b feature/nueva-caracteristica`)
3. Commit sus cambios (`git commit -am 'Añadir nueva característica'`)
4. Push a la rama (`git push origin feature/nueva-caracteristica`)
5. Abra un Pull Request

## Licencia

Este proyecto está licenciado bajo la Licencia MIT. Consulte el archivo [LICENSE](LICENSE) para más detalles.

## Autor

Desarrollado por [Felipe Vega](https://github.com/fvegah)

---

<p align="center">
  <sub>Hecho con ❤️ usando Go y Svelte</sub>
</p>

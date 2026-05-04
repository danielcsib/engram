[← Volver al README](../README.md)

# Guía definitiva del codebase de Engram

**Engram es un sistema local-first de memoria persistente para agentes de código.** El centro del producto es un binario Go que escribe en SQLite + FTS5; CLI, MCP, API HTTP, TUI, plugins, sync, cloud y dashboard son interfaces alrededor de ese núcleo.

Esta guía es para maintainers y contributors que necesitan entender **dónde vive cada responsabilidad, qué invariantes no se negocian y qué archivo abrir cuando hay que cambiar algo**.

---

## Modelo mental en 90 segundos

Engram existe para resolver un problema concreto: los agentes olvidan el contexto cuando termina una sesión o se compacta la conversación. Engram les da una memoria curada, buscable y portable.

```text
Agente de código
  Claude Code / OpenCode / Gemini CLI / Codex / VS Code / Antigravity / Cursor / Windsurf
        │
        │ MCP stdio, plugin hooks o API local
        ▼
cmd/engram
  CLI + runtime local + runtime cloud
        │
        ├── internal/mcp        herramientas mem_* para agentes
        ├── internal/server     JSON API local: engram serve
        ├── internal/tui        UI terminal Bubbletea
        ├── internal/setup      instalación de integraciones automatizadas
        │
        ▼
internal/store
  SQLite + FTS5 + sesiones + observaciones + prompts + relaciones + sync mutations
        │
        ├── internal/sync                   chunks git-friendly / transporte
        └── internal/cloud/autosync         push/pull background opcional
                │
                ▼
        internal/cloud/remote ── HTTP ── engram cloud serve
                                      │
                                      ├── internal/cloud/cloudserver
                                      ├── internal/cloud/cloudstore  Postgres
                                      ├── internal/cloud/dashboard   HTML/HTMX
                                      └── internal/cloud/auth        bearer + sesión dashboard
```

La frase que ordena todo el repo:

> **SQLite local es la fuente de verdad. Cloud es replicación y acceso compartido opt-in, no el dueño del dato.**

Si una decisión contradice eso, probablemente está mal ubicada o mal diseñada.

---

## Qué es Engram y qué NO es

### Qué es

| Es | Qué significa en el código |
|---|---|
| **Memoria persistente para agentes** | Los agentes guardan observaciones estructuradas con `mem_save`, `mem_session_summary`, `mem_save_prompt` y herramientas relacionadas en `internal/mcp`. |
| **Local-first** | `internal/store` persiste en SQLite; las interfaces leen/escriben ahí primero. |
| **Un binario Go** | `cmd/engram` compone store, server, MCP, TUI, setup, sync y cloud. |
| **SQLite + FTS5** | `internal/store/store.go` define sesiones, observaciones, prompts, FTS, dedupe, topic upserts y soft deletes. |
| **Agent-agnostic** | Las integraciones pasan por MCP, configuración manual de MCP o plugins finos en `plugin/`; `internal/setup` cubre solo los flujos automatizados implementados. |
| **Cloud opcional** | `engram cloud serve` expone transporte de sync, dashboard y auth, pero no reemplaza el store local. |
| **Documentado por comportamiento actual** | `DOCS.md`, `docs/ARCHITECTURE.md`, `docs/AGENT-SETUP.md`, `docs/PLUGINS.md` y `docs/engram-cloud/*` son referencias vivas. |

### Qué NO es

| No es | Por qué importa |
|---|---|
| **No es cloud-only** | Nunca diseñes una feature que requiera cloud para que la memoria local funcione. |
| **No es un recorder crudo de tool calls** | El agente decide qué vale la pena guardar; Engram no persigue un firehose indiscriminado. |
| **No es una UI que simula política** | Si un toggle cambia permisos o sync, tiene que estar enforceado en servidor/cloudstore, no solo en HTML. |
| **No es un monolito sin fronteras** | Store, server, cloudstore, cloudserver, dashboard, autosync, plugins y setup tienen ownership explícito. |
| **No es una API reference duplicada acá** | Para endpoints, schemas y parámetros completos usá [DOCS.md](../DOCS.md). Esta guía explica dónde encaja cada cosa. |

---

## Mapa rápido: si necesitás X, leé Y

| Si necesitás... | Abrí primero | Después mirá |
|---|---|---|
| Entender el producto en 5 minutos | `README.md` | `docs/ARCHITECTURE.md` |
| Ver referencia técnica completa | `DOCS.md` | Esta guía para ownership y guardrails |
| Cambiar herramientas MCP | `internal/mcp/mcp.go` | `internal/mcp/*_test.go`, `docs/AGENT-SETUP.md` |
| Cambiar persistencia local | `internal/store/store.go` | `internal/store/*_test.go`, `DOCS.md#database-schema` |
| Cambiar API local | `internal/server/server.go` | `internal/server/*_test.go`, `DOCS.md#http-api-endpoints` |
| Cambiar sync por chunks | `internal/sync/sync.go` | `internal/sync/*_test.go`, `README.md#git-sync` |
| Cambiar autosync cloud | `internal/cloud/autosync/manager.go` | `internal/cloud/remote/transport.go`, `DOCS.md#cloud-autosync` |
| Cambiar transporte cloud | `internal/cloud/cloudserver/cloudserver.go` | `internal/cloud/cloudstore/cloudstore.go`, `docs/engram-cloud/README.md` |
| Cambiar dashboard | `internal/cloud/dashboard/dashboard.go` | `internal/cloud/dashboard/*_templ.go`, `internal/cloud/dashboard/static/styles.css` |
| Cambiar auth cloud/dashboard | `internal/cloud/auth/auth.go` | `cmd/engram/cloud.go`, `internal/cloud/cloudserver/cloudserver.go` |
| Cambiar setup de agentes | `internal/setup/setup.go` | `plugin/`, `docs/AGENT-SETUP.md`, `docs/PLUGINS.md` |
| Cambiar detección de proyecto | `internal/project/detect.go` | `internal/project/similar.go`, `docs/AGENT-SETUP.md` |
| Cambiar TUI | `internal/tui/model.go` | `internal/tui/update.go`, `internal/tui/view.go`, `internal/tui/styles.go` |
| Cambiar Obsidian | `internal/obsidian/` | `plugin/obsidian/`, `docs/beta/obsidian-brain.md` |
| Preparar una feature grande | `openspec/changes/*` | `openspec/specs/*`, `CONTRIBUTING.md` |

---

## Separación del código por responsabilidad

### Tabla de ownership

| Área | Responsabilidad | No debería hacer |
|---|---|---|
| `cmd/engram` | Parseo CLI, composición de runtimes, wiring entre paquetes, flags, comandos de usuario. | Meter SQL o reglas de negocio profundas. |
| `internal/store` | Fuente de verdad local: SQLite, FTS5, sesiones, observaciones, prompts, relaciones, mutaciones, dedupe, topic upserts, diagnóstico local. | Renderizar HTML, hablar HTTP cloud directo, decidir UX. |
| `internal/mcp` | Exponer herramientas MCP, resolver proyecto por cwd, perfilar herramientas (`agent`, `admin`, `all`), traducir llamadas de agente a operaciones store. | Persistir por fuera del store o duplicar lógica de store. |
| `internal/server` | API JSON local (`engram serve`), endpoints locales, notificación a autosync después de writes. | Exponer rutas cloud/dashboard o usar Postgres. |
| `internal/sync` | Export/import de chunks, manifest, transporte abstracto, bootstrap/upgrade de sync. | Decidir auth cloud o renderizar dashboard. |
| `internal/cloud/chunkcodec` | Canonicalización compartida de chunks, IDs y decodificación de payloads de mutaciones usada por sync/cloud. | Decidir transporte, persistencia o políticas de sync. |
| `internal/cloud/remote` | Cliente HTTP hacia cloud para chunks/mutaciones. | Guardar estado local directamente salvo a través de interfaces esperadas. |
| `internal/cloud/autosync` | Orquestar background push/pull con leases, backoff, cursores y estado degradado. | Implementar transporte HTTP o queries SQL concretas. |
| `internal/cloud/cloudserver` | Runtime cloud HTTP: `/sync/*`, auth boundary, dashboard mount, enforcement server-side. | Guardar datos local-first o mezclar HTML/SQL en handlers. |
| `internal/cloud/cloudstore` | Persistencia cloud en Postgres, chunks, mutaciones, read-model del dashboard, controles, auditoría. | Decidir rutas HTTP o interacción de browser. |
| `internal/cloud/dashboard` | UI browser server-rendered, rutas `/dashboard/*`, HTMX, componentes templ, navegación. | Enforcear políticas solo visualmente; si es regla real, debe llegar a server/cloudstore. |
| `internal/cloud/auth` | Bearer token, project scope authorizer, sesiones firmadas para dashboard. | Crear reglas de producto fuera de su contrato de auth. |
| `internal/project` | Detección y normalización de proyecto. | Acceder al store para corregir datos. |
| `internal/setup` | Instalación idempotente de integraciones de agentes. | Orquestar cloud enrollment/login automático si no está implementado y documentado. |
| `plugin/` | Adaptadores finos por agente/host. | Contener comportamiento core que debería vivir en Go. |
| `skills/` | Guardrails para agentes contributors. | Reemplazar specs, tests o código. |
| `docs/` | Guías de uso, arquitectura, cloud, plugins, instalación, doctor. | Documentar aspiraciones no implementadas. |
| `openspec/` | Propuestas, specs, diseños y tareas por cambio. | Ser documentación de usuario final. |

### Regla de oro de ubicación

```text
¿Es persistencia local?        → internal/store
¿Es contrato HTTP local?       → internal/server
¿Es herramienta para agentes?  → internal/mcp
¿Es chunk/export/import?       → internal/sync
¿Es canonicalización de chunks? → internal/cloud/chunkcodec
¿Es cliente cloud?             → internal/cloud/remote
¿Es orquestación background?   → internal/cloud/autosync
¿Es transporte cloud/server?   → internal/cloud/cloudserver
¿Es Postgres/read-model cloud? → internal/cloud/cloudstore
¿Es pantalla browser?          → internal/cloud/dashboard
¿Es auth/sesión?               → internal/cloud/auth
¿Es instalación de agentes?    → internal/setup
```

Si un cambio necesita dos áreas, separá el contrato: por ejemplo, un toggle de dashboard que pausa sync debe tener UI en `dashboard`, enforcement en `cloudserver`, estado en `cloudstore` y tests cruzando esa frontera.

---

## Runtime split: local vs cloud

Engram tiene dos runtimes que no conviene mezclar.

```text
Runtime local: engram serve
  Escucha JSON API local en 127.0.0.1:7437 por defecto
  Usa internal/server
  Lee/escribe internal/store SQLite
  Expone /sync/status para estado local/autosync

Runtime cloud: engram cloud serve
  Escucha cloud transport + dashboard
  Usa internal/cloud/cloudserver
  Persiste en internal/cloud/cloudstore Postgres
  Monta internal/cloud/dashboard bajo /dashboard/*
  Aplica auth/policy de proyecto en el borde cloud
```

| Runtime | Comando | Paquetes principales | Tipo de dato |
|---|---|---|---|
| Local API | `engram serve` | `cmd/engram`, `internal/server`, `internal/store` | JSON local, SQLite |
| MCP stdio | `engram mcp` | `cmd/engram`, `internal/mcp`, `internal/store` | Tools MCP, SQLite |
| CLI directa | `engram search`, `engram save`, `engram sync`, etc. | `cmd/engram`, `internal/store`, `internal/sync` | stdout/stderr + SQLite/chunks |
| TUI | `engram tui` | `internal/tui`, `internal/store` | Terminal Bubbletea |
| Cloud | `engram cloud serve` | `internal/cloud/*`, `cmd/engram/cloud.go` | HTTP cloud, Postgres, dashboard |

Para lista exacta y actualizada de endpoints, no copies esta tabla: usá [DOCS.md — HTTP API Endpoints](../DOCS.md#http-api-endpoints).

---

## Flujo principal: guardar y recuperar memoria

El flujo de memoria no empieza en la base. Empieza en el agente decidiendo que algo merece ser recordado.

```text
1. El agente termina trabajo significativo
   bugfix, decisión, discovery, config, convención, resumen de sesión

2. El agente llama una tool MCP
   mem_save / mem_session_summary / mem_save_prompt / mem_capture_passive

3. internal/mcp resuelve proyecto y valida el contrato
   cwd → .engram/config.json → git remote/root → child repo → basename

4. internal/store persiste
   sessions / observations / user_prompts / memory_relations / sync_mutations
   FTS5 indexa para búsqueda

5. Próxima sesión
   mem_context → mem_search → mem_get_observation si hace falta detalle completo
```

### Entidades mentales del store

| Entidad | Propósito | Archivos relevantes |
|---|---|---|
| `sessions` | Agrupa trabajo de una sesión de agente. | `internal/store/store.go`, `internal/mcp/activity.go` |
| `observations` | Memorias curadas: decisiones, bugs, patterns, discoveries, summaries. | `internal/store/store.go`, `internal/store/store_test.go` |
| `observations_fts` | Índice FTS5 para búsqueda. | `internal/store/store.go`, `DOCS.md#database-schema` |
| `user_prompts` / `prompts_fts` | Prompt del usuario como contexto recuperable. | `internal/store/store.go`, `internal/server/server.go` |
| `memory_relations` | Relaciones/judgments entre memorias para conflicto semántico. | `internal/store/relations.go`, `internal/mcp/mcp_judge_test.go` |
| `sync_mutations` | Cola de cambios para sync/autosync. | `internal/store/store.go`, `internal/sync/sync.go`, `internal/cloud/autosync/manager.go` |
| `sync_apply_deferred` | Mutaciones pull diferidas por dependencias faltantes. | `internal/store/sync_apply_test.go`, `internal/server/server.go` |

### Invariantes de memoria

- El protocolo de agente y las guías de herramienta esperan contenido estructurado en `mem_save`: **What / Why / Where / Learned**. La capa de persistencia no rechaza automáticamente prosa mal formada; la disciplina vive en la instrucción del agente y la revisión.
- `topic_key` es para temas evolutivos; no se mezclan decisiones distintas en una misma key.
- `scope=project` es el default; `scope=personal` existe para memoria no compartida.
- Soft delete (`deleted_at`) oculta sin borrar físicamente salvo hard delete explícito.
- Las herramientas de escritura resuelven proyecto desde cwd/config; no se debe inventar proyecto si hay ambigüedad.
- La búsqueda es progresiva: resultados compactos primero, `mem_get_observation` recién cuando necesitás contenido completo.

---

## Interfaces de acceso

### CLI: `cmd/engram`

`cmd/engram/main.go` y archivos vecinos son el punto de entrada del binario. Ahí se conectan store, HTTP, MCP, TUI, sync, autosync, setup, doctor, conflictos, cloud y Obsidian.

No metas comportamiento core en el comando si puede vivir en un paquete testeable. El comando debería coordinar, parsear flags y adaptar errores para humanos.

### MCP: `internal/mcp`

`internal/mcp/mcp.go` expone Engram a agentes por stdio. Tiene perfiles de herramientas:

| Perfil | Uso |
|---|---|
| `all` | Default de `engram mcp`; registra todas las herramientas. |
| `agent` | Herramientas normales de agente, como save/search/context/session summary/current project/judge/doctor. |
| `admin` | Curation manual: stats, delete, timeline, merge. |

Puntos importantes:

- `mem_current_project` es la primera llamada recomendada para confirmar detección.
- Escrituras normales no deberían pasar `project` como override arbitrario.
- La recuperación de `ambiguous_project` exige que el usuario elija un proyecto exacto.
- Si `mem_save` devuelve candidatos de conflicto, el agente debe juzgar con `mem_judge` o preguntar cuando la relación sea sensible.

### API local: `internal/server`

`internal/server/server.go` es una API JSON simple sobre el store local. También expone `GET /sync/status` para visibilidad de autosync/degraded state.

Usala para plugins, hooks o clientes externos locales. No la confundas con cloud: el runtime cloud tiene su propio server.

### TUI: `internal/tui`

La TUI usa Bubbletea y lee del store local. La separación es clásica:

| Archivo | Rol |
|---|---|
| `internal/tui/model.go` | Estado, pantallas, inicialización. |
| `internal/tui/update.go` | Manejo de input/transiciones. |
| `internal/tui/view.go` | Render de pantallas. |
| `internal/tui/styles.go` | Estilos Lipgloss. |

---

## Sync local y cloud sync

Engram tiene dos ideas relacionadas pero distintas:

1. **Git sync por chunks**: export/import a `.engram/manifest.json` y `.engram/chunks/*.jsonl.gz`.
2. **Cloud sync opt-in**: push/pull contra `engram cloud serve` por proyecto explícito.

```text
SQLite local
   │
   ├── engram sync
   │     └── .engram/manifest.json + chunks gzip JSONL
   │
   └── engram sync --cloud --project <name>
         └── internal/cloud/remote
                └── HTTP /sync/*
                       └── internal/cloud/cloudserver
                              └── internal/cloud/cloudstore Postgres
```

### Git-friendly chunks: `internal/sync`

`internal/sync/sync.go` evita un gran JSON compartido. Cada sync crea chunks nuevos y un manifest chico. Eso reduce conflictos de merge y permite que múltiples máquinas generen memoria en paralelo.

Guardrails:

- No modificar chunks viejos para “actualizarlos”.
- No asumir que todos los proyectos se exportan salvo que el comando lo indique.
- Mantener tracking de chunks importados para evitar duplicados.

### Autosync cloud: `internal/cloud/autosync`

`internal/cloud/autosync/manager.go` corre en procesos largos y coordina:

- lease SQLite para evitar workers duplicados,
- push de mutaciones pendientes,
- pull por cursor,
- replay de diferidos,
- backoff con jitter,
- estado degradado con `reason_code` y mensaje.

Regla de negocio: **si sync está bloqueado, se falla fuerte y visible**. Nada de drops silenciosos.

### Cloud transport: `internal/cloud/remote` + `internal/cloud/cloudserver`

`internal/cloud/remote/transport.go` es el cliente. `internal/cloud/cloudserver/cloudserver.go` es el server. El server monta:

- `GET /health`
- `GET /sync/pull`
- `GET /sync/pull/{chunkID}`
- `POST /sync/push`
- `POST /sync/mutations/push`
- `GET /sync/mutations/pull`
- `/dashboard/*`

### Cloud store: `internal/cloud/cloudstore`

`internal/cloud/cloudstore/cloudstore.go` persiste en Postgres, materializa chunks/mutaciones y alimenta read-models del dashboard. Si una política organizacional importa, el estado vive acá o se enforcea desde `cloudserver` contra datos de acá.

---

## Dashboard cloud

El dashboard vive en `internal/cloud/dashboard` y se monta desde `internal/cloud/cloudserver`. Es server-rendered con templ y usa HTMX para parciales.

```text
Browser
  GET /dashboard/*
      │
      ▼
internal/cloud/cloudserver
  frontera de auth/sesión/admin
      │
      ▼
internal/cloud/dashboard
  handlers + templ components + static assets
      │
      ▼
DashboardStore interface
      │
      ▼
internal/cloud/cloudstore
  Postgres read-model + controls + audit log
```

### Archivos clave

| Archivo | Responsabilidad |
|---|---|
| `internal/cloud/dashboard/dashboard.go` | Route mount, handlers, `DashboardStore` interface. |
| `internal/cloud/dashboard/*_templ.go` | Componentes templ generados y verificados. |
| `internal/cloud/dashboard/static/styles.css` | Estilos del dashboard. |
| `internal/cloud/dashboard/middleware.go` | Sesión/protección de rutas. |
| `internal/cloud/dashboard/principal.go` | Identidad visual del operador. |
| `internal/cloud/cloudserver/cloudserver.go` | Frontera de autenticación, montaje del dashboard y transporte de sync. |
| `internal/cloud/cloudstore/dashboard_queries.go` | Queries/read-model del dashboard. |
| `internal/cloud/cloudstore/project_controls.go` | Controles por proyecto. |
| `internal/cloud/cloudstore/audit_log.go` | Auditoría de eventos relevantes. |

### Invariante central del dashboard

La UI no puede mentir. Si muestra “sync pausado”, todos los caminos de push deben estar bloqueados del lado servidor, incluyendo `POST /sync/push` y `POST /sync/mutations/push`. Si muestra controles de administración, esos controles deben mapear a comportamiento aplicable y testeable.

---

## Integraciones de agentes y plugins

Engram soporta agentes por MCP y por plugins que agregan lifecycle/session management.

```text
Agente
  │
  ├── Bare MCP
  │     └── engram mcp --tools=agent
  │
  ├── OpenCode plugin
  │     ├── plugin/opencode/engram.ts
  │     └── internal/setup setup opencode
  │
  ├── Claude Code plugin
  │     ├── plugin/claude-code/.claude-plugin/plugin.json
  │     ├── plugin/claude-code/hooks/hooks.json
  │     ├── plugin/claude-code/scripts/*.sh / *.ps1
  │     └── plugin/claude-code/skills/memory/SKILL.md
  │
  ├── Gemini / Codex setup
  │     └── internal/setup/setup.go
  │
  └── VS Code / Antigravity / Cursor / Windsurf MCP manual
        └── configuración JSON documentada en docs/AGENT-SETUP.md
```

`internal/setup` no instala todas las integraciones posibles. VS Code, Antigravity, Cursor y Windsurf son caminos de configuración MCP manual documentados en `docs/AGENT-SETUP.md`.

### Principio de plugin fino

Los plugins pueden:

- arrancar o encontrar `engram serve`,
- crear sesiones,
- importar chunks,
- inyectar Memory Protocol,
- persistir summaries en compaction,
- limpiar tags privados,
- registrar MCP.

Los plugins **no** deberían implementar semántica core de memoria. Si hay una regla de dedupe, prompt capture, relation judgment o project resolution, tiene que estar en Go.

Para detalles por agente, usá [docs/AGENT-SETUP.md](./AGENT-SETUP.md) y [docs/PLUGINS.md](./PLUGINS.md).

---

## Funcionalidades principales del producto

| Funcionalidad | Qué hace | Dónde vive |
|---|---|---|
| Guardado proactivo | Memorias estructuradas por agente. | `internal/mcp`, `internal/store` |
| Búsqueda FTS5 | Recupera memorias por texto, tipo, proyecto, scope. | `internal/store`, `internal/server`, `internal/mcp` |
| Contexto reciente | Resumen de sesiones y contexto para nueva sesión. | `internal/store`, `internal/mcp` |
| Prompt capture | Guarda prompts del usuario como contexto recuperable. | `internal/store`, `internal/mcp`, plugins |
| Topic upserts | Actualiza decisiones evolutivas con `topic_key`. | `internal/store`, `internal/mcp` |
| Conflict surfacing | Detecta/juzga relaciones entre memorias. | `internal/store/relations.go`, `internal/mcp`, `cmd/engram/conflicts.go` |
| Doctor/repair | Diagnósticos operacionales y reparación guiada. | `internal/diagnostic`, `cmd/engram/doctor.go`, `docs/DOCTOR.md` |
| Git sync | Export/import de chunks en `.engram/`. | `internal/sync` |
| Cloud sync | Replicación por proyecto contra cloud. | `internal/cloud/*`, `internal/sync` |
| Autosync | Push/pull background con degraded status. | `internal/cloud/autosync` |
| Dashboard | Visibilidad web para proyectos, actividad, admin, auditoría. | `internal/cloud/dashboard`, `internal/cloud/cloudstore` |
| Agent setup | Instalación automatizada de MCP/plugins y configuración MCP manual por IDE. | `internal/setup`, `plugin/`, `docs/AGENT-SETUP.md` |
| TUI | Navegación terminal de memorias. | `internal/tui` |
| Obsidian beta | Export/plugin experimental hacia Obsidian. | `internal/obsidian`, `plugin/obsidian`, `docs/beta/obsidian-brain.md` |

---

## Cómo navegar el repo como maintainer

### Primer recorrido recomendado

1. **Producto y comandos**: `README.md`.
2. **Referencia técnica completa**: `DOCS.md`.
3. **Arquitectura existente**: `docs/ARCHITECTURE.md`.
4. **Entrada del binario**: `cmd/engram/main.go` y `cmd/engram/cloud.go`.
5. **Núcleo de datos**: `internal/store/store.go`.
6. **Superficies de acceso**: `internal/mcp/mcp.go`, `internal/server/server.go`, `internal/tui/`.
7. **Sync y cloud**: `internal/sync/sync.go`, `internal/cloud/*`.
8. **Integraciones**: `internal/setup/setup.go`, `plugin/`.
9. **Specs históricas/activas**: `openspec/changes/` y `openspec/specs/`.

### Para revisar un PR

Usá esta lectura en capas:

```text
1. ¿Qué comportamiento dice cambiar?
2. ¿Qué paquete debería ser dueño?
3. ¿El cambio toca la fuente de verdad correcta?
4. ¿Hay test en la frontera afectada?
5. ¿La doc pública quedó alineada?
6. ¿La UI, si existe, representa comportamiento real?
```

---

## Guardrails que no se rompen

### Local-first

- SQLite local es la fuente de verdad del usuario.
- Cloud es opt-in y project-scoped.
- `engram sync --cloud --project <name>` es explícito; evitar “sync everything” implícito.
- Un cliente local debe poder seguir funcionando sin cloud.

### Boundaries técnicos

- Store no renderiza HTML.
- Dashboard no escribe SQL directo si corresponde a cloudstore.
- Handlers HTTP no concentran SQL + HTML + transporte.
- Plugins no contienen reglas core.
- Adapters son finos; comportamiento reusable vive en Go.

### Sync y políticas

- Enrollment controla qué puede sincronizar.
- Pausa cloud controla qué permite la organización ahora.
- Sync bloqueado falla con error visible y reason code; no hay drop silencioso.
- Si una regla es org-level, debe enforcearse server-side.

### Docs

- Docs describen comportamiento actual, no intención.
- No dupliques la API completa fuera de `DOCS.md`.
- Si cambiás endpoint, comando o setup, actualizá docs en el mismo cambio.
- Validá nombres de comandos, rutas y archivos antes de publicar.

---

## Checklists por tipo de cambio

### Cambio en store local

- [ ] La regla pertenece realmente a `internal/store`.
- [ ] Migración/schema está cubierta por tests existentes o nuevos.
- [ ] FTS/dedupe/topic/scope/soft delete siguen coherentes.
- [ ] Si toca sync, se encolan o aplican mutaciones correctamente.
- [ ] `internal/store/*_test.go` cubre el flujo esperado y casos límite.
- [ ] `DOCS.md#database-schema` se actualiza si cambia schema o semántica pública.

### Cambio en MCP tools

- [ ] La tool usa store como fuente de verdad.
- [ ] Project resolution respeta `.engram/config.json`, cwd y flujo `ambiguous_project`.
- [ ] El perfil `agent`/`admin` sigue intencional.
- [ ] Errores devuelven envelopes útiles para agentes.
- [ ] Tests en `internal/mcp/*_test.go` cubren contrato.
- [ ] `docs/AGENT-SETUP.md`, `docs/ARCHITECTURE.md` o `DOCS.md` se actualizan si cambia comportamiento visible.

### Cambio en API local

- [ ] La ruta pertenece a `engram serve`, no a cloud.
- [ ] `internal/server/server.go` solo orquesta request/response y llama store/servicios.
- [ ] Status codes y JSON errors son determinísticos.
- [ ] Tests en `internal/server/*_test.go` cubren errores y éxito.
- [ ] `DOCS.md#http-api-endpoints` se actualiza si hay endpoint público nuevo/modificado.

### Cambio en sync/cloud

- [ ] Local SQLite sigue siendo fuente de verdad.
- [ ] Cloud sync es project-scoped.
- [ ] Push y pull están cubiertos si cambia contrato de sync.
- [ ] Bloqueos/policies fallan fuerte con reason code.
- [ ] `internal/cloud/autosync/*_test.go`, `internal/cloud/remote/*_test.go`, `internal/cloud/cloudserver/*_test.go` o `internal/cloud/cloudstore/*_test.go` cubren frontera afectada.
- [ ] Docs cloud (`docs/engram-cloud/*`, `DOCS.md#cloud-cli-opt-in`, `DOCS.md#cloud-autosync`) quedan alineadas.

### Cambio en dashboard

- [ ] La UI representa estado real, no fake control.
- [ ] Política admin se enforcea en `cloudserver`/`cloudstore`, no solo en templ/HTMX.
- [ ] Handlers quedan en `internal/cloud/dashboard`.
- [ ] Queries/read-model quedan en `internal/cloud/cloudstore`.
- [ ] Componentes templ generados se mantienen junto al cambio cuando aplique.
- [ ] Tests cubren rutas, autenticación/sesión/administración, parciales HTMX y casos límite.

### Cambio en plugins/setup

- [ ] El plugin sigue siendo adapter fino.
- [ ] Comportamiento core vive en Go, no en shell/TypeScript duplicado.
- [ ] Setup es idempotente.
- [ ] Windows/macOS/Linux o paths documentados siguen correctos.
- [ ] `docs/AGENT-SETUP.md` y `docs/PLUGINS.md` reflejan exactamente el flujo actual.
- [ ] No prometas cloud bootstrap automático si todavía es CLI-first.

### Cambio en documentación

- [ ] El payoff aparece al inicio.
- [ ] Se documenta comportamiento actual, no roadmap.
- [ ] Los paths existen.
- [ ] Los comandos y rutas son actuales.
- [ ] No se duplica referencia completa de API; se linkea a `DOCS.md`.
- [ ] Hay tabla/checklist si el lector debe decidir o actuar.

---

## Playbook de maintainer

### Antes de aprobar arquitectura

Preguntas duras:

1. **¿Quién es la fuente de verdad?** Si la respuesta no es SQLite local para memoria local, revisá el diseño.
2. **¿Dónde se enforcea la regla?** Si es política org-level y solo está en UI, está mal.
3. **¿Qué frontera cambia?** Store/server/cloudstore/cloudserver/dashboard/autosync/plugin.
4. **¿Hay test de frontera?** Los bugs caros aparecen entre paquetes, no en helpers aislados.
5. **¿La doc quedó sincronizada?** Si el usuario puede verlo o ejecutarlo, tiene que estar documentado.

### Señales de alerta

| Olor | Probable problema | Corrección esperada |
|---|---|---|
| SQL dentro de handler dashboard | Mezcla de concerns | Mover query a `cloudstore`. |
| Toggle admin que solo cambia HTML | Fake control | Persistir estado y enforcear en server. |
| Plugin implementa dedupe/sync policy | Adapter demasiado gordo | Mover a Go. |
| Cloud requerido para feature local | Rompe local-first | Diseñar local primero, cloud después. |
| Endpoint nuevo sin docs/tests | Contrato invisible | Agregar tests y actualizar `DOCS.md`. |
| Helper genérico con acoplamiento oculto | Cleverness local | Ubicar comportamiento en owner explícito. |

### Decisión rápida de ownership

```text
¿El cambio afecta datos guardados?         store/cloudstore
¿Afecta qué entra/sale por HTTP?          server/cloudserver
¿Afecta experiencia browser?              dashboard
¿Afecta background replication?           autosync
¿Afecta transporte remoto?                remote/cloudserver
¿Afecta agente o host específico?         plugin/setup
¿Afecta comandos humanos?                 cmd/engram + docs
```

---

## Apéndice trazable: documentos y archivos fuente

### Documentos principales

| Documento | Uso |
|---|---|
| `README.md` | Landing, quickstart, modelo de producto, links principales. |
| `DOCS.md` | Referencia técnica completa: schema, endpoints, tools MCP, CLI/cloud. |
| `CONTRIBUTING.md` | Flujo de contribución y estándares generales. |
| `SECURITY.md` | Reporte de seguridad. |
| `CHANGELOG.md` | Historial de cambios. |
| `docs/ARCHITECTURE.md` | Arquitectura existente, lifecycle, CLI reference, rutas cloud/dashboard. |
| `docs/AGENT-SETUP.md` | Setup por agente, project detection, compaction survival. |
| `docs/PLUGINS.md` | OpenCode/Claude plugin details y límites actuales. |
| `docs/INSTALLATION.md` | Instalación por plataforma. |
| `docs/DOCTOR.md` | Diagnóstico y reparación operacional. |
| `docs/COMPARISON.md` | Comparación con alternativas. |
| `docs/BETA_TESTING.md` | Flujos beta aislados. |
| `docs/intended-usage.md` | Uso esperado/product framing. |
| `docs/engram-cloud/README.md` | Landing de cloud. |
| `docs/engram-cloud/quickstart.md` | Camino recomendado para cloud. |
| `docs/engram-cloud/troubleshooting.md` | Fallos y recuperación cloud. |

### Archivos fuente principales

| Archivo/directorio | Por qué importa |
|---|---|
| `cmd/engram/main.go` | Wiring principal del binario. |
| `cmd/engram/cloud.go` | Subcomandos y runtime cloud. |
| `cmd/engram/doctor.go` | CLI doctor. |
| `cmd/engram/conflicts.go` | CLI de conflictos. |
| `internal/store/store.go` | Núcleo de persistencia local. |
| `internal/store/relations.go` | Relaciones/judgments de memoria. |
| `internal/mcp/mcp.go` | MCP tools y perfiles. |
| `internal/mcp/activity.go` | Tracking de actividad/sesión MCP. |
| `internal/server/server.go` | API JSON local. |
| `internal/sync/sync.go` | Chunks, manifest, import/export, bootstrap. |
| `internal/sync/transport.go` | Abstracción de transporte sync. |
| `internal/cloud/config.go` | Config cloud desde entorno. |
| `internal/cloud/chunkcodec/` | Canonicalización de chunks, IDs y decodificación de payloads de mutaciones. |
| `internal/cloud/remote/transport.go` | Cliente remote sync/mutations. |
| `internal/cloud/autosync/manager.go` | Manager background push/pull. |
| `internal/cloud/cloudserver/cloudserver.go` | Runtime cloud HTTP + dashboard mount. |
| `internal/cloud/cloudserver/mutations.go` | Endpoints/contrato de mutaciones cloud. |
| `internal/cloud/cloudstore/cloudstore.go` | Postgres cloud store. |
| `internal/cloud/cloudstore/dashboard_queries.go` | Read-model dashboard. |
| `internal/cloud/cloudstore/project_controls.go` | Controles de sync por proyecto. |
| `internal/cloud/cloudstore/audit_log.go` | Auditoría cloud/dashboard. |
| `internal/cloud/auth/auth.go` | Auth bearer/sesión. |
| `internal/cloud/dashboard/dashboard.go` | Rutas y handlers dashboard. |
| `internal/cloud/dashboard/static/styles.css` | Estilos dashboard. |
| `internal/project/detect.go` | Detección de proyecto. |
| `internal/project/similar.go` | Similaridad/consolidación de nombres. |
| `internal/setup/setup.go` | Instalación de integraciones. |
| `internal/tui/` | TUI Bubbletea. |
| `internal/diagnostic/` | Checks/repair operacionales. |
| `internal/llm/` | Runners para escaneo semántico con CLIs de agentes. |
| `internal/obsidian/` | Export/watch/hub Obsidian beta. |
| `plugin/opencode/engram.ts` | Adapter OpenCode. |
| `plugin/claude-code/` | Plugin Claude Code, hooks y skill. |
| `plugin/obsidian/` | Plugin Obsidian experimental. |
| `skills/` | Reglas de contribución para agentes. |
| `openspec/` | Specs/diseños/tareas por cambio. |

---

## Cierre: la brújula del codebase

Cuando dudes, volvé a esta brújula:

```text
Local-first antes que cloud-first.
Comportamiento real antes que UI linda.
Owner explícito antes que helper cómodo.
Adapter fino antes que plugin inteligente.
Docs actuales antes que promesas.
Tests de frontera antes que confianza oral.
```

Engram se mantiene sano cuando cada paquete hace una cosa clara y todas las superficies cuentan la misma historia: **un agente guarda memorias curadas, SQLite local las conserva, y cloud solo replica o permite verlas cuando el usuario lo elige explícitamente**.

# godarch — Architecture intelligence for Godot projects

A desktop app that statically analyses a Godot project — code **and** scenes, resources,
assets, and editor configuration — to build a typed dependency graph of the whole game, then
surfaces domain sprawl, coupling hotspots, fragile wiring, and dead configuration.

Modelled on [`archi`](../archi), which does this for microservice backends. This document
distills what archi does, translates its model to the Godot domain, and proposes an
architecture for godarch.

---

## 1. What archi actually is (the transferable architecture)

archi is a **static-analysis pipeline that produces a typed graph and then runs graph analyses
over it.** Strip away the web/Elixir/Ruby specifics and the shape is:

```
discover → extract (per language) → emit schema-conformant artifacts
        → stitch (resolve cross-unit references via match keys)
        → ingest into a graph store
        → derive analyses (walks, cycles, hubs, flow patterns, metrics)
        → present (UI + query facade + optional MCP)
```

Key design decisions worth stealing wholesale:

- **Boundary-oriented model, not just a call graph.** archi's core abstractions are *ingress*
  (typed entry points: HTTP routes, jobs, message consumers) and *egress* (typed side-effect
  call sites: HTTP calls, DB queries, publishes, cache, filesystem). The call graph exists only
  to connect ingress → egress. Coupling is reasoned about at the boundary, not at every function.
- **Language-native parsing, pluggable per stack.** Each analyzer uses the language's own AST
  (`Code.string_to_quoted` for Elixir, `RubyVM::AST` for Ruby, `ast` for Python). Ingress and
  egress detectors are a **plugin registry** keyed by framework (Phoenix router, Oban worker,
  Tesla client, Ecto…). Adding a language = implement the analyzer contract + emit the schema.
- **Match keys as deterministic strings.** Cross-unit linking ("stitching") never traverses
  code across units. Each boundary point is reduced to a normalized string like
  `http:post:/api/users` or `messaging:orders/created`. Two points link iff their match keys are
  equal. Normalization (param tokens, variants) is the entire secret sauce. Confidence scores +
  exact/variant/fuzzy strategies handle ambiguity.
- **Walks = rooted DFS from ingress through the call graph to egress**, with cycle cutoff and a
  precomputed reachability cache. Walks crossing unit boundaries are "platform walks".
- **Generic node/edge store with typed properties.** Postgres `core_nodes` + `core_edges`, each
  with a `label`/`type` and a JSONB `properties` validated against per-label embedded schemas.
  Derived layers (walks, cycles, flow patterns, node properties) are **computed post-ingestion**
  as separate persisted passes, not at query time.
- **Flow patterns = equivalence classes of walks** by interaction signature → recurring shapes.
- **Cycles via Tarjan SCC; hubs/leaves/isolated via node-degree properties; blast radius via
  forward reachability.** These are the actual "architecture smells" outputs.
- **Manual override files** (`http_overrides.yml`, `service_registry.yml`, fixtures) patch what
  static analysis can't resolve, and lock regression-prone matches as ground-truth tests.

The single most important idea: **reduce a heterogeneous system to typed boundary points +
normalized match keys + a resolution pass, then do graph theory on the result.**

---

## 2. The translation problem: where does coupling *live* in Godot?

In a microservice backend the "edges" sit at the **process boundary** — services are separate
deployables that can only talk over HTTP / messaging / shared DB. Those channels are explicit
and few, which is why archi can enumerate them.

A Godot game is (mostly) a **single process**. So the unit of analysis is not a deployable — it's
a **logical module / domain** (a cohesive cluster of scripts + scenes + resources + assets). And
the coupling between modules runs through a *much wider and stranger* set of channels than HTTP.
Crucially:

> **A large fraction of a Godot project's wiring lives outside the code — in `.tscn` connections,
> exported-and-inspector-assigned properties, autoload registration, the input map, animation
> tracks, and group membership.** A code-only tool (call-graph analyzer, the GDScript LSP) is
> *blind* to all of it.

That blindness is godarch's reason to exist. The differentiator is **resolving editor-configured
edges against the code and surfacing the mismatches** — the signal connected in the inspector to
a method that no longer exists; the `@export var` never assigned in any scene; the group called in
code that no node ever joins; the input action referenced in code but absent from `project.godot`.

So the conceptual mapping is:

| archi | godarch |
|---|---|
| Service (one repo, one process) | **Module / domain** — an inferred cluster of files (and the whole project is the "platform") |
| Ingress (route, job, consumer) | Signal handler, lifecycle/virtual method (`_ready`, `_process`…), editor-wired connection target, `@rpc` method, group-call target, input-action handler |
| Egress (HTTP call, query, publish) | `emit_signal`/`.emit()`, `connect`, resource `load`/`preload`, `change_scene_to_*`, autoload access, `get_node`/NodePath reach, `call_group`, `@rpc` call, file/`user://` save |
| Match key (`http:post:/users`) | `signal:<name>` + target, `res://path` or `uid://…`, `action:<name>`, `group:<name>`, `autoload:<Name>`, node path |
| Platform edge (cross-service link) | Cross-module/cross-scene resolved link (emit in A → handler in B; `load` in A → resource owned by B) |
| Stitching | Resolve emit→connected handlers; `load("res://x")`→the scene/resource node; editor `[connection]`→method symbol; `@export NodePath`→target node; `action`→input map; `group`→members |
| Walk | Causal chain: input action → handler → emit → connected handlers → scene change → … |
| Flow pattern | Recurring interaction shapes ("UI button → autoload signal → N listeners") |
| Cycle / hub / blast radius | Module dependency cycle / god-autoload / "what breaks if I change this scene" |

---

## 3. The "edges" of a Godot project (the deep dive)

This is the heart of the analysis. Entities (nodes) first, then the typed edges between them.

### 3.1 Entities (graph nodes)

- **Scripts** — `.gd` / `.cs`, with their classes (`class_name`), methods, signals, exported vars.
- **Scenes** — `.tscn` (and binary `.scn`), with their internal node tree.
- **Scene-internal nodes** — each `[node]` in a scene; has a type, a parent path, optional script,
  groups, and property values.
- **Resources** — `.tres`/`.res`, including custom `extends Resource` data classes.
- **Autoloads / singletons** — scripts or scenes registered in `project.godot` `[autoload]`.
- **Assets** — textures, audio, models (`.glb`/`.gltf`), fonts, shaders (`.gdshader`), themes.
- **`.import` configs** — the import-pipeline settings paired with each asset.
- **Named concepts** — input actions, groups, collision layers, signal names. These are strings
  defined in one place and referenced (often by string) elsewhere; treat them as first-class nodes
  so "defined but unused" / "used but undefined" become graph queries.
- **Class registry** — `class_name` globals + engine built-in types + GDExtension-registered
  native classes.
- **GDExtension / native** — `.gdextension` files and the classes they register (C/C++ entry).

### 3.2 Typed edges — and which are code vs editor-configured

The **C/E** column is the whole point: many edges are invisible to code-only tools.

| Edge type | From → To | Where it's declared | Notes / smell potential |
|---|---|---|---|
| `instances` | scene → scene | **Editor** (`[node instance=ExtResource]`) | composition; high fan-in = a "god scene" |
| `attaches_script` | node/scene → script | **Editor** (`script = ExtResource`) | links scene wiring to code |
| `extends` | script → script/class | Code (`extends`) | inheritance graph |
| `connects_signal` (editor) | emitter node → handler method | **Editor** (`[connection signal=… from=… to=… method=…]`) | **invisible in code**; dangling if method removed |
| `connects_signal` (code) | object → callable | Code (`.connect()`, `sig.connect`) | dynamic targets often unresolvable |
| `emits_signal` | method → signal | Code (`emit_signal`, `sig.emit()`) | egress; fan-out = event bus pressure |
| `declares_signal` | script → signal name | Code (`signal foo`) | unused signal = dead |
| `calls` | method → method | Code | the in-module call graph |
| `references_node` | script → node path | Code (`get_node`, `$Path`, `%Unique`) | cross-tree reach = fragile structural coupling |
| `exports_var` / `binds_export` | script var ↔ inspector value | Code declares (`@export`), **Editor** assigns (`.tscn` property / `@export NodePath`) | **the canonical editor↔code seam**; unassigned export = dead config |
| `accesses_autoload` | script → autoload | Code (global name) | global coupling; the domain-sprawl magnet |
| `loads_resource` / `preloads` | script/scene → resource/scene/asset | Code (`load`/`preload`) + **Editor** (`ext_resource`) | dynamic string path = unresolvable egress |
| `changes_scene_to` | script → scene | Code (`change_scene_to_*`) | the scene/navigation flow graph |
| `in_group` | node → group | Code (`add_to_group`) + **Editor** (`groups=[…]`) | broadcast membership |
| `calls_group` | script → group | Code (`call_group`, `get_nodes_in_group`) | implicit pub/sub by string |
| `uses_action` | script → input action | Code (`Input.is_action_*`) + **Config** (`project.godot [input]`) | undefined/unused action smells |
| `uses_layer` | node → collision layer | **Editor** (bitmask) + **Config** (layer names) | implicit physics interaction graph |
| `rpc_call` / `rpc_endpoint` | method → `@rpc` method | Code (`@rpc`, `rpc()`/`rpc_id()`) | true network edge (multiplayer) |
| `animates` | AnimationPlayer track → node prop/method | **Editor** (track path/method by string) | invisible behavior; calls methods that may not exist |
| `uses_asset` | resource/scene → asset file | **Editor** (`ext_resource`) | unused asset = dead weight |
| `imports` | asset ↔ `.import` | Build config | import settings drift |
| `uses_shader` / `uses_material` / `uses_theme` | resource → asset | **Editor** | asset dependency chains |

### 3.3 The `.tscn` / `.tres` / `project.godot` parsing surface

These are an INI-like text format and are the richest source of editor-configured edges. The
extractor must read, at minimum:

- **Header** — `[gd_scene format=3 uid="uid://…"]`. The `uid://` is the stable identity that
  survives file moves; the resolution layer must maintain a UID→path map (from `.godot/uid_cache`
  or by scanning headers).
- **`[ext_resource type=… uid=… path=… id=…]`** — external dependency. `type="Script"` →
  `attaches_script`/code dep; `type="PackedScene"` → `instances`; `type="Texture2D"` etc → asset.
- **`[sub_resource type=… id=…]`** — inline resources (materials, shapes, animations) with their
  own property references.
- **`[node name=… type=… parent=… instance=ExtResource("…") groups=[…]]`** then property lines like
  `script = ExtResource("…")`, `target = NodePath("../Player")`, `shape = SubResource("…")`,
  `some_export = …`. This yields `attaches_script`, `instances`, `in_group`, `exports`/binds,
  and NodePath references.
- **`[connection signal="pressed" from="UI/Button" to="." method="_on_pressed" flags=… binds=[…]]`**
  — the editor-wired signal graph. `from`/`to` are node paths within the scene; `method` is a
  symbol that must resolve against the script attached to the `to` node.
- **`[editable path="Child"]`** — overrides into an instanced sub-scene.
- **`project.godot`** — `[autoload]` (singletons), `[input]` (action map), `[layer_names]`,
  `application/run/main_scene` (the root of the whole scene-flow graph), `[global_group]`.
- **`.import`** — `[remap] importer=…`, `[deps] source_file=…`, import params per asset.
- **`plugin.cfg` / `.gdextension`** — editor plugins and native class registration.

---

## 4. The godarch domain model

Mirror archi's structure precisely so the analyses transfer:

- **Node** — `{ id, kind, file, line?, identity{…}, properties{…} }`. `kind` ∈ {script, scene,
  scene_node, resource, autoload, asset, action, group, layer, signal, class, extension}.
- **Boundary points**:
  - **Ingress** — `{ type, symbol, file, line }`, `type` ∈ {lifecycle, signal_handler,
    editor_connection, rpc_endpoint, group_target, input_handler, notification}.
  - **Egress** — `{ type, target_match_key, file, line }`, `type` ∈ {signal_emit, signal_connect,
    resource_load, scene_change, autoload_access, node_reach, group_call, rpc_call, file_io}.
- **Match key** (the normalization contract — the secret sauce, as in archi):
  - signals: `signal:<emitter_type_or_*>:<name>`
  - resources: canonicalize to `res://…` via UID map; `res:<normalized_path>`
  - actions: `action:<name>`, groups: `group:<name>`, autoloads: `autoload:<Name>`
  - node paths: resolved against the owning scene tree to a concrete node id where possible
- **Edge** — `{ type, source_id, target_id, origin: code|editor|config, confidence,
  resolved: bool, properties{…} }`.
- **Module / domain** — inferred cluster (see §5). The "service" analogue.
- **Walk** — rooted causal chain from an ingress through edges to terminal egress.
- **Flow pattern** — walk-signature equivalence class.
- **Unresolved edge** — dynamic `load`, dynamic `connect`, NodePath that can't be resolved →
  the diagnostic list (= archi's unresolved egress).

---

## 5. What "domain sprawl & coupling" mean in Godot terms

The analyses to compute post-ingestion (each is a graph query over the typed multigraph):

**Structural / coupling**
- **Afferent/efferent coupling per module** (`Ca`/`Ce`) and **instability** `Ce/(Ca+Ce)`, over the
  resolved edge graph. The classic.
- **Autoload fan-in** — how many scripts touch each singleton. A singleton touched by everything is
  a god object and the #1 sprawl source in real Godot projects.
- **Scene instancing fan-in** — scenes instanced everywhere = de-facto global components.
- **Signal fan-out / event-bus pressure** — emitters with many listeners, especially via an
  autoload "event bus" (a common Godot pattern that quietly becomes untraceable).
- **Cross-tree NodePath reach distance** — `$"../../../Manager"` is fragile structural coupling;
  score by how far up/across the tree a reference walks.
- **Module dependency cycles** — Tarjan SCC over module edges (scenes/scripts/autoloads in a
  reference cycle).

**Domain inference & sprawl**
- Infer domains from **directory structure as a prior**, then refine with **community detection
  (Louvain/modularity)** on the resolved coupling graph.
- **Sprawl** = a domain whose members are physically scattered across the tree, *or* a domain with
  high efferent coupling to many others, *or* an autoload that accumulates members from many
  unrelated domains (low cohesion). Report cross-domain signal/autoload chatter.

**Editor↔code integrity (the unique value)**
- Dangling editor connection (`[connection] method=` → missing method).
- Dead `@export` (declared, never assigned in any scene/inspector).
- Broken/unused group (called in code, no members; or members, never called).
- Undefined/unused input action; orphaned collision layer.
- Dynamic-but-unresolvable `load`/`connect` (flag, don't fail).
- Asset with no inbound `uses_asset` (dead asset); script with no `attaches_script` (dead script).

**Performance-adjacent (cheap wins from the same graph)**
- Per-frame cost surface: scripts with `_process`/`_physics_process` and what they touch.
- Scenes with very large node counts / deep nesting.

**Blast radius** — forward + reverse reachability from any node: "if I change this scene/signal/
autoload, what is affected?"

---

## 6. Proposed architecture

```
┌─ Discovery ─────────────────────────────────────────────────────────────┐
│ walk project dir → classify files → parse project.godot → UID↔path map   │
└───────────────────────────────────────────────────────────────────────────┘
        │
┌─ Extraction (pluggable, per file-type / per language) ────────────────────┐
│ • GDScript extractor   → scripts, methods, signals, exports, calls, egress │
│ • Scene extractor      → node tree, instances, connections, groups, binds  │
│ • Resource extractor   → .tres graphs, custom Resource refs                │
│ • Project/config extr. → autoloads, input map, layers, main scene          │
│ • Asset/import extr.    → assets + .import deps                             │
│ • (later) C# extractor (Roslyn), GDExtension extractor                     │
│   → each emits schema-conformant node + raw-edge + boundary-point artifacts│
└───────────────────────────────────────────────────────────────────────────┘
        │
┌─ Resolution / stitching ──────────────────────────────────────────────────┐
│ match keys → resolve: emit→handlers, load→target, [connection]→method,     │
│ @export NodePath→node, action→input map, group→members, class_name→uses    │
│ → resolved edges + unresolved diagnostics + confidence                     │
└───────────────────────────────────────────────────────────────────────────┘
        │
┌─ Graph build + derived analyses ──────────────────────────────────────────┐
│ typed multigraph → modules (Louvain) → walks → cycles → hubs → metrics →   │
│ editor↔code integrity checks → flow patterns                               │
└───────────────────────────────────────────────────────────────────────────┘
        │
┌─ Store (embedded, no server) ──────────────────────────────────────────────┐
│ SQLite (nodes/edges/properties tables, archi-style generic schema)         │
│ + in-memory gonum/graph for analysis                                       │
└───────────────────────────────────────────────────────────────────────────┘
        │
┌─ Desktop UI + optional MCP ───────────────────────────────────────────────┐
│ graph explorer · domain map · smell report · blast-radius · editor-vs-code │
│ diff view · "open in Godot" deep links                                     │
└───────────────────────────────────────────────────────────────────────────┘
```

### 6.1 Stack — Go (decided)

**Core: Go. Shell: Wails (Go backend + web frontend). Parser: tree-sitter-gdscript via cgo.**
Chosen for developer fluency on a greenfield project, a clean single-binary distributable (the
"non-dev double-clicks it" goal), and a parser path equivalent to any Rust core. The analysis core
is a plain Go library with a thin CLI; the Wails UI is just one front-end over it.

**Parsing.**
- **GDScript** → `tree-sitter-gdscript` through Go bindings (`smacker/go-tree-sitter`). Compile the
  grammar, run tree queries to extract the edge model (signals, `@export`, calls, `load`/`preload`,
  groups). The semantic resolution godarch needs is structural tree-pattern matching, which
  tree-sitter handles well. (`gdtoolkit` (Scony) — the lark-based Python parser + `gdradon`
  complexity metrics — is the reference for grammar/metric behaviour, *not* a runtime dependency.)
- **Scene / resource / config** (`.tscn`, `.tres`, `project.godot`, `.import`, `plugin.cfg`,
  `.gdextension`) → hand-written Go parser; the format is simple INI-like text and is where the
  unique editor-edge value comes from.

**Fidelity levers** (GDScript is dynamically typed — a pure parse can't resolve every symbol):
1. **`class_name` + `extends` + `@export` type hints** give a usable static type lattice for the
   common cases — enough to resolve most `emit`→handler links and method calls.
2. **Optional Godot-assisted resolution** — locate/bundle a Godot **headless** binary for
   ground-truth (the editor maintains a full dependency DB, UID cache, and an LSP that answers
   where-defined queries). An optional accuracy booster (archi-override style), never a hard dep.

**What we build rather than import** (the cost of leaving the Python ecosystem):
- **Graph algorithms** → `gonum/graph` covers Tarjan SCC (cycles), topo sort, connected components,
  traversal.
- **Community detection (Louvain)** for domain inference → vendor or implement (~a few hundred
  lines; well-specified algorithm). The one notable gap vs Python/Rust.
- **Cyclomatic complexity** → own AST pass over the tree-sitter tree (mirror `gdradon`'s rules).

**Store:** SQLite (archi's generic `nodes`/`edges`/`properties` schema) loaded into an in-memory
`gonum` graph for the analysis passes. No external DB — it's a desktop app.

**Distribution wrinkle:** tree-sitter needs **cgo**, which disables Go's effortless cross-compile.
Build per-OS on a CI runner matrix (GitHub Actions) and sign there. Standard, just not "GOOS and
done".

**Alternatives considered:** *Python + webview* — fastest to value via gdtoolkit's ready parser +
metrics, but heaviest/fiddliest to package cleanly. *Rust + Tauri* — technically near-identical to
the Go plan, marginally better graph/community libs, but less fluent. *Godot-native UI* — poetic
and trivially shippable to gamedevs (`GraphEdit`/`Tree` controls), weaker for dense table/diff UX.

### 6.3 Extensibility to C# and C/C++

The whole point of the boundary+match-key model is language-independence at the graph layer.
Adding a language = a new **extractor** that emits the same node/edge/boundary artifacts:

- **C#** — same edge model: `[Signal]`/`EmitSignal`, `GetNode`, `[Export]`, `Connect`. From a Go
  core, extract via `tree-sitter-c-sharp` (consistent with the GDScript path), or shell out to a
  small Roslyn-based helper for full semantic fidelity. Either way C# and GDScript edges land in
  the *same* graph and stitch together (a C# `[Export]` assigned in a `.tscn`, a signal emitted in
  GDScript handled in C#).
- **C/C++ GDExtension** — parse `.gdextension` + the registered class table (and optionally headers)
  to add native classes as nodes; calls into them from GDScript/C# become resolvable edges.

The scene/resource/config extractors are **language-agnostic** — they already work regardless of
which language a script is in, because the `.tscn` only references scripts by resource path.

### 6.4 Manual overrides (archi's lesson)

Provide a `godarch.overrides.yml` for: dynamic `load` paths the tool can't resolve, custom
event-bus signal conventions, domain assignments to override inferred clusters, and ignore globs
(`addons/`, generated files). Plus a fixtures file locking expected resolutions as regression tests.

---

## 7. Suggested phasing

1. **v0 — extract & graph.** Discovery + GDScript extractor (gdtoolkit) + scene/project extractors.
   Build the typed graph. Output: a navigable node/edge explorer. No analyses yet. This alone —
   *one graph spanning code and editor wiring* — is already more than existing tools show.
2. **v1 — resolution & integrity.** Match-key resolution; the editor↔code integrity report
   (dangling connections, dead exports, broken groups/actions, unresolved loads). High, immediate,
   uniquely-godarch value.
3. **v2 — coupling & domains.** Module inference (Louvain), coupling metrics, cycles, autoload
   fan-in, signal fan-out, blast radius, scene-flow graph. The "domain sprawl" story.
4. **v3 — polish & reach.** Flow patterns, performance surface, "open in Godot" deep links,
   optional MCP endpoint (expose the graph to an LLM for Q&A and refactor suggestions, as archi
   does), C# extractor.
5. **later — GDExtension/C++**, Godot-assisted resolution, CI mode (graph diff between commits,
   "coupling regression" gate — archi has a snapshot/diff harness worth copying), and
   **documentation-drift detection** (see §8a) once the graph + snapshot/diff exist. The cheap
   slice of it — a *docs reference linter* (frontmatter anchors must resolve against the graph) —
   can land early alongside the integrity report, since it's the same check family.

---

## 8a. Later feature — documentation drift

The user keeps docs in-repo: dev log, game-design docs, level docs, system notes. Goal: a
consistent format across doc types + LLM integration to detect **documentation drift** over time.

### The core idea — docs-vs-graph reconciliation, not generic staleness

Generic "is this doc stale?" LLM checks are weak: no ground truth, hallucination-prone. godarch's
differentiator is that it already owns an **authoritative structural graph** of the actual project.
So drift is defined as **divergence between what a doc asserts and what the graph shows** — and that
is *verifiable*. Docs reference the same entities godarch already resolves (UIDs, `res://` paths,
signal/group/action names, `class_name`s), so those references are **join keys between prose and
graph — the documentation analogue of match keys.**

### Consistent doc format = anchors the tool can join on

Define a small set of **doc types**, each with lightweight frontmatter that links the prose to graph
entities (schema-driven, exactly like the analyzer artifacts). Frontmatter is what turns "some
markdown" into something mechanically checkable:

```markdown
---
type: system            # system | level | design | devlog | adr
domain: combat          # links to an inferred/declared unit
covers:                 # entity anchors — resolved against the graph
  - res://combat/weapons/weapon.gd
  - uid://b8x2k...
  - signal:Weapon:fired
  - autoload:CombatState
updated: 2026-06-20
---
The Weapon system emits `fired`, which CombatState listens for to apply damage…
```

- `system` — describes a domain/module; `covers` lists its entities.
- `level` — describes a level scene; anchors to the `.tscn`.
- `design` — higher-level game design; links to systems.
- `devlog` — dated entries; captures **intent/decisions** ("we decoupled combat from UI").
- `adr` — architecture decision records with status.

### Two drift mechanisms, two precisions

1. **Identifier-anchored (mechanical, high precision — no LLM needed).** Any `res://`/`uid://`/
   `signal:`/`group:`/`class_name` reference in a doc is resolved against the graph. Findings:
   reference to a deleted/renamed entity; a doc claiming a connection the graph doesn't contain;
   the reverse — a significant graph structure (e.g. a high-fan-in autoload) with no doc coverage.
   This is the **same "X references Y that doesn't exist" check family as the integrity report** —
   a "docs reference linter" is a cheap early win that needs zero LLM.
2. **Prose claims (LLM, grounded).** For natural-language assertions not tied to a hard anchor, an
   LLM extracts claims and checks them — but **grounded**: it's given the doc *plus the relevant
   subgraph* (the entities the doc covers and their current edges). This is RAG over the structural
   graph, not freeform judgement — grounding is what keeps it reliable.

### Division of labor — graph finds, LLM explains

The snapshot/diff harness (copied from archi's snapshot tooling) drives "drift over time": when the
graph diffs between commits (entity renamed, edge removed, signal deleted), godarch knows **which
docs reference the changed entity** → those become drift candidates mechanically and precisely. The
LLM is only invoked to *judge, explain, and draft the correction* on candidates — not to hunt blind.

### LLM integration

- **Pluggable provider** (Claude / OpenAI / local), API key in config — the analogue of archi's
  async enrichment layer.
- **Optional & offline-friendly:** mechanical reference-tracking works with no LLM; the LLM is
  enrichment, not a hard dependency.
- **Outputs:** drift findings (claim, doc location, what the graph says, confidence, suggested
  edit), optionally an auto-drafted doc-fix diff.

### Risks / sequencing

- The noisy part is extracting claims from freeform prose; the reliable core is identifier-anchored
  references. **Lean on the frontmatter convention to maximise mechanical precision.**
- Depends on: the graph, the entity-identifier scheme, and the snapshot/diff harness → naturally a
  **post-v2 feature**. But **define the doc format early** so conforming docs (and the history that
  makes "drift over time" meaningful) start accumulating before the feature ships.

## 8. Open decisions

- **Stack:** ✅ **Go core + Wails shell + tree-sitter-gdscript** (see §6.1). Python/Rust/Godot-native
  recorded as alternatives considered.
- **Unit of analysis:** confirm "module/domain" (inferred) is the right archi-"service" analogue, vs
  treating each top-level scene as a unit, vs user-declared domains.
- **Godot-assisted resolution:** optional booster vs hard dependency vs never.
- **Scope of v1:** ✅ lead with the **editor↔code integrity report** (unique, fast to demo, and
  largely unit-agnostic — see below).

---
title: "gowt public roadmap"
id: 2026-11-05-library-roadmap
author: "Andrey Sobolev"
date: 2026-11-05
state: ready
status: ПЛАН
tasks: TSK-2026-11-05-library-roadmap-01 - TSK-2026-11-05-library-roadmap-11
---

# gowt public roadmap

This is the public development plan for gowt. Task numbers are stable and are not reused. Priorities describe the recommended order, not release promises.

- **P0:** maintainability, navigation and verification infrastructure.
- **P1:** user-facing gaps and reliability on the three supported systems.
- **P2:** larger platform evolution after the current gaps are under control.

Baseline: `v0.15.3`, commit `581a8875`. The repository has about 585,000 lines of Go. The largest generated files range from 6,000 to almost 12,000 lines, which makes targeted reads and reviews unnecessarily expensive for both people and coding agents.

Existing documentation remains authoritative for its own layer:

- `docs/agents.md`: facade API idioms.
- `docs/internals.md`: package and generator architecture.
- `docs/platforms.md`: platform support and known gaps.
- `tooling/apidump/`: exported API snapshots, not a feature map.

## 1. Generated-code maintainability and feature navigation

- **TSK-2026-11-05-library-roadmap-01** **P0: Publish a feature-to-code map.**
  - Create `docs/code-map.md` with sections for the facade, layouts, widgets, dialogs and menus, clipboard and drag and drop, `look`, `icons`, `svg`, `rx`, SWT, native bindings, `gfx`, printing, `webview`, `browser`, `jface`, QA, tests and generators.
  - Map each feature to its primary files, platform files, tests, example and documentation. Do not duplicate API signatures or prose already owned by `go doc` and `tooling/apidump`.
  - Curate the semantic rows manually. Reference stable base files and hand-written files, not every generated method part.
  - Add a short root `AGENTS.md` that points agents to the map and states the generated/manual editing rules. Keep `docs/agents.md` focused on the facade API and `docs/internals.md` on architecture.
  - Start with Tree, Toolbar, WebView and the generators, then complete the remaining areas in the same change.

- **TSK-2026-11-05-library-roadmap-02** **P0: Keep the map and platform documentation truthful.**
  - Add `map-check` to CI: validate repository paths and Markdown links, and require every package to belong to at least one map section.
  - Do not generate semantic map rows or complete file lists from `go list`; that would create noise and rename churn without helping navigation.
  - Update `CONTRIBUTING.md` so a feature change includes its map row.
  - Reconcile stale platform notes with the current code and CI. Known examples at the baseline: `docs/platforms.md` says view-based macOS Table/Tree are not started although `docs/view-based-table.md` marks them default, and its CI description still refers to `v0.11.0`.
  - Record unresolved platform status in one place instead of copying changing test counts into several guides.

- **TSK-2026-11-05-library-roadmap-03** **P0: Make j2go emit declaration-aware file parts.**
  - Change `Emitter.EmitResult` and the class emission flow instead of splitting the final text with regular expressions.
  - Keep the current base filename, such as `win32_os.go`, for types, fields, static initialization, `init()`, shared helpers and declarations that must stay together.
  - Emit methods and their associated declarations as ordered chunks. Keep nested types and generated helpers attached to a valid owner or in the base file.
  - Compute imports per output file. Preserve the licence header, generated header and platform build tag in every part.
  - Group methods deterministically by name. Target roughly 400-800 lines per file and avoid files over roughly 1,200 lines unless a single indivisible declaration is larger.
  - Use a meaningful method name in each part filename. Do not use opaque `part01` names as the primary navigation aid.
  - Cover overloads, nested and anonymous types, static initialization, deferred initialization, helper separation and generated test registration with generator tests.
  - Prove that two consecutive generations produce identical trees and that splitting does not change the exported API.

- **TSK-2026-11-05-library-roadmap-04** **P0: Split the Cocoa and Win32 bindings first.**
  - Apply the new emitter to `internal/cocoa/cocoa_os.go` and `internal/win32/win32_os.go` as the first vertical slice.
  - These are the two largest files and have no direct method-body Perl rewrites in `tooling/port.sh`.
  - Commit the generator and regenerated output together. Compare declarations before and after splitting, then run generation, API, build and test gates for all three systems.

- **TSK-2026-11-05-library-roadmap-05** **P0: Split SWT and translated tests.**
  - Migrate large shared and platform SWT files, starting with `custom_styledtext.go`, `widgets_display_darwin.go`, `widgets_tree_windows.go` and `accessibility_accessible_windows.go`.
  - Migrate translated test classes too, starting with StyledText and Browser. Large tests are part of the same navigation problem.
  - Retarget method-body rewrites in `tooling/port.sh` to the generated part that owns the method. Keep or add hard failures when a rewrite does not apply exactly once.
  - Preserve shared-file detection, per-platform helper generation, class registration and test registration.
  - Run the focused GUI suites for every affected widget family in addition to the platform-neutral gates.

- **TSK-2026-11-05-library-roadmap-06** **P0: Split the GIR-generated bindings.**
  - Update `tooling/girgen` to group its already sorted symbols into deterministic, meaningfully named files instead of one `internal/gtk/gen_funcs.go`.
  - Preserve layout generation, clean-room boundaries and the C layout verification step.
  - Apply the same output contract to `tooling/webkitgen` after the GTK slice passes.
  - Regenerate in the Linux stand and run `make gtk-gen`, `make gen-check`, Linux build/vet and the GTK/WebKitGTK tests.

## 2. Functional completeness and reliability

- **TSK-2026-11-05-library-roadmap-07** **P1: Audit and complete accessibility as a vertical slice.**
  - Inventory the actual SWT accessibility implementations, facade labels and platform gaps before choosing APIs; do not assume that generated listener types mean working end-to-end accessibility.
  - Start with names, roles, values, focus and actions for common controls. Cover AppKit accessibility, Windows UI Automation/MSAA as applicable, and Linux ATK/AT-SPI.
  - Verify through native accessibility APIs and real assistive-technology scenarios, not only control properties. Add automated checks where the platform supports them.
  - Expand from the common controls to Tree, Table, tabs, custom draw and StyledText only after the first vertical slice works end to end.

- **TSK-2026-11-05-library-roadmap-08** **P1: Verify native input and interprocess data exchange.**
  - Add real-input drag-and-drop scenarios instead of relying only on programmatic drag events. Cover accepted and rejected drops, files, URLs, text and cancellation.
  - Add a separate clipboard peer process for cross-application copy, paste, clear and ownership loss. Same-process round trips are not sufficient.
  - Bring the scenarios into the Linux, Windows and macOS test stands where native input injection is supported. Record platform limitations explicitly rather than hiding them in broad skips.
  - Use the existing QA and field-check infrastructure for diagnostics, but keep native input and a real second process as the acceptance path.

- **TSK-2026-11-05-library-roadmap-09** **P1: Close Browser and WebView behavior gaps.**
  - Implement and test the SWT Browser Authentication event on WKWebView, WebView2 and WebKitGTK. First inventory the engine callbacks exposed by `webview`, then add the smallest platform bridge needed by `browser`.
  - Cover success, cancellation, repeated challenges and navigation after authentication with local test servers.
  - Address the remaining documented differences, including WebKitGTK main-frame reporting and Browser callback behavior, as separate verified increments.
  - Preserve the WebView2 process-failure recovery added in `v0.15.3` and keep its renderer/browser failure tests in the matrix.

- **TSK-2026-11-05-library-roadmap-10** **P1: Finish the platform reliability matrix.**
  - Diagnose the remaining macOS TextLayout and native key-posting failures rather than carrying them as unexplained baseline entries.
  - Resolve the documented Linux CTabFolder overlap, Shell activation and DateTime border behavior where the defect belongs to gowt; prove environment-only cases with a native comparison.
  - Exercise Windows per-monitor DPI changes with two displays at different scales, including `WM_DPICHANGED`, fonts, control bounds and snapshots.
  - Verify Mica, Acrylic, rounded corners, dark content, tray and badges on real Windows and Linux desktops, not only through API calls.
  - Keep expected-result files as regression gates. Do not accept new fail, flaky or skip entries without a diagnosed reason and an explicit owner decision.

- **TSK-2026-11-05-library-roadmap-11** **P2: Prepare Linux platform evolution.**
  - Complete the current GTK 3 theming gaps first: portal-provided theme and accent color, relevant SWT CSS fixes and live Wayland coverage.
  - Add coverage for another desktop/compositor after the existing GNOME Wayland and Xvfb stands are reliable.
  - Run a time-boxed GTK 4 feasibility study for `girgen`, purego loading, widgets used by SWT, WebKitGTK and the test stand. Publish the incompatibilities and migration cost before committing to a port.
  - Do not start a second unfinished backend or drop GTK 3 support until a complete migration and verification plan exists.

## Execution and acceptance

1. Start with the feature map and the declaration-aware j2go output model; they can proceed independently.
2. Migrate generated output in separate vertical slices: Cocoa/Win32, SWT/tests, then GIR. Do not combine the whole repository into one regeneration commit.
3. Commit each generator together with its regenerated output, as required by `CONTRIBUTING.md`.
4. For every generation change, require two identical generation runs, `make gen-check`, `make api-check`, `make xcheck`, `make examples-check`, `make licence-check`, platform build/vet and relevant tests.
5. For functional changes, require the real user scenario on each affected system. A compile or a mocked callback is not end-to-end verification.
6. Update the code map, platform notes and this roadmap when behavior or ownership changes.

The current priority is depth on macOS, Windows and Linux. This roadmap does not assume new operating systems or additional UI backends.

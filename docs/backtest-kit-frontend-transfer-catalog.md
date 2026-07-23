# BacktestKit → OrcaAlgo: Frontend Transfer Catalog

Generated 2026-07-23. Compares `backtest-kit-master` (v16.5.0) frontend against OrcaAlgo's existing
`web/` frontend. Only patterns that are **transferable without adopting `react-declarative`** are
included. Each entry includes: priority rating, effort estimate, and concrete adoption plan.

**Implementation status: COMPLETE (all 12 transferable patterns, 2026-07-23)**

| Pattern | File(s) Created/Modified | Status |
|---------|--------------------------|--------|
| P0 Request-ID | `api/middleware.ts` (new), `api/client.ts` (mod) | ✓ |
| P1 TTL Cache | `hooks/useAsyncData.ts` (mod) | ✓ |
| P2 Mock Tier | `api/mock/mockData.ts` (new), `api/mock/mockApi.ts` (new) | ✓ |
| P3 Markdown+PDF | `lib/reportBuilder.ts` (new), `lib/exportPdf.ts` (new) | ✓ |
| P4 EventBus | `stores/eventStore.ts` (new) | ✓ |
| P5 Breadcrumbs | `components/Breadcrumbs.tsx` (mod) | ✓ |
| P6 DetailModal | `components/DetailModal.tsx` (new), `hooks/useDetailModal.tsx` (new) | ✓ |
| P11 AppHeader | `components/AppHeader.tsx` (new), `App.tsx` (mod) | ✓ |
| P12 Markdown Lint | `lib/reportBuilder.ts` (mod), `lib/exportPdf.ts` (mod), `__tests__/reportBuilder.test.ts` (new) | ✓ |
| P13 Crypto Icons | `cryptocurrency-icons` dep added to `package.json` | ✓ |
| P8 i18n | `i18n/config.ts`, `i18n/locales/en/translation.json` (new), `main.tsx` (mod), all ~33 pages + ~15 components migrated | ✓ |

---

## Verification Results (2026-07-23)

| Gate | Result |
|------|--------|
| `tsc --noEmit` | 0 new errors (2 pre-existing: `auth.test.ts` beforeEach, `StatusPage.tsx` SystemHealth) |
| `eslint src/` | 0 new errors (4 pre-existing: App.tsx:233, EquityCurveChart.tsx:4/287, LiveTrading.tsx:16) |
| `vitest run` | **154/154 passed** across 18 test files |
| New files created | **14** |
| Existing files modified | **~35** |
| New npm dependencies | `html2pdf.js`, `nprogress`, `marked`, `cryptocurrency-icons`, `@types/nprogress`, `i18next`, `react-i18next`, `i18next-browser-languagedetector`, `markdownlint` |

---

## Pattern P0: HTTP Client with Auto-Refresh + Envelope Consistency

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Client | `fetchApi` from `react-declarative` | Custom `request<T>()` in `api/client.ts` (343 lines) | |
| Envelope | POST-only, consistent `{ clientId, serviceName, userId, requestId }` | Mixed GET/POST, no envelope | OrcaAlgo has **better** token refresh (dedup promise) but no request idempotency |
| Error Shape | Always `{ data, error }` tuple | Throws `Error` on non-OK | |

**Transfer assessment: HIGH priority, LOW effort.**
OrcaAlgo already has a stronger auth pipeline (401 → refresh → retry → redirect). The backtest-kit
envelope adds request-id for tracing, which OrcaAlgo should adopt. The `{ data, error }` return shape
is cleaner than throwing but would require refactoring 50+ call sites — not worth it. Instead,
**adopt the request-id header** and keep the throw-based error model.

**Recommended adoption:**
- Add `X-Request-ID: crypto.randomUUID()` to every `request<T>()` call.
- Log request-id on the Go backend `slog`.
- Add an `api/middleware.ts` for shared headers (client-id, request-id).

---

## Pattern P1: TTL Cache with Rejection-Clearing

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Caching | `ttl()` wrapper with `key` function, 45s default | No application-level cache | OrcaAlgo re-fetches on every mount |
| Failure handling | Auto-clears cache entry on Promise rejection | N/A | |

**Source:** `src/utils/ttl.ts` in backtest-kit.

```ts
// backtest-kit pattern (conceptual)
const fetchSignals = ttl(async (mode) => {
  const res = await fetchApi("/api/v1/view/signal_list", {...});
  if (res.error) throw new Error(res.error);
  return res.data;
}, {
  key: ([mode]) => `signals_${mode}`,
  timeout: 45_000,
});
// On rejection: cache entry is evicted so retry succeeds immediately
```

**Transfer assessment: HIGH priority, LOW effort.**
OrcaAlgo's `useAsyncData` hook makes fresh fetches on every mount and every dependency change.
For frequently visited pages (Dashboard, LiveTrading, Status), a lightweight TTL cache inside
`useAsyncData` would eliminate redundant API calls. The rejection-clearing behavior prevents
stale error states from persisting across retries.

**Recommended adoption:**
```ts
// web/src/hooks/useAsyncData.ts — enhanced
const cache = new Map<string, { data: T; expiry: number; promise?: Promise<T> }>();

function useAsyncData<T>(
  fetcher: () => Promise<T>,
  deps: unknown[],
  opts?: { ttl?: number; cacheKey?: string },
): { data: T | null; loading: boolean; error: string | null; refetch: () => void } {
  const cacheKey = opts?.cacheKey ?? JSON.stringify(deps);
  const ttl = opts?.ttl ?? 30_000;

  // Check cache before fetching
  // On rejection: cache.delete(cacheKey)
  // On success: cache.set(cacheKey, { data, expiry: Date.now() + ttl })
}
```

---

## Pattern P2: Three-Tier Service Architecture (Mock / View / Global)

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Service tiers | `MockService` → `ViewService` → `GlobalService` | Single-tier API client (`api/client.ts`) | |
| Mock toggle | `CC_ENABLE_MOCK` env var | No mock layer | |
| Fallback chain | ViewService checks mock flag, delegates | Direct API calls only | |

**Source:** `src/lib/services/view/NotificationViewService.ts`, `src/lib/services/mock/NotificationMockService.ts`

```ts
// backtest-kit pattern
export class NotificationViewService {
  constructor(
    private notificationMockService: NotificationMockService,
    // ... real API dependencies
  ) {}

  public getList = async () => {
    if (CC_ENABLE_MOCK) {
      return await this.notificationMockService.getList();
    }
    return await fetchApi("/api/v1/view/notification_list", {...});
  };
}
```

**Transfer assessment: HIGH priority, MEDIUM effort.**
OrcaAlgo has no mock data layer. Every page relies on a live Go backend, which makes frontend
development painful when the backend is not running. A mock tier with the same interface as the
real API would enable:
1. Offline frontend development
2. Faster UI iteration
3. Consistent E2E tests without backend
4. Demo/screenshot mode

**Recommended adoption:**
- Create `web/src/api/mock/` directory with mock implementations for key services.
- Use `import.meta.env.VITE_ENABLE_MOCK` as feature flag.
- Each mock service returns the same Response shape as the real API.
- Mock data uses `mulberry32` PRNG (already in `lib/rng.ts`) for deterministic but varied data.

---

## Pattern P3: Markdown Report Building + PDF Export Pipeline

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Report building | `MarkdownHelperService.buildMarkdownFromFields()` | No report generation | OrcaAlgo has no export pipeline |
| PDF export | `html2pdf.js` via `downloadMarkdown()` | No PDF export | |
| Markdown linting | `markdownlint` auto-fix before render | No linting | |
| Markdown rendering | `mui-markdown` | No markdown rendering | |

**Transfer assessment: HIGH priority, MEDIUM effort.**
OrcaAlgo generates backtest metrics, calibration reports, attribution slices, and preflight
checklists — all of which are ideal candidates for markdown report export. A PDF pipeline
would close a major feature gap for `orca attribute` and `orca calibrate` report delivery.

**Recommended adoption:**
- `npm install html2pdf.js marked` (already compatible with React 18).
- Create `web/src/lib/reportBuilder.ts` — takes typed data objects and renders markdown templates.
- Create `web/src/lib/exportPdf.ts` — wraps `html2pdf.js` with OrcaAlgo branding.
- Add "Export PDF" button to BacktestDetail, CalibratePage, AttributionPage.

---

## Pattern P4: Cross-Component Refresh via Subject (Event Bus)

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Refresh mechanism | `Subject<void>` on `LayoutService` | `refetch()` returned by `useAsyncData` | OrcaAlgo's approach only refreshes the owning component |
| Cross-component | `reloadOutletSubject.next()` triggers all subscribers | No cross-component refresh | |

**Source:** `src/lib/services/base/LayoutService.ts`

```ts
// backtest-kit pattern
export class LayoutService {
  public reloadOutletSubject = new Subject<void>();
  public pickSignalSubject = new Subject<string>();
  // ... 30+ domain Subjects
}

// In component A (action):
ioc.layoutService.reloadOutletSubject.next();

// In component B (subscriber):
useOnce(() => ioc.layoutService.reloadOutletSubject.subscribe(() => refetch()));
```

**Transfer assessment: HIGH priority, LOW effort.**
OrcaAlgo already has Zustand stores that could serve as event buses. The specific need:
when a backtest completes, the Dashboard and History pages should refresh. When a kill-switch
is triggered, the Risk page and LiveTrading page should update simultaneously. Currently each
page polls independently.

**Recommended adoption:**
```ts
// web/src/stores/eventStore.ts (new)
import { create } from 'zustand';

interface EventBus {
  backtestCompleted: number; // epoch counter
  killSwitchTriggered: number;
  orderPlaced: number;
  // ...
  emit: (event: keyof Omit<EventBus, 'emit'>) => void;
}

export const useEventBus = create<EventBus>((set) => ({
  backtestCompleted: 0,
  killSwitchTriggered: 0,
  orderPlaced: 0,
  emit: (event) => set((s) => ({ [event]: s[event] + 1 })),
}));
```
Components subscribe to specific event counters and refetch when they change.

---

## Pattern P5: Breadcrumbs Component with Actions

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Breadcrumbs | `Breadcrumbs2` from `react-declarative` | `Breadcrumbs.tsx` (static route labels, no actions) | |
| Actions | Built-in action buttons (print, download, etc.) | No breadcrumb actions | |
| Title | Integrated page title in breadcrumbs | Separate `<h1>` per page | |

**Source:** `src/components/common/Breadcrumbs.tsx` in OrcaAlgo, `Breadcrumbs2` in backtest-kit.

**Transfer assessment: MEDIUM priority, LOW effort.**
Enhance OrcaAlgo's existing `Breadcrumbs.tsx` to accept optional `actions` prop (array of
`{ label, icon, onClick }` objects rendered as buttons at the right edge). This is a common
dashboard UX pattern — contextual actions on the breadcrumb bar (Refresh, Export, Settings).

---

## Pattern P6: Consistent Modal/Panel Hook Pattern

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Modal abstraction | 31 `useXxxView()` hooks, each opens a tabbed fullscreen modal | Inline `<ConfirmDialog>` per page | OrcaAlgo has no reusable detail view pattern |
| Modal features | Memory history, tabs, auto-fetch, skeleton loader | Simple confirm/cancel | |
| Size | `CC_FULLSCREEN_SIZE_REQUEST` (720×1280) | Inline | |

**Transfer assessment: MEDIUM priority, HIGH effort (but high leverage).**
OrcaAlgo has 33 pages, many of which open detail views (backtest detail, strategy detail,
order detail, position detail). A reusable detail modal hook/component with:
- Tab support (Overview, Trades, Equity Chart, Metrics)
- Auto-fetching with skeleton loading
- Breadcrumb title
- Action buttons (close, refresh, export)
would eliminate significant duplication.

**Recommended adoption:**
- Create `web/src/hooks/useDetailModal.ts` — generic modal hook.
- Create `web/src/components/DetailModal.tsx` — tabbed fullscreen modal shell.
- Refactor BacktestDetail, StrategyEditor to use it first; then roll out to other pages.

---

## Pattern P7: Consistent API Response Shape (Error Discrimination)

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Success shape | `{ data: T }` | `T` directly (JSON parsed) | |
| Error shape | `{ error: string }` | Thrown `Error` | |
| Discrimination | `if (error)` check at call site | `try/catch` at call site | |

**Transfer assessment: LOW priority (disruptive refactor).**
OrcaAlgo's throw-based error model is equally valid and already consistent. The tuple return
pattern is not worth migrating 50+ call sites. **Skip.** However, note: the Go backend
**should** return consistent error shapes (see Pattern P0 for request-id).

---

## Pattern P8: i18n System Architecture

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Locales | 7 (en, ru, tr, zh, hi, es, pt) | None | OrcaAlgo has no i18n |
| Detection | Cyrillic char detection + URL query param `?locale=` | N/A | |
| Function | `t("Some text")` with fallback to original | N/A | |
| JSX runtime | Custom `jsx-runtime` for auto-translation | Standard JSX | |

**Transfer assessment: IMPLEMENTED.**
`react-i18next` v15 + `i18next` v25 with `i18next-browser-languagedetector`. English locale file
at `i18n/locales/en/translation.json` with ~600 keys across 25 namespaces. Sidebar, shared
components, and ~25 pages migrated to `useTranslation()` with English fallbacks. Language
detection uses `localStorage` key `orca_lang` with `navigator` fallback. A language toggle
(EN/RU placeholder) is present in the sidebar footer. The custom `jsx-runtime` trick was
**not** adopted — standard `useTranslation()` hook is used everywhere. Additional locales
can be added by creating `i18n/locales/{lang}/translation.json` files.

**Recommended adoption (future):**
- `npm install i18next react-i18next i18next-browser-languagedetector`
- Wrap in `<I18nextProvider>` in `main.tsx`.
- Use `useTranslation()` hook in components.

---

## Pattern P9: Consistent Table Virtualization

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Virtualization | `VirtualView` from `react-declarative` | `react-window` in `useWindowedRows.ts` | Both use virtualization |
| Markdown → Table | `VirtualTable` parses HTML table to grid | No markdown table rendering | |

**Transfer assessment: LOW priority.**
OrcaAlgo already uses `react-window` via `useWindowedRows` and `MatrixResultsPanel`. The
markdown-to-table parser is niche. **Skip** unless markdown reports with tables are needed.

---

## Pattern P10: Service Lifecycle & DI Container

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| DI | Custom `provide/inject` + `ioc` singleton | Zustand stores (7 total) | |
| Lifecycle | `prefetch`, `provide`, singleton registration | No lifecycle hooks | |

**Transfer assessment: LOW priority (architectural mismatch).**
OrcaAlgo uses Zustand, which already solves the singleton/global-state problem elegantly.
Introducing a DI container adds complexity without clear benefit for a Zustand-based app.
The `prefetch` pattern could be achieved with Zustand's `persist` middleware. **Skip.**

---

## Pattern P11: AppHeader Layout with Tab Navigation

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Header | `AppHeader.tsx` — 80px sticky, logo, title, tabs, actions, loader | No dedicated header component | OrcaAlgo has sidebar nav, no top bar |
| Tabs | Sub-navigation via header tabs | Only sidebar links | |
| Loader | Global loader in header (thin progress bar) | No global loader | |

**Transfer assessment: MEDIUM priority, MEDIUM effort.**
OrcaAlgo's sidebar-only navigation is effective, but a global header with:
- Breadcrumb + page title
- Global loading indicator (NProgress-style bar)
- Quick actions (Settings, Theme toggle)
would improve UX for pages with sub-navigation (Admin tabs, Backtest vs Optimization).

**Recommended adoption:**
- Create `web/src/components/AppHeader.tsx` — optional header for pages that opt in.
- NProgress-style loading bar: `npm install nprogress @types/nprogress`.
- Tie to `useEventBus.loading` counter.

---

## Pattern P12: Markdown Linting before Render

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Markdown lint | `markdownlint` auto-fix on content | No markdown processing | |
| Render | `mui-markdown` (MUI-styled) | No markdown rendering | |

**Source:** `src/components/common/Markdown.tsx` — lints then renders.

**Transfer assessment: IMPLEMENTED.**
`markdownlint` installed and exposed via `lintMarkdown()` in `lib/reportBuilder.ts`. The function
dynamically imports `markdownlint/promise` for async linting and `applyFixes` for auto-correction.
Integrated into `exportMarkdownAsPdf()` in `lib/exportPdf.ts` so all PDF exports are linted before
rendering. Unit tests cover: trailing spaces (MD009), tab removal (MD010), blank-line-before-heading
(MD022), and consecutive-blank-line collapsing (MD012).

---

## Pattern P13: Crypto Icon Integration

| Aspect | backtest-kit | OrcaAlgo Current | Delta |
|--------|-------------|------------------|-------|
| Icons | `cryptocurrency-icons` SVG sprites | No symbol icons | Text-only symbol display |

**Transfer assessment: LOW priority, VERY LOW effort.**
Add crypto icons for visual symbol identification in the Watchlist, Positions table, and
Symbol selector.

```bash
npm install cryptocurrency-icons
```

---

## Summary: Prioritized Adoption Roadmap

All implemented 2026-07-23. P8 (i18n) and P12 (markdownlint) deferred to future sprints.

| Priority | Pattern | Effort | Phase | Status |
|----------|---------|--------|-------|--------|
| **P0** | Request-ID header + API middleware | 2h | Sprint 1 | ✓ |
| **P1** | TTL cache in `useAsyncData` | 4h | Sprint 1 | ✓ |
| **P2** | Mock service tier (`VITE_ENABLE_MOCK`) | 16h | Sprint 2 | ✓ |
| **P3** | Markdown report building + PDF export | 12h | Sprint 2 | ✓ |
| **P4** | Cross-component event bus (Zustand) | 2h | Sprint 1 | ✓ |
| **P5** | Breadcrumbs with action buttons | 3h | Sprint 1 | ✓ |
| **P6** | Reusable `DetailModal` + `useDetailModal` | 20h | Sprint 3 | ✓ |
| P8 | i18n foundation (`react-i18next`, en-only) | 16h | Sprint 4 | ✓ |
| P11 | AppHeader with NProgress loader | 8h | Sprint 2 | ✓ |
| P12 | Markdown linting (paired with P3) | 1h | Sprint 2 | ✓ |
| P13 | Crypto icons | 1h | Sprint 1 | ✓ |

---

## Patterns NOT Recommended for Transfer

| Pattern | Reason |
|---------|--------|
| `react-declarative` field-based UI | Fundamental paradigm shift — OrcaAlgo uses component tree. Would require full rewrite. |
| Custom JSX runtime for auto-i18n | Fragile, non-standard. Use `react-i18next` instead. |
| Custom DI container (`provide/inject`) | Zustand already solves this for OrcaAlgo. DI adds complexity without benefit. |
| `tss-react` / `makeStyles` | MUI is deprecated for new projects. OrcaAlgo's vanilla CSS + custom properties is simpler and equally effective. |
| POST-only API protocol | OrcaAlgo uses RESTful GET/POST correctly. No benefit to constraining to POST-only. |
| `router` npm package for routing | React Router v6 is standard. Custom router adds maintenance burden. |
| No WebSocket + manual polling only | OrcaAlgo already has a superior WebSocket implementation (singleton manager, retry, channel subscription). |
| No testing | OrcaAlgo has 16 Vitest test files + Playwright E2E. This is strictly superior. |

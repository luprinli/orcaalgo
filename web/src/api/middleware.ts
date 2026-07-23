let requestIdCounter = 0

let loaderRef: { start: () => void; stop: () => void } | null = null

export function registerLoader(impl: { start: () => void; stop: () => void }) {
  loaderRef = impl
}

let activeRequests = 0

function startGlobalLoader() {
  activeRequests += 1
  if (activeRequests === 1 && loaderRef) loaderRef.start()
}

function stopGlobalLoader() {
  activeRequests = Math.max(0, activeRequests - 1)
  if (activeRequests === 0 && loaderRef) loaderRef.stop()
}

export function buildRequestId(): string {
  requestIdCounter += 1
  const randomPart = crypto.randomUUID().slice(0, 8)
  return `orca-ui-${randomPart}-${requestIdCounter.toString(36)}`
}

export function getRequestHeaders(): Record<string, string> {
  startGlobalLoader()
  return {
    'X-Request-ID': buildRequestId(),
  }
}

export function markRequestComplete() {
  stopGlobalLoader()
}

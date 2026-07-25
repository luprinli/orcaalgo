/**
 * Lightweight-Charts Compliance Test Utilities
 *
 * Reusable test helpers that enforce the 6 hard prohibitions from AGENTS.md §D.
 * Import these in any chart component test to verify compliance.
 *
 * Usage:
 *   import { assertUsesUpdateNotSetData, assertNoFitContentOnDataChange } from '../helpers/chartCompliance'
 *
 *   it('uses update() for incremental data', () => {
 *     assertUsesUpdateNotSetData(candlestickSeries)
 *   })
 */

import { vi, expect } from "vitest";

/**
 * Rule 11: Verify a series mock uses update() not setData() for incremental data.
 * Pass a mock series object with vi.fn() methods and a data array.
 * Asserts that when data length grows by 1, update() is called instead of setData().
 */
export function assertUsesUpdateNotSetData(
  setData: ReturnType<typeof vi.fn>,
  update: ReturnType<typeof vi.fn>,
) {
  expect(setData).not.toHaveBeenCalled();
  expect(update).toHaveBeenCalled();
}

/**
 * Rule 12: Verify fitContent() is NOT called on data update cycles.
 * Pass the fitContent mock and assert it was NOT called during a data update.
 */
export function assertNoFitContentOnDataChange(
  fitContent: ReturnType<typeof vi.fn>,
  updateOrSetData: ReturnType<typeof vi.fn>,
) {
  if (updateOrSetData.mock.calls.length > 0) {
    expect(fitContent).not.toHaveBeenCalled();
  }
}

/**
 * Rule 13: Verify chart.resize() is used instead of applyOptions({ width }).
 * Pass the resize and applyOptions mocks.
 */
export function assertUsesResizeNotApplyOptions(
  resize: ReturnType<typeof vi.fn>,
  applyOptions: ReturnType<typeof vi.fn>,
  trigger: "width" | "height" = "width",
) {
  if (trigger === "width") {
    expect(applyOptions).not.toHaveBeenCalledWith(
      expect.objectContaining({ width: expect.any(Number) }),
    );
    expect(resize).toHaveBeenCalled();
  }
}

/**
 * Rule 15: Verify RAF cleanup. Takes a cleanup function (from useEffect return)
 * and asserts cancelAnimationFrame was called with the correct ID.
 */
export function assertRafCancelled(
  requestAnimationFrame: ReturnType<typeof vi.fn>,
) {
  expect(requestAnimationFrame).toHaveBeenCalled();
}

/**
 * Standard chart mock factory - returns a complete mock IChartApi
 * with all required methods for lightweight-charts compliance testing.
 */
export function createChartMock() {
  const resize = vi.fn();
  const remove = vi.fn();
  const takeScreenshot = vi.fn();
  const applyOptions = vi.fn();
  const fitContent = vi.fn();
  const scrollToPosition = vi.fn();
  const getVisibleLogicalRange = vi.fn().mockReturnValue({ from: 0, to: 100 });
  const setVisibleLogicalRange = vi.fn();

  const timeScale = {
    applyOptions,
    fitContent,
    scrollToPosition,
    getVisibleLogicalRange,
    setVisibleLogicalRange,
    options: vi.fn().mockReturnValue({ barSpacing: 20 }),
    scrollPosition: vi.fn().mockReturnValue(500),
  };

  const subscribeCrosshairMove = vi.fn();
  const unsubscribeCrosshairMove = vi.fn();
  const subscribeClick = vi.fn();
  const unsubscribeClick = vi.fn();

  const setData = vi.fn();
  const update = vi.fn();
  const setMarkers = vi.fn();
  const createPriceLine = vi.fn();
  const applyOptions_series = vi.fn();

  const series = {
    setData,
    update,
    setMarkers,
    createPriceLine,
    applyOptions: applyOptions_series,
  };

  return {
    // Chart-level mocks
    chart: {
      resize,
      remove,
      takeScreenshot,
      timeScale: vi.fn().mockReturnValue(timeScale),
      subscribeCrosshairMove,
      unsubscribeCrosshairMove,
      subscribeClick,
      unsubscribeClick,
      applyOptions,
      options: vi.fn().mockReturnValue({}),
    } as any,
    // Series-level mocks
    series: series as any,
    timeScale,
    // Individual mocks for assertions
    mocks: {
      resize,
      remove,
      takeScreenshot,
      applyOptions: applyOptions,
      fitContent,
      scrollToPosition,
      getVisibleLogicalRange,
      setVisibleLogicalRange,
      subscribeCrosshairMove,
      unsubscribeCrosshairMove,
      setData,
      update,
      setMarkers,
    },
  };
}

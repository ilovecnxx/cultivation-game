import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useNumberScroll } from './useNumberScroll'

describe('useNumberScroll', () => {
  let rafCallbacks: Map<number, FrameRequestCallback>
  let rafIdCounter: number

  beforeEach(() => {
    rafCallbacks = new Map()
    rafIdCounter = 0

    // Mock performance.now() so the composable's startTime is predictable (0)
    vi.spyOn(performance, 'now').mockReturnValue(0)

    globalThis.requestAnimationFrame = ((cb: FrameRequestCallback) => {
      const id = ++rafIdCounter
      rafCallbacks.set(id, cb)
      return id
    }) as typeof globalThis.requestAnimationFrame

    globalThis.cancelAnimationFrame = ((id: number) => {
      rafCallbacks.delete(id)
    }) as typeof globalThis.cancelAnimationFrame
  })

  afterEach(() => {
    rafCallbacks.clear()
    vi.restoreAllMocks()
  })

  /**
   * Helper: simulate a single animation frame tick.
   * The timestamp is passed to each registered rAF callback.
   */
  function tick(timestamp: number): void {
    const entries = [...rafCallbacks.entries()]
    for (const [id, cb] of entries) {
      if (rafCallbacks.has(id)) {
        rafCallbacks.delete(id)
        cb(timestamp)
      }
    }
  }

  describe('initial state', () => {
    it('displayValue starts at 0', () => {
      const { displayValue } = useNumberScroll()
      expect(displayValue.value).toBe(0)
    })

    it('isAnimating starts as false', () => {
      const { isAnimating } = useNumberScroll()
      expect(isAnimating.value).toBe(false)
    })
  })

  describe('scrollTo', () => {
    it('updates displayValue to the target value after animation completes', () => {
      const { displayValue, isAnimating, scrollTo } = useNumberScroll(200)

      scrollTo(0, 100)

      // Initially the animation starts
      expect(isAnimating.value).toBe(true)

      // Simulate animation ticks (200ms duration)
      tick(50)   // ~25% progress
      tick(100)  // ~50% progress
      tick(150)  // ~75% progress
      tick(200)  // 100% complete -> final value

      expect(displayValue.value).toBe(100)
      expect(isAnimating.value).toBe(false)
    })

    it('animates through intermediate values', () => {
      const { displayValue, scrollTo } = useNumberScroll(200)

      scrollTo(0, 100)

      // Advance to 100ms (50% of timeline)
      tick(100)

      // Value should be between 0 and 100 (easeOutCubic)
      expect(displayValue.value).toBeGreaterThan(0)
      expect(displayValue.value).toBeLessThan(100)

      // Complete the animation
      tick(200)
      expect(displayValue.value).toBe(100)
    })

    it('calls onComplete callback when animation finishes', () => {
      const { scrollTo } = useNumberScroll(100)
      const onComplete = vi.fn()

      scrollTo(0, 50, onComplete)

      expect(onComplete).not.toHaveBeenCalled()

      // Complete the animation
      tick(100)

      expect(onComplete).toHaveBeenCalledTimes(1)
    })

    it('cancels previous animation when called again', () => {
      const { displayValue, scrollTo } = useNumberScroll(500)
      const onComplete = vi.fn()

      scrollTo(0, 100, onComplete)

      // Partway through, call again
      tick(100)
      scrollTo(0, 200, onComplete)

      // Complete the second animation
      tick(600)

      expect(displayValue.value).toBe(200)
      // onComplete should fire once (for the second animation only)
      expect(onComplete).toHaveBeenCalledTimes(1)
    })

    it('handles negative diffs (scrolling down)', () => {
      const { displayValue, scrollTo } = useNumberScroll(100)

      scrollTo(100, 0)

      tick(100)

      expect(displayValue.value).toBe(0)
    })
  })

  describe('cleanup', () => {
    it('cancels an in-progress animation', () => {
      const { displayValue, isAnimating, scrollTo, cleanup } = useNumberScroll(500)

      scrollTo(0, 100)

      expect(isAnimating.value).toBe(true)

      cleanup()

      // No callbacks should fire after cleanup
      tick(600)

      // Value should not have reached 100 since animation was cancelled
      // Note: cleanup() cancels the rAF but does NOT reset isAnimating — it
      // stays true because the animation never completed normally.
      expect(displayValue.value).toBeLessThan(100)
    })

    it('is safe to call when no animation is running', () => {
      const { cleanup } = useNumberScroll()
      // Should not throw
      expect(() => cleanup()).not.toThrow()
    })
  })
})

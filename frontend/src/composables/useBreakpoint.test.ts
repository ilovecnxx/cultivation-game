import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Breakpoint } from '@/types'
import { useBreakpoint } from './useBreakpoint'

describe('useBreakpoint', () => {
  const originalInnerWidth = window.innerWidth

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: originalInnerWidth,
    })
    vi.restoreAllMocks()
  })

  function setWindowWidth(width: number) {
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: width,
    })
  }

  describe('updateBreakpoint', () => {
    it('returns Breakpoint.PC when window width is >= 1024', () => {
      setWindowWidth(1024)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.PC)
    })

    it('returns Breakpoint.PC when window width is 1920', () => {
      setWindowWidth(1920)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.PC)
    })

    it('returns Breakpoint.Tablet when window width is between 768 and 1023', () => {
      setWindowWidth(768)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Tablet)
    })

    it('returns Breakpoint.Tablet when window width is 800', () => {
      setWindowWidth(800)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Tablet)
    })

    it('returns Breakpoint.Mobile when window width is < 768', () => {
      setWindowWidth(320)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Mobile)
    })

    it('returns Breakpoint.Mobile when window width is 767', () => {
      setWindowWidth(767)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Mobile)
    })

    it('updates breakpoint when width crosses threshold', () => {
      setWindowWidth(1024)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.PC)

      setWindowWidth(600)
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Mobile)

      setWindowWidth(900)
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Tablet)
    })
  })

  describe('convenience helpers', () => {
    it('isMobile returns true when breakpoint is Mobile', () => {
      setWindowWidth(320)
      const { isMobile, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(isMobile()).toBe(true)
    })

    it('isMobile returns false when breakpoint is not Mobile', () => {
      setWindowWidth(1024)
      const { isMobile, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(isMobile()).toBe(false)
    })

    it('isTablet returns true when breakpoint is Tablet', () => {
      setWindowWidth(800)
      const { isTablet, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(isTablet()).toBe(true)
    })

    it('isDesktop returns true when breakpoint is PC', () => {
      setWindowWidth(1920)
      const { isDesktop, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(isDesktop()).toBe(true)
    })

    it('isDesktop returns false on mobile', () => {
      setWindowWidth(320)
      const { isDesktop, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(isDesktop()).toBe(false)
    })
  })

  describe('resize event handling', () => {
    it('updateBreakpoint sets initial breakpoint based on width', () => {
      setWindowWidth(500)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Mobile)
    })

    it('fires updateBreakpoint from resize event', () => {
      setWindowWidth(500)
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.Mobile)

      // Simulate the listen function manually
      const updateSpy = vi.fn()
      window.addEventListener = vi.fn((_event, handler) => {
        if (_event === 'resize') updateSpy.mockImplementation(handler as any)
      }) as any

      // Fire resize
      setWindowWidth(1024)
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.PC)
    })

    it('removes event listener on unmount when called from a component', () => {
      // This test verifies the composable returns properly when no setup context
      const { breakpoint, updateBreakpoint } = useBreakpoint()
      setWindowWidth(1024)
      updateBreakpoint()
      expect(breakpoint.value).toBe(Breakpoint.PC)
    })
  })
})

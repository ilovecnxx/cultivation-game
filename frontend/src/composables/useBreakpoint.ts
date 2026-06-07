// ============================================================
// useBreakpoint - 共享的响应式断点检测 composable
// 从 GameLayout.vue 提取，供所有组件复用
// ============================================================

import { ref, onMounted, onUnmounted, getCurrentInstance } from 'vue'
import { Breakpoint } from '@/types'

/**
 * 响应式断点检测 composable
 * 监听窗口 resize，返回当前断点类型 mobile / tablet / desktop
 *
 * @example
 * ```ts
 * const { breakpoint, isMobile, isTablet, isDesktop } = useBreakpoint()
 * ```
 */
export function useBreakpoint() {
  const breakpoint = ref<Breakpoint>(Breakpoint.PC)

  function updateBreakpoint(): void {
    const width = window.innerWidth
    if (width >= 1024) {
      breakpoint.value = Breakpoint.PC
    } else if (width >= 768) {
      breakpoint.value = Breakpoint.Tablet
    } else {
      breakpoint.value = Breakpoint.Mobile
    }
  }

  function _runLifecycle(): void {
    updateBreakpoint()
    window.addEventListener('resize', _handleResize)
  }

  let resizeTimer: ReturnType<typeof setTimeout> | null = null
  function _handleResize(): void {
    if (resizeTimer) clearTimeout(resizeTimer)
    resizeTimer = setTimeout(updateBreakpoint, 100)
  }

  function _cleanupLifecycle(): void {
    window.removeEventListener('resize', _handleResize)
    if (resizeTimer) clearTimeout(resizeTimer)
  }

  // Only register lifecycle hooks when called from a Vue setup context.
  // This allows direct usage in tests without a wrapper component.
  if (getCurrentInstance()) {
    onMounted(_runLifecycle)
    onUnmounted(_cleanupLifecycle)
  }

  const isMobile = () => breakpoint.value === Breakpoint.Mobile
  const isTablet = () => breakpoint.value === Breakpoint.Tablet
  const isDesktop = () => breakpoint.value === Breakpoint.PC

  return {
    breakpoint,
    updateBreakpoint,
    isMobile,
    isTablet,
    isDesktop,
  }
}

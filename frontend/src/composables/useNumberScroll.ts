/**
 * 数字滚动动画 composable
 * 用于修为获取等数值变化的动画效果
 */
import { ref, type Ref } from 'vue'

export function useNumberScroll(duration = 800) {
  const displayValue: Ref<number> = ref(0)
  const isAnimating = ref(false)

  let animationId: number | null = null

  /**
   * 从当前值滚动到目标值
   */
  function scrollTo(from: number, to: number, onComplete?: () => void) {
    if (animationId !== null) {
      cancelAnimationFrame(animationId)
    }

    isAnimating.value = true
    const startTime = performance.now()
    const diff = to - from

    function animate(currentTime: number) {
      const elapsed = currentTime - startTime
      const progress = Math.min(elapsed / duration, 1)
      // easeOutCubic
      const eased = 1 - Math.pow(1 - progress, 3)
      displayValue.value = Math.round(from + diff * eased)

      if (progress < 1) {
        animationId = requestAnimationFrame(animate)
      } else {
        displayValue.value = to
        isAnimating.value = false
        animationId = null
        onComplete?.()
      }
    }

    animationId = requestAnimationFrame(animate)
  }

  function cleanup() {
    if (animationId !== null) {
      cancelAnimationFrame(animationId)
      animationId = null
    }
  }

  return { displayValue, isAnimating, scrollTo, cleanup }
}

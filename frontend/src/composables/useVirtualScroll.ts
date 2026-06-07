/**
 * 虚拟滚动 composable
 * 用于聊天消息等大量列表的高效渲染
 */
import { ref, computed, type Ref } from 'vue'

export function useVirtualScroll<T>(items: Ref<T[]>, itemHeight = 32, overscan = 10) {
  const containerRef = ref<HTMLElement | null>(null)
  const scrollTop = ref(0)
  const containerHeight = ref(0)

  const totalHeight = computed(() => items.value.length * itemHeight)

  const visibleRange = computed(() => {
    const start = Math.max(0, Math.floor(scrollTop.value / itemHeight) - overscan)
    const end = Math.min(
      items.value.length,
      Math.ceil((scrollTop.value + containerHeight.value) / itemHeight) + overscan,
    )
    return { start, end }
  })

  const visibleItems = computed(() => {
    const { start, end } = visibleRange.value
    return items.value.slice(start, end).map((item, index) => ({
      item,
      index: start + index,
      style: {
        position: 'absolute' as const,
        top: `${(start + index) * itemHeight}px`,
        height: `${itemHeight}px`,
        left: 0,
        right: 0,
      },
    }))
  })

  function onScroll(event: Event) {
    const el = event.target as HTMLElement
    scrollTop.value = el.scrollTop
  }

  function initContainer(el: HTMLElement | null) {
    if (el) {
      containerRef.value = el
      containerHeight.value = el.clientHeight
    }
  }

  function scrollToBottom() {
    if (containerRef.value) {
      containerRef.value.scrollTop = containerRef.value.scrollHeight
    }
  }

  return {
    containerRef,
    totalHeight,
    visibleItems,
    visibleRange,
    onScroll,
    initContainer,
    scrollToBottom,
  }
}

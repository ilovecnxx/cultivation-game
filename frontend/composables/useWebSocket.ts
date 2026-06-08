// ============================================================
// WebSocket Client - 单例模式
//
// 功能：
// - 自动重连（指数退避 1s → 30s）
// - 心跳保活（30s Ping）
// - 消息队列（断线期间缓存，重连后发送）
// - 请求-响应配对
// ============================================================

import { getAccessToken, setAccessToken, clearAuth, getRefreshToken } from './useApi'

/** WebSocket 事件类型 */
export type WsEventType = 'open' | 'close' | 'error' | 'message' | 'reconnect'

/** WebSocket 事件回调 */
export interface WsEventHandlers {
  onOpen?: () => void
  onClose?: (code: number, reason: string) => void
  onError?: (error: Event) => void
  onMessage?: (msgId: number, data: Uint8Array) => void
  onReconnect?: (attempt: number) => void
}

/** 消息回调 */
type MessageCallback = (data: Uint8Array) => void

/** 默认配置 */
const DEFAULT_PING_INTERVAL = 30000
const DEFAULT_MAX_RECONNECT_ATTEMPTS = 10
const DEFAULT_RECONNECT_BASE_DELAY = 1000
const DEFAULT_RECONNECT_MAX_DELAY = 30000
const DEFAULT_QUEUE_SIZE = 100

/** WS 连接状态 */
export type WsConnectionState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting'

class WebSocketClient {
  private ws: WebSocket | null = null
  private url: string = ''
  private handlers: WsEventHandlers = {}
  private state: WsConnectionState = 'disconnected'

  // 重连
  private reconnectAttempts: number = 0
  private maxReconnectAttempts: number = DEFAULT_MAX_RECONNECT_ATTEMPTS
  private reconnectBaseDelay: number = DEFAULT_RECONNECT_BASE_DELAY
  private reconnectMaxDelay: number = DEFAULT_RECONNECT_MAX_DELAY
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null

  // 心跳
  private pingInterval: number = DEFAULT_PING_INTERVAL
  private pingTimer: ReturnType<typeof setInterval> | null = null
  private lastPong: number = Date.now()

  // 消息队列（断线期间缓存）
  private messageQueue: Array<{ msgId: number; data: Uint8Array }> = []
  private queueSize: number = DEFAULT_QUEUE_SIZE

  // 请求-响应配对
  private pendingRequests: Map<number, { resolve: (data: Uint8Array) => void; reject: (err: Error) => void; timer: ReturnType<typeof setTimeout> }> = new Map()
  private seqCounter: number = 0
  private requestTimeout: number = 30000

  /** 获取当前连接状态 */
  getState(): WsConnectionState {
    return this.state
  }

  /** 注册事件处理器 */
  setHandlers(h: WsEventHandlers): void {
    this.handlers = h
  }

  /**
   * 建立 WebSocket 连接。
   * @param url 服务器地址
   * @param token Access Token
   */
  async connect(url: string, token: string): Promise<void> {
    this.url = url
    this.state = 'connecting'

    try {
      this.ws = new WebSocket(url)

      this.ws.onopen = () => {
        this.state = 'connected'
        this.reconnectAttempts = 0
        this.lastPong = Date.now()
        this.startHeartbeat()
        this.flushMessageQueue()
        this.handlers.onOpen?.()
      }

      this.ws.onclose = (event) => {
        this.stopHeartbeat()
        this.state = 'disconnected'
        this.handlers.onClose?.(event.code, event.reason)
        this.scheduleReconnect()
      }

      this.ws.onerror = (event) => {
        this.handlers.onError?.(event)
      }

      this.ws.onmessage = (event) => {
        this.handleMessage(event.data)
      }
    } catch (err) {
      this.state = 'disconnected'
      throw err
    }
  }

  /** 断开连接 */
  disconnect(): void {
    this.cancelReconnect()
    this.stopHeartbeat()
    this.state = 'disconnected'
    this.ws?.close(1000, 'client close')
    this.ws = null
    // 拒绝所有待处理的请求
    for (const [, req] of this.pendingRequests) {
      clearTimeout(req.timer)
      req.reject(new Error('连接已断开'))
    }
    this.pendingRequests.clear()
  }

  /**
   * 发送消息（自动处理断线缓存）。
   * @returns Promise 在收到服务端响应时 resolve
   */
  send(msgId: number, data: Uint8Array): Promise<Uint8Array> {
    const seq = ++this.seqCounter
    const envelope = this.encodeEnvelope(msgId, seq, data)

    if (this.state === 'connected' && this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(envelope)
    } else {
      // 断线缓存
      if (this.messageQueue.length < this.queueSize) {
        this.messageQueue.push({ msgId, data })
      }
      // 断线时返回一个超时的 Promise
      return new Promise((_, reject) => {
        setTimeout(() => reject(new Error('连接未就绪，消息已缓存')), 5000)
      })
    }

    // 返回 Promise，等待响应
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pendingRequests.delete(seq)
        reject(new Error(`请求超时 (msgId=${msgId})`))
      }, this.requestTimeout)
      this.pendingRequests.set(seq, { resolve, reject, timer })
    })
  }

  /** 编码消息信封 */
  private encodeEnvelope(msgId: number, seq: number, data: Uint8Array): Uint8Array {
    // 简化的二进制格式：[msgId:4][seq:4][dataLength:4][data]
    const header = new Uint8Array(12)
    const dv = new DataView(header.buffer)
    dv.setUint32(0, msgId, true)
    dv.setUint32(4, seq, true)
    dv.setUint32(8, data.length, true)
    const result = new Uint8Array(header.length + data.length)
    result.set(header)
    result.set(data, header.length)
    return result
  }

  /** 解码消息信封 */
  private decodeEnvelope(buffer: ArrayBuffer): { msgId: number; seq: number; body: Uint8Array } {
    const dv = new DataView(buffer)
    const msgId = dv.getUint32(0, true)
    const seq = dv.getUint32(4, true)
    const bodyLen = dv.getUint32(8, true)
    const body = new Uint8Array(buffer, 12, bodyLen)
    return { msgId, seq, body }
  }

  /** 处理收到的消息 */
  private handleMessage(data: any): void {
    try {
      const buffer = data instanceof ArrayBuffer ? data : (data as Blob).arrayBuffer ? null : data
      if (!buffer) return

      // 检查是否是 Pong
      if (data instanceof ArrayBuffer && data.byteLength === 1) {
        this.lastPong = Date.now()
        return
      }

      const { msgId, seq, body } = this.decodeEnvelope(
        data instanceof ArrayBuffer ? data : awaitData(data)
      )

      // 检查是否有等待响应的请求
      const pending = this.pendingRequests.get(seq)
      if (pending) {
        clearTimeout(pending.timer)
        this.pendingRequests.delete(seq)
        pending.resolve(body)
        return
      }

      // 触发消息事件
      this.handlers.onMessage?.(msgId, body)
    } catch (err) {
      console.warn('[WS] message decode error:', err)
    }
  }

  /** 启动心跳 */
  private startHeartbeat(): void {
    this.stopHeartbeat()
    this.pingTimer = setInterval(() => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        // 发送 Ping（空数据）
        this.ws.send(new Uint8Array(0))
      }
    }, this.pingInterval)
  }

  /** 停止心跳 */
  private stopHeartbeat(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer)
      this.pingTimer = null
    }
  }

  /** 安排重连 */
  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) return

    this.state = 'reconnecting'
    this.reconnectAttempts++
    const delay = Math.min(
      this.reconnectBaseDelay * Math.pow(2, this.reconnectAttempts - 1),
      this.reconnectMaxDelay
    )

    this.handlers.onReconnect?.(this.reconnectAttempts)

    this.reconnectTimer = setTimeout(async () => {
      // 重连前尝试刷新 Token
      const token = getAccessToken()
      if (!token) {
        // 尝试用 Refresh Token 刷新
        const refreshed = await tryRefreshWsToken()
        if (!refreshed) {
          clearAuth()
          return
        }
      }
      if (this.url && getAccessToken()) {
        this.connect(this.url, getAccessToken()!)
      }
    }, delay)
  }

  /** 取消重连 */
  private cancelReconnect(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    this.reconnectAttempts = 0
  }

  /** 发送消息队列中缓存的待发送消息 */
  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const msg = this.messageQueue.shift()
      if (msg && this.ws?.readyState === WebSocket.OPEN) {
        this.ws.send(msg.data)
      }
    }
  }
}

/** 辅助函数：等待 Blob 转为 ArrayBuffer */
function awaitData(data: Blob): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result as ArrayBuffer)
    reader.onerror = reject
    reader.readAsArrayBuffer(data)
  })
}

/** 辅助函数：尝试用 Refresh Token 刷新 WebSocket 连接的 Access Token */
async function tryRefreshWsToken(): Promise<boolean> {
  const refreshToken = sessionStorage.getItem('refresh_token')
  if (!refreshToken) return false

  try {
    const res = await fetch('/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false
    const data = await res.json()
    if (data.access_token) {
      setAccessToken(data.access_token)
      if (data.refresh_token) {
        sessionStorage.setItem('refresh_token', data.refresh_token)
      }
      return true
    }
    return false
  } catch {
    return false
  }
}

// 单例导出
export const wsClient = new WebSocketClient()

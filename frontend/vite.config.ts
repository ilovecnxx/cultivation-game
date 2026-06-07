import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  optimizeDeps: {
    esbuildOptions: {
      tsconfigRaw: {
        compilerOptions: {
          useDefineForClassFields: false,
          experimentalDecorators: true,
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      // Auth: 直接到 Gateway
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // WebSocket: 到 Gateway
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
      // Player 服务
      '/api/v1/player': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      // Cultivation 服务
      '/api/v1/cultivate': { target: 'http://localhost:8083', changeOrigin: true },
      '/api/v1/breakthrough': { target: 'http://localhost:8083', changeOrigin: true },
      '/api/v1/technique': { target: 'http://localhost:8083', changeOrigin: true },
      '/api/v1/meditate': { target: 'http://localhost:8083', changeOrigin: true },
      '/api/v1/alchemy': { target: 'http://localhost:8083', changeOrigin: true },
      '/api/v1/tribulation': { target: 'http://localhost:8083', changeOrigin: true },
      // Combat 服务
      '/api/v1/combat': { target: 'http://localhost:8084', changeOrigin: true },
      '/api/v1/arena': { target: 'http://localhost:8084', changeOrigin: true },
      // World 服务
      '/api/v1/world': { target: 'http://localhost:8086', changeOrigin: true },
      '/api/v1/quest': { target: 'http://localhost:8086', changeOrigin: true },
      // Trade 服务
      '/api/v1/trade': { target: 'http://localhost:8087', changeOrigin: true },
      // Ranking 服务
      '/api/v1/ranking': { target: 'http://localhost:8088', changeOrigin: true },
      // Social 服务
      '/api/v1/social': { target: 'http://localhost:8085', changeOrigin: true },
      '/api/v1/chat': { target: 'http://localhost:8085', changeOrigin: true },
      '/api/v1/friend': { target: 'http://localhost:8085', changeOrigin: true },
      '/api/v1/sect': { target: 'http://localhost:8085', changeOrigin: true },
      '/api/v1/mail': { target: 'http://localhost:8085', changeOrigin: true },
      // 健康检查
      '/health': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      // GM 管理后台
      '/api/v1/gm': {
        target: 'http://localhost:18082',
        changeOrigin: true,
      },
    },
  },
  css: {
    preprocessorOptions: {
      scss: {
        // 全局注入 SCSS 变量和混入，每个组件自动可用
        additionalData: `
          @use "sass:color";
          @use "@/styles/variables.scss" as *;
          @use "@/styles/mixins.scss" as *;
        `,
      },
    },
  },
})

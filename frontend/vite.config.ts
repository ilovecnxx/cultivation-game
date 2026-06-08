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
      // 单体模式 — 所有 API 统一到 :8080
      '/auth':          { target: 'http://localhost:8080', changeOrigin: true },
      '/ws':            { target: 'ws://localhost:8080', ws: true },
      '/api':           { target: 'http://localhost:8080', changeOrigin: true },
      '/health':        { target: 'http://localhost:8080', changeOrigin: true },
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

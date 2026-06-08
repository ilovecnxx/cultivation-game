// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  ssr: false,

  modules: [
    '@vant/nuxt', // Vant 4 移动端 UI 组件库
    '@pinia/nuxt', // Pinia 状态管理
  ],

  app: {
    head: {
      title: '修仙世界',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover' },
        { name: 'theme-color', content: '#1a1a2e' },
        { name: 'apple-mobile-web-app-capable', content: 'yes' },
        { name: 'apple-mobile-web-app-status-bar-style', content: 'black-translucent' },
      ],
      link: [
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'stylesheet', href: 'https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;700;900&display=swap' },
      ],
    },
  },

  css: ['vant/lib/index.css', '~/styles/vant-theme.scss', '~/styles/main.scss'],

  vite: {
    server: {
      proxy: {
        '/auth': 'http://localhost:8080',
        '/api': 'http://localhost:8080',
        '/health': 'http://localhost:8080',
        '/ws': { target: 'ws://localhost:8080', ws: true },
      },
    },
    resolve: {
      alias: { '@styles': '/root/projects/cultivation-game/frontend/styles' },
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `
            @use "sass:color";
            @use "@styles/variables.scss" as *;
            @use "@styles/mixins.scss" as *;
          `,
        },
      },
    },
  },

  compatibilityDate: '2025-06-08',
})
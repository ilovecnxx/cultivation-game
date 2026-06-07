import { createRouter, createWebHashHistory } from 'vue-router'
import LandingView from '@/views/LandingView.vue'
import GameHome from '@/views/GameHome.vue'

const routes = [
  { path: '/', name: 'Landing', component: LandingView, meta: { title: '修仙世界' } },
  { path: '/home', name: 'GameHome', component: GameHome, meta: { title: '修仙世界', requiresAuth: true } },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to, _from, next) => {
  document.title = `${to.meta.title ?? '修仙世界'} | 修仙世界`
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) { next({ name: 'Landing' }); return }
  if (token && (to.name === 'Landing')) { next({ name: 'GameHome' }); return }
  next()
})

export default router

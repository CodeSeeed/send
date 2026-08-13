import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'upload', component: () => import('../views/Upload.vue') },
    { path: '/s/:code', name: 'share', component: () => import('../views/Share.vue') },
    { path: '/admin/login', name: 'admin-login', component: () => import('../views/AdminLogin.vue') },
    {
      // Auth guard is done inside AdminDashboard via /api/admin/check
      // (the session lives in an HttpOnly cookie, invisible to the router)
      path: '/admin',
      name: 'admin',
      component: () => import('../views/AdminDashboard.vue'),
    },
  ],
})

export default router
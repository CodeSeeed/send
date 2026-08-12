import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'upload', component: () => import('../views/Upload.vue') },
    { path: '/s/:code', name: 'share', component: () => import('../views/Share.vue') },
    { path: '/admin/login', name: 'admin-login', component: () => import('../views/AdminLogin.vue') },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/AdminDashboard.vue'),
      beforeEnter: () => {
        if (!localStorage.getItem('admin_token')) {
          return { name: 'admin-login' }
        }
        return true
      },
    },
  ],
})

export default router

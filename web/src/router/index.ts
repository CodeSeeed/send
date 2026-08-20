import { createRouter, createWebHashHistory } from 'vue-router'
import { adminCheck } from '../api/admin'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('../views/Home.vue'),
    },
    {
      path: '/upload',
      name: 'upload',
      component: () => import('../views/Upload.vue'),
      meta: { requiresAuth: true },
    },
    { path: '/s/:code', name: 'share', component: () => import('../views/Share.vue') },
    { path: '/admin/login', name: 'admin-login', component: () => import('../views/AdminLogin.vue') },
    {
      path: '/admin',
      name: 'admin',
      component: () => import('../views/AdminDashboard.vue'),
      meta: { requiresAuth: true },
    },
  ],
})

// Auth guard: require an active admin session for protected routes.  The
// session lives in an HttpOnly cookie (invisible to the router), so each
// navigation performs a lightweight /api/admin/check call.  Doing this in the
// routing guard *before* the component mounts prevents a logged-out visitor
// from briefly seeing the protected page before being redirected away.
router.beforeEach(async (to) => {
  if (to.meta.requiresAuth) {
    try {
      await adminCheck()
      return true
    } catch {
      // Use a query parameter to signal the login page to show the "请登录"
      // hint — the ElMessage call is kept in the component context where it
      // is guaranteed to work (after the app has been fully mounted).
      return { name: 'admin-login', query: { reason: 'auth' } }
    }
  }
  return true
})

export default router

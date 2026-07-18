import { createRouter, createWebHashHistory } from 'vue-router'
import Upload from '../views/Upload.vue'
import Share from '../views/Share.vue'
import AdminLogin from '../views/AdminLogin.vue'
import AdminDashboard from '../views/AdminDashboard.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'upload', component: Upload },
    { path: '/s/:code', name: 'share', component: Share },
    { path: '/admin/login', name: 'admin-login', component: AdminLogin },
    { path: '/admin', name: 'admin', component: AdminDashboard },
  ],
})

export default router

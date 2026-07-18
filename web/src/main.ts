import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)

// eslint-disable-next-line @typescript-eslint/no-explicit-any
app.use(ElementPlus, { locale: zhCn as any })
app.use(router)
app.mount('#app')

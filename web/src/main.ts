import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/theme-chalk/dark/css-vars.css'
import 'element-plus/theme-chalk/el-message.css'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
// Components are auto-imported by unplugin, but the locale must be configured
// globally for component-provided text (pagination, date pickers, …) to be Chinese.
app.use(ElementPlus, { locale: zhCn })
app.use(router)
app.mount('#app')

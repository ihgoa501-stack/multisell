import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

// Ant Design Vue 全局样式（CSS-in-JS 基础重置）
import 'ant-design-vue/dist/reset.css'

// 全局样式
import './assets/main.css'
import './styles/fonts.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

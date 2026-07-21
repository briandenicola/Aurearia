import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/styles/main.css'

// Apply saved theme on load
const savedTheme = localStorage.getItem('theme') || 'dark'
document.documentElement.setAttribute('data-theme', savedTheme)

// Register service worker in prompt mode; PwaUpdateBanner surfaces onNeedRefresh
// so users can choose when to apply a newly deployed version.
import '@/composables/usePwaUpdate'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

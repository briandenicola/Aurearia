import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import '@fontsource/inter/300.css'
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/cinzel/400.css'
import '@fontsource/cinzel/500.css'
import '@fontsource/cinzel/600.css'
import '@fontsource/cinzel/700.css'
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

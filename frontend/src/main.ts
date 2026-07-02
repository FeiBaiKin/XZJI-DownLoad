import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './styles/main.css'
import './styles/home.css'
import './styles/auth.css'
import './styles/verification.css'
import './styles/modal.css'
import './styles/admin-layout.css'
import './styles/admin-keyword.css'
import './styles/admin-dashboard.css'
import './styles/admin-orders.css'
import './styles/admin-extra.css'
import './styles/admin-membership.css'
import './styles/admin-payment.css'
import './styles/admin-email.css'
import './styles/rich-text.css'
import './styles/content-page.css'
import './styles/user.css'
import './styles/membership-public.css'
import './styles/toast.css'
import './styles/responsive.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHashHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

// FontAwesome
import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import {
  faPlus, faMessage, faSun, faMoon, faRocket, faGear,
  faRightFromBracket, faSpinner, faPaperPlane, faRobot,
  faChevronDown, faChevronRight, faCheck, faCircleCheck,
  faFile, faFileLines, faPenToSquare, faTerminal,
  faList, faCalendar, faMagnifyingGlass, faCode,
  faPaperclip, faXmark, faCircleXmark, faImage,
  faVideo, faMusic, faDownload, faArrowLeft, faArrowRight,
  faFolderOpen, faEllipsis,
} from '@fortawesome/free-solid-svg-icons'

library.add(
  faPlus, faMessage, faSun, faMoon, faRocket, faGear,
  faRightFromBracket, faSpinner, faPaperPlane, faRobot,
  faChevronDown, faChevronRight, faCheck, faCircleCheck,
  faFile, faFileLines, faPenToSquare, faTerminal,
  faList, faCalendar, faMagnifyingGlass, faCode,
  faPaperclip, faXmark, faCircleXmark, faImage,
  faVideo, faMusic, faDownload, faArrowLeft, faArrowRight,
  faFolderOpen, faEllipsis,
)

// Markstream CSS
import 'markstream-vue/index.css'

import AppLayout from './components/AppLayout.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', redirect: '/login' },
    { path: '/login', component: () => import('./views/LoginView.vue') },
    {
      path: '/',
      component: AppLayout,
      children: [
        { path: 'chat', component: () => import('./views/ChatView.vue') },
        { path: 'settings', component: () => import('./views/SettingsView.vue') },
      ],
    },
  ],
})

const app = createApp(App)
app.component('FontAwesomeIcon', FontAwesomeIcon)
app.use(createPinia())
app.use(router)
app.mount('#app')

import { createRouter, createWebHistory } from 'vue-router'


const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    name: 'Login',
    path: '/login',
    component: () => import('@/pages/login/index.vue')
  }, {
    name: 'Main',
    component: () => import('@/pages/mainSection/index.vue'),
    path: '/main',
    redirect: '/main/chat',
    meta: {
      breadcrumb: '主菜单'
    },
    children: [{
      name: 'chat',
      path: 'chat',
      component: () => import('@/pages/chat/index.vue'),
      meta: {
        breadcrumb: '对话'
      }
    }, {
      name: 'home',
      path: 'home',
      component: () => import('@/pages/home/index.vue'),
      meta: {
        breadcrumb: '主页'
      }
    }, {
      name: 'models',
      path: 'models',
      component: () => import('@/pages/models/index.vue'),
      meta: {
        breadcrumb: '模型管理'
      }
    }, {
      name: 'settings',
      path: 'settings',
      component: () => import('@/pages/settings/index.vue'),
      meta: {
        breadcrumb: '设置'
      }
    }, {
      name: 'mcp',
      path: 'mcp',
      component: () => import('@/pages/mcp/index.vue'),
      meta: {
        breadcrumb: 'MCP'
      }
    }, {
      name: 'platform',
      path: 'platform',
      component: () => import('@/pages/platform/index.vue'),
      meta: {
        breadcrumb: '平台'
      }
    }]
  }

]


const router = createRouter({
  history: createWebHistory(),
  routes,
})
router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.fullPath !== '/login') {
    return token ? true : { name: 'Login' }
  } else {
    return token ? { path:'Main' } : true
  }
})

export default router
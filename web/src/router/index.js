const routes = [
  {
    path: '/mobile/image',
    component: () => import('@/views/mobile/ImageView.vue'),
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/mobile/chat/list',
    component: () => import('@/views/mobile/ChatList.vue'),
    meta: {
      requiresAuth: true
    }
  },
  {
    path: '/mobile/c2c',
    component: () => import('@/views/mobile/c2c.vue'),
    meta: {
      requiresAuth: true
    }
  }
]

export default routes
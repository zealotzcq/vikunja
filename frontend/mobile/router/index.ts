import type { RouteRecordRaw } from 'vue-router';

// Simple auth guard: redirect to mobile login if no token
const requireAuth = (to: any, from: any, next: any) => {
  const token = typeof window !== 'undefined'
    ? localStorage.getItem('token')
    : null;
  if (token) {
    next();
  } else {
    next('/mobile/login');
  }
};

export default [
  {
    path: '/mobile/login',
    name: 'MobileLogin',
    component: () => import('../views/MobileLoginView.vue'),
  },
  {
    path: '/mobile',
    component: () => import('../components/MobileTabLayout.vue'),
    beforeEnter: requireAuth,
    redirect: '/mobile/home',
    children: [
      {
        path: 'home',
        name: 'MobileHome',
        component: () => import('../views/MobileHomeView.vue'),
      },
      {
        path: 'chat',
        name: 'MobileChat',
        component: () => import('../views/MobileChatView.vue'),
      },
      {
        path: 'projects',
        name: 'MobileProjects',
        component: () => import('../views/MobileProjectsView.vue'),
      },
      {
        path: 'project/:projectId',
        name: 'MobileProjectDetail',
        component: () => import('../views/MobileProjectDetailView.vue'),
      },
      {
        path: 'task/:taskId',
        name: 'MobileTaskDetail',
        component: () => import('../views/MobileTaskDetailView.vue'),
      },
    ],
  },
  {
    path: '/mobile/:pathMatch(.*)*',
    redirect: '/mobile/home',
  },
] as Array<RouteRecordRaw>;

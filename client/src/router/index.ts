import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuth } from '@/composables/useAuth'

import DashboardView from '@/components/DashboardView.vue'
import LoginView from '@/views/LoginView.vue'
import UnauthorizedView from '@/views/UnauthorizedView.vue'
import CoursesView from '@/views/CoursesView.vue'
import SettingsView from '@/views/SettingsView.vue'
import GenericView from '@/views/GenericView.vue'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginView,
    meta: {
      guestOnly: true,
      hideNavigation: true,
      title: 'Sign In',
    },
  },
  {
    path: '/unauthorized',
    name: 'Unauthorized',
    component: UnauthorizedView,
    meta: {
      hideNavigation: true,
      title: 'Unauthorized Access',
    },
  },
  // Protected Routes
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: DashboardView,
    meta: {
      requiresAuth: true,
      title: 'Dashboard',
    },
  },
  {
    path: '/courses',
    name: 'Courses',
    component: CoursesView,
    meta: {
      requiresAuth: true,
      title: 'My Courses',
    },
  },
  {
    path: '/groups',
    name: 'Groups',
    component: GenericView,
    props: {
      title: 'Groups',
      icon: 'mdi-account-group',
    },
    meta: {
      requiresAuth: true,
      title: 'Groups',
    },
  },
  {
    path: '/messages',
    name: 'Messages',
    component: GenericView,
    props: {
      title: 'Messages',
      icon: 'mdi-message',
    },
    meta: {
      requiresAuth: true,
      title: 'Messages',
    },
  },
  {
    path: '/grades',
    name: 'Grades',
    component: GenericView,
    props: {
      title: 'Grades',
      icon: 'mdi-chart-bar',
    },
    meta: {
      requiresAuth: true,
      title: 'Grades',
    },
  },
  {
    path: '/settings',
    name: 'Settings',
    component: SettingsView,
    meta: {
      requiresAuth: true,
      title: 'Settings',
    },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: GenericView,
    props: {
      title: 'Admin Control Panel',
      icon: 'mdi-shield-crown-outline',
    },
    meta: {
      requiresAuth: true,
      roles: ['admin', 'superadmin'],
      title: 'Admin',
    },
  },
  // Fallback catch-all route
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

// Navigation Guard
router.beforeEach((to, _from, next) => {
  const { isAuthenticated, currentUser } = useAuth()

  // 1. Check if route requires authentication
  if (to.meta.requiresAuth) {
    if (!isAuthenticated.value) {
      // Redirect to unauthorized page with destination preserved in query
      return next({
        path: '/unauthorized',
        query: { redirect: to.fullPath },
      })
    }

    // 2. Check role authorization if specified
    const allowedRoles = to.meta.roles as string[] | undefined
    if (allowedRoles && allowedRoles.length > 0) {
      const userRole = currentUser.value?.role?.toLowerCase()
      const hasPermission = userRole && allowedRoles.map(r => r.toLowerCase()).includes(userRole)
      if (!hasPermission) {
        return next({
          path: '/unauthorized',
          query: { redirect: to.fullPath },
        })
      }
    }
  }

  // 3. Prevent authenticated users from visiting guest-only pages (e.g. login)
  if (to.meta.guestOnly && isAuthenticated.value) {
    return next({ path: '/dashboard' })
  }

  // 4. Update document title
  if (to.meta.title) {
    document.title = `${to.meta.title} — DAT250 Learning Platform`
  }

  return next()
})

export default router

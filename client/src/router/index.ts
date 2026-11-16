import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { getStoredAccessToken, registerAuthFailureHandler } from '@/api/client'
import { useAuth } from '@/composables/useAuth'

declare module 'vue-router' {
  interface RouteMeta {
    requiresAuth?: boolean
    guestOnly?: boolean
    hideNavigation?: boolean
    title?: string
    roles?: string[]
  }
}

function paramAsString(value: string | string[] | undefined): string {
  if (Array.isArray(value)) {
    return value[0] ?? ''
  }
  return value ?? ''
}

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: '/dashboard',
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: {
      guestOnly: true,
      hideNavigation: true,
      title: 'Sign In',
    },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/RegisterView.vue'),
    meta: {
      guestOnly: true,
      hideNavigation: true,
      title: 'Sign Up',
    },
  },
  {
    path: '/unauthorized',
    name: 'Unauthorized',
    component: () => import('@/views/UnauthorizedView.vue'),
    meta: {
      hideNavigation: true,
      title: 'Unauthorized Access',
    },
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Dashboard',
    },
  },
  {
    path: '/courses',
    name: 'Courses',
    component: () => import('@/views/CoursesView.vue'),
    meta: {
      requiresAuth: true,
      title: 'My Courses',
    },
  },
  {
    path: '/catalog',
    name: 'CourseCatalog',
    component: () => import('@/views/CourseCatalogView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Course Catalog',
    },
  },
  {
    path: '/courses/:uuid',
    name: 'CourseDetail',
    component: () => import('@/views/CourseDetailView.vue'),
    props: route => ({ courseUuid: paramAsString(route.params.uuid) }),
    meta: {
      requiresAuth: true,
      title: 'Course Details',
    },
  },
  {
    path: '/courses/:uuid/modules/:moduleUuid/files/:fileUuid',
    name: 'FileView',
    component: () => import('@/views/FileView.vue'),
    props: route => ({
      courseUuid: paramAsString(route.params.uuid),
      moduleUuid: paramAsString(route.params.moduleUuid),
      fileUuid: paramAsString(route.params.fileUuid),
    }),
    meta: {
      requiresAuth: true,
      title: 'Course Material',
    },
  },
  {
    path: '/courses/:uuid/modules/:moduleUuid/quizzes/:quizUuid/edit',
    name: 'QuizEdit',
    component: () => import('@/views/QuizEditorView.vue'),
    props: route => ({
      courseUuid: paramAsString(route.params.uuid),
      moduleUuid: paramAsString(route.params.moduleUuid),
      quizUuid: paramAsString(route.params.quizUuid),
    }),
    meta: {
      requiresAuth: true,
      title: 'Edit Quiz',
    },
  },
  {
    path: '/courses/:uuid/modules/:moduleUuid/quizzes/:quizUuid/take',
    name: 'QuizTake',
    component: () => import('@/views/QuizTakeView.vue'),
    props: route => ({
      courseUuid: paramAsString(route.params.uuid),
      moduleUuid: paramAsString(route.params.moduleUuid),
      quizUuid: paramAsString(route.params.quizUuid),
    }),
    meta: {
      requiresAuth: true,
      title: 'Quiz',
    },
  },
  {
    path: '/courses/:uuid/modules/:moduleUuid/quizzes/:quizUuid/results',
    name: 'QuizResults',
    component: () => import('@/views/QuizResultsView.vue'),
    props: route => ({
      courseUuid: paramAsString(route.params.uuid),
      moduleUuid: paramAsString(route.params.moduleUuid),
      quizUuid: paramAsString(route.params.quizUuid),
    }),
    meta: {
      requiresAuth: true,
      title: 'Quiz Results',
    },
  },
  {
    path: '/assignments',
    name: 'Assignments',
    component: () => import('@/views/AssignmentsView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Assignments',
    },
  },
  {
    path: '/groups',
    name: 'Groups',
    component: () => import('@/views/GroupsView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Groups',
    },
  },
  {
    path: '/grades',
    name: 'Grades',
    component: () => import('@/views/GradesView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Grades',
    },
  },
  {
    path: '/settings',
    name: 'Settings',
    component: () => import('@/views/SettingsView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Settings',
    },
  },
  {
    path: '/admin',
    name: 'Admin',
    component: () => import('@/views/GenericView.vue'),
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
  {
    path: '/notification/send',
    name: 'NotificationSend',
    component: () => import('@/views/NotificationSendView.vue'),
    meta: {
      requiresAuth: true,
      title: 'Send notification',
    },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/dashboard',
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
})

registerAuthFailureHandler(() => {
  const currentRoute = router.currentRoute.value
  if (currentRoute.path !== '/login' && currentRoute.path !== '/unauthorized') {
    void router.replace({
      path: '/login',
      query: { redirect: currentRoute.fullPath },
    })
  }
})

router.beforeEach(async to => {
  const { isAuthenticated, isTokenExpired, hasAnyRole, refresh } = useAuth()

  if (to.meta.requiresAuth) {
    if (!isAuthenticated.value || isTokenExpired()) {
      const refreshed = await refresh()
      if (!refreshed || !isAuthenticated.value || isTokenExpired()) {
        return {
          path: '/login',
          query: { redirect: to.fullPath },
        }
      }
    }

    const allowedRoles = to.meta.roles
    if (allowedRoles && allowedRoles.length > 0 && !hasAnyRole(allowedRoles)) {
      return {
        path: '/unauthorized',
        query: { redirect: to.fullPath },
      }
    }
  }

  if (to.meta.guestOnly) {
    if ((!isAuthenticated.value || isTokenExpired()) && getStoredAccessToken()) {
      await refresh()
    }
    if (isAuthenticated.value && !isTokenExpired()) {
      return { path: '/dashboard' }
    }
  }

  return undefined
})

router.afterEach(to => {
  document.title = to.meta.title
    ? `${to.meta.title} — DAT250 Learning Platform`
    : 'DAT250 Learning Platform'
})

export default router

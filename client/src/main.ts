/**
 * main.ts
 *
 * Bootstraps Vuetify and other plugins then mounts the App.
 */

import { createApp } from 'vue'
import { useToast } from '@/composables/useToast'
import { registerPlugins } from '@/plugins'
import router from '@/router'
import App from './App.vue'
import 'unfonts.css'
import '@/styles/global.scss'

const app = createApp(App)

app.config.errorHandler = (err, _instance, info) => {
  console.error(`Unhandled application error (${info}):`, err)
  const toast = useToast()
  toast.error(err instanceof Error ? err.message : 'An unexpected error occurred.')
}

registerPlugins(app)

router
  .isReady()
  .then(() => {
    app.mount('#app')
  })
  .catch((error: unknown) => {
    console.error('Failed to initialize router:', error)
    const root = document.querySelector('#app')
    if (root) {
      root.textContent = 'Unable to start the application. Please refresh the page.'
    }
  })

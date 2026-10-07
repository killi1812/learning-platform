/**
 * plugins/vuetify.ts
 *
 * Framework documentation: https://vuetifyjs.com`
 */

// Composables
import { createVuetify } from 'vuetify'
// Styles
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify({
  theme: {
    defaultTheme: 'system',
    themes: {
      light: {
        colors: {
          primary: '#E85D75',
          'on-primary': '#FFFFFF',
          secondary: '#6BAA5E',
          'on-secondary': '#FFFFFF',
          accent: '#6BAA5E',
          'on-accent': '#FFFFFF',
          background: '#FFFFFF',
          'on-background': '#1a1a1a',
          surface: '#FFFFFF',
          'on-surface': '#1a1a1a',
          'surface-variant': '#f5f5f5',
        },
      },
      dark: {
        colors: {
          primary: '#E85D75',
          'on-primary': '#FFFFFF',
          secondary: '#6BAA5E',
          'on-secondary': '#FFFFFF',
          accent: '#6BAA5E',
          'on-accent': '#FFFFFF',
          background: '#0a0a0a',
          'on-background': '#f5f5f5',
          surface: '#1a1a1a',
          'on-surface': '#f5f5f5',
          'surface-variant': '#2a2a2a',
        },
      },
    },
  },
})

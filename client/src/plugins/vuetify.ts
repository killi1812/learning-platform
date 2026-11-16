/**
 * plugins/vuetify.ts
 *
 * Framework documentation: https://vuetifyjs.com
 */

import { createVuetify } from 'vuetify'
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'

export default createVuetify({
  theme: {
    defaultTheme: 'system',
    themes: {
      light: {
        colors: {
          primary: '#E85D75',
          'on-primary': '#1A1A1A',
          secondary: '#6BAA5E',
          'on-secondary': '#1A1A1A',
          accent: '#6BAA5E',
          'on-accent': '#1A1A1A',
          background: '#FFFFFF',
          'on-background': '#1A1A1A',
          surface: '#FFFFFF',
          'on-surface': '#1A1A1A',
          'surface-variant': '#F5F5F5',
          'on-surface-variant': '#1A1A1A',
        },
      },
      dark: {
        colors: {
          primary: '#E85D75',
          'on-primary': '#1A1A1A',
          secondary: '#6BAA5E',
          'on-secondary': '#1A1A1A',
          accent: '#6BAA5E',
          'on-accent': '#1A1A1A',
          background: '#0A0A0A',
          'on-background': '#F5F5F5',
          surface: '#1A1A1A',
          'on-surface': '#F5F5F5',
          'surface-variant': '#2A2A2A',
          'on-surface-variant': '#F5F5F5',
        },
      },
    },
  },
})

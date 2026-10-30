/// <reference types="vite/client" />
/// <reference types="vite-plugin-vue-layouts-next/client" />

declare module '@vue-office/docx' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{ src: unknown }, {}, unknown>
  export default component
}

declare module '@vue-office/pptx' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{ src: unknown }, {}, unknown>
  export default component
}

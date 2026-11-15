import vuetify from 'eslint-config-vuetify'

export default vuetify(
  {
    stylistic: false,
  },
  {
    name: 'prettier-compat',
    rules: {
      'vue/first-attribute-linebreak': 'off',
      'vue/html-closing-bracket-newline': 'off',
      'vue/html-indent': 'off',
      'vue/html-self-closing': 'off',
      'vue/max-attributes-per-line': 'off',
      'vue/padding-line-between-tags': 'off',
      'vue/script-indent': 'off',
    },
  },
)

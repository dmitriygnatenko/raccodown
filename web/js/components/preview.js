import { renderMarkdown } from '../data/markdown.js'

export default {
  props: { content: { type: String, default: '' } },
  template: '<div class="markdown-preview" v-html="html"></div>',
  data() {
    return { html: '' }
  },
  watch: {
    content: {
      immediate: true,
      handler(content) {
        this.html = renderMarkdown(content)
      },
    },
  },
}

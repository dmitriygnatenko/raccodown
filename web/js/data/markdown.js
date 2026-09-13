import MarkdownIt from 'markdown-it'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: false,
})

export function renderMarkdown(content) {
  return md.render(content)
}

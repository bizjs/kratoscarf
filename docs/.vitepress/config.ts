import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'kratoscarf',
  description: 'Kratos 微服务框架的 Web 开发增强层',
  lang: 'zh-CN',
  base: '/kratoscarf/',

  head: [
    ['meta', { name: 'theme-color', content: '#3a7afe' }],
  ],

  themeConfig: {
    nav: [
      { text: '指南', link: '/guide/' },
      { text: 'API', link: '/router' },
      { text: 'GitHub', link: 'https://github.com/bizjs/kratoscarf' },
    ],

    sidebar: [
      {
        text: '入门',
        items: [
          { text: '简介', link: '/guide/' },
          { text: '快速开始', link: '/guide/getting-started' },
        ],
      },
      {
        text: '核心',
        items: [
          { text: '路由', link: '/router' },
          { text: '响应', link: '/response' },
          { text: '验证', link: '/validation' },
        ],
      },
      {
        text: '认证',
        items: [
          { text: 'JWT / Session', link: '/auth' },
        ],
      },
      {
        text: '中间件',
        items: [
          { text: 'CORS / Secure / RequestID', link: '/middleware' },
        ],
      },
      {
        text: '工具',
        items: [
          { text: '健康检查', link: '/health' },
          { text: 'ID / 加密', link: '/util' },
        ],
      },
    ],

    outline: {
      level: [2, 3],
      label: '目录',
    },

    search: {
      provider: 'local',
    },

    footer: {
      message: 'Released under the MIT License.',
    },
  },
})

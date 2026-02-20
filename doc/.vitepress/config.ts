import { defineConfig } from 'vitepress'

export default defineConfig({
  title: 'Bonds',
  description: 'Personal Relationship Manager — Go + React',
  base: '/bonds/',

  ignoreDeadLinks: [
    /localhost/,
  ],

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/bonds/logo.svg' }],
  ],

  lastUpdated: true,

  themeConfig: {
    search: {
      provider: 'local',
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/naiba/bonds' },
    ],
    footer: {
      message: 'Released under the <a href="https://github.com/naiba/bonds/blob/main/LICENSE">BSL-1.1 License</a> (converts to AGPL-3.0 on 2030-02-17)',
      copyright: '© 2026 <a href="https://github.com/naiba">Naiba</a>',
    },
  },

  locales: {
    root: {
      label: 'English',
      lang: 'en',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/introduction' },
          { text: 'Features', link: '/features/contacts' },
          {
            text: 'Links',
            items: [
              { text: 'GitHub', link: 'https://github.com/naiba/bonds' },
              { text: 'Releases', link: 'https://github.com/naiba/bonds/releases' },
              { text: 'Docker Hub', link: 'https://github.com/naiba/bonds/pkgs/container/bonds' },
            ],
          },
        ],
        sidebar: {
          '/guide/': [
            {
              text: 'Guide',
              items: [
                { text: 'Introduction', link: '/guide/introduction' },
                { text: 'Getting Started', link: '/guide/getting-started' },
                { text: 'Configuration', link: '/guide/configuration' },
                { text: 'Development', link: '/guide/development' },
              ],
            },
          ],
          '/features/': [
            {
              text: 'Features',
              items: [
                { text: 'Contacts', link: '/features/contacts' },
                { text: 'Vaults', link: '/features/vaults' },
                { text: 'Reminders', link: '/features/reminders' },
                { text: 'Full-text Search', link: '/features/search' },
                { text: 'CardDAV / CalDAV', link: '/features/dav' },
                { text: 'Import / Export', link: '/features/import-export' },
                { text: 'Files & Avatars', link: '/features/files' },
                { text: 'Authentication', link: '/features/authentication' },
                { text: 'Admin & Settings', link: '/features/admin' },
                { text: 'More Features', link: '/features/more' },
              ],
            },
          ],
        },
        editLink: {
          pattern: 'https://github.com/naiba/bonds/edit/main/doc/:path',
          text: 'Edit this page on GitHub',
        },
      },
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/zh/',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/introduction' },
          { text: '功能', link: '/zh/features/contacts' },
          {
            text: '链接',
            items: [
              { text: 'GitHub', link: 'https://github.com/naiba/bonds' },
              { text: '发布', link: 'https://github.com/naiba/bonds/releases' },
            ],
          },
        ],
        sidebar: {
          '/zh/guide/': [
            {
              text: '指南',
              items: [
                { text: '简介', link: '/zh/guide/introduction' },
                { text: '快速开始', link: '/zh/guide/getting-started' },
                { text: '配置', link: '/zh/guide/configuration' },
                { text: '开发', link: '/zh/guide/development' },
              ],
            },
          ],
          '/zh/features/': [
            {
              text: '功能',
              items: [
                { text: '联系人', link: '/zh/features/contacts' },
                { text: '多 Vault', link: '/zh/features/vaults' },
                { text: '提醒', link: '/zh/features/reminders' },
                { text: '全文搜索', link: '/zh/features/search' },
                { text: 'CardDAV / CalDAV', link: '/zh/features/dav' },
                { text: '导入 / 导出', link: '/zh/features/import-export' },
                { text: '文件与头像', link: '/zh/features/files' },
                { text: '认证', link: '/zh/features/authentication' },
                { text: '管理面板', link: '/zh/features/admin' },
                { text: '更多功能', link: '/zh/features/more' },
              ],
            },
          ],
        },
        editLink: {
          pattern: 'https://github.com/naiba/bonds/edit/main/doc/:path',
          text: '在 GitHub 上编辑此页',
        },
        lastUpdated: {
          text: '最后更新',
        },
        outline: {
          label: '页面导航',
        },
        docFooter: {
          prev: '上一页',
          next: '下一页',
        },
      },
    },
  },
})

import { defineConfig } from 'vitepress'

// https://vitepress.dev/reference/site-config
export default defineConfig({
  title: "Goliac project",
  description: "Github Organization IAC made simple",
  ignoreDeadLinks: true,
  head: [['link', { rel: 'icon', href: '/favicon.ico' }]],
  base: '/goliac',
  themeConfig: {
    outline: 'deep',
//    outline: false, // Disable/enable right sidebar globally
    logo: '/logo_small.png', 
    // https://vitepress.dev/reference/default-theme-config
    nav: [
      { text: 'Home', link: '/' },
      { text: 'Documentation',
        items: [
            { text: 'Why Goliac', link: '/why_goliac' },
            { text: 'Quick start', link: '/quick_start' },
            { text: 'Installation', link: '/installation' },
            { text: 'Security', link: '/security' },
            { text: 'Troubleshooting', link: '/troubleshooting' }
          ]
        }
      ],

    sidebar: [
      {
        text: 'Documentation',
        items: [
          { text: 'Why Goliac', link: '/why_goliac' },
          { text: 'Quick start', link: '/quick_start' },
          { text: 'Installation', link: '/installation' },
          { text: 'Security', link: '/security' },
          { text: 'Troubleshooting', link: '/troubleshooting' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Alayacare/goliac' }
    ]
  }
})

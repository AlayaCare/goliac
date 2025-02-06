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
            { text: 'What is Goliac', link: '/what_is_goliac' },
            { text: 'Quick start', link: '/quick_start' },
            { text: 'Installation', link: '/installation' },
            { text: 'Day to day Usage', link: '/usage' },
            { text: 'Security', link: '/security' },
            { text: 'Troubleshooting', link: '/troubleshooting' }
          ]
        }
      ],

    sidebar: [
      {
        text: 'Documentation',
        items: [
          { text: 'What is Goliac', link: '/what_is_goliac' },
          { text: 'Quick start', link: '/quick_start' },
          { text: 'Installation', link: '/installation' },
          { text: 'Day to day Usage', link: '/usage' },
          { text: 'Security', link: '/security' },
          { text: 'Troubleshooting', link: '/troubleshooting' }
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Alayacare/goliac' }
    ],
    
    footer: {
      message: 'Sponsored by <a href="https://www.alayacare.com">AlayaCare</a>',
      copyright: '<a href="https://github.com/AlayaCare/goliac/blob/main/LICENSE">MIT License</a>'
    }
  }
})

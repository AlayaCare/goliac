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
          { text: 'Admin Usage', link: '/admin_usage' },
          { text: 'Security', link: '/security' },
          { text: 'Troubleshooting', link: '/troubleshooting' },
          { text: 'APIs', link: '/apis' },
        ]
      },
      {
        text: 'Goliac Resources',
        items: [
          { text: 'User', link: '/resource_user'},
          { text: 'Team', link: '/resource_team'},
          { text: 'Repository', link: '/resource_repository'},
          { text: 'Ruleset', link: '/resource_ruleset'},
          { text: 'Workflow', link: '/resource_workflow'}
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
          {
            text: 'Goliac Resources',
            items: [
              { text: 'User', link: '/resource_user'},
              { text: 'Team', link: '/resource_team'},
              { text: 'Repository', link: '/resource_repository'},
              { text: 'Ruleset', link: '/resource_ruleset'},
              { text: 'Workflow', link: '/resource_workflow'}
            ]
          },
          { text: 'Admin Usage', link: '/admin_usage' },
          { text: 'PR Breaking glass', link: '/breakingglass' },
          { text: 'Security', link: '/security' },
          { text: 'Troubleshooting', link: '/troubleshooting' },
          { text: 'APIs', link: '/apis' },
          { text: 'API docs', link: '/api_docs', target: '_self'}
        ]
      }
    ],

    socialLinks: [
      { icon: 'github', link: 'https://github.com/goliac-project/goliac' }
    ],
    
    footer: {
      copyright: '<a href="https://github.com/goliac-project/goliac/blob/main/LICENSE">MIT License</a>'
    }
  }
})

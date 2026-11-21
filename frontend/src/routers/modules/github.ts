// fork: top-level GitHub section. The route name must match the core
// HideMenu label "GitHub-Menu" (core/init/migration/helper/github_menu.go).
import { Layout } from '@/routers/constant';

const githubRouter = {
    sort: 9.5,
    path: '/github',
    name: 'GitHub-Menu',
    component: Layout,
    redirect: '/github/projects',
    meta: {
        icon: 'p-huaban88',
        title: 'github.menu',
        adminOnly: true,
    },
    children: [
        {
            path: '/github',
            name: 'GithubIndex',
            redirect: '/github/projects',
            component: () => import('@/views/github/index.vue'),
            meta: {},
            children: [
                {
                    path: 'projects',
                    name: 'GithubProjects',
                    component: () => import('@/views/github/projects/index.vue'),
                    hidden: true,
                    meta: { activeMenu: '/github', title: 'github.projects', adminOnly: true },
                },
                {
                    path: 'logs',
                    name: 'GithubLogs',
                    component: () => import('@/views/github/runs/index.vue'),
                    hidden: true,
                    meta: { activeMenu: '/github', title: 'github.logs', adminOnly: true },
                },
                {
                    path: 'keys',
                    name: 'GithubKeys',
                    component: () => import('@/views/github/keys/index.vue'),
                    hidden: true,
                    meta: { activeMenu: '/github', title: 'github.keys', adminOnly: true },
                },
                {
                    path: 'updates',
                    name: 'GithubUpdates',
                    component: () => import('@/views/github/updates/index.vue'),
                    hidden: true,
                    meta: { activeMenu: '/github', title: 'github.updates', adminOnly: true },
                },
                {
                    path: 'settings',
                    name: 'GithubSettings',
                    component: () => import('@/views/github/settings/index.vue'),
                    hidden: true,
                    meta: { activeMenu: '/github', title: 'github.settings', adminOnly: true },
                },
            ],
        },
    ],
};

export default githubRouter;

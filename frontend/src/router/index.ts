import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/pages/HomePage.vue') },
    { path: '/repositories', name: 'repositories', component: () => import('@/pages/RepositoriesPage.vue') },
    { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
  ],
  scrollBehavior: (_to, _from, saved) => saved ?? { top: 0 },
})

export default router

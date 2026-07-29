import { createRouter, createWebHistory } from 'vue-router';
import type { RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  { path: '/', redirect: '/today' },
  {
    path: '/today',
    name: 'today',
    component: () => import('@/views/TodayView.vue'),
    meta: { title: 'Today' },
  },
  {
    path: '/lesson',
    name: 'lesson',
    component: () => import('@/views/LessonView.vue'),
    meta: { title: 'Lesson' },
  },
  {
    path: '/quiz',
    name: 'quiz',
    component: () => import('@/views/QuizView.vue'),
    meta: { title: 'Knowledge check' },
  },
  {
    path: '/progress',
    name: 'progress',
    component: () => import('@/views/ProgressView.vue'),
    meta: { title: 'Progress' },
  },
  {
    path: '/library',
    name: 'library',
    component: () => import('@/views/LibraryView.vue'),
    meta: { title: 'Knowledge library' },
  },
  {
    path: '/interview',
    name: 'interview',
    component: () => import('@/views/InterviewView.vue'),
    meta: { title: 'Mock interview' },
  },
  {
    path: '/preferences',
    name: 'preferences',
    component: () => import('@/views/PreferencesView.vue'),
    meta: { title: 'Preferences' },
  },
  { path: '/:pathMatch(.*)*', redirect: '/today' },
];

const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.afterEach((to) => {
  document.title = `${String(to.meta.title || 'Coach')} — Nora`;
});

export default router;

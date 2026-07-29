<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useCoachStore } from '@/stores/coach';

const store = useCoachStore();
const mobileOpen = ref(false);
const notificationsOpen = ref(false);
const links = [
  { to: '/today', icon: '⌂', label: 'Today' },
  { to: '/lesson', icon: '◫', label: 'Learn' },
  { to: '/quiz', icon: '✓', label: 'Practice' },
  { to: '/progress', icon: '⌁', label: 'Progress' },
  { to: '/library', icon: '▤', label: 'Library' },
  { to: '/interview', icon: '◉', label: 'Interview' },
];

onMounted(() => store.initialize());
function retryWorkspace() {
  void store.initialize(true);
}
</script>

<template>
  <div class="app-shell">
    <button class="mobile-menu icon-button" aria-label="Open navigation" @click="mobileOpen = true">
      ☰
    </button>
    <div v-if="mobileOpen" class="scrim" @click="mobileOpen = false" />
    <aside class="sidebar" :class="{ open: mobileOpen }">
      <div class="brand">
        <span class="brand-mark">N</span
        ><span><strong>Nora</strong><small>AI learning coach</small></span>
      </div>
      <nav aria-label="Main navigation">
        <RouterLink v-for="link in links" :key="link.to" :to="link.to" @click="mobileOpen = false"
          ><span aria-hidden="true">{{ link.icon }}</span
          >{{ link.label }}</RouterLink
        >
      </nav>
      <RouterLink class="profile" to="/preferences" @click="mobileOpen = false"
        ><span class="avatar">A</span
        ><span
          ><strong>{{ store.preferences?.name || 'Learner' }}</strong
          ><small>{{ store.preferences?.mode || 'Learning preferences' }}</small></span
        ><span class="profile-arrow">›</span></RouterLink
      >
    </aside>

    <main class="main-content">
      <header class="topbar">
        <div class="topbar-stats">
          <span
            >🔥 <strong>{{ store.overview?.streak ?? 0 }}</strong> day streak</span
          ><span
            >✦ <strong>{{ (store.overview?.xp ?? 0).toLocaleString() }}</strong> XP</span
          >
        </div>
        <div class="notification-wrap">
          <button
            class="icon-button"
            aria-label="Notifications"
            :aria-expanded="notificationsOpen"
            @click="notificationsOpen = !notificationsOpen"
          >
            <span>◌</span><i v-if="store.unread">{{ store.unread }}</i>
          </button>
          <section v-if="notificationsOpen" class="notification-panel" aria-label="Notifications">
            <div class="panel-heading">
              <strong>Notifications</strong
              ><button class="text-button" @click="store.markNotificationsRead">
                Mark all read
              </button>
            </div>
            <p class="local-only-note">Notifications are kept in this browser session.</p>
            <article
              v-for="item in store.notifications"
              :key="item.id"
              :class="{ unread: !item.read }"
            >
              <span class="notice-dot" :class="item.tone" />
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.body }}</p>
                <small>{{ item.time }}</small>
              </div>
            </article>
          </section>
        </div>
      </header>
      <div v-if="store.error && !store.loading" class="workspace-notice" role="status">
        <span>Some workspace data could not be refreshed.</span
        ><button class="text-button" @click="retryWorkspace">Try again</button>
      </div>
      <RouterView />
    </main>
  </div>
</template>

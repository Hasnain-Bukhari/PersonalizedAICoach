<script setup lang="ts">
import { computed } from 'vue'; import { useCoachStore } from '@/stores/coach'; import PageHeading from '@/components/PageHeading.vue'; import ProgressRing from '@/components/ProgressRing.vue'
const store = useCoachStore(); const session = computed(() => store.session); const firstName = computed(() => store.preferences?.name.split(' ')[0] || 'Learner')
const date = new Intl.DateTimeFormat('en', { weekday: 'long', month: 'long', day: 'numeric' }).format(new Date())
</script>
<template><div class="page today-page" v-if="session">
  <PageHeading :eyebrow="date" :title="`Good evening, ${firstName}.`" description="Your coach has shaped today around the concepts that will move you forward most." />
  <section class="hero-session">
    <div class="hero-copy"><span class="tag accent">Today’s focus</span><h2>{{ session.title }}</h2><p>{{ session.summary }}</p><div class="session-meta"><span>◷ {{ session.duration }} min</span><span>◫ 1 lesson</span><span>✓ {{ session.questions.length }} questions</span></div><RouterLink class="button primary" to="/lesson">Continue session <span>→</span></RouterLink></div>
    <ProgressRing :value="session.progress" label="today" />
  </section>
  <div class="content-grid">
    <section class="card"><div class="section-heading"><div><span class="eyebrow">YOUR ITINERARY</span><h2>Built for today</h2></div><span class="quiet">{{ session.duration }} min total</span></div>
      <ol class="itinerary"><li><span class="step complete">✓</span><div><strong>Warm-up revision</strong><p>Consistency models · 8 min</p></div><span class="tag">Complete</span></li><li><span class="step current">02</span><div><strong>{{ session.lesson.title }}</strong><p>Guided lesson · 22 min</p></div><span class="tag accent">In progress</span></li><li><span class="step">03</span><div><strong>Knowledge check</strong><p>Adaptive quiz · 10 min</p></div></li><li><span class="step">04</span><div><strong>Reflect & lock it in</strong><p>Confidence check · 5 min</p></div></li></ol>
    </section>
    <aside class="card coach-note"><span class="coach-orb">✦</span><span class="eyebrow">NOTE FROM NORA</span><h2>Why this, today?</h2><p>Your last quiz showed that delivery guarantees are still fuzzy. Today connects that gap to a production scenario—so it sticks.</p><div class="insight"><span>↗</span><div><strong>Recovery opportunity</strong><p>A score above 80% moves this topic out of your weak queue.</p></div></div></aside>
  </div>
</div></template>

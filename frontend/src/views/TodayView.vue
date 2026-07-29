<script setup lang="ts">
import { computed } from 'vue';
import { useCoachStore } from '@/stores/coach';
import PageHeading from '@/components/PageHeading.vue';
import ProgressRing from '@/components/ProgressRing.vue';

const store = useCoachStore();
const session = computed(() => store.session);
const state = computed(() => store.resources.session);
const firstName = computed(() => store.preferences?.name.split(' ')[0] || 'Learner');
const date = new Intl.DateTimeFormat('en', {
  weekday: 'long',
  month: 'long',
  day: 'numeric',
}).format(new Date());
function retry() {
  void store.retrySession();
}
</script>

<template>
  <div class="page today-page">
    <section
      v-if="!session && (state.status === 'idle' || state.status === 'loading')"
      class="route-state"
      aria-live="polite"
    >
      <span class="spinner" />
      <h1>
        {{ store.sessionGeneration ? 'Building today’s session…' : 'Preparing your itinerary…' }}
      </h1>
      <p v-if="store.sessionGeneration">
        Generation step {{ store.sessionGeneration.attempt }} of
        {{ store.sessionGeneration.maxAttempts }}. You can stay on this page.
      </p>
      <p v-else>Loading your lesson, quiz, and learning priorities.</p>
    </section>
    <section
      v-else-if="!session && state.status === 'error'"
      class="route-state error-state"
      role="alert"
    >
      <span class="state-icon">!</span>
      <h1>
        {{
          store.sessionGeneration
            ? 'Today’s session is still being built'
            : 'Today’s session is unavailable'
        }}
      </h1>
      <p>{{ state.error }}</p>
      <p v-if="store.sessionGeneration" class="quiet">
        The coach reached the automatic check limit after
        {{ store.sessionGeneration.maxAttempts }} attempts. Checking again is safe and resumes from
        the same workflow.
      </p>
      <button class="button primary" @click="retry">
        {{ store.sessionGeneration ? 'Check again' : 'Try again' }}
      </button>
    </section>
    <section v-else-if="!session" class="route-state empty-state">
      <span class="state-icon">○</span>
      <h1>Nothing scheduled yet</h1>
      <p>Your coach has not published a session for today. Try refreshing in a moment.</p>
      <button class="button" @click="retry">Refresh session</button>
    </section>
    <template v-else>
      <PageHeading
        :eyebrow="date"
        :title="`Good evening, ${firstName}.`"
        description="Your coach has shaped today around the concepts that will move you forward most."
      />
      <section class="hero-session">
        <div class="hero-copy">
          <span class="tag accent">Today’s focus</span>
          <h2>{{ session.title }}</h2>
          <p>{{ session.summary }}</p>
          <div class="session-meta">
            <span>◷ {{ session.duration }} min</span><span>◫ 1 lesson</span
            ><span>✓ {{ session.questions.length }} questions</span>
          </div>
          <RouterLink class="button primary" to="/lesson"
            >{{ session.progress >= 100 ? 'Review session' : 'Continue session' }}
            <span>→</span></RouterLink
          >
        </div>
        <ProgressRing :value="session.progress" label="today" />
      </section>
      <div class="content-grid">
        <section class="card">
          <div class="section-heading">
            <div>
              <span class="eyebrow">YOUR ITINERARY</span>
              <h2>Built for today</h2>
            </div>
            <span class="quiet">{{ session.duration }} min total</span>
          </div>
          <ol class="itinerary">
            <li>
              <span class="step current">01</span>
              <div>
                <strong>{{ session.lesson.title }}</strong>
                <p>Guided lesson</p>
              </div>
              <span class="tag accent">Lesson</span>
            </li>
            <li>
              <span class="step">02</span>
              <div>
                <strong>Knowledge check</strong>
                <p>{{ session.questions.length }} adaptive questions</p>
              </div>
              <span class="tag">Quiz</span>
            </li>
          </ol>
        </section>
        <aside class="card coach-note">
          <span class="coach-orb">✦</span><span class="eyebrow">TODAY’S OBJECTIVES</span>
          <h2>What you’ll practice</h2>
          <ul v-if="session.objectives.length" class="objective-list">
            <li v-for="objective in session.objectives" :key="objective">{{ objective }}</li>
          </ul>
          <p v-else>Your lesson contains the key concepts selected for today.</p>
        </aside>
      </div>
    </template>
  </div>
</template>

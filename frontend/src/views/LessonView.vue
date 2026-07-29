<script setup lang="ts">
import { computed, ref } from 'vue';
import { useCoachStore } from '@/stores/coach';

const store = useCoachStore();
const lesson = computed(() => store.session?.lesson);
const state = computed(() => store.resources.session);
const sourceOpen = ref(false);
function retry() {
  void store.initialize(true);
}
</script>

<template>
  <div class="page lesson-page">
    <section
      v-if="!lesson && (state.status === 'idle' || state.status === 'loading')"
      class="route-state"
      aria-live="polite"
    >
      <span class="spinner" />
      <h1>Opening your lesson…</h1>
      <p>Loading today’s explanations and sources.</p>
    </section>
    <section
      v-else-if="!lesson && state.status === 'error'"
      class="route-state error-state"
      role="alert"
    >
      <span class="state-icon">!</span>
      <h1>Lesson unavailable</h1>
      <p>{{ state.error }}</p>
      <div class="state-actions">
        <button class="button primary" @click="retry">Try again</button
        ><RouterLink class="button" to="/today">Back to today</RouterLink>
      </div>
    </section>
    <section v-else-if="!lesson" class="route-state empty-state">
      <span class="state-icon">○</span>
      <h1>No lesson is ready</h1>
      <p>Open Today to generate or refresh your current session.</p>
      <RouterLink class="button primary" to="/today">Go to Today</RouterLink>
    </section>
    <template v-else>
      <div class="lesson-header">
        <RouterLink class="back-link" to="/today">← Today’s itinerary</RouterLink
        ><span class="tag accent">Lesson · 1 of 2</span>
        <h1>{{ lesson.title }}</h1>
        <p>{{ lesson.eyebrow }}</p>
        <div class="progress-track"><i style="width: 50%" /></div>
      </div>
      <div class="lesson-layout">
        <article class="lesson-content">
          <section
            v-for="(section, index) in lesson.sections"
            :id="`section-${index}`"
            :key="section.title"
          >
            <span class="section-number">0{{ index + 1 }}</span>
            <h2>{{ section.title }}</h2>
            <p>{{ section.body }}</p>
          </section>
          <section id="reference-architecture">
            <span class="section-number">{{
              String(lesson.sections.length + 1).padStart(2, '0')
            }}</span>
            <h2>Reference architecture</h2>
            <pre class="architecture" aria-label="Architecture diagram">{{ lesson.diagram }}</pre>
          </section>
          <div class="lesson-panels">
            <section class="callout danger">
              <strong>Production pitfalls</strong>
              <ul v-if="lesson.pitfalls.length">
                <li v-for="item in lesson.pitfalls" :key="item">{{ item }}</li>
              </ul>
              <p v-else>No additional pitfalls were provided.</p>
            </section>
            <section class="callout success">
              <strong>Keep this close</strong>
              <ul v-if="lesson.cheatSheet.length">
                <li v-for="item in lesson.cheatSheet" :key="item">{{ item }}</li>
              </ul>
              <p v-else>No cheat-sheet notes were provided.</p>
            </section>
          </div>
          <section class="sources">
            <button
              class="source-button"
              :aria-expanded="sourceOpen"
              @click="sourceOpen = !sourceOpen"
            >
              <span>▤</span
              ><span
                ><strong
                  >{{ lesson.sources.length }}
                  {{ lesson.sources.length === 1 ? 'source' : 'sources' }} used</strong
                ><small>{{
                  lesson.sources.length
                    ? 'Open the evidence behind this lesson'
                    : 'No uploaded source was cited for this lesson'
                }}</small></span
              ><span>{{ sourceOpen ? '⌃' : '⌄' }}</span>
            </button>
            <div v-if="sourceOpen && !lesson.sources.length" class="source-detail">
              <p>
                This lesson used general model knowledge. No private document citation was attached.
              </p>
            </div>
            <div
              v-for="source in sourceOpen ? lesson.sources : []"
              :key="source.id"
              class="source-detail"
            >
              <strong>{{ source.title }}</strong
              ><span>{{ source.location }}</span>
              <p>“{{ source.excerpt }}”</p>
            </div>
          </section>
          <div class="lesson-actions">
            <RouterLink class="button" to="/today">Back to itinerary</RouterLink
            ><RouterLink v-if="store.session?.questions.length" class="button primary" to="/quiz"
              >Start knowledge check →</RouterLink
            ><span v-else class="quiet">No knowledge check was included.</span>
          </div>
        </article>
        <aside class="lesson-outline card">
          <span class="eyebrow">IN THIS LESSON</span
          ><a v-for="(section, i) in lesson.sections" :key="section.title" :href="`#section-${i}`"
            ><span>0{{ i + 1 }}</span
            >{{ section.title }}</a
          ><a href="#reference-architecture"
            ><span>{{ String(lesson.sections.length + 1).padStart(2, '0') }}</span
            >Reference architecture</a
          >
        </aside>
      </div>
    </template>
  </div>
</template>

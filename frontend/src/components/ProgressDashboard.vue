<template>
  <div class="progress-dashboard">
    <h1>Multi-Topic Progress Dashboard</h1>

    <!-- Graphical Completion Stats -->
    <section class="completion-stats">
      <h2>Completion Stats</h2>
      <div class="stat-card">
        <h3>Total Topics Completed</h3>
        <p>{{ totalTopicsCompleted }}</p>
      </div>
      <div class="stat-card">
        <h3>Current Streak</h3>
        <p>{{ currentStreak }}</p>
      </div>
    </section>

    <!-- Historical Streak Calendar Visualizer -->
    <section class="streak-calendar">
      <h2>Historical Streak Calendar</h2>
      <vue-full-calendar :events="calendarEvents" />
    </section>

    <!-- Dynamic Topic Switching Cards -->
    <section class="topic-cards">
      <h2>Select a Topic</h2>
      <div v-for="topic in topics" :key="topic.id" class="topic-card" @click="selectTopic(topic)">
        <h3>{{ topic.name }}</h3>
        <p>{{ topic.description }}</p>
      </div>
    </section>

    <!-- Quick Action Buttons -->
    <section class="quick-actions">
      <h2>Quick Actions</h2>
      <button @click="launchPendingTasks">Launch Pending Tasks</button>
    </section>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref } from 'vue';
import VueFullCalendar from '@fullcalendar/vue3';

export default defineComponent({
  components: {
    VueFullCalendar,
  },
  setup() {
    const totalTopicsCompleted = ref(10);
    const currentStreak = ref(5);
    const topics = [
      { id: 1, name: 'Mathematics', description: 'Learn advanced mathematical concepts.' },
      { id: 2, name: 'Science', description: 'Explore the natural world and scientific principles.' },
      { id: 3, name: 'History', description: 'Study past events and civilizations.' },
    ];
    const selectedTopic = ref(null);

    const calendarEvents = [
      { title: 'Streak Break', start: '2023-10-01' },
      { title: 'Streak Continue', start: '2023-10-02' },
      // Add more events as needed
    ];

    const selectTopic = (topic) => {
      selectedTopic.value = topic;
    };

    const launchPendingTasks = () => {
      alert('Launching pending tasks...');
      // Implement logic to launch today's pending workspace tasks
    };

    return {
      totalTopicsCompleted,
      currentStreak,
      topics,
      calendarEvents,
      selectTopic,
      launchPendingTasks,
    };
  },
});
</script>

<style scoped>
.progress-dashboard {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.completion-stats, .streak-calendar, .topic-cards, .quick-actions {
  margin-bottom: 2rem;
}

.stat-card, .topic-card {
  border: 1px solid #ccc;
  padding: 1rem;
  cursor: pointer;
}

.topic-card:hover {
  background-color: #f0f0f0;
}
</style>
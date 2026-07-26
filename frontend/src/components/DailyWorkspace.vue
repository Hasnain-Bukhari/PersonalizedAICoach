<template>
  <div class="daily-workspace">
    <h1>Daily Learning Workspace</h1>
    
    <!-- Markdown Previewer -->
    <div class="markdown-preview">
      <h2>Markdown Previewer</h2>
      <textarea v-model="markdownText" @input="updatePreview"></textarea>
      <div v-html="compiledMarkdown"></div>
    </div>

    <!-- YouTube Embed Player with Takeaway Points -->
    <div class="youtube-player">
      <h2>YouTube Video</h2>
      <iframe
        :src="`https://www.youtube.com/embed/${videoId}`"
        frameborder="0"
        allowfullscreen
      ></iframe>
      <ul>
        <li v-for="(point, index) in takeawayPoints" :key="index">
          {{ point }}
        </li>
      </ul>
    </div>

    <!-- GitHub Architecture Summary Card -->
    <div class="github-summary">
      <h2>GitHub Architecture Summary</h2>
      <p>{{ githubSummary }}</p>
    </div>

    <!-- Interactive Quiz Widget with Instant Validation -->
    <div class="quiz-widget">
      <h2>Interactive Quiz</h2>
      <form @submit.prevent="validateQuiz">
        <div v-for="(question, index) in quizQuestions" :key="index">
          <p>{{ question.text }}</p>
          <input
            type="text"
            v-model="question.answer"
            placeholder="Your answer"
          />
        </div>
        <button type="submit">Submit</button>
      </form>
      <p v-if="quizResult">{{ quizResult }}</p>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, ref, computed } from 'vue';
import MarkdownIt from 'markdown-it';
import Prism from 'prismjs';

export default defineComponent({
  name: 'DailyWorkspace',
  setup() {
    const markdownText = ref('');
    const videoId = ref('dQw4w9WgXcQ'); // Example YouTube video ID
    const takeawayPoints = ref(['Point 1', 'Point 2', 'Point 3']);
    const githubSummary = ref('GitHub is a platform for version control and collaboration. It allows developers to store their code, manage projects, and collaborate with others.');
    
    const quizQuestions = ref([
      { text: 'What is Vue.js?', answer: '' },
      { text: 'What is the purpose of GitHub?', answer: '' }
    ]);
    
    const quizResult = ref('');

    const markdownIt = new MarkdownIt({
      html: true,
      linkify: true,
      typographer: true,
      highlight: function (str, lang) {
        if (lang && Prism.languages[lang]) {
          return `<pre class="language-${lang}"><code>${Prism.highlight(str, Prism.languages[lang], lang)}</code></pre>`;
        }
        return `<pre class="language-text"><code>${Prism.util.encode(str)}</code></pre>`;
      }
    });

    const updatePreview = () => {
      compiledMarkdown.value = markdownIt.render(markdownText.value);
    };

    const validateQuiz = () => {
      let correctAnswers = 0;
      quizQuestions.value.forEach((question, index) => {
        if (question.answer.trim().toLowerCase() === question.text.split('?')[1].trim().toLowerCase()) {
          correctAnswers++;
        }
      });
      quizResult.value = `You got ${correctAnswers} out of ${quizQuestions.value.length} questions right.`;
    };

    return {
      markdownText,
      videoId,
      takeawayPoints,
      githubSummary,
      quizQuestions,
      quizResult,
      updatePreview,
      validateQuiz
    };
  }
});
</script>

<style scoped>
.daily-workspace {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px;
}

.markdown-preview, .youtube-player, .github-summary, .quiz-widget {
  margin: 20px;
  padding: 10px;
  border: 1px solid #ccc;
  width: 80%;
}

textarea {
  width: 100%;
  height: 200px;
}
</style>
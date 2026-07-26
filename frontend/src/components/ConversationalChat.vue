<template>
  <div class="chat-container">
    <h2>Conversational Chat</h2>
    <div ref="chatBox" class="chat-box"></div>
    <input v-model="userInput" @keyup.enter="sendMessage" placeholder="Type a message..." />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';

const chatBox = ref<HTMLElement | null>(null);
const userInput = ref('');
const sseUrl = '/api/generate-plan'; // Adjust the URL as needed

onMounted(() => {
  const eventSource = new EventSource(sseUrl);

  eventSource.onmessage = (event) => {
    if (chatBox.value) {
      chatBox.value.innerHTML += `<p>${event.data}</p>`;
      chatBox.value.scrollTop = chatBox.value.scrollHeight;
    }
  };

  eventSource.onerror = (error) => {
    console.error('EventSource failed:', error);
    eventSource.close();
  };
});

const sendMessage = () => {
  if (userInput.value.trim() !== '') {
    // Simulate sending a message to the server
    const messageElement = document.createElement('p');
    messageElement.textContent = `User: ${userInput.value}`;
    chatBox.value?.appendChild(messageElement);
    userInput.value = '';
  }
};
</script>

<style scoped>
.chat-container {
  border: 1px solid #ccc;
  padding: 20px;
  margin-top: 20px;
}

.chat-box {
  height: 300px;
  overflow-y: scroll;
  border-bottom: 1px solid #ccc;
  padding-bottom: 10px;
}

input {
  width: calc(100% - 22px);
  padding: 10px;
  margin-top: 10px;
}
</style>
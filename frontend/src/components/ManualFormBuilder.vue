<template>
  <div class="form-container">
    <h2>Manual Form Builder</h2>
    <form @submit.prevent="handleSubmit">
      <div>
        <label for="topic">Topic:</label>
        <input v-model="formData.topic" id="topic" type="text" required />
      </div>
      <div>
        <label for="duration">Duration (in hours):</label>
        <input v-model.number="formData.duration" id="duration" type="number" required />
      </div>
      <div>
        <label for="dailyHours">Daily Hours:</label>
        <input v-model.number="formData.dailyHours" id="dailyHours" type="number" required />
      </div>
      <button type="submit">Submit</button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';

const formData = ref({
  topic: '',
  duration: 0,
  dailyHours: 0
});

const handleSubmit = () => {
  // Handle form submission logic here
  console.log('Form Data:', formData.value);
};
</script>

<style scoped>
.form-container {
  border: 1px solid #ccc;
  padding: 20px;
  margin-top: 20px;
}

form {
  display: flex;
  flex-direction: column;
}

label {
  margin-top: 10px;
}

input {
  padding: 10px;
  margin-top: 5px;
}

button {
  margin-top: 20px;
}
</style>
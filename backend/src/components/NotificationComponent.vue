<template>
  <div>
    <button @click="requestPermission">Request Permission</button>
    <button @click="sendLocalNotification">Send Local Notification</button>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { Plugins } from '@capacitor/core';
const { PushNotifications } = Plugins;

export default defineComponent({
  name: 'NotificationComponent',
  methods: {
    async requestPermission() {
      const status = await PushNotifications.requestPermissions();
      if (status.granted) {
        console.log('Permission granted');
      } else {
        console.log('Permission denied');
      }
    },
    async sendLocalNotification() {
      await PushNotifications.register();
      await PushNotifications.send({
        title: 'Hello',
        body: 'This is a local notification',
      });
    },
  },
});
</script>
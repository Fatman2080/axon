<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import LayoutShell from './components/LayoutShell.vue';

const route = useRoute();
const isLoginPage = computed(() => route.path === '/login');
</script>

<template>
  <LayoutShell v-if="!isLoginPage">
    <RouterView />
  </LayoutShell>
  <RouterView v-else />
</template>

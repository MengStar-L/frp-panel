<template>
  <router-view v-slot="{ Component }">
    <transition name="fade-slide" mode="out-in">
      <component :is="Component" />
    </transition>
  </router-view>
  <ToastHost />
</template>

<script setup>
import ToastHost from './components/ToastHost.vue'
</script>

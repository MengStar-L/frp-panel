<template>
  <div class="progress" :class="{ indeterminate }">
    <div class="bar" :style="indeterminate ? {} : { width: pct + '%' }"></div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  value: { type: Number, default: 0 },
  indeterminate: { type: Boolean, default: false },
})

const pct = computed(() => Math.max(0, Math.min(100, props.value)))
</script>

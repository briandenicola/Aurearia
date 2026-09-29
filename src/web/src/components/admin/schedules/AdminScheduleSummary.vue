<template>
  <span class="form-hint" data-testid="schedule-summary">{{ summary }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { describeSchedule } from '@/composables/useScheduleTimezone'

const props = defineProps<{
  startTime?: string
  defaultStartTime: string
  zone?: string
  intervalMinutes?: string | number
  intervalDays?: string | number
}>()

const summary = computed(() => describeSchedule(props))
</script>

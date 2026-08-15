<template>
  <div class="container">
    <div class="mx-auto max-w-[900px]">
      <div class="page-header">
        <h1>Deep Analysis</h1>
        <div class="pwa-actions">
          <RouterLink class="pwa-icon-btn" to="/lookup" title="Identify Coin" aria-label="Identify Coin">
            <Search :size="22" />
          </RouterLink>
        </div>
      </div>

      <p v-if="!jobId" class="rounded-md border border-border-subtle bg-card p-6 text-base text-text-secondary shadow-[var(--shadow-card)]">
        Start a Deep Analysis from Identify Coin or an existing saved coin to see progress here.
      </p>

      <template v-else>
        <p v-if="loading" class="text-body text-text-secondary">Loading Deep Analysis job...</p>
        <p v-else-if="loadError" role="alert" class="text-body text-byzantine">{{ loadError }}</p>

        <section v-else-if="job" class="card grid gap-4">
          <div class="flex items-center justify-between">
            <h2 class="m-0 text-lg text-heading">Job #{{ job.id }}</h2>
            <BaseBadge>{{ job.status }}</BaseBadge>
          </div>
          <p class="m-0 text-body text-text-secondary">
            Deep Analysis runs Nomisma and Numista automatically. NGC results link out only; OCRE and RPC
            remain manual for this job. Progress and results will appear here as the job runs.
          </p>
        </section>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { Search } from 'lucide-vue-next'
import BaseBadge from '@/components/ui/BaseBadge.vue'
import { useDeepIdentification } from '@/composables/useDeepIdentification'

const route = useRoute()
const jobId = computed(() => {
  const raw = route.params.jobId
  const value = Array.isArray(raw) ? raw[0] : raw
  const parsed = value ? Number(value) : NaN
  return Number.isFinite(parsed) ? parsed : null
})

const { job, loading, error: loadError, refresh } = useDeepIdentification()

onMounted(async () => {
  if (jobId.value !== null) {
    await refresh(jobId.value)
  }
})
</script>

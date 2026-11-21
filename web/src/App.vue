<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { api, getToken, setToken, type Audit, type Channel, type Member } from './api'
import { useI18n, type Locale, type MessageKey } from './i18n'

type Tab = 'channels' | 'members' | 'audits'

const { locale, t, setLocale } = useI18n()

const statusKeys: Record<string, MessageKey> = {
  active: 'value.active',
  disabled: 'value.disabled',
}
const resultKeys: Record<string, MessageKey> = {
  success: 'value.success',
  policy_rejected: 'value.policy_rejected',
  quota_exceeded: 'value.quota_exceeded',
  rate_limited: 'value.rate_limited',
  unknown: 'value.unknown',
}
const sourceKeys: Record<string, MessageKey> = {
  enterprise: 'value.enterprise',
  personal: 'value.personal',
}

function translateValue(map: Record<string, MessageKey>, raw: string): string {
  const key = map[raw]
  return key ? t(key) : raw
}

function statusLabel(raw: string): string {
  return translateValue(statusKeys, raw)
}

function resultLabel(raw: string): string {
  return translateValue(resultKeys, raw)
}

function sourceLabel(raw: string): string {
  return translateValue(sourceKeys, raw)
}

const tab = ref<Tab>('channels')
const tokenInput = ref(getToken())
const error = ref('')
const busy = ref(false)

const channels = ref<Channel[]>([])
const channelForm = reactive({
  name: 'local-fake',
  base_url: 'http://127.0.0.1:9099',
  api_key: '',
  models: 'gpt-4o-mini',
})
const testResults = reactive<Record<number, boolean>>({})

const members = ref<Member[]>([])
const memberForm = reactive({ name: '', enterprise_limit: 10000 })
const issuedToken = ref('')

const audits = ref<Audit[]>([])

async function run(action: () => Promise<void>): Promise<void> {
  error.value = ''
  busy.value = true
  try {
    await action()
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}

function saveToken(): void {
  setToken(tokenInput.value.trim())
  refreshAll()
}

function refreshAll(): void {
  run(async () => {
    await Promise.all([loadChannels(), loadMembers(), loadAudits()])
  })
}

async function loadChannels(): Promise<void> {
  channels.value = await api.channels()
}

async function loadMembers(): Promise<void> {
  members.value = await api.members()
}

async function loadAudits(): Promise<void> {
  audits.value = await api.audits()
}

function addChannel(): void {
  run(async () => {
    await api.createChannel({
      name: channelForm.name,
      base_url: channelForm.base_url,
      api_key: channelForm.api_key,
      models: channelForm.models.split(',').map((m) => m.trim()).filter(Boolean),
    })
    channelForm.api_key = ''
    await loadChannels()
  })
}

function toggleChannel(ch: Channel): void {
  run(async () => {
    await api.setChannelEnabled(ch.id, !ch.enabled)
    await loadChannels()
  })
}

function testChannel(ch: Channel): void {
  run(async () => {
    const res = await api.testChannel(ch.id)
    testResults[ch.id] = res.ok
  })
}

function addMember(): void {
  run(async () => {
    const created = await api.createMember({
      name: memberForm.name,
      enterprise_limit: Number(memberForm.enterprise_limit),
    })
    issuedToken.value = created.token
    memberForm.name = ''
    await loadMembers()
  })
}

function switchTab(next: Tab): void {
  tab.value = next
  if (next === 'audits') run(loadAudits)
  if (next === 'members') run(loadMembers)
  if (next === 'channels') run(loadChannels)
}

function chooseLocale(next: Locale): void {
  setLocale(next)
}

onMounted(() => {
  tokenInput.value = getToken()
  if (tokenInput.value) refreshAll()
})
</script>

<template>
  <header class="topbar">
    <div>
      <h1>OneMore</h1>
      <p class="sub">{{ t('app.subtitle') }}</p>
    </div>
    <div class="top-actions">
      <div class="locale" role="group" aria-label="Language">
        <button :class="{ active: locale === 'en' }" @click="chooseLocale('en')">EN</button>
        <button :class="{ active: locale === 'zh' }" @click="chooseLocale('zh')">中文</button>
      </div>
      <div class="token-box">
        <label for="admin-token">{{ t('auth.label') }}</label>
        <input id="admin-token" v-model="tokenInput" type="password" placeholder="dev-admin-token" />
        <button @click="saveToken">{{ t('auth.save') }}</button>
      </div>
    </div>
  </header>

  <p v-if="error" class="error">{{ error }}</p>
  <p v-if="busy" class="status">{{ t('status.working') }}</p>

  <nav class="tabs">
    <button :class="{ active: tab === 'channels' }" @click="switchTab('channels')">{{ t('nav.channels') }}</button>
    <button :class="{ active: tab === 'members' }" @click="switchTab('members')">{{ t('nav.members') }}</button>
    <button :class="{ active: tab === 'audits' }" @click="switchTab('audits')">{{ t('nav.audit') }}</button>
  </nav>

  <main>
    <section v-if="tab === 'channels'" class="panel">
      <h2>{{ t('channels.title') }}</h2>
      <form class="form-grid" @submit.prevent="addChannel">
        <label>{{ t('channels.name') }}<input v-model="channelForm.name" required /></label>
        <label>{{ t('channels.baseUrl') }}<input v-model="channelForm.base_url" required /></label>
        <label>{{ t('channels.apiKey') }}<input v-model="channelForm.api_key" type="password" required /></label>
        <label>{{ t('channels.models') }}<input v-model="channelForm.models" required /></label>
        <button type="submit">{{ t('channels.add') }}</button>
      </form>

      <table>
        <thead>
          <tr>
            <th>ID</th><th>{{ t('channels.name') }}</th><th>{{ t('channels.baseUrl') }}</th>
            <th>{{ t('channels.key') }}</th><th>{{ t('channels.models') }}</th><th>{{ t('channels.enabled') }}</th><th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="ch in channels" :key="ch.id">
            <td>{{ ch.id }}</td>
            <td>{{ ch.name }}</td>
            <td class="mono">{{ ch.base_url }}</td>
            <td class="mono">{{ ch.api_key_masked }}</td>
            <td>{{ ch.models.join(', ') }}</td>
            <td>{{ ch.enabled ? t('common.yes') : t('common.no') }}</td>
            <td class="actions">
              <button @click="toggleChannel(ch)">
                {{ ch.enabled ? t('common.disable') : t('common.enable') }}
              </button>
              <button @click="testChannel(ch)">{{ t('common.test') }}</button>
              <span v-if="testResults[ch.id] !== undefined" class="hint">
                {{ t(testResults[ch.id] ? 'channels.reachable' : 'channels.unreachable') }}
              </span>
            </td>
          </tr>
          <tr v-if="channels.length === 0"><td colspan="7" class="empty">{{ t('channels.empty') }}</td></tr>
        </tbody>
      </table>
    </section>

    <section v-if="tab === 'members'" class="panel">
      <h2>{{ t('members.title') }}</h2>
      <form class="form-grid" @submit.prevent="addMember">
        <label>{{ t('members.name') }}<input v-model="memberForm.name" required /></label>
        <label>{{ t('members.quota') }}<input v-model.number="memberForm.enterprise_limit" type="number" min="0" required /></label>
        <button type="submit">{{ t('members.create') }}</button>
      </form>

      <div v-if="issuedToken" class="issued">
        <strong>{{ t('members.tokenNotice') }}</strong>
        <code>{{ issuedToken }}</code>
      </div>

      <table>
        <thead>
          <tr>
            <th>ID</th><th>{{ t('members.name') }}</th><th>{{ t('members.status') }}</th><th>{{ t('members.quota') }}</th>
            <th>{{ t('members.used') }}</th><th>{{ t('members.reserved') }}</th><th>{{ t('members.remaining') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m.id">
            <td>{{ m.id }}</td>
            <td>{{ m.name }}</td>
            <td>{{ statusLabel(m.status) }}</td>
            <td>{{ m.enterprise_limit }}</td>
            <td>{{ m.enterprise_used }}</td>
            <td>{{ m.enterprise_reserved }}</td>
            <td>{{ m.remaining }}</td>
          </tr>
          <tr v-if="members.length === 0"><td colspan="7" class="empty">{{ t('members.empty') }}</td></tr>
        </tbody>
      </table>
    </section>

    <section v-if="tab === 'audits'" class="panel">
      <div class="panel-head">
        <h2>{{ t('audit.title') }}</h2>
        <button @click="run(loadAudits)">{{ t('common.refresh') }}</button>
      </div>
      <table>
        <thead>
          <tr>
            <th>{{ t('audit.requestId') }}</th><th>{{ t('audit.time') }}</th><th>{{ t('audit.member') }}</th>
            <th>{{ t('audit.channel') }}</th><th>{{ t('audit.source') }}</th><th>{{ t('audit.model') }}</th>
            <th>{{ t('audit.result') }}</th><th>{{ t('audit.code') }}</th><th>{{ t('audit.http') }}</th>
            <th>{{ t('audit.tokens') }}</th><th>{{ t('audit.ms') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in audits" :key="a.request_id">
            <td class="mono">{{ a.request_id }}</td>
            <td>{{ a.created_at }}</td>
            <td>{{ a.member_name }}</td>
            <td>{{ a.channel_id || '-' }}</td>
            <td>{{ sourceLabel(a.quota_source) }}</td>
            <td>{{ a.model }}</td>
            <td>{{ resultLabel(a.result) }}</td>
            <td class="mono">{{ a.error_code || '-' }}</td>
            <td>{{ a.http_status }}</td>
            <td>{{ a.total_tokens }}</td>
            <td>{{ a.latency_ms }}</td>
          </tr>
          <tr v-if="audits.length === 0"><td colspan="11" class="empty">{{ t('audit.empty') }}</td></tr>
        </tbody>
      </table>
    </section>
  </main>
</template>

import { ref } from 'vue'

export type Locale = 'en' | 'zh'

const en = {
  'app.subtitle': 'Enterprise AI gateway console · M1',
  'app.title': 'OneMore Console',
  'auth.label': 'Admin token',
  'auth.save': 'Save',
  'nav.channels': 'Channels',
  'nav.members': 'Members',
  'nav.audit': 'Audit',
  'status.working': 'working…',
  'common.yes': 'yes',
  'common.no': 'no',
  'common.enable': 'Enable',
  'common.disable': 'Disable',
  'common.test': 'Test',
  'common.refresh': 'Refresh',
  'channels.title': 'Model channels',
  'channels.name': 'Name',
  'channels.baseUrl': 'Base URL',
  'channels.apiKey': 'API key',
  'channels.models': 'Models (comma separated)',
  'channels.key': 'Key',
  'channels.enabled': 'Enabled',
  'channels.add': 'Add channel',
  'channels.empty': 'No channels yet.',
  'channels.reachable': 'reachable',
  'channels.unreachable': 'unreachable',
  'members.title': 'Members',
  'members.name': 'Name',
  'members.status': 'Status',
  'members.quota': 'Enterprise quota',
  'members.create': 'Create member',
  'members.tokenNotice': 'Copy this member token now; it is shown once.',
  'members.empty': 'No members yet.',
  'members.used': 'Used',
  'members.reserved': 'Reserved',
  'members.remaining': 'Remaining',
  'audit.title': 'Call audit',
  'audit.requestId': 'Request ID',
  'audit.time': 'Time',
  'audit.member': 'Member',
  'audit.channel': 'Channel',
  'audit.source': 'Source',
  'audit.model': 'Model',
  'audit.result': 'Result',
  'audit.code': 'Code',
  'audit.http': 'HTTP',
  'audit.tokens': 'Tokens',
  'audit.ms': 'ms',
  'audit.empty': 'No calls recorded yet.',
  'value.active': 'active',
  'value.disabled': 'disabled',
  'value.success': 'success',
  'value.policy_rejected': 'policy blocked',
  'value.quota_exceeded': 'quota exceeded',
  'value.rate_limited': 'upstream rate limited',
  'value.unknown': 'result unknown',
  'value.enterprise': 'enterprise',
  'value.personal': 'personal',
} as const

export type MessageKey = keyof typeof en

const zh: Record<MessageKey, string> = {
  'app.subtitle': '企业 AI 网关控制台 · M1',
  'app.title': 'OneMore 控制台',
  'auth.label': '管理员 Token',
  'auth.save': '保存',
  'nav.channels': '模型渠道',
  'nav.members': '成员',
  'nav.audit': '调用审计',
  'status.working': '处理中…',
  'common.yes': '是',
  'common.no': '否',
  'common.enable': '启用',
  'common.disable': '停用',
  'common.test': '测试',
  'common.refresh': '刷新',
  'channels.title': '模型渠道',
  'channels.name': '名称',
  'channels.baseUrl': '基础 URL',
  'channels.apiKey': 'API Key',
  'channels.models': '模型（逗号分隔）',
  'channels.key': '密钥',
  'channels.enabled': '启用状态',
  'channels.add': '添加渠道',
  'channels.empty': '暂无渠道。',
  'channels.reachable': '可连通',
  'channels.unreachable': '不可连通',
  'members.title': '成员',
  'members.name': '名称',
  'members.status': '状态',
  'members.quota': '企业额度',
  'members.create': '创建成员',
  'members.tokenNotice': '请立即复制成员 Token，它只显示一次。',
  'members.empty': '暂无成员。',
  'members.used': '已用',
  'members.reserved': '预留',
  'members.remaining': '剩余',
  'audit.title': '调用审计',
  'audit.requestId': '请求 ID',
  'audit.time': '时间',
  'audit.member': '成员',
  'audit.channel': '渠道',
  'audit.source': '额度来源',
  'audit.model': '模型',
  'audit.result': '结果',
  'audit.code': '错误码',
  'audit.http': 'HTTP',
  'audit.tokens': 'Token',
  'audit.ms': '毫秒',
  'audit.empty': '暂无调用记录。',
  'value.active': '启用',
  'value.disabled': '停用',
  'value.success': '成功',
  'value.policy_rejected': '策略拦截',
  'value.quota_exceeded': '额度不足',
  'value.rate_limited': '上游限流',
  'value.unknown': '结果未知',
  'value.enterprise': '企业',
  'value.personal': '个人',
}

const messages: Record<Locale, Record<MessageKey, string>> = { en, zh }

const LOCALE_KEY = 'onemore.locale'

function initialLocale(): Locale {
  const stored = localStorage.getItem(LOCALE_KEY)
  if (stored === 'en' || stored === 'zh') return stored
  return navigator.language.toLowerCase().startsWith('zh') ? 'zh' : 'en'
}

const locale = ref<Locale>(initialLocale())

function applyDocument(l: Locale): void {
  document.documentElement.lang = l === 'zh' ? 'zh-CN' : 'en'
  document.title = messages[l]['app.title']
}

applyDocument(locale.value)

export function useI18n() {
  function t(key: MessageKey): string {
    return messages[locale.value][key] ?? messages.en[key]
  }

  function setLocale(next: Locale): void {
    locale.value = next
    localStorage.setItem(LOCALE_KEY, next)
    applyDocument(next)
  }

  return { locale, t, setLocale }
}

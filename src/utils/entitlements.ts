import type { Page } from '../types/navigation';
import type { PlatformId } from '../types/platform';
import type { EffectiveEntitlements } from '../stores/useAuthStore';

export function platformFeatureKey(platformId: PlatformId): string {
  return `platform.${platformId}`;
}

export function hasFeature(
  entitlements: EffectiveEntitlements | null,
  key: string,
): boolean {
  return !!entitlements?.has_subscription && entitlements.features[key] === true;
}

const PAGE_FEATURES: Partial<Record<Page, string>> = {
  'api-relay': 'feature.api_relay',
  overview: 'platform.antigravity',
  instances: 'platform.antigravity',
  wakeup: 'platform.antigravity',
  verification: 'platform.antigravity',
  codex: 'platform.codex',
  'codex-api-service': 'platform.codex_api_service',
  claude: 'platform.claude_manager',
  'claude-cli': 'platform.claude_manager',
  zed: 'platform.zed',
  'github-copilot': 'platform.github-copilot',
  windsurf: 'platform.windsurf',
  kiro: 'platform.kiro',
  cursor: 'platform.cursor',
  grok: 'platform.grok',
  codebuddy: 'platform.codebuddy',
  'codebuddy-cn': 'platform.codebuddy_cn',
  qoder: 'platform.qoder',
  zcode: 'platform.zcode',
  trae: 'platform.trae',
  'trae-solo': 'platform.trae_solo',
  'trae-cn': 'platform.trae_cn',
  'trae-solo-cn': 'platform.trae_solo_cn',
  workbuddy: 'platform.workbuddy',
};

export function canAccessPage(
  entitlements: EffectiveEntitlements | null,
  page: Page,
): boolean {
  const feature = PAGE_FEATURES[page];
  if (!feature) return true;
  if (!hasFeature(entitlements, feature)) return false;
  return page === 'api-relay' || hasFeature(entitlements, 'feature.local_switch');
}

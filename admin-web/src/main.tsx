import { FormEvent, useCallback, useEffect, useState } from 'react';
import { createRoot } from 'react-dom/client';
import { BarChart3, CreditCard, Database, KeyRound, LogOut, Monitor, Package, Plus, RefreshCw, ShieldCheck, Ticket, Users, X } from 'lucide-react';
import './styles.css';
import './permissions.css';

const API_BASE = import.meta.env.VITE_AISWITCH_API_BASE || 'https://clawbotchat.com/aiswitch/api/v1';
const TOKEN_KEY = 'aiswitch.admin.access_token';
const REFRESH_KEY = 'aiswitch.admin.refresh_token';
const USER_KEY = 'aiswitch.admin.user';

type User = { id: string; email: string; nickname?: string; is_admin: boolean; is_active: boolean; created_at: string };
type Tab = 'overview' | 'users' | 'plans' | 'accounts' | 'cards' | 'payments';
type Feature = { key: string; name: string; category: string };
type Plan = { id: string; name: string; duration_days: number; max_accounts: number; max_devices: number; price_cents: number; is_active: boolean; entitlements: Record<string, boolean> };

function storedUser(): User | null {
  try { return JSON.parse(localStorage.getItem(USER_KEY) || 'null'); } catch { return null; }
}

async function request(path: string, options: RequestInit = {}) {
  const token = localStorage.getItem(TOKEN_KEY);
  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', ...(token ? { Authorization: `Bearer ${token}` } : {}), ...options.headers },
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `请求失败（${response.status}）`);
  }
  return response.json();
}

function Login({ onAuthenticated }: { onAuthenticated: (user: User) => void }) {
  const [email, setEmail] = useState('admin@clawbotchat.com');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const submit = async (event: FormEvent) => {
    event.preventDefault(); setBusy(true); setError('');
    try {
      const data = await request('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) });
      if (!data.user?.is_admin) throw new Error('此账号没有管理后台权限');
      localStorage.setItem(TOKEN_KEY, data.access_token);
      localStorage.setItem(REFRESH_KEY, data.refresh_token);
      localStorage.setItem(USER_KEY, JSON.stringify(data.user));
      onAuthenticated(data.user);
    } catch (err) { setError(err instanceof Error ? err.message : '登录失败'); }
    finally { setBusy(false); }
  };
  return <main className="login-shell"><section className="login-brand"><div className="brand-mark">↯</div><p className="eyebrow">AISWITCH CONTROL</p><h1>你的订阅服务，<br /><span>一处掌控。</span></h1><p className="brand-copy">管理账号池、订阅、卡密与用户访问权限。所有运营动作都留有记录。</p><div className="brand-points"><span>● 安全账号分配</span><span>● 订阅生命周期</span><span>● 实时运营数据</span></div></section><section className="login-panel"><form className="login-card" onSubmit={submit}><div className="mobile-brand"><div className="brand-mark">↯</div><strong>AISwitch</strong></div><p className="eyebrow">管理员入口</p><h2>欢迎回来</h2><p className="muted">使用管理员账号登录控制台</p>{error && <p className="form-error">{error}</p>}<label>邮箱<input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required autoComplete="username" /></label><label>密码<input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required autoComplete="current-password" /></label><button className="primary" disabled={busy}>{busy ? '正在验证…' : '进入管理后台 →'}</button><p className="security-note"><ShieldCheck size={15} /> 仅限已授权管理员访问</p></form></section></main>;
}

function Overview() {
  const [stats, setStats] = useState<Record<string, number> | null>(null);
  const [error, setError] = useState('');
  const load = useCallback(async () => { try { setError(''); setStats(await request('/admin/dashboard')); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, []);
  useEffect(() => { void load(); }, [load]);
  const cards = [['总用户', stats?.total_users, Users], ['活跃订阅', stats?.active_subscriptions, ShieldCheck], ['可分配账号', stats?.available_accounts, Database], ['已分配账号', stats?.assigned_accounts, KeyRound], ['本月收入', stats ? `¥${((stats.monthly_revenue_cents || 0) / 100).toFixed(2)}` : '—', CreditCard]] as const;
  return <section><div className="page-heading"><div><p className="eyebrow">运营概览</p><h2>今天的业务状态</h2></div><button className="secondary" onClick={() => void load()}><RefreshCw size={16} /> 刷新数据</button></div>{error && <p className="form-error">{error}</p>}<div className="stat-grid">{cards.map(([label, value, Icon]) => <article className="stat-card" key={label}><span className="stat-icon"><Icon size={19} /></span><p>{label}</p><strong>{value ?? '—'}</strong></article>)}</div><article className="welcome-card"><div><p className="eyebrow">准备开始</p><h3>先导入账号，再生成套餐卡密</h3><p>账号进入“待分配”状态后，用户兑换卡密即可自动领取。</p></div><Database size={42} /></article></section>;
}

function UsersPage() {
  const [items, setItems] = useState<User[]>([]); const [error, setError] = useState(''); const [selected, setSelected] = useState<User | null>(null);
  const load = useCallback(async () => { try { setItems(await request('/admin/users')); setError(''); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, []);
  useEffect(() => { void load(); }, [load]);
  const toggle = async (item: User) => { await request(`/admin/users/${item.id}`, { method: 'PUT', body: JSON.stringify({ is_active: !item.is_active }) }); await load(); };
  return <section><div className="page-heading"><div><p className="eyebrow">用户管理</p><h2>用户与访问状态</h2></div><button className="secondary" onClick={() => void load()}><RefreshCw size={16} /> 刷新</button></div>{error && <p className="form-error">{error}</p>}<Table headers={['邮箱', '昵称', '权限', '状态', '注册时间', '操作']}>{items.map((item) => <tr key={item.id}><td>{item.email}</td><td>{item.nickname || '—'}</td><td><Badge tone={item.is_admin ? 'violet' : 'neutral'}>{item.is_admin ? '管理员' : '用户'}</Badge></td><td><Badge tone={item.is_active ? 'green' : 'red'}>{item.is_active ? '正常' : '已禁用'}</Badge></td><td>{new Date(item.created_at).toLocaleDateString('zh-CN')}</td><td><div className="action-row"><button className="text-button" onClick={() => setSelected(item)}>权限与设备</button><button className="text-button" onClick={() => void toggle(item)}>{item.is_active ? '禁用' : '启用'}</button></div></td></tr>)}</Table>{selected && <UserAccessPanel user={selected} onClose={() => setSelected(null)} />}</section>;
}

function UserAccessPanel({ user, onClose }: { user: User; onClose: () => void }) {
  const [features, setFeatures] = useState<Feature[]>([]); const [detail, setDetail] = useState<any>(null); const [devices, setDevices] = useState<any[]>([]); const [overrides, setOverrides] = useState<Record<string, boolean>>({}); const [maxAccounts, setMaxAccounts] = useState(''); const [maxDevices, setMaxDevices] = useState(''); const [error, setError] = useState('');
  const load = useCallback(async () => { try { const [featureData, accessData, deviceData] = await Promise.all([request('/admin/features'), request(`/admin/users/${user.id}/entitlements`), request(`/admin/users/${user.id}/devices`)]); setFeatures(featureData); setDetail(accessData); setDevices(deviceData); setOverrides(accessData.override?.entitlements || {}); setMaxAccounts(accessData.override?.max_accounts?.toString() || ''); setMaxDevices(accessData.override?.max_devices?.toString() || ''); setError(''); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, [user.id]);
  useEffect(() => { void load(); }, [load]);
  const save = async () => { try { await request(`/admin/users/${user.id}/entitlements`, { method: 'PUT', body: JSON.stringify({ entitlements: overrides, max_accounts: maxAccounts === '' ? null : Number(maxAccounts), max_devices: maxDevices === '' ? null : Number(maxDevices) }) }); await load(); } catch (err) { setError(err instanceof Error ? err.message : '保存失败'); } };
  const setOverride = (key: string, value: string) => { setOverrides((current) => { const next = { ...current }; if (value === 'inherit') delete next[key]; else next[key] = value === 'allow'; return next; }); };
  const revoke = async (deviceId: string) => { await request(`/admin/users/${user.id}/devices/${deviceId}`, { method: 'DELETE' }); await load(); };
  return <div className="drawer-backdrop"><div className="access-panel"><div className="page-heading"><div><p className="eyebrow">用户权限</p><h2>{user.email}</h2></div><button className="secondary icon-only" onClick={onClose}><X size={18} /></button></div>{error && <p className="form-error">{error}</p>}{detail && <div className="effective-summary"><span>当前套餐：<strong>{detail.effective.plan_name || '无有效套餐'}</strong></span><span>账号上限：<strong>{detail.effective.max_accounts}</strong></span><span>设备上限：<strong>{detail.effective.max_devices}</strong></span></div>}<div className="limit-grid"><label>账号上限覆盖<input type="number" min="0" placeholder="跟随套餐" value={maxAccounts} onChange={(event) => setMaxAccounts(event.target.value)} /></label><label>设备上限覆盖<input type="number" min="1" placeholder="跟随套餐" value={maxDevices} onChange={(event) => setMaxDevices(event.target.value)} /></label></div><div className="feature-list">{features.map((feature) => <label key={feature.key}><span><small>{feature.category}</small>{feature.name}</span><select value={Object.prototype.hasOwnProperty.call(overrides, feature.key) ? (overrides[feature.key] ? 'allow' : 'deny') : 'inherit'} onChange={(event) => setOverride(feature.key, event.target.value)}><option value="inherit">跟随套餐</option><option value="allow">额外开放</option><option value="deny">强制关闭</option></select></label>)}</div><button className="primary compact" onClick={() => void save()}>保存用户权限</button><div className="panel-section"><h3><Monitor size={17} /> 已登记设备</h3>{devices.length === 0 ? <p className="muted">暂无设备</p> : devices.map((device) => <div className="device-row" key={device.id}><div><strong>{device.device_name || device.platform || '未知设备'}</strong><small>{device.device_id}<br />最后活跃：{new Date(device.last_seen_at).toLocaleString('zh-CN')}</small></div>{device.revoked_at ? <Badge tone="neutral">已解绑</Badge> : <button className="text-button" onClick={() => void revoke(device.id)}>解绑</button>}</div>)}</div></div></div>;
}

function PlansPage() {
  const [plans, setPlans] = useState<Plan[]>([]); const [features, setFeatures] = useState<Feature[]>([]); const [editing, setEditing] = useState<Plan | null>(null); const [error, setError] = useState('');
  const load = useCallback(async () => { try { const [planData, featureData] = await Promise.all([request('/admin/plans'), request('/admin/features')]); setPlans(planData); setFeatures(featureData); setError(''); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, []);
  useEffect(() => { void load(); }, [load]);
  const create = () => setEditing({ id: '', name: '新套餐', duration_days: 30, max_accounts: 1, max_devices: 1, price_cents: 0, is_active: true, entitlements: Object.fromEntries(features.map((feature) => [feature.key, false])) });
  const save = async () => { if (!editing) return; const method = editing.id ? 'PUT' : 'POST'; const path = editing.id ? `/admin/plans/${editing.id}` : '/admin/plans'; try { await request(path, { method, body: JSON.stringify(editing) }); setEditing(null); await load(); } catch (err) { setError(err instanceof Error ? err.message : '保存失败'); } };
  return <section><div className="page-heading"><div><p className="eyebrow">商品权限</p><h2>套餐与开放功能</h2></div><button className="primary compact" onClick={create}><Plus size={16} /> 新建套餐</button></div>{error && <p className="form-error">{error}</p>}<div className="plan-grid">{plans.map((plan) => <article className="plan-card" key={plan.id}><div><Badge tone={plan.is_active ? 'green' : 'neutral'}>{plan.is_active ? '销售中' : '已下架'}</Badge><h3>{plan.name}</h3><p>¥{(plan.price_cents / 100).toFixed(2)} / {plan.duration_days} 天</p></div><div className="plan-limits"><span>{plan.max_accounts} 个账号</span><span>{plan.max_devices} 台设备</span><span>{Object.values(plan.entitlements || {}).filter(Boolean).length} 项功能</span></div><button className="secondary" onClick={() => setEditing({ ...plan, entitlements: { ...plan.entitlements } })}>编辑套餐</button></article>)}</div>{editing && <div className="drawer-backdrop"><div className="access-panel"><div className="page-heading"><div><p className="eyebrow">套餐配置</p><h2>{editing.id ? '编辑套餐' : '新建套餐'}</h2></div><button className="secondary icon-only" onClick={() => setEditing(null)}><X size={18} /></button></div><div className="plan-form"><label>套餐名称<input value={editing.name} onChange={(event) => setEditing({ ...editing, name: event.target.value })} /></label><label>有效天数<input type="number" min="1" value={editing.duration_days} onChange={(event) => setEditing({ ...editing, duration_days: Number(event.target.value) })} /></label><label>价格（分）<input type="number" min="0" value={editing.price_cents} onChange={(event) => setEditing({ ...editing, price_cents: Number(event.target.value) })} /></label><label>账号上限<input type="number" min="0" value={editing.max_accounts} onChange={(event) => setEditing({ ...editing, max_accounts: Number(event.target.value) })} /></label><label>设备上限<input type="number" min="1" value={editing.max_devices} onChange={(event) => setEditing({ ...editing, max_devices: Number(event.target.value) })} /></label><label className="toggle-label"><input type="checkbox" checked={editing.is_active} onChange={(event) => setEditing({ ...editing, is_active: event.target.checked })} /> 对外销售</label></div><div className="feature-list checkbox-list">{features.map((feature) => <label key={feature.key}><span><small>{feature.category}</small>{feature.name}</span><input type="checkbox" checked={!!editing.entitlements?.[feature.key]} onChange={(event) => setEditing({ ...editing, entitlements: { ...editing.entitlements, [feature.key]: event.target.checked } })} /></label>)}</div><button className="primary compact" onClick={() => void save()}>保存套餐</button></div></div>}</section>;
}

function AccountsPage() {
  const [items, setItems] = useState<any[]>([]); const [error, setError] = useState(''); const [showForm, setShowForm] = useState(false); const [form, setForm] = useState({ platform: 'codex', email: '', access_token: '', refresh_token: '', token_expiry: '' });
  const load = useCallback(async () => { try { setItems(await request('/admin/accounts')); setError(''); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, []);
  useEffect(() => { void load(); }, [load]);
  const add = async (event: FormEvent) => { event.preventDefault(); await request('/admin/accounts', { method: 'POST', body: JSON.stringify({ ...form, token_expiry: Number(form.token_expiry || 0) }) }); setShowForm(false); setForm({ platform: 'codex', email: '', access_token: '', refresh_token: '', token_expiry: '' }); await load(); };
  return <section><div className="page-heading"><div><p className="eyebrow">账号池</p><h2>可分配账号</h2></div><div className="action-row"><button className="secondary" onClick={() => void load()}><RefreshCw size={16} /> 刷新</button><button className="primary compact" onClick={() => setShowForm(true)}><Plus size={16} /> 导入账号</button></div></div>{error && <p className="form-error">{error}</p>}{showForm && <form className="inline-form" onSubmit={add}><select value={form.platform} onChange={(event) => setForm({ ...form, platform: event.target.value })}><option value="codex">Codex</option><option value="cursor">Cursor</option><option value="claude">Claude</option></select><input placeholder="账号邮箱（选填）" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} /><textarea placeholder="Access Token（必填）" required value={form.access_token} onChange={(event) => setForm({ ...form, access_token: event.target.value })} /><textarea placeholder="Refresh Token（选填）" value={form.refresh_token} onChange={(event) => setForm({ ...form, refresh_token: event.target.value })} /><input type="number" placeholder="过期时间戳（秒）" value={form.token_expiry} onChange={(event) => setForm({ ...form, token_expiry: event.target.value })} /><div className="action-row"><button type="button" className="secondary" onClick={() => setShowForm(false)}>取消</button><button className="primary compact">保存入池</button></div></form>}<Table headers={['平台', '邮箱', '状态', '过期时间', '导入时间']}>{items.map((item) => <tr key={item.id}><td><Badge tone="violet">{item.platform}</Badge></td><td>{item.email || '—'}</td><td><Badge tone={item.status === 'ready' ? 'green' : 'neutral'}>{item.status}</Badge></td><td>{item.token_expiry ? new Date(item.token_expiry * 1000).toLocaleString('zh-CN') : '—'}</td><td>{item.created_at}</td></tr>)}</Table></section>;
}

function CardsPage() {
  const [plans, setPlans] = useState<any[]>([]); const [codes, setCodes] = useState<any[]>([]); const [planId, setPlanId] = useState(''); const [count, setCount] = useState(1); const [created, setCreated] = useState<string[]>([]); const [error, setError] = useState('');
  const load = useCallback(async () => { try { const [planData, codeData] = await Promise.all([request('/subscriptions/plans'), request('/admin/redemption-codes')]); setPlans(planData); setCodes(codeData); setPlanId((current) => current || planData[0]?.id || ''); } catch (err) { setError(err instanceof Error ? err.message : '加载失败'); } }, []);
  useEffect(() => { void load(); }, [load]);
  const create = async () => { try { const data = await request('/admin/redemption-codes', { method: 'POST', body: JSON.stringify({ plan_id: planId, count }) }); setCreated(data.codes || []); await load(); } catch (err) { setError(err instanceof Error ? err.message : '生成失败'); } };
  return <section><div className="page-heading"><div><p className="eyebrow">卡密中心</p><h2>套餐与兑换卡密</h2></div></div><div className="generator"><div><label>套餐<select value={planId} onChange={(event) => setPlanId(event.target.value)}>{plans.map((plan) => <option key={plan.id} value={plan.id}>{plan.name} · ¥{(plan.price_cents / 100).toFixed(2)}</option>)}</select></label><label>数量<input type="number" min="1" max="1000" value={count} onChange={(event) => setCount(Math.max(1, Number(event.target.value)))} /></label></div><button className="primary compact" disabled={!planId} onClick={() => void create()}><Ticket size={16} /> 生成卡密</button></div>{error && <p className="form-error">{error}</p>}{created.length > 0 && <div className="codes"><strong>请立即复制保存，卡密明文仅显示这一次：</strong><textarea readOnly value={created.join('\n')} /><button className="secondary" onClick={() => void navigator.clipboard.writeText(created.join('\n'))}>复制全部</button></div>}<Table headers={['套餐', '状态', '兑换用户', '兑换时间', '到期时间']}>{codes.map((code) => <tr key={code.id}><td>{code.plan_name}</td><td><Badge tone={code.status === 'active' ? 'green' : 'neutral'}>{code.status}</Badge></td><td>{code.redeemed_by || '—'}</td><td>{code.redeemed_at ? new Date(code.redeemed_at).toLocaleString('zh-CN') : '—'}</td><td>{code.expires_at ? new Date(code.expires_at).toLocaleString('zh-CN') : '永久有效'}</td></tr>)}</Table></section>;
}

function PaymentsPage() { const [items, setItems] = useState<any[]>([]); useEffect(() => { void request('/admin/payments').then(setItems); }, []); return <section><div className="page-heading"><div><p className="eyebrow">支付记录</p><h2>订阅订单</h2></div></div><Table headers={['用户', '金额', '状态', '支付时间', '创建时间']}>{items.map((item) => <tr key={item.id}><td>{item.user_email}</td><td>¥{(item.amount_cents / 100).toFixed(2)}</td><td><Badge tone={item.status === 'completed' ? 'green' : 'neutral'}>{item.status}</Badge></td><td>{item.paid_at ? new Date(item.paid_at).toLocaleString('zh-CN') : '—'}</td><td>{item.created_at}</td></tr>)}</Table></section>; }
function Badge({ children, tone }: { children: React.ReactNode; tone: string }) { return <span className={`badge ${tone}`}>{children}</span>; }
function Table({ headers, children }: { headers: string[]; children: React.ReactNode }) { return <div className="table-wrap"><table><thead><tr>{headers.map((header) => <th key={header}>{header}</th>)}</tr></thead><tbody>{children}</tbody></table></div>; }
function Dashboard({ user, onLogout }: { user: User; onLogout: () => void }) { const [tab, setTab] = useState<Tab>('overview'); const tabs: [Tab, string, typeof BarChart3][] = [['overview', '概览', BarChart3], ['users', '用户', Users], ['plans', '套餐权限', Package], ['accounts', '账号池', Database], ['cards', '卡密', Ticket], ['payments', '支付', CreditCard]]; return <div className="app-shell"><aside><div className="side-brand"><div className="brand-mark">↯</div><span>AISwitch</span></div><nav>{tabs.map(([id, label, Icon]) => <button className={tab === id ? 'active' : ''} key={id} onClick={() => setTab(id)}><Icon size={18} />{label}</button>)}</nav><div className="side-user"><span>{user.email}</span><button onClick={onLogout}><LogOut size={15} /> 退出</button></div></aside><main className="dashboard-main">{tab === 'overview' && <Overview />}{tab === 'users' && <UsersPage />}{tab === 'plans' && <PlansPage />}{tab === 'accounts' && <AccountsPage />}{tab === 'cards' && <CardsPage />}{tab === 'payments' && <PaymentsPage />}</main></div>; }
function App() { const [user, setUser] = useState<User | null>(storedUser); if (!user) return <Login onAuthenticated={setUser} />; return <Dashboard user={user} onLogout={() => { localStorage.removeItem(TOKEN_KEY); localStorage.removeItem(REFRESH_KEY); localStorage.removeItem(USER_KEY); setUser(null); }} />; }
createRoot(document.getElementById('root')!).render(<App />);

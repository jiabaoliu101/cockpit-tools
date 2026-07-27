import { useState, useEffect } from 'react';
import { useAuthStore } from '../stores/useAuthStore';
import { RefreshCw, Users, Database, CreditCard, BarChart3, Plus, Trash2, Ticket } from 'lucide-react';
import { logout } from '../services/authService';

type Tab = 'dashboard' | 'users' | 'accounts' | 'payments' | 'cards';

const API_BASE = (typeof window !== 'undefined' && (window as any).__AISWITCH_API_BASE__)
  || import.meta.env.VITE_AISWITCH_API_BASE
  || 'http://localhost:8080/api/v1';

function getToken() {
  return useAuthStore.getState().accessToken || '';
}

async function apiGet(path: string) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function apiPost(path: string, body: any) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${getToken()}` },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function apiPut(path: string, body: any) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${getToken()}` },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

async function apiDelete(path: string) {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'DELETE',
    headers: { Authorization: `Bearer ${getToken()}` },
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

// ── Dashboard Tab ──

function DashboardTab() {
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try { setStats(await apiGet('/admin/dashboard')); } catch {}
    setLoading(false);
  };
  useEffect(() => { load(); }, []);

  if (loading) return <div className="flex items-center justify-center p-12"><span className="loading loading-spinner loading-lg" /></div>;

  return (
    <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
      <StatCard title="总用户" value={stats?.total_users ?? 0} color="text-primary" />
      <StatCard title="活跃订阅" value={stats?.active_subscriptions ?? 0} color="text-success" />
      <StatCard title="可用账号" value={stats?.available_accounts ?? 0} color="text-info" />
      <StatCard title="已分配" value={stats?.assigned_accounts ?? 0} color="text-warning" />
      <StatCard title="月收入" value={`¥${((stats?.monthly_revenue_cents ?? 0) / 100).toFixed(2)}`} color="text-accent" />
    </div>
  );
}

function StatCard({ title, value, color }: { title: string; value: number | string; color: string }) {
  return (
    <div className="card bg-base-100 shadow">
      <div className="card-body p-4 text-center">
        <div className={`text-3xl font-bold ${color}`}>{value}</div>
        <div className="text-xs text-base-content/60">{title}</div>
      </div>
    </div>
  );
}

// ── Users Tab ──

function UsersTab() {
  const [users, setUsers] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try { setUsers(await apiGet('/admin/users')); } catch {}
    setLoading(false);
  };
  useEffect(() => { load(); }, []);

  const toggleActive = async (id: string, isActive: boolean) => {
    try { await apiPut(`/admin/users/${id}`, { is_active: !isActive }); load(); } catch {}
  };

  if (loading) return <div className="flex items-center justify-center p-12"><span className="loading loading-spinner loading-lg" /></div>;

  return (
    <div className="overflow-x-auto">
      <table className="table table-zebra">
        <thead>
          <tr>
            <th>邮箱</th>
            <th>昵称</th>
            <th>管理员</th>
            <th>状态</th>
            <th>注册时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u: any) => (
            <tr key={u.id}>
              <td>{u.email}</td>
              <td>{u.nickname || '-'}</td>
              <td>{u.is_admin ? <span className="badge badge-warning badge-xs">管理员</span> : <span className="badge badge-ghost badge-xs">用户</span>}</td>
              <td>{u.is_active ? <span className="badge badge-success badge-xs">正常</span> : <span className="badge badge-error badge-xs">已禁用</span>}</td>
              <td className="text-xs text-base-content/50">{new Date(u.created_at).toLocaleDateString('zh-CN')}</td>
              <td>
                <button className="btn btn-xs btn-ghost" onClick={() => toggleActive(u.id, u.is_active)}>
                  {u.is_active ? '禁用' : '启用'}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ── Accounts Tab (Pool Management) ──

function AccountsTab() {
  const [accounts, setAccounts] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [showAdd, setShowAdd] = useState(false);
  const [showBulk, setShowBulk] = useState(false);

  const load = async () => {
    setLoading(true);
    try { setAccounts(await apiGet('/admin/accounts')); } catch {}
    setLoading(false);
  };
  useEffect(() => { load(); }, []);

  const handleDelete = async (id: string) => {
    if (!confirm('确认删除这个账号？')) return;
    try { await apiDelete(`/admin/accounts/${id}`); load(); } catch {}
  };

  const markReady = async (id: string) => {
    try {
      await apiPut(`/admin/accounts/${id}`, { status: 'ready' });
      load();
    } catch (e: any) {
      alert('更新账号状态失败: ' + e.message);
    }
  };

  if (loading) return <div className="flex items-center justify-center p-12"><span className="loading loading-spinner loading-lg" /></div>;

  return (
    <div>
      <div className="flex gap-2 mb-4">
        <button className="btn btn-primary btn-sm" onClick={() => setShowAdd(true)}>
          <Plus className="w-4 h-4" /> 添加账号
        </button>
        <button className="btn btn-outline btn-sm" onClick={() => setShowBulk(true)}>
          <Database className="w-4 h-4" /> 批量导入
        </button>
        <button className="btn btn-ghost btn-sm" onClick={load}>
          <RefreshCw className="w-4 h-4" /> 刷新
        </button>
      </div>

      <div className="overflow-x-auto">
        <table className="table table-zebra">
          <thead>
            <tr>
              <th>平台</th>
              <th>邮箱</th>
              <th>状态</th>
              <th>有效期</th>
              <th>导入时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {accounts.map((a: any) => (
              <tr key={a.id}>
                <td><span className="badge">{a.platform}</span></td>
                <td className="text-sm">{a.email || '-'}</td>
                <td>
                  {a.status === 'ready' ? <span className="badge badge-success badge-xs">待分配</span>
                   : a.status === 'assigned' ? <span className="badge badge-info badge-xs">已分配</span>
                   : a.status === 'staging' ? <span className="badge badge-warning badge-xs">待校验</span>
                   : a.status === 'expiring' ? <span className="badge badge-warning badge-xs">到期处理</span>
                   : a.status === 'revoking' ? <span className="badge badge-warning badge-xs">撤销中</span>
                   : a.status === 'quarantined' ? <span className="badge badge-error badge-xs">已隔离</span>
                   : <span className="badge badge-ghost badge-xs">{a.status}</span>}
                </td>
                <td className="text-xs">{a.token_expiry ? new Date(a.token_expiry * 1000).toLocaleDateString('zh-CN') : '-'}</td>
                <td className="text-xs text-base-content/50">{a.created_at}</td>
                <td>
                  {(a.status === 'staging' || a.status === 'verified_revoked') && (
                    <button
                      className="btn btn-xs btn-ghost text-success mr-1"
                      title="确认凭据有效且可安全分配后再入池"
                      onClick={() => markReady(a.id)}
                    >
                      入池
                    </button>
                  )}
                  <button className="btn btn-xs btn-ghost text-error" onClick={() => handleDelete(a.id)}>
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                </td>
              </tr>
            ))}
            {accounts.length === 0 && (
              <tr><td colSpan={6} className="text-center text-base-content/40 py-8">暂无账号，请先导入</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {/* Add Single Account Modal */}
      {showAdd && <AddAccountModal onClose={() => { setShowAdd(false); load(); }} />}
      {/* Bulk Import Modal */}
      {showBulk && <BulkImportModal onClose={() => { setShowBulk(false); load(); }} />}
    </div>
  );
}

function AddAccountModal({ onClose }: { onClose: () => void }) {
  const [form, setForm] = useState({ platform: 'codex', email: '', access_token: '', refresh_token: '', token_expiry: 0 });
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await apiPost('/admin/accounts', form);
      onClose();
    } catch (e: any) {
      alert('添加失败: ' + e.message);
    }
    setSaving(false);
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box">
        <h3 className="text-lg font-bold mb-3">添加账号到池</h3>
        <form onSubmit={handleSubmit} className="space-y-3">
          <label className="form-control">
            <span className="label-text text-xs font-medium mb-1">平台</span>
            <select className="select select-bordered select-sm" value={form.platform} onChange={e => setForm({...form, platform: e.target.value})}>
              <option value="codex">Codex</option>
              <option value="cursor">Cursor</option>
              <option value="claude">Claude</option>
            </select>
          </label>
          <label className="form-control">
            <span className="label-text text-xs font-medium mb-1">邮箱（选填）</span>
            <input className="input input-bordered input-sm" placeholder="account@example.com" value={form.email} onChange={e => setForm({...form, email: e.target.value})} />
          </label>
          <label className="form-control">
            <span className="label-text text-xs font-medium mb-1">Access Token *</span>
            <textarea className="textarea textarea-bordered textarea-sm" rows={2} required placeholder="oauth access token" value={form.access_token} onChange={e => setForm({...form, access_token: e.target.value})} />
          </label>
          <label className="form-control">
            <span className="label-text text-xs font-medium mb-1">Refresh Token</span>
            <textarea className="textarea textarea-bordered textarea-sm" rows={2} placeholder="refresh token" value={form.refresh_token} onChange={e => setForm({...form, refresh_token: e.target.value})} />
          </label>
          <label className="form-control">
            <span className="label-text text-xs font-medium mb-1">Token 过期时间戳（秒）</span>
            <input className="input input-bordered input-sm" type="number" value={form.token_expiry} onChange={e => setForm({...form, token_expiry: Number(e.target.value)})} />
          </label>
          <div className="modal-action">
            <button type="button" className="btn btn-ghost btn-sm" onClick={onClose}>取消</button>
            <button type="submit" className="btn btn-primary btn-sm" disabled={saving}>
              {saving ? <span className="loading loading-spinner loading-xs" /> : null}
              添加
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function BulkImportModal({ onClose }: { onClose: () => void }) {
  const [text, setText] = useState('');
  const [importing, setImporting] = useState(false);

  const handleImport = async () => {
    setImporting(true);
    try {
      const lines = text.trim().split('\n').filter(l => l.trim());
      const accounts = lines.map(line => {
        const parts = line.split('|');
        return {
          platform: parts[0]?.trim() || 'codex',
          email: parts[1]?.trim() || '',
          access_token: parts[2]?.trim() || '',
          refresh_token: parts[3]?.trim() || '',
          token_expiry: parseInt(parts[4] || '0'),
        };
      }).filter(a => a.access_token);

      if (accounts.length === 0) {
        alert('未解析到有效账号。格式: platform|email|access_token|refresh_token|expiry');
        setImporting(false);
        return;
      }

      const result = await apiPost('/admin/accounts/bulk', accounts);
      alert(`导入完成: ${result.added}/${result.total} 个账号`);
      onClose();
    } catch (e: any) {
      alert('导入失败: ' + e.message);
    }
    setImporting(false);
  };

  return (
    <div className="modal modal-open">
      <div className="modal-box">
        <h3 className="text-lg font-bold mb-3">批量导入账号</h3>
        <p className="text-xs text-base-content/60 mb-2">
          每行一个账号，格式: <code>平台|邮箱|access_token|refresh_token|过期时间戳</code><br />
          示例: <code>codex|user@mail.com|sk-xxx|rt-xxx|1735689600</code>
        </p>
        <textarea
          className="textarea textarea-bordered w-full"
          rows={10}
          placeholder="codex|user@mail.com|sk-xxx|rt-xxx|1735689600"
          value={text}
          onChange={e => setText(e.target.value)}
        />
        <div className="modal-action">
          <button className="btn btn-ghost btn-sm" onClick={onClose}>取消</button>
          <button className="btn btn-primary btn-sm" onClick={handleImport} disabled={importing || !text.trim()}>
            {importing ? <span className="loading loading-spinner loading-xs" /> : null}
            批量导入
          </button>
        </div>
      </div>
    </div>
  );
}

// ── Payments Tab ──

function PaymentsTab() {
  const [payments, setPayments] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try { setPayments(await apiGet('/admin/payments')); } catch {}
    setLoading(false);
  };
  useEffect(() => { load(); }, []);

  if (loading) return <div className="flex items-center justify-center p-12"><span className="loading loading-spinner loading-lg" /></div>;

  return (
    <div className="overflow-x-auto">
      <table className="table table-zebra">
        <thead>
          <tr>
            <th>用户</th>
            <th>金额</th>
            <th>状态</th>
            <th>支付时间</th>
            <th>创建时间</th>
          </tr>
        </thead>
        <tbody>
          {payments.map((p: any) => (
            <tr key={p.id}>
              <td className="text-sm">{p.user_email}</td>
              <td>¥{(p.amount_cents / 100).toFixed(2)}</td>
              <td>
                {p.status === 'completed' ? <span className="badge badge-success badge-xs">已完成</span>
                 : p.status === 'pending' ? <span className="badge badge-warning badge-xs">待支付</span>
                 : p.status === 'refunded' ? <span className="badge badge-ghost badge-xs">已退款</span>
                 : <span className="badge badge-error badge-xs">{p.status}</span>}
              </td>
              <td className="text-xs">{p.paid_at ? new Date(p.paid_at).toLocaleString('zh-CN') : '-'}</td>
              <td className="text-xs text-base-content/50">{p.created_at}</td>
            </tr>
          ))}
          {payments.length === 0 && (
            <tr><td colSpan={5} className="text-center text-base-content/40 py-8">暂无支付记录</td></tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function CardCodesTab() {
  const [plans, setPlans] = useState<any[]>([]);
  const [codes, setCodes] = useState<any[]>([]);
  const [planId, setPlanId] = useState('');
  const [count, setCount] = useState(1);
  const [expiresAt, setExpiresAt] = useState('');
  const [batchName, setBatchName] = useState('');
  const [note, setNote] = useState('');
  const [creating, setCreating] = useState(false);
  const [newCodes, setNewCodes] = useState<string[]>([]);

  const load = async () => {
    try {
      const [planData, codeData] = await Promise.all([
        apiGet('/subscriptions/plans'),
        apiGet('/admin/redemption-codes'),
      ]);
      setPlans(planData);
      setCodes(codeData);
      if (!planId && planData[0]?.id) setPlanId(planData[0].id);
    } catch {}
  };
  useEffect(() => { void load(); }, []);

  const generate = async () => {
    if (!planId) return;
    setCreating(true);
    try {
      const result = await apiPost('/admin/redemption-codes', {
        plan_id: planId,
        count,
        batch_name: batchName,
        note,
        expires_at: expiresAt ? new Date(expiresAt).toISOString() : null,
      });
      setNewCodes(result.codes || []);
      await load();
    } catch (e: any) {
      alert('生成卡密失败: ' + e.message);
    }
    setCreating(false);
  };

  const downloadCodes = () => {
    const csv = `card_code\n${newCodes.map((code) => `"${code}"`).join('\n')}`;
    const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `aiswitch-cards-${new Date().toISOString().slice(0, 10)}.csv`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  const disable = async (id: string) => {
    if (!confirm('确认作废此未使用卡密？')) return;
    try { await apiDelete(`/admin/redemption-codes/${id}`); await load(); } catch (e: any) { alert('作废失败: ' + e.message); }
  };

  return (
    <div>
      <div className="card bg-base-100 shadow mb-5">
        <div className="card-body">
          <h2 className="card-title text-lg">生成套餐卡密</h2>
          <div className="flex flex-col md:flex-row gap-3 items-end">
            <label className="form-control flex-1">
              <span className="label-text text-xs mb-1">套餐</span>
              <select className="select select-bordered select-sm" value={planId} onChange={(e) => setPlanId(e.target.value)}>
                {plans.map((plan) => <option key={plan.id} value={plan.id}>{plan.name}（¥{(plan.price_cents / 100).toFixed(2)}）</option>)}
              </select>
            </label>
            <label className="form-control w-full md:w-32">
              <span className="label-text text-xs mb-1">数量</span>
              <input className="input input-bordered input-sm" type="number" min={1} max={1000} value={count} onChange={(e) => setCount(Math.max(1, Math.min(1000, Number(e.target.value))))} />
            </label>
            <label className="form-control w-full md:w-56">
              <span className="label-text text-xs mb-1">有效期（可选）</span>
              <input className="input input-bordered input-sm" type="datetime-local" value={expiresAt} onChange={(e) => setExpiresAt(e.target.value)} />
            </label>
            <button className="btn btn-primary btn-sm" disabled={creating || !planId} onClick={() => void generate()}>
              {creating ? <span className="loading loading-spinner loading-xs" /> : <Ticket className="w-4 h-4" />} 生成卡密
            </button>
          </div>
          <div className="grid md:grid-cols-2 gap-3 mt-3">
            <input className="input input-bordered input-sm" placeholder="批次名称（例如：7 月首批）" maxLength={100} value={batchName} onChange={(e) => setBatchName(e.target.value)} />
            <input className="input input-bordered input-sm" placeholder="内部备注" maxLength={500} value={note} onChange={(e) => setNote(e.target.value)} />
          </div>
        </div>
      </div>

      {newCodes.length > 0 && (
        <div className="alert alert-warning mb-5 block">
          <div className="font-semibold mb-2">请立即复制并保存：卡密明文仅显示这一次。</div>
          <textarea className="textarea textarea-bordered w-full font-mono text-sm" rows={Math.min(12, newCodes.length + 1)} readOnly value={newCodes.join('\n')} />
          <div className="flex gap-2 mt-2">
            <button className="btn btn-sm" onClick={() => void navigator.clipboard.writeText(newCodes.join('\n'))}>复制全部</button>
            <button className="btn btn-sm btn-outline" onClick={downloadCodes}>导出 CSV</button>
          </div>
        </div>
      )}

      <div className="overflow-x-auto">
        <table className="table table-zebra">
          <thead><tr><th>套餐 / 批次</th><th>状态</th><th>兑换用户</th><th>兑换时间</th><th>到期</th><th>操作</th></tr></thead>
          <tbody>{codes.map((code) => (
            <tr key={code.id}>
              <td><div>{code.plan_name}</div><div className="text-xs text-base-content/50">{code.batch_name || code.note || '-'}</div></td>
              <td><span className={`badge badge-xs ${code.status === 'active' ? 'badge-success' : code.status === 'redeemed' ? 'badge-info' : 'badge-ghost'}`}>{code.status === 'active' ? '未使用' : code.status === 'redeemed' ? '已兑换' : '已作废'}</span></td>
              <td>{code.redeemed_by || '-'}</td>
              <td className="text-xs">{code.redeemed_at ? new Date(code.redeemed_at).toLocaleString('zh-CN') : '-'}</td>
              <td className="text-xs">{code.expires_at ? new Date(code.expires_at).toLocaleString('zh-CN') : '永久有效'}</td>
              <td>{code.status === 'active' && <button className="btn btn-xs btn-ghost text-error" onClick={() => void disable(code.id)}>作废</button>}</td>
            </tr>
          ))}</tbody>
        </table>
      </div>
    </div>
  );
}

// ── Main Admin Page ──

export function AdminPage() {
  const user = useAuthStore((s) => s.user);
  const [tab, setTab] = useState<Tab>('dashboard');

  const tabs: { id: Tab; label: string; icon: React.ReactNode }[] = [
    { id: 'dashboard', label: '概览', icon: <BarChart3 className="w-4 h-4" /> },
    { id: 'users', label: '用户', icon: <Users className="w-4 h-4" /> },
    { id: 'accounts', label: '账号池', icon: <Database className="w-4 h-4" /> },
    { id: 'payments', label: '支付', icon: <CreditCard className="w-4 h-4" /> },
    { id: 'cards', label: '卡密', icon: <Ticket className="w-4 h-4" /> },
  ];

  return (
    <div className="min-h-screen bg-base-200">
      {/* Header */}
      <div className="bg-base-100 border-b border-base-300 px-6 py-3 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-lg font-bold text-primary">AISwitch 管理后台</h1>
          <span className="text-xs text-base-content/40">{user?.email}</span>
        </div>
        <button className="btn btn-ghost btn-sm" onClick={() => { logout(); }}>
          退出登录
        </button>
      </div>

      {/* Tabs */}
      <div className="tabs tabs-bordered px-6 pt-3 bg-base-100">
        {tabs.map(t => (
          <button
            key={t.id}
            className={`tab ${tab === t.id ? 'tab-active' : ''}`}
            onClick={() => setTab(t.id)}
          >
            {t.icon}
            <span className="ml-1.5">{t.label}</span>
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="p-6">
        {tab === 'dashboard' && <DashboardTab />}
        {tab === 'users' && <UsersTab />}
        {tab === 'accounts' && <AccountsTab />}
        {tab === 'payments' && <PaymentsTab />}
        {tab === 'cards' && <CardCodesTab />}
      </div>
    </div>
  );
}

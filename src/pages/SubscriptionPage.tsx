import { useState, useEffect } from 'react';
import { useAuthStore } from '../stores/useAuthStore';
import { fetchCurrentSubscription, fetchSubscriptionPlans, redeemCardCode } from '../services/authService';
import { Crown, Clock, Users, Loader2 } from 'lucide-react';

interface Props {
  onBack: () => void;
}

export function SubscriptionPage({ onBack }: Props) {
  const { subscription: cachedSub } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [plans, setPlans] = useState<any[]>([]);
  const [cardCode, setCardCode] = useState('');
  const [redeeming, setRedeeming] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const plansData = await fetchSubscriptionPlans();
      setPlans(plansData);
      await fetchCurrentSubscription();
    } catch {}
    setLoading(false);
  };

  const handleRedeem = async () => {
    if (!cardCode.trim()) return;
    setRedeeming(true);
    try {
      await redeemCardCode(cardCode);
      setCardCode('');
      await loadData();
      alert('兑换成功，订阅已生效');
    } catch (e: any) {
      alert('兑换失败: ' + (e.message || '请检查卡密后重试'));
    }
    setRedeeming(false);
  };

  const sub = cachedSub?.subscription;

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-base-200">
        <span className="loading loading-spinner loading-lg" />
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-base-200 p-4 md:p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-3 mb-6">
          <button className="btn btn-ghost btn-sm" onClick={onBack}>
            ← 返回
          </button>
          <h1 className="text-2xl font-bold">订阅管理</h1>
        </div>

        {/* Current subscription card */}
        {sub && (
          <div className="card bg-base-100 shadow mb-6">
            <div className="card-body">
              <div className="flex items-center gap-2 text-primary">
                <Crown className="w-5 h-5" />
                <span className="text-lg font-semibold">当前套餐: {sub.plan_name}</span>
                <span className={`badge ${sub.status === 'active' ? 'badge-success' : 'badge-error'}`}>
                  {sub.status === 'active' ? '生效中' : sub.status}
                </span>
              </div>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mt-3">
                <div className="flex items-center gap-2">
                  <Clock className="w-4 h-4 text-base-content/50" />
                  <span className="text-sm">到期: {new Date(sub.expires_at).toLocaleDateString('zh-CN')}</span>
                </div>
                <div className="flex items-center gap-2">
                  <Users className="w-4 h-4 text-base-content/50" />
                  <span className="text-sm">账号: {sub.assigned_count}/{sub.max_accounts}</span>
                </div>
              </div>
            </div>
          </div>
        )}

        {!sub && (
          <div className="alert alert-warning mb-6">
            <span>您当前没有有效的订阅，请选择一个套餐开始使用</span>
          </div>
        )}

        <div className="card bg-base-100 shadow mb-6">
          <div className="card-body">
            <h2 className="card-title text-lg">兑换卡密</h2>
            <p className="text-sm text-base-content/60">从管理员购买套餐卡密后，在这里兑换。卡密只能使用一次。</p>
            <div className="flex flex-col sm:flex-row gap-2 mt-2">
              <input
                className="input input-bordered flex-1 uppercase"
                placeholder="例如 AS-ABCDE-FGHIJ-KLMNP-QRSTU"
                value={cardCode}
                onChange={(event) => setCardCode(event.target.value)}
                onKeyDown={(event) => { if (event.key === 'Enter') void handleRedeem(); }}
              />
              <button className="btn btn-primary" disabled={redeeming || !cardCode.trim()} onClick={() => void handleRedeem()}>
                {redeeming && <Loader2 className="w-4 h-4 animate-spin" />}
                兑换并开通
              </button>
            </div>
          </div>
        </div>

        {/* Plan selection */}
        <h2 className="text-xl font-semibold mb-4">
          {sub ? '升级套餐' : '选择套餐'}
        </h2>
        <div className="grid md:grid-cols-3 gap-4">
          {plans.map((plan: any) => (
            <div key={plan.id} className={`card bg-base-100 shadow ${sub?.plan_name === plan.name ? 'ring-2 ring-primary' : ''}`}>
              <div className="card-body">
                <h3 className="card-title">{plan.name}</h3>
                <div className="text-3xl font-bold text-primary">
                  ¥{(plan.price_cents / 100).toFixed(2)}
                </div>
                <div className="text-sm text-base-content/60">
                  {plan.duration_days >= 365
                    ? `${plan.duration_days / 365} 年`
                    : `${plan.duration_days} 天`}
                  · 最多 {plan.max_accounts} 个账号
                </div>
                <p className="text-sm mt-3 text-base-content/60">请联系管理员购买此套餐的卡密。</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

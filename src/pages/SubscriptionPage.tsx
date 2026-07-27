import { useState, useEffect } from 'react';
import { useAuthStore } from '../stores/useAuthStore';
import { fetchCurrentSubscription, fetchSubscriptionPlans, createOrder, checkOrderStatus } from '../services/authService';
import { Crown, Clock, Users, RefreshCw, Loader2 } from 'lucide-react';

interface Props {
  onBack: () => void;
}

export function SubscriptionPage({ onBack }: Props) {
  const { subscription: cachedSub } = useAuthStore();
  const [loading, setLoading] = useState(true);
  const [plans, setPlans] = useState<any[]>([]);
  const [ordering, setOrdering] = useState<string | null>(null);
  const [orderResult, setOrderResult] = useState<any>(null);
  const [pollingOrderId, setPollingOrderId] = useState<string | null>(null);

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

  const handleOrder = async (planId: string) => {
    setOrdering(planId);
    try {
      const result = await createOrder(planId);
      setOrderResult(result);
      setPollingOrderId(result.order_id);
    } catch (e: any) {
      alert('创建订单失败: ' + (e.message || '未知错误'));
    }
    setOrdering(null);
  };

  // Poll payment status every 3 seconds when waiting for payment
  useEffect(() => {
    if (!pollingOrderId) return;
    const timer = setInterval(async () => {
      try {
        const status = await checkOrderStatus(pollingOrderId);
        if (status.status === 'completed') {
          setPollingOrderId(null);
          setOrderResult(null);
          await loadData(); // Refresh subscription
          alert('支付成功！您的订阅已激活');
        } else if (status.status === 'failed') {
          setPollingOrderId(null);
          alert('支付失败，请重试');
        }
      } catch {}
    }, 3000);
    return () => clearInterval(timer);
  }, [pollingOrderId]);

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
                <button
                  className="btn btn-primary mt-3"
                  disabled={ordering === plan.id || !!pollingOrderId}
                  onClick={() => handleOrder(plan.id)}
                >
                  {ordering === plan.id && <Loader2 className="w-4 h-4 animate-spin" />}
                  {sub?.plan_name === plan.name ? '续费' : '购买'}
                </button>
              </div>
            </div>
          ))}
        </div>

        {/* Payment QR Code Modal */}
        {orderResult && (
          <div className="modal modal-open">
            <div className="modal-box text-center">
              <h3 className="text-lg font-bold mb-3">扫码支付</h3>
              <p className="text-sm mb-2">金额: ¥{(orderResult.amount / 100).toFixed(2)}</p>
              <div className="bg-base-200 p-4 rounded-lg inline-block mb-3">
                <div className="w-48 h-48 bg-base-300 flex items-center justify-center mx-auto">
                  <span className="text-base-content/40 text-sm">支付二维码</span>
                </div>
                <p className="text-xs text-base-content/50 mt-1">
                  集成 PayJS 后将显示真实二维码
                </p>
              </div>
              <p className="text-xs text-base-content/50 mb-3">
                订单号: {orderResult.order_id}
              </p>
              {pollingOrderId && (
                <div className="flex items-center justify-center gap-2 text-sm text-primary">
                  <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  等待支付...
                </div>
              )}
              <div className="modal-action">
                <button
                  className="btn btn-ghost btn-sm"
                  onClick={() => {
                    setOrderResult(null);
                    setPollingOrderId(null);
                  }}
                >
                  取消
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

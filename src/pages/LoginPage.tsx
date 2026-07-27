import { useState, type FormEvent } from 'react';
import { useAuthStore } from '../stores/useAuthStore';
import { login, register } from '../services/authService';
import { KeyRound, Mail, User, Loader2, Eye, EyeOff } from 'lucide-react';

interface Props {
  onAuthenticated: () => void;
}

export function LoginPage({ onAuthenticated }: Props) {
  const [isRegister, setIsRegister] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [nickname, setNickname] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const setAuthLoading = useAuthStore((s) => s.setAuthLoading);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      if (isRegister) {
        await register(email, password, nickname || email.split('@')[0]);
      } else {
        await login(email, password);
      }
      onAuthenticated();
    } catch (err: any) {
      setError(err.message || '操作失败，请重试');
    } finally {
      setLoading(false);
      setAuthLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-base-200">
      <div className="card w-full max-w-md bg-base-100 shadow-xl">
        <div className="card-body">
          {/* Logo / Title */}
          <div className="text-center mb-4">
            <div className="text-4xl font-bold text-primary mb-1">AISwitch</div>
            <p className="text-sm text-base-content/60">
              {isRegister ? '创建您的账号，开始使用换号服务' : '登录以管理您的 IDE 账号'}
            </p>
          </div>

          {/* Error */}
          {error && (
            <div className="alert alert-error text-sm">
              <span>{error}</span>
            </div>
          )}

          {/* Form */}
          <form onSubmit={handleSubmit} className="space-y-3">
            {isRegister && (
              <label className="input input-bordered flex items-center gap-2">
                <User className="w-4 h-4 text-base-content/50" />
                <input
                  type="text"
                  className="grow"
                  placeholder="昵称（选填）"
                  value={nickname}
                  onChange={(e) => setNickname(e.target.value)}
                />
              </label>
            )}

            <label className="input input-bordered flex items-center gap-2">
              <Mail className="w-4 h-4 text-base-content/50" />
              <input
                type="email"
                className="grow"
                placeholder="邮箱"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </label>

            <label className="input input-bordered flex items-center gap-2">
              <KeyRound className="w-4 h-4 text-base-content/50" />
              <input
                type={showPassword ? 'text' : 'password'}
                className="grow"
                placeholder="密码（至少 6 位）"
                required
                minLength={6}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
              <button
                type="button"
                className="btn btn-ghost btn-xs btn-circle"
                onClick={() => setShowPassword(!showPassword)}
              >
                {showPassword ? <EyeOff className="w-3.5 h-3.5" /> : <Eye className="w-3.5 h-3.5" />}
              </button>
            </label>

            <button
              type="submit"
              className="btn btn-primary w-full"
              disabled={loading}
            >
              {loading && <Loader2 className="w-4 h-4 animate-spin" />}
              {isRegister ? '注册' : '登录'}
            </button>
          </form>

          {/* Toggle */}
          <div className="text-center text-sm mt-2">
            {isRegister ? (
              <span>
                已有账号？{' '}
                <button
                  type="button"
                  className="link link-primary"
                  onClick={() => { setIsRegister(false); setError(''); }}
                >
                  去登录
                </button>
              </span>
            ) : (
              <span>
                没有账号？{' '}
                <button
                  type="button"
                  className="link link-primary"
                  onClick={() => { setIsRegister(true); setError(''); }}
                >
                  注册新账号
                </button>
              </span>
            )}
          </div>

          <div className="text-center text-xs text-base-content/40 mt-2">
            注册即表示您同意我们的服务条款和隐私政策
          </div>
        </div>
      </div>
    </div>
  );
}

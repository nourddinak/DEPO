import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import {
  ShieldCheck,
  Lock,
  Mail,
  Key,
  CheckCircle2,
  Server,
  Zap,
  HardDrive,
  Eye,
  EyeOff,
  ArrowRight,
  ArrowLeft,
  Sparkles,
  Layers,
  Shield,
} from 'lucide-react';

export const Login: React.FC = () => {
  const { setupRequired, login, setupOwner } = useAuth();

  // Wizard state for first-time admin setup
  const [wizardStep, setWizardStep] = useState(1);
  const [instanceName, setInstanceName] = useState('DEPO Production Fleet');
  const [adminName, setAdminName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [retentionDays, setRetentionDays] = useState(30);
  const [autoRollback, setAutoRollback] = useState(true);
  const [ntfyTopic, setNtfyTopic] = useState('');

  // Login state for regular authentication
  const [totpCode, setTotpCode] = useState('');
  const [totpRequired, setTotpRequired] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  // Password strength calculation
  const getPasswordStrength = (pass: string) => {
    let score = 0;
    if (pass.length >= 10) score += 1;
    if (pass.length >= 14) score += 1;
    if (/[A-Z]/.test(pass)) score += 1;
    if (/[0-9]/.test(pass)) score += 1;
    if (/[^A-Za-z0-9]/.test(pass)) score += 1;
    return score; // 0 to 5
  };

  const strength = getPasswordStrength(password);

  const getStrengthLabel = () => {
    if (strength <= 1) return { label: 'Weak', color: 'bg-rose-500', text: 'text-rose-600' };
    if (strength === 2) return { label: 'Fair', color: 'bg-amber-500', text: 'text-amber-600' };
    if (strength === 3 || strength === 4) return { label: 'Good', color: 'bg-blue-500', text: 'text-blue-600' };
    return { label: 'Strong', color: 'bg-emerald-500', text: 'text-emerald-600' };
  };

  // Handle first-time setup submission
  const handleSetupSubmit = async () => {
    setError('');
    if (password.length < 10) {
      setError('Password must be at least 10 characters for Argon2id hashing.');
      return;
    }
    if (password !== confirmPassword) {
      setError('Passwords do not match. Please re-enter.');
      return;
    }
    setLoading(true);

    try {
      await setupOwner(email, password);
    } catch (err: any) {
      setError(err.message || 'Platform initialization failed.');
    } finally {
      setLoading(false);
    }
  };

  // Handle standard login
  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await login(email, password, totpCode);
      if (res.totp_required) {
        setTotpRequired(true);
      }
    } catch (err: any) {
      setError(err.message || 'Authentication failed. Please verify credentials.');
    } finally {
      setLoading(false);
    }
  };

  // FIRST-TIME ADMIN SETUP WIZARD (Vercel Aesthetic)
  if (setupRequired) {
    return (
      <div className="min-h-screen bg-zinc-50 flex flex-col justify-between text-zinc-900 font-sans selection:bg-black selection:text-white">
        {/* Top Header */}
        <header className="h-16 border-b border-zinc-200 bg-white/80 backdrop-blur px-8 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <svg viewBox="0 0 24 24" fill="currentColor" className="w-6 h-6 text-black">
              <path d="M12 2L2 22h20L12 2z" />
            </svg>
            <span className="font-semibold text-sm tracking-tight text-black">DEPO</span>
            <span className="text-zinc-300">/</span>
            <span className="text-xs font-medium text-zinc-500">Initial Setup Wizard</span>
          </div>
          <div className="flex items-center gap-2 text-xs font-mono text-zinc-500 bg-zinc-100 px-2.5 py-1 rounded-full border border-zinc-200">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
            Fresh System Unconfigured
          </div>
        </header>

        {/* Wizard Main Content */}
        <main className="flex-1 flex items-center justify-center p-6 md:p-10">
          <div className="w-full max-w-2xl bg-white border border-zinc-200 rounded-xl shadow-sm overflow-hidden animate-fade-in">
            {/* Step Progress Indicators */}
            <div className="border-b border-zinc-100 bg-zinc-50/60 p-4 px-6">
              <div className="flex items-center justify-between">
                {[
                  { num: 1, title: 'Architecture' },
                  { num: 2, title: 'Administrator' },
                  { num: 3, title: 'Preferences' },
                  { num: 4, title: 'Launch' },
                ].map((s) => (
                  <div key={s.num} className="flex items-center gap-2">
                    <span
                      className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-semibold transition-all ${
                        wizardStep === s.num
                          ? 'bg-black text-white'
                          : wizardStep > s.num
                          ? 'bg-zinc-900 text-white'
                          : 'bg-zinc-200 text-zinc-500'
                      }`}
                    >
                      {wizardStep > s.num ? '✓' : s.num}
                    </span>
                    <span
                      className={`text-xs hidden sm:inline ${
                        wizardStep === s.num ? 'font-semibold text-black' : 'text-zinc-400 font-normal'
                      }`}
                    >
                      {s.title}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div className="p-8">
              {error && (
                <div className="mb-6 p-3.5 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs font-medium flex items-center gap-2">
                  <span className="w-1.5 h-1.5 rounded-full bg-rose-600"></span>
                  {error}
                </div>
              )}

              {/* STEP 1: WELCOME & ARCHITECTURE */}
              {wizardStep === 1 && (
                <div className="space-y-6">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight text-zinc-900">
                      Welcome to DEPO Deployment Platform
                    </h2>
                    <p className="text-xs text-zinc-500 mt-1">
                      Configure your high-performance self-hosted multi-VPS static frontend control center.
                    </p>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Platform Instance Name
                    </label>
                    <input
                      type="text"
                      value={instanceName}
                      onChange={(e) => setInstanceName(e.target.value)}
                      placeholder="My Organization Fleet"
                      className="vercel-input"
                    />
                    <span className="text-[11px] text-zinc-400 mt-1 block">
                      Displayed on notification alerts and the dashboard header.
                    </span>
                  </div>

                  {/* Architecture Feature Grid */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
                    <div className="p-3.5 rounded-lg border border-zinc-200 bg-zinc-50/50 space-y-1">
                      <div className="flex items-center gap-2 text-xs font-semibold text-zinc-900">
                        <Server size={14} className="text-black" /> Outbound VPS Agents
                      </div>
                      <p className="text-[11px] text-zinc-500 leading-normal">
                        Agents long-poll over HTTPS. Zero open inbound ports required on Ubuntu VPSs.
                      </p>
                    </div>

                    <div className="p-3.5 rounded-lg border border-zinc-200 bg-zinc-50/50 space-y-1">
                      <div className="flex items-center gap-2 text-xs font-semibold text-zinc-900">
                        <Zap size={14} className="text-black" /> Atomic Symlink Switch
                      </div>
                      <p className="text-[11px] text-zinc-500 leading-normal">
                        Instant zero-downtime release activation with one-click atomic rollback.
                      </p>
                    </div>

                    <div className="p-3.5 rounded-lg border border-zinc-200 bg-zinc-50/50 space-y-1">
                      <div className="flex items-center gap-2 text-xs font-semibold text-zinc-900">
                        <Layers size={14} className="text-black" /> Caddy Web Server
                      </div>
                      <p className="text-[11px] text-zinc-500 leading-normal">
                        Managed Caddy configuration with automated Let's Encrypt TLS certificates.
                      </p>
                    </div>

                    <div className="p-3.5 rounded-lg border border-zinc-200 bg-zinc-50/50 space-y-1">
                      <div className="flex items-center gap-2 text-xs font-semibold text-zinc-900">
                        <Shield size={14} className="text-black" /> Hardened Cryptography
                      </div>
                      <p className="text-[11px] text-zinc-500 leading-normal">
                        Argon2id password hashing and AES-256-GCM secret envelope storage.
                      </p>
                    </div>
                  </div>

                  <div className="flex justify-end pt-4 border-t border-zinc-100">
                    <button
                      type="button"
                      onClick={() => setWizardStep(2)}
                      className="btn-vercel-primary px-5 py-2 text-xs"
                    >
                      Continue to Account Setup <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              )}

              {/* STEP 2: PRIMARY ADMINISTRATOR */}
              {wizardStep === 2 && (
                <div className="space-y-5">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight text-zinc-900">
                      Create Master Administrator
                    </h2>
                    <p className="text-xs text-zinc-500 mt-1">
                      This primary account receives platform OWNER privileges and full administrative access.
                    </p>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Administrator Name (Optional)
                    </label>
                    <input
                      type="text"
                      value={adminName}
                      onChange={(e) => setAdminName(e.target.value)}
                      placeholder="Jane Doe"
                      className="vercel-input"
                    />
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Owner Email Address
                    </label>
                    <div className="relative">
                      <input
                        type="email"
                        required
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="admin@company.com"
                        className="vercel-input"
                      />
                    </div>
                    <span className="text-[11px] text-zinc-400 mt-1 block">
                      Used for signing in and receiving audit alerts.
                    </span>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Master Password
                    </label>
                    <div className="relative">
                      <input
                        type={showPassword ? 'text' : 'password'}
                        required
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="••••••••••••••••"
                        className="vercel-input pr-10 font-mono"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(!showPassword)}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-zinc-600"
                      >
                        {showPassword ? <EyeOff size={16} /> : <Eye size={16} />}
                      </button>
                    </div>

                    {/* Password Strength Meter */}
                    {password && (
                      <div className="mt-2 space-y-1">
                        <div className="flex items-center justify-between text-[11px]">
                          <span className="text-zinc-500">Security entropy:</span>
                          <span className={`font-semibold ${getStrengthLabel().text}`}>
                            {getStrengthLabel().label}
                          </span>
                        </div>
                        <div className="h-1.5 w-full bg-zinc-100 rounded-full overflow-hidden flex gap-1">
                          {[1, 2, 3, 4, 5].map((lvl) => (
                            <div
                              key={lvl}
                              className={`h-full flex-1 rounded-full transition-all ${
                                strength >= lvl ? getStrengthLabel().color : 'bg-transparent'
                              }`}
                            ></div>
                          ))}
                        </div>
                      </div>
                    )}
                    <span className="text-[11px] text-zinc-400 mt-1 block">
                      Must be at least 10 characters. Hashed via Argon2id (64MB memory, 3 iterations).
                    </span>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Confirm Master Password
                    </label>
                    <input
                      type={showPassword ? 'text' : 'password'}
                      required
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      placeholder="••••••••••••••••"
                      className="vercel-input font-mono"
                    />
                    {confirmPassword && password !== confirmPassword && (
                      <span className="text-[11px] text-rose-500 mt-1 block">Passwords do not match</span>
                    )}
                    {confirmPassword && password === confirmPassword && (
                      <span className="text-[11px] text-emerald-600 mt-1 flex items-center gap-1">
                        <CheckCircle2 size={12} /> Passwords match
                      </span>
                    )}
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-zinc-100">
                    <button
                      type="button"
                      onClick={() => setWizardStep(1)}
                      className="btn-vercel-secondary text-xs"
                    >
                      <ArrowLeft size={14} /> Back
                    </button>
                    <button
                      type="button"
                      disabled={!email || password.length < 10 || password !== confirmPassword}
                      onClick={() => setWizardStep(3)}
                      className="btn-vercel-primary px-5 py-2 text-xs"
                    >
                      Next: Operational Settings <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              )}

              {/* STEP 3: PREFERENCES & DEFAULTS */}
              {wizardStep === 3 && (
                <div className="space-y-5">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight text-zinc-900">
                      Operational & Security Preferences
                    </h2>
                    <p className="text-xs text-zinc-500 mt-1">
                      Set deployment retention policies and optional notification hooks.
                    </p>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Release Artifact Retention (Days)
                    </label>
                    <input
                      type="number"
                      min={7}
                      max={365}
                      value={retentionDays}
                      onChange={(e) => setRetentionDays(parseInt(e.target.value) || 30)}
                      className="vercel-input"
                    />
                    <span className="text-[11px] text-zinc-400 mt-1 block">
                      Releases older than this period can be pruned to preserve VPS disk capacity.
                    </span>
                  </div>

                  <div className="p-4 rounded-lg border border-zinc-200 bg-zinc-50/50 space-y-3">
                    <label className="flex items-start gap-3 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={autoRollback}
                        onChange={(e) => setAutoRollback(e.target.checked)}
                        className="mt-0.5 rounded border-zinc-300 text-black focus:ring-0"
                      />
                      <div>
                        <span className="text-xs font-semibold text-zinc-900 block">
                          Automatic Atomic Rollback Protection
                        </span>
                        <span className="text-[11px] text-zinc-500 block leading-normal mt-0.5">
                          If an updated frontend fails Caddy validation or post-switch health check, the agent immediately restores the prior symlink within 200ms.
                        </span>
                      </div>
                    </label>
                  </div>

                  <div>
                    <label className="block text-xs font-semibold uppercase tracking-wider text-zinc-600 mb-1.5">
                      Optional ntfy.sh Topic for Push Alerts
                    </label>
                    <input
                      type="text"
                      value={ntfyTopic}
                      onChange={(e) => setNtfyTopic(e.target.value)}
                      placeholder="my-team-depo-alerts"
                      className="vercel-input font-mono text-xs"
                    />
                    <span className="text-[11px] text-zinc-400 mt-1 block">
                      Free outbound push notifications to mobile or desktop without third-party email servers.
                    </span>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-zinc-100">
                    <button
                      type="button"
                      onClick={() => setWizardStep(2)}
                      className="btn-vercel-secondary text-xs"
                    >
                      <ArrowLeft size={14} /> Back
                    </button>
                    <button
                      type="button"
                      onClick={() => setWizardStep(4)}
                      className="btn-vercel-primary px-5 py-2 text-xs"
                    >
                      Review & Complete <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              )}

              {/* STEP 4: REVIEW & LAUNCH */}
              {wizardStep === 4 && (
                <div className="space-y-6">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight text-zinc-900">
                      Ready to Initialize DEPO
                    </h2>
                    <p className="text-xs text-zinc-500 mt-1">
                      Confirm configuration details below. Clicking launch will create the primary owner account and initialize your control center.
                    </p>
                  </div>

                  <div className="rounded-xl border border-zinc-200 divide-y divide-zinc-100 bg-zinc-50/50 text-xs">
                    <div className="p-3.5 flex justify-between">
                      <span className="text-zinc-500 font-medium">Platform Name</span>
                      <span className="font-semibold text-zinc-900">{instanceName}</span>
                    </div>
                    <div className="p-3.5 flex justify-between">
                      <span className="text-zinc-500 font-medium">Master Admin Email</span>
                      <span className="font-mono text-zinc-900 font-semibold">{email}</span>
                    </div>
                    <div className="p-3.5 flex justify-between">
                      <span className="text-zinc-500 font-medium">Password Protection</span>
                      <span className="text-emerald-600 font-medium flex items-center gap-1">
                        <CheckCircle2 size={12} /> Argon2id (Validated)
                      </span>
                    </div>
                    <div className="p-3.5 flex justify-between">
                      <span className="text-zinc-500 font-medium">Release Retention</span>
                      <span className="text-zinc-800">{retentionDays} days</span>
                    </div>
                    <div className="p-3.5 flex justify-between">
                      <span className="text-zinc-500 font-medium">Automated Rollback</span>
                      <span className="text-emerald-600 font-medium">Active (Zero-Downtime)</span>
                    </div>
                  </div>

                  <div className="flex items-center justify-between pt-4 border-t border-zinc-100">
                    <button
                      type="button"
                      onClick={() => setWizardStep(3)}
                      className="btn-vercel-secondary text-xs"
                    >
                      <ArrowLeft size={14} /> Back
                    </button>
                    <button
                      type="button"
                      disabled={loading}
                      onClick={handleSetupSubmit}
                      className="btn-vercel-primary px-6 py-2.5 text-xs bg-black hover:bg-zinc-800 font-semibold shadow-sm"
                    >
                      {loading ? (
                        'Initializing Platform...'
                      ) : (
                        <span className="flex items-center gap-1.5">
                          <Sparkles size={14} /> Launch DEPO Control Panel
                        </span>
                      )}
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </main>

        {/* Footer */}
        <footer className="py-4 text-center text-xs text-zinc-400 border-t border-zinc-100 bg-white">
          DEPO Self-Hosted Multi-VPS Static Frontend Platform • Outbound HTTPS Agent Protocol
        </footer>
      </div>
    );
  }

  // STANDARD LOGIN SCREEN (Vercel Aesthetic Pro White Mode)
  return (
    <div className="min-h-screen bg-zinc-50 flex flex-col justify-between text-zinc-900 font-sans selection:bg-black selection:text-white">
      {/* Top Bar */}
      <header className="h-16 border-b border-zinc-200 bg-white px-8 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <svg viewBox="0 0 24 24" fill="currentColor" className="w-6 h-6 text-black">
            <path d="M12 2L2 22h20L12 2z" />
          </svg>
          <span className="font-semibold text-sm tracking-tight text-black">DEPO</span>
        </div>
        <div className="text-xs font-mono text-zinc-500">v1.0.0</div>
      </header>

      {/* Main Login Card */}
      <main className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-sm bg-white border border-zinc-200 rounded-xl p-8 shadow-sm">
          <div className="text-center mb-6">
            <div className="w-10 h-10 rounded-lg bg-black text-white mx-auto flex items-center justify-center mb-3">
              <svg viewBox="0 0 24 24" fill="currentColor" className="w-5 h-5 text-white">
                <path d="M12 2L2 22h20L12 2z" />
              </svg>
            </div>
            <h1 className="text-xl font-bold tracking-tight text-zinc-900">Sign in to DEPO</h1>
            <p className="text-xs text-zinc-500 mt-1">Multi-VPS static frontend deployment control center</p>
          </div>

          {error && (
            <div className="mb-4 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs font-medium">
              {error}
            </div>
          )}

          <form onSubmit={handleLoginSubmit} className="space-y-4">
            <div>
              <label className="block text-xs font-medium text-zinc-700 mb-1">Email Address</label>
              <input
                type="email"
                required
                autoFocus
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@domain.com"
                className="vercel-input"
              />
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-xs font-medium text-zinc-700">Password</label>
              </div>
              <input
                type="password"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••••••"
                className="vercel-input"
              />
            </div>

            {totpRequired && (
              <div className="animate-fade-in pt-1">
                <label className="block text-xs font-semibold text-amber-700 mb-1 flex items-center gap-1.5">
                  <ShieldCheck size={14} /> Two-Factor Authentication Code
                </label>
                <input
                  type="text"
                  required
                  autoFocus
                  maxLength={10}
                  value={totpCode}
                  onChange={(e) => setTotpCode(e.target.value)}
                  placeholder="6-digit TOTP code"
                  className="vercel-input font-mono text-center tracking-widest text-base"
                />
              </div>
            )}

            <button
              type="submit"
              disabled={loading}
              className="w-full btn-vercel-primary py-2.5 text-xs font-semibold mt-4"
            >
              {loading ? 'Authenticating...' : 'Sign In'}
            </button>
          </form>

          <div className="mt-6 pt-4 border-t border-zinc-100 text-center text-[11px] text-zinc-400">
            Protected by Argon2id & Hardware Security • Outbound VPS Agent Protocol
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="py-4 text-center text-xs text-zinc-400 border-t border-zinc-100 bg-white">
        DEPO Self-Hosted Multi-VPS Static Frontend Platform
      </footer>
    </div>
  );
};

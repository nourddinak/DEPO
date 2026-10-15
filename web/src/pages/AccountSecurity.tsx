import React, { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { api } from '../api/client';
import { ShieldCheck, Key, Copy, Check, AlertTriangle, Download } from 'lucide-react';

export const AccountSecurity: React.FC = () => {
  const { user, refresh } = useAuth();
  const [totpSecret, setTotpSecret] = useState('');
  const [totpUrl, setTotpUrl] = useState('');
  const [verifyCode, setVerifyCode] = useState('');
  const [recoveryCodes, setRecoveryCodes] = useState<string[]>([]);
  const [step, setStep] = useState<'idle' | 'setup' | 'completed'>('idle');
  const [copiedCodes, setCopiedCodes] = useState(false);
  const [error, setError] = useState('');

  const handleStartTOTP = async () => {
    setError('');
    try {
      const res = await api.setupTOTP();
      setTotpSecret(res.secret);
      setTotpUrl(res.url);
      setStep('setup');
    } catch (err: any) {
      setError(err.message || 'Failed to initiate 2FA setup');
    }
  };

  const handleVerifyTOTP = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const res = await api.verifyTOTP(verifyCode);
      setRecoveryCodes(res.recovery_codes);
      setStep('completed');
      refresh();
    } catch (err: any) {
      setError(err.message || 'Invalid verification code');
    }
  };

  const downloadRecoveryCodes = () => {
    const text = `DEPO Control Panel Recovery Codes\nUser: ${user?.email}\nGenerated: ${new Date().toISOString()}\n\n` +
      recoveryCodes.map((c, i) => `${i + 1}. ${c}`).join('\n') +
      '\n\nKeep these codes safe in case you lose access to your authenticator app.';
    const blob = new Blob([text], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `depo-recovery-codes-${user?.email}.txt`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6 animate-fade-in max-w-4xl text-zinc-900">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Account Security</h1>
        <p className="text-xs text-zinc-500 mt-0.5">
          Manage two-factor authentication (TOTP) and cryptographic credentials
        </p>
      </div>

      {/* Account Info Card */}
      <div className="vercel-card p-6 space-y-4">
        <h2 className="text-sm font-bold text-zinc-900">Profile Overview</h2>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs font-mono">
          <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-200">
            <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">Email</span>
            <span className="text-zinc-900 font-medium">{user?.email}</span>
          </div>
          <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-200">
            <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">Role</span>
            <span className="text-black font-semibold uppercase">{user?.role}</span>
          </div>
          <div className="p-3 bg-zinc-50 rounded-lg border border-zinc-200">
            <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">2FA Status</span>
            <span className={user?.totp_enabled ? 'text-emerald-700 font-bold' : 'text-zinc-500'}>
              {user?.totp_enabled ? 'Active' : 'Not Enabled'}
            </span>
          </div>
        </div>
      </div>

      {/* Two-Factor Authentication Card */}
      <div className="vercel-card p-6 space-y-5">
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
              <ShieldCheck size={18} className="text-zinc-700" />
              Two-Factor Authentication (TOTP)
            </h2>
            <p className="text-xs text-zinc-500 mt-1">
              Add a second layer of security using an authenticator app (Google Authenticator, 1Password, Aegis, Authy).
            </p>
          </div>
          {user?.totp_enabled && (
            <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
              Protected
            </span>
          )}
        </div>

        {error && (
          <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-lg">{error}</div>
        )}

        {step === 'idle' && (
          <div>
            {user?.totp_enabled ? (
              <div className="text-xs text-zinc-600 flex items-center gap-2">
                <Check size={16} className="text-emerald-600" />
                Two-factor authentication is active on your account. You will be prompted for a TOTP code upon login.
              </div>
            ) : (
              <button onClick={handleStartTOTP} className="btn-vercel-primary text-xs">
                <Key size={14} /> Enable Two-Factor Authentication
              </button>
            )}
          </div>
        )}

        {step === 'setup' && (
          <div className="space-y-4 pt-2 border-t border-zinc-100 text-xs animate-fade-in">
            <div className="space-y-2">
              <span className="font-semibold text-zinc-800 block">1. Enter Secret Key into Authenticator</span>
              <div className="p-3 bg-zinc-50 rounded-xl border border-zinc-200 flex items-center justify-between font-mono text-xs text-zinc-800">
                <span className="font-bold">{totpSecret}</span>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(totpSecret);
                    alert('Secret copied to clipboard');
                  }}
                  className="btn-vercel-secondary py-1 px-2.5 text-[11px]"
                >
                  <Copy size={12} /> Copy
                </button>
              </div>
              <span className="text-zinc-500 block text-[11px]">
                Or enter URI into your authenticator: <code className="text-zinc-700 font-mono break-all">{totpUrl}</code>
              </span>
            </div>

            <form onSubmit={handleVerifyTOTP} className="space-y-3 pt-2">
              <span className="font-semibold text-zinc-800 block">2. Verify 6-digit Code</span>
              <div className="flex gap-3 max-w-sm">
                <input
                  type="text"
                  required
                  maxLength={6}
                  value={verifyCode}
                  onChange={(e) => setVerifyCode(e.target.value.replace(/\D/g, ''))}
                  placeholder="123456"
                  className="vercel-input font-mono text-center text-lg tracking-widest"
                />
                <button type="submit" disabled={verifyCode.length < 6} className="btn-vercel-primary text-xs shrink-0">
                  Verify & Activate
                </button>
              </div>
            </form>
          </div>
        )}

        {step === 'completed' && (
          <div className="space-y-4 pt-2 border-t border-zinc-100 text-xs animate-fade-in">
            <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 font-medium">
              Two-Factor Authentication is now enabled! Save your recovery codes below.
            </div>

            <div className="p-4 rounded-xl bg-zinc-50 border border-zinc-200 space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-amber-800 flex items-center gap-1.5 uppercase tracking-wider text-[11px]">
                  <AlertTriangle size={14} className="text-amber-600" /> One-Time Emergency Recovery Codes
                </span>
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(recoveryCodes.join('\n'));
                      setCopiedCodes(true);
                      setTimeout(() => setCopiedCodes(false), 2000);
                    }}
                    className="btn-vercel-secondary py-1 px-2.5 text-[11px]"
                  >
                    {copiedCodes ? <Check size={12} className="text-emerald-600" /> : <Copy size={12} />}
                    {copiedCodes ? 'Copied' : 'Copy All'}
                  </button>
                  <button onClick={downloadRecoveryCodes} className="btn-vercel-primary py-1 px-2.5 text-[11px]">
                    <Download size={12} /> Download (.txt)
                  </button>
                </div>
              </div>

              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 font-mono text-zinc-900 text-center py-2">
                {recoveryCodes.map((code, idx) => (
                  <div key={idx} className="p-2 bg-white rounded border border-zinc-200 font-bold shadow-2xs">
                    {code}
                  </div>
                ))}
              </div>
              <span className="text-[11px] text-zinc-500 block">
                Each code can only be used once. If you ever lose your TOTP app, you can sign in with any of these codes.
              </span>
            </div>

            <button onClick={() => setStep('idle')} className="btn-vercel-secondary text-xs">
              Return to Account Overview
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

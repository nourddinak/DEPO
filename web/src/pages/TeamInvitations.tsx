import React, { useState, useEffect } from 'react';
import { User, Invitation } from '../types';
import { api } from '../api/client';
import { Modal } from '../components/Modal';
import { Users, UserPlus, Copy, Check, ShieldCheck, Clock } from 'lucide-react';

export const TeamInvitations: React.FC = () => {
  const [users, setUsers] = useState<User[]>([]);
  const [invitations, setInvitations] = useState<Invitation[]>([]);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [email, setEmail] = useState('');
  const [role, setRole] = useState<'admin' | 'member'>('member');
  const [loading, setLoading] = useState(true);
  const [inviteResult, setInviteResult] = useState<{
    invitation: Invitation;
    invitation_token: string;
    invitation_url: string;
  } | null>(null);
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState('');

  const loadData = async () => {
    try {
      const [u, i] = await Promise.all([api.getUsers(), api.getInvitations()]);
      setUsers(u || []);
      setInvitations(i || []);
    } catch (err) {
      console.error('Failed to load team data', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleInvite = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      const res = await api.createInvitation(email, role);
      setInviteResult(res);
      loadData();
    } catch (err: any) {
      setError(err.message || 'Failed to create invitation');
    }
  };

  return (
    <div className="space-y-6 animate-fade-in max-w-5xl text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Team & Invitations</h1>
          <p className="text-xs text-zinc-500 mt-0.5">Manage owner and team member access with role-based permissions</p>
        </div>
        <button
          onClick={() => {
            setInviteResult(null);
            setEmail('');
            setShowInviteModal(true);
          }}
          className="btn-vercel-primary text-xs"
        >
          <UserPlus size={14} /> Invite Member
        </button>
      </div>

      {/* Users Table */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
          <Users size={16} className="text-zinc-700" /> Active Team Members
        </h2>
        <div className="vercel-card overflow-hidden">
          <table className="w-full text-left text-xs">
            <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
              <tr>
                <th className="px-5 py-3 text-[11px]">Member Email</th>
                <th className="px-5 py-3 text-[11px]">Role</th>
                <th className="px-5 py-3 text-[11px]">2FA Status</th>
                <th className="px-5 py-3 text-[11px] text-right">Joined</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {(users || []).map((u) => (
                <tr key={u.id} className="hover:bg-zinc-50/80 transition">
                  <td className="px-5 py-3.5 font-semibold text-zinc-900">{u.email}</td>
                  <td className="px-5 py-3.5">
                    <span className="capitalize px-2 py-0.5 rounded text-[11px] font-mono font-medium bg-zinc-100 text-zinc-700 border border-zinc-200">
                      {u.role}
                    </span>
                  </td>
                  <td className="px-5 py-3.5">
                    {u.totp_enabled ? (
                      <span className="text-emerald-700 font-medium flex items-center gap-1">
                        <ShieldCheck size={14} className="text-emerald-600" /> Enabled
                      </span>
                    ) : (
                      <span className="text-zinc-400">Disabled</span>
                    )}
                  </td>
                  <td className="px-5 py-3.5 text-zinc-500 font-mono text-right">
                    {new Date(u.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pending Invitations Table */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold text-zinc-900">Invitations</h2>
        <div className="vercel-card overflow-hidden">
          {(invitations || []).length === 0 ? (
            <div className="p-8 text-center text-zinc-400 text-xs">No pending invitations</div>
          ) : (
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Invited Email</th>
                  <th className="px-5 py-3 text-[11px]">Assigned Role</th>
                  <th className="px-5 py-3 text-[11px]">Expires</th>
                  <th className="px-5 py-3 text-[11px] text-right">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {(invitations || []).map((inv) => (
                  <tr key={inv.id} className="hover:bg-zinc-50/80 transition">
                    <td className="px-5 py-3.5 font-semibold text-zinc-900">{inv.email}</td>
                    <td className="px-5 py-3.5 capitalize text-zinc-700">{inv.role}</td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono flex items-center gap-1">
                      <Clock size={12} className="text-zinc-400" />
                      {new Date(inv.expires_at).toLocaleDateString()}
                    </td>
                    <td className="px-5 py-3.5 text-right font-mono">
                      {inv.accepted_at ? (
                        <span className="text-emerald-700 font-semibold">Accepted</span>
                      ) : (
                        <span className="text-amber-700 font-semibold">Pending</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Invite Modal */}
      <Modal
        isOpen={showInviteModal}
        onClose={() => setShowInviteModal(false)}
        title={inviteResult ? 'Invitation Created' : 'Invite Team Member'}
        maxWidth="max-w-lg"
      >
        {!inviteResult ? (
          <form onSubmit={handleInvite} className="space-y-4">
            {error && (
              <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-lg">{error}</div>
            )}
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Member Email Address
              </label>
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="colleague@domain.com"
                className="vercel-input text-xs"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Access Role
              </label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value as any)}
                className="vercel-input text-xs"
              >
                <option value="member">Member (view & deploy assigned projects)</option>
                <option value="admin">Admin (manage servers, projects & invitations)</option>
              </select>
            </div>
            <div className="flex justify-end gap-3 pt-4 border-t border-zinc-100">
              <button type="button" onClick={() => setShowInviteModal(false)} className="btn-vercel-secondary text-xs">
                Cancel
              </button>
              <button type="submit" disabled={!email} className="btn-vercel-primary text-xs">
                Send Invitation
              </button>
            </div>
          </form>
        ) : (
          <div className="space-y-4 text-xs">
            <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 font-medium rounded-xl">
              Invitation link generated for {inviteResult.invitation.email}.
            </div>
            <div className="space-y-1">
              <span className="font-semibold text-zinc-700 uppercase tracking-wider text-[11px] block">
                Share this onboarding link with the user:
              </span>
              <div className="relative">
                <input
                  type="text"
                  readOnly
                  value={window.location.origin + inviteResult.invitation_url}
                  className="vercel-input font-mono text-xs pr-20 bg-white select-all"
                />
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(window.location.origin + inviteResult.invitation_url);
                    setCopied(true);
                    setTimeout(() => setCopied(false), 2000);
                  }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 btn-vercel-secondary py-1 px-2.5 text-[10px]"
                >
                  {copied ? <Check size={12} className="text-emerald-600" /> : <Copy size={12} />}
                  {copied ? 'Copied' : 'Copy Link'}
                </button>
              </div>
            </div>
            <div className="flex justify-end pt-2 border-t border-zinc-100">
              <button onClick={() => setShowInviteModal(false)} className="btn-vercel-primary text-xs">
                Done
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

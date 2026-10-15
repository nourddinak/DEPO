import React, { useState } from 'react';
import { Server } from '../types';
import { api } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { Modal } from '../components/Modal';
import {
  Server as ServerIcon,
  Plus,
  Cpu,
  Activity,
  HardDrive,
  Copy,
  Check,
  Trash2,
  ShieldCheck,
} from 'lucide-react';

interface ServersProps {
  servers: Server[];
  onRefresh: () => void;
}

export const Servers: React.FC<ServersProps> = ({ servers = [], onRefresh }) => {
  const safeServers = servers || [];
  const [showEnrollModal, setShowEnrollModal] = useState(false);
  const [serverName, setServerName] = useState('');
  const [serverHost, setServerHost] = useState('');
  const [enrollData, setEnrollData] = useState<{
    server: Server;
    enrollment_token: string;
    install_command: string;
  } | null>(null);
  const [copiedCmd, setCopiedCmd] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const res = await api.createServer(serverName, serverHost);
      setEnrollData(res);
      onRefresh();
    } catch (err: any) {
      setError(err.message || 'Failed to create server');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove server '${name}'?`)) return;
    try {
      await api.deleteServer(id);
      onRefresh();
    } catch (err: any) {
      alert('Failed to delete server: ' + err.message);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">VPS Fleet</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Outbound-connected deployment agents running on target Ubuntu servers
          </p>
        </div>
        <button
          onClick={() => {
            setEnrollData(null);
            setServerName('');
            setServerHost('');
            setShowEnrollModal(true);
          }}
          className="btn-vercel-primary text-xs"
        >
          <Plus size={14} /> Enroll New Server
        </button>
      </div>

      {safeServers.length === 0 ? (
        <div className="vercel-card p-12 text-center">
          <div className="w-12 h-12 rounded-full bg-zinc-100 text-zinc-400 mx-auto flex items-center justify-center mb-3">
            <ServerIcon size={24} />
          </div>
          <h3 className="text-sm font-semibold text-zinc-900">No servers connected</h3>
          <p className="text-xs text-zinc-500 mt-1 max-w-md mx-auto">
            DEPO uses outbound-only authenticated HTTPS polling. No incoming firewall ports are needed on your VPS.
          </p>
          <button
            onClick={() => setShowEnrollModal(true)}
            className="btn-vercel-primary text-xs mt-4"
          >
            <Plus size={14} /> Enroll Your First VPS
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {safeServers.map((s) => (
            <div key={s.id} className="vercel-card vercel-card-hover p-5 space-y-4 flex flex-col justify-between">
              <div className="space-y-3">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-bold text-zinc-900 text-sm">{s.name}</h3>
                    <span className="text-xs font-mono text-zinc-400 block mt-0.5">
                      {s.host ? s.host : 'Outbound Long-Poll Worker'}
                    </span>
                  </div>
                  <StatusBadge status={s.status} size="sm" />
                </div>

                <div className="grid grid-cols-2 gap-2 text-[11px] font-mono text-zinc-600 bg-zinc-50 p-2.5 rounded-lg border border-zinc-200">
                  <div>
                    <span className="text-zinc-400 block text-[10px] uppercase">OS</span>
                    <span className="text-zinc-800 capitalize font-medium">{s.os}</span>
                  </div>
                  <div>
                    <span className="text-zinc-400 block text-[10px] uppercase">Agent</span>
                    <span className="text-zinc-800 font-medium">{s.agent_version || 'v1.0.0'}</span>
                  </div>
                  <div className="col-span-2 pt-1 border-t border-zinc-200">
                    <span className="text-zinc-400 block text-[10px] uppercase">Last Heartbeat</span>
                    <span className="text-zinc-700">
                      {s.last_heartbeat_at ? new Date(s.last_heartbeat_at).toLocaleTimeString() : 'Never'}
                    </span>
                  </div>
                </div>

                {/* Telemetry Progress Bars */}
                <div className="space-y-2 pt-2 border-t border-zinc-100 text-xs">
                  <div>
                    <div className="flex justify-between text-zinc-500 mb-1">
                      <span className="flex items-center gap-1">
                        <Cpu size={12} /> CPU
                      </span>
                      <span className="font-mono font-medium text-zinc-700">{s.cpu_pct.toFixed(1)}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-zinc-100 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          s.cpu_pct > 80 ? 'bg-rose-500' : 'bg-black'
                        }`}
                        style={{ width: `${Math.min(s.cpu_pct, 100)}%` }}
                      ></div>
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between text-zinc-500 mb-1">
                      <span className="flex items-center gap-1">
                        <Activity size={12} /> Memory
                      </span>
                      <span className="font-mono font-medium text-zinc-700">{s.ram_pct.toFixed(1)}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-zinc-100 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          s.ram_pct > 85 ? 'bg-rose-500' : 'bg-emerald-600'
                        }`}
                        style={{ width: `${Math.min(s.ram_pct, 100)}%` }}
                      ></div>
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between text-zinc-500 mb-1">
                      <span className="flex items-center gap-1">
                        <HardDrive size={12} /> Disk ({formatBytes(s.disk_free_bytes)} free)
                      </span>
                      <span className="font-mono font-medium text-zinc-700">{s.disk_pct.toFixed(1)}%</span>
                    </div>
                    <div className="w-full h-1.5 bg-zinc-100 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all ${
                          s.disk_pct > 90 ? 'bg-rose-500' : 'bg-blue-600'
                        }`}
                        style={{ width: `${Math.min(s.disk_pct, 100)}%` }}
                      ></div>
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex justify-end pt-3 border-t border-zinc-100">
                <button
                  onClick={() => handleDelete(s.id, s.name)}
                  className="p-1.5 text-zinc-400 hover:text-rose-600 rounded transition"
                  title="Remove server"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Enrollment Modal */}
      <Modal
        isOpen={showEnrollModal}
        onClose={() => setShowEnrollModal(false)}
        title={enrollData ? 'Install Agent on Ubuntu VPS' : 'Enroll New Server'}
        maxWidth="max-w-xl"
      >
        {!enrollData ? (
          <form onSubmit={handleCreate} className="space-y-4">
            {error && (
              <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-lg">{error}</div>
            )}
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Server Display Name
              </label>
              <input
                type="text"
                required
                value={serverName}
                onChange={(e) => setServerName(e.target.value)}
                placeholder="vps-lon-production-01"
                className="vercel-input font-mono text-sm"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Host IP or Domain (Optional note)
              </label>
              <input
                type="text"
                value={serverHost}
                onChange={(e) => setServerHost(e.target.value)}
                placeholder="203.0.113.195"
                className="vercel-input font-mono text-sm"
              />
            </div>
            <div className="p-3 rounded-xl bg-zinc-50 border border-zinc-200 text-zinc-600 text-xs flex items-start gap-2">
              <ShieldCheck size={16} className="shrink-0 mt-0.5 text-black" />
              <span>
                Agents connect outbound over authenticated HTTPS. You do not need to open any incoming ports in your VPS firewall or security groups.
              </span>
            </div>
            <div className="flex justify-end gap-3 pt-4 border-t border-zinc-100">
              <button type="button" onClick={() => setShowEnrollModal(false)} className="btn-vercel-secondary text-xs">
                Cancel
              </button>
              <button type="submit" disabled={loading || !serverName} className="btn-vercel-primary text-xs">
                {loading ? 'Generating Token...' : 'Generate Enrollment Token'}
              </button>
            </div>
          </form>
        ) : (
          <div className="space-y-5 text-xs">
            <div className="p-3 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 font-medium">
              Single-use enrollment token generated for '{enrollData.server.name}'.
            </div>

            <div className="space-y-2">
              <span className="font-semibold text-zinc-700 uppercase tracking-wider text-[11px] block">
                Run this command on your Ubuntu VPS (via SSH):
              </span>
              <div className="relative">
                <pre className="p-4 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 select-all overflow-x-auto">
                  {enrollData.install_command}
                </pre>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(enrollData.install_command);
                    setCopiedCmd(true);
                    setTimeout(() => setCopiedCmd(false), 2000);
                  }}
                  className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                >
                  {copiedCmd ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                  {copiedCmd ? 'Copied' : 'Copy Command'}
                </button>
              </div>
            </div>

            <div className="p-3 rounded-xl bg-zinc-50 border border-zinc-200 text-zinc-600 text-[11px] space-y-1">
              <div className="font-semibold text-zinc-900">What this installer does:</div>
              <ul className="list-disc pl-4 space-y-0.5">
                <li>Installs the lightweight Go <code className="text-black font-semibold font-mono">depo-agent</code> binary to <code className="text-zinc-800 font-mono">/usr/local/bin/depo-agent</code></li>
                <li>Exchanges the single-use token for a persistent agent credential</li>
                <li>Installs and enables the <code className="text-black font-semibold font-mono">depo-agent.service</code> systemd service</li>
                <li>Begins sending heartbeats and polling for deployment releases</li>
              </ul>
            </div>

            <div className="flex justify-end pt-2">
              <button
                onClick={() => {
                  setShowEnrollModal(false);
                  onRefresh();
                }}
                className="btn-vercel-primary text-xs"
              >
                Done
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

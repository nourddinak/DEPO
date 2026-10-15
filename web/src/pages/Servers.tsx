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
  Terminal,
  Settings,
  HelpCircle,
  RefreshCw,
  AlertTriangle,
  Code2,
} from 'lucide-react';

interface ServersProps {
  servers: Server[];
  onRefresh: () => void;
}

export const Servers: React.FC<ServersProps> = ({ servers = [], onRefresh }) => {
  const safeServers = servers || [];
  const [showEnrollModal, setShowEnrollModal] = useState(false);
  const [showGuideModal, setShowGuideModal] = useState(false);
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [serverSettingsTab, setServerSettingsTab] = useState<'install' | 'uninstall' | 'diagnostics'>('install');

  const [serverName, setServerName] = useState('');
  const [serverHost, setServerHost] = useState('');
  const [enrollData, setEnrollData] = useState<{
    server: Server;
    enrollment_token: string;
    install_command: string;
  } | null>(null);

  const [reinstallData, setReinstallData] = useState<{
    enrollment_token: string;
    install_command: string;
  } | null>(null);

  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const currentOrigin = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:8080';

  const copyToClipboard = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

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

  const handleGenerateNewToken = async (serverId: string) => {
    setLoading(true);
    try {
      const res = await api.resetServerEnrollToken(serverId);
      setReinstallData({
        enrollment_token: res.enrollment_token,
        install_command: res.install_command,
      });
      onRefresh();
    } catch (err: any) {
      alert('Failed to generate new enrollment token: ' + (err.message || 'Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to remove server '${name}' from your fleet?`)) return;
    try {
      await api.deleteServer(id);
      if (selectedServer?.id === id) {
        setSelectedServer(null);
      }
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
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">VPS Fleet</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Outbound-connected deployment agents running on target Ubuntu/Debian VPS servers
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setShowGuideModal(true)}
            className="btn-vercel-secondary text-xs flex items-center gap-1.5"
          >
            <HelpCircle size={14} /> Agent & VPS Guide
          </button>
          <button
            onClick={() => {
              setEnrollData(null);
              setServerName('');
              setServerHost('');
              setShowEnrollModal(true);
            }}
            className="btn-vercel-primary text-xs flex items-center gap-1.5"
          >
            <Plus size={14} /> Enroll New Server
          </button>
        </div>
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

              {/* Bottom Actions */}
              <div className="flex items-center justify-between pt-3 border-t border-zinc-100">
                <button
                  onClick={() => {
                    setSelectedServer(s);
                    setReinstallData(null);
                    setServerSettingsTab('install');
                  }}
                  className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-zinc-700 bg-zinc-100 hover:bg-zinc-200 rounded-lg transition"
                >
                  <Terminal size={13} />
                  <span>VPS Settings</span>
                </button>
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
                <pre className="p-4 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 select-all overflow-x-auto whitespace-pre-wrap break-all">
                  {enrollData.install_command}
                </pre>
                <button
                  onClick={() => copyToClipboard(enrollData.install_command, 'enroll_cmd')}
                  className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                >
                  {copiedKey === 'enroll_cmd' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                  {copiedKey === 'enroll_cmd' ? 'Copied' : 'Copy Command'}
                </button>
              </div>
            </div>

            <div className="p-3 rounded-xl bg-zinc-50 border border-zinc-200 text-zinc-600 text-[11px] space-y-1">
              <div className="font-semibold text-zinc-900">What this installer does:</div>
              <ul className="list-disc pl-4 space-y-0.5">
                <li>Installs the lightweight Go <code className="text-black font-semibold font-mono">depo-agent</code> binary to <code className="text-zinc-800 font-mono">/usr/local/bin/depo-agent</code></li>
                <li>Ensures Caddy web server is installed and active</li>
                <li>Exchanges the single-use token for a persistent agent credential</li>
                <li>Installs and enables the <code className="text-black font-semibold font-mono">depo-agent.service</code> systemd service</li>
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

      {/* VPS Settings & Agent Commands Modal for Specific Server */}
      {selectedServer && (
        <Modal
          isOpen={!!selectedServer}
          onClose={() => {
            setSelectedServer(null);
            setReinstallData(null);
          }}
          title={`VPS Settings: ${selectedServer.name}`}
          maxWidth="max-w-2xl"
        >
          <div className="space-y-4 text-xs">
            {/* Server Quick Info Bar */}
            <div className="flex flex-wrap items-center justify-between gap-2 p-3 bg-zinc-50 border border-zinc-200 rounded-xl text-zinc-700">
              <div className="flex items-center gap-2">
                <StatusBadge status={selectedServer.status} size="sm" />
                <span className="font-mono text-zinc-800 font-semibold">{selectedServer.name}</span>
                <span className="text-zinc-400">({selectedServer.host || 'Outbound Long-Poll'})</span>
              </div>
              <div className="font-mono text-[11px] text-zinc-500">
                Agent: {selectedServer.agent_version || 'v1.0.0'} | OS: {selectedServer.os}
              </div>
            </div>

            {/* Navigation Tabs */}
            <div className="flex border-b border-zinc-200 gap-4 text-xs font-semibold">
              <button
                onClick={() => setServerSettingsTab('install')}
                className={`pb-2 border-b-2 transition ${
                  serverSettingsTab === 'install'
                    ? 'border-black text-black'
                    : 'border-transparent text-zinc-400 hover:text-zinc-700'
                }`}
              >
                Install / Re-install Agent
              </button>
              <button
                onClick={() => setServerSettingsTab('uninstall')}
                className={`pb-2 border-b-2 transition ${
                  serverSettingsTab === 'uninstall'
                    ? 'border-rose-600 text-rose-600'
                    : 'border-transparent text-zinc-400 hover:text-zinc-700'
                }`}
              >
                Uninstall Agent
              </button>
              <button
                onClick={() => setServerSettingsTab('diagnostics')}
                className={`pb-2 border-b-2 transition ${
                  serverSettingsTab === 'diagnostics'
                    ? 'border-black text-black'
                    : 'border-transparent text-zinc-400 hover:text-zinc-700'
                }`}
              >
                Commands & Logs
              </button>
            </div>

            {/* Tab 1: Install / Re-install */}
            {serverSettingsTab === 'install' && (
              <div className="space-y-4 pt-1">
                <p className="text-zinc-600">
                  Run this automated command via SSH on the target Ubuntu VPS to install or re-link the DEPO edge agent daemon.
                </p>

                {reinstallData ? (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="font-semibold text-zinc-700 text-[11px]">Active One-Line Installation Command:</span>
                      <span className="text-[10px] text-emerald-600 font-medium">New single-use token active</span>
                    </div>
                    <div className="relative">
                      <pre className="p-3.5 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 select-all overflow-x-auto whitespace-pre-wrap break-all">
                        {reinstallData.install_command}
                      </pre>
                      <button
                        onClick={() => copyToClipboard(reinstallData.install_command, 'reinstall_cmd')}
                        className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                      >
                        {copiedKey === 'reinstall_cmd' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                        {copiedKey === 'reinstall_cmd' ? 'Copied' : 'Copy Command'}
                      </button>
                    </div>
                  </div>
                ) : (
                  <div className="p-4 bg-zinc-50 border border-zinc-200 rounded-xl space-y-3">
                    <div className="text-zinc-700">
                      Need to connect or re-install the agent on this VPS? Generating a fresh enrollment token will reset this server's connection credentials.
                    </div>
                    <button
                      onClick={() => handleGenerateNewToken(selectedServer.id)}
                      disabled={loading}
                      className="btn-vercel-primary text-xs flex items-center gap-1.5"
                    >
                      <RefreshCw size={13} className={loading ? 'animate-spin' : ''} />
                      {loading ? 'Generating...' : 'Generate New Installation Command'}
                    </button>
                  </div>
                )}

                <div className="p-3 bg-zinc-50 border border-zinc-200 rounded-xl text-zinc-600 space-y-1.5">
                  <span className="font-semibold text-zinc-900 block">Generic Script Syntax:</span>
                  <pre className="p-2 bg-white border border-zinc-200 rounded font-mono text-[11px] text-zinc-800 select-all overflow-x-auto">
                    curl -sSL {currentOrigin}/install-agent.sh | sudo bash -s -- --server {currentOrigin} --token &lt;ENROLL_TOKEN&gt;
                  </pre>
                </div>
              </div>
            )}

            {/* Tab 2: Uninstall Agent */}
            {serverSettingsTab === 'uninstall' && (
              <div className="space-y-4 pt-1">
                <div className="p-3 bg-rose-50 border border-rose-200 text-rose-800 rounded-xl text-xs space-y-1">
                  <div className="font-semibold flex items-center gap-1.5">
                    <AlertTriangle size={14} className="text-rose-600" /> Completely Uninstall Agent from VPS
                  </div>
                  <p className="text-[11px] text-rose-700">
                    Executing these commands on your VPS will stop the <code className="font-mono bg-rose-100 px-1 py-0.5 rounded">depo-agent.service</code>, remove the binary from <code className="font-mono bg-rose-100 px-1 py-0.5 rounded">/usr/local/bin/depo-agent</code>, and purge all credentials in <code className="font-mono bg-rose-100 px-1 py-0.5 rounded">/etc/depo-agent</code>.
                  </p>
                </div>

                {/* 1-Line Uninstall Script */}
                <div className="space-y-1.5">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold text-zinc-800 text-[11px] uppercase tracking-wider">
                      Option A: Automated 1-Line Uninstaller (Recommended)
                    </span>
                  </div>
                  <div className="relative">
                    <pre className="p-3 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 select-all overflow-x-auto whitespace-pre-wrap break-all">
                      curl -sSL {currentOrigin}/uninstall-agent.sh | sudo bash
                    </pre>
                    <button
                      onClick={() => copyToClipboard(`curl -sSL ${currentOrigin}/uninstall-agent.sh | sudo bash`, 'uninstall_cmd')}
                      className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                    >
                      {copiedKey === 'uninstall_cmd' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                      {copiedKey === 'uninstall_cmd' ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </div>

                {/* With purge www */}
                <div className="space-y-1.5">
                  <span className="font-semibold text-zinc-700 text-[11px]">
                    Option B: Uninstall & Purge All Web Deploys (/var/www):
                  </span>
                  <div className="relative">
                    <pre className="p-3 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 select-all overflow-x-auto whitespace-pre-wrap break-all">
                      curl -sSL {currentOrigin}/uninstall-agent.sh | sudo bash -s -- --purge-www
                    </pre>
                    <button
                      onClick={() => copyToClipboard(`curl -sSL ${currentOrigin}/uninstall-agent.sh | sudo bash -s -- --purge-www`, 'uninstall_purge_cmd')}
                      className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                    >
                      {copiedKey === 'uninstall_purge_cmd' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                      {copiedKey === 'uninstall_purge_cmd' ? 'Copied' : 'Copy'}
                    </button>
                  </div>
                </div>

                {/* Manual Steps */}
                <div className="space-y-1.5">
                  <span className="font-semibold text-zinc-700 text-[11px]">
                    Option C: Manual Shell Commands:
                  </span>
                  <div className="relative">
                    <pre className="p-3 bg-zinc-100 border border-zinc-200 rounded-xl font-mono text-[11px] text-zinc-800 select-all overflow-x-auto">
{`sudo systemctl stop depo-agent
sudo systemctl disable depo-agent
sudo rm -f /etc/systemd/system/depo-agent.service /usr/local/bin/depo-agent
sudo rm -rf /etc/depo-agent /var/lib/depo-agent
sudo systemctl daemon-reload`}
                    </pre>
                    <button
                      onClick={() => copyToClipboard(
`sudo systemctl stop depo-agent
sudo systemctl disable depo-agent
sudo rm -f /etc/systemd/system/depo-agent.service /usr/local/bin/depo-agent
sudo rm -rf /etc/depo-agent /var/lib/depo-agent
sudo systemctl daemon-reload`,
                        'manual_uninstall_cmd'
                      )}
                      className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
                    >
                      {copiedKey === 'manual_uninstall_cmd' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                      {copiedKey === 'manual_uninstall_cmd' ? 'Copied' : 'Copy Steps'}
                    </button>
                  </div>
                </div>
              </div>
            )}

            {/* Tab 3: Diagnostics & Commands */}
            {serverSettingsTab === 'diagnostics' && (
              <div className="space-y-3.5 pt-1">
                <p className="text-zinc-600">
                  Useful systemd and debugging commands you can run via SSH on this VPS:
                </p>

                <div className="space-y-2">
                  <div>
                    <span className="text-[11px] font-semibold text-zinc-700 block mb-1">Check Live Agent Service Status:</span>
                    <pre className="p-2.5 bg-zinc-950 text-zinc-200 rounded-lg font-mono text-[11px] select-all">
                      sudo systemctl status depo-agent
                    </pre>
                  </div>

                  <div>
                    <span className="text-[11px] font-semibold text-zinc-700 block mb-1">Stream Live Agent Output Logs:</span>
                    <pre className="p-2.5 bg-zinc-950 text-zinc-200 rounded-lg font-mono text-[11px] select-all">
                      journalctl -u depo-agent -f -n 100
                    </pre>
                  </div>

                  <div>
                    <span className="text-[11px] font-semibold text-zinc-700 block mb-1">Restart Agent Daemon:</span>
                    <pre className="p-2.5 bg-zinc-950 text-zinc-200 rounded-lg font-mono text-[11px] select-all">
                      sudo systemctl restart depo-agent
                    </pre>
                  </div>

                  <div>
                    <span className="text-[11px] font-semibold text-zinc-700 block mb-1">Validate Caddy Reverse Proxy Config:</span>
                    <pre className="p-2.5 bg-zinc-950 text-zinc-200 rounded-lg font-mono text-[11px] select-all">
                      caddy validate --config /etc/caddy/Caddyfile
                    </pre>
                  </div>
                </div>
              </div>
            )}

            <div className="flex justify-end pt-3 border-t border-zinc-100">
              <button
                onClick={() => {
                  setSelectedServer(null);
                  setReinstallData(null);
                }}
                className="btn-vercel-primary text-xs"
              >
                Close
              </button>
            </div>
          </div>
        </Modal>
      )}

      {/* Global Agent & VPS Setup Guide Modal */}
      <Modal
        isOpen={showGuideModal}
        onClose={() => setShowGuideModal(false)}
        title="DEPO Agent & VPS Reference Guide"
        maxWidth="max-w-2xl"
      >
        <div className="space-y-5 text-xs">
          <div className="p-3 bg-zinc-50 border border-zinc-200 rounded-xl space-y-1 text-zinc-600">
            <span className="font-semibold text-zinc-900 block">Outbound-Only Architecture</span>
            <p>
              DEPO edge agents run on your VPS instances and communicate outbound via long-polling HTTPS requests to this central control plane. No incoming ports, firewalls, or SSH keys need to be opened to DEPO.
            </p>
          </div>

          {/* Section: Agent Install Command */}
          <div className="space-y-2">
            <span className="font-semibold text-zinc-900 text-sm block">1. Installing the Agent on a VPS</span>
            <p className="text-zinc-600">
              To add a server, click <strong>Enroll New Server</strong> in the top-right to generate an installation command with a secure single-use enrollment token:
            </p>
            <pre className="p-3 bg-zinc-950 text-zinc-200 rounded-xl font-mono text-[11px] select-all overflow-x-auto whitespace-pre-wrap break-all">
              curl -sSL {currentOrigin}/install-agent.sh | sudo bash -s -- --server {currentOrigin} --token &lt;TOKEN&gt;
            </pre>
          </div>

          {/* Section: Agent Uninstall Command */}
          <div className="space-y-2">
            <span className="font-semibold text-zinc-900 text-sm block">2. Uninstalling the Agent from a VPS</span>
            <p className="text-zinc-600">
              To fully remove the agent daemon, systemd unit, and credentials from any connected VPS:
            </p>
            <div className="relative">
              <pre className="p-3 bg-zinc-950 text-zinc-200 rounded-xl font-mono text-[11px] select-all overflow-x-auto whitespace-pre-wrap break-all">
                curl -sSL {currentOrigin}/uninstall-agent.sh | sudo bash
              </pre>
              <button
                onClick={() => copyToClipboard(`curl -sSL ${currentOrigin}/uninstall-agent.sh | sudo bash`, 'guide_uninstall_agent')}
                className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
              >
                {copiedKey === 'guide_uninstall_agent' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                {copiedKey === 'guide_uninstall_agent' ? 'Copied' : 'Copy'}
              </button>
            </div>
          </div>

          {/* Section: Central Server Uninstall Command */}
          <div className="space-y-2">
            <span className="font-semibold text-zinc-900 text-sm block">3. Uninstalling the Central Control Plane</span>
            <p className="text-zinc-600">
              If you ever need to remove the DEPO central server itself from its host VPS:
            </p>
            <div className="relative">
              <pre className="p-3 bg-zinc-950 text-zinc-200 rounded-xl font-mono text-[11px] select-all overflow-x-auto whitespace-pre-wrap break-all">
                curl -sSL {currentOrigin}/uninstall.sh | sudo bash
              </pre>
              <button
                onClick={() => copyToClipboard(`curl -sSL ${currentOrigin}/uninstall.sh | sudo bash`, 'guide_uninstall_server')}
                className="absolute top-2 right-2 btn-vercel-secondary py-1 px-2 text-[10px]"
              >
                {copiedKey === 'guide_uninstall_server' ? <Check size={12} className="text-emerald-500" /> : <Copy size={12} />}
                {copiedKey === 'guide_uninstall_server' ? 'Copied' : 'Copy'}
              </button>
            </div>
            <p className="text-[11px] text-zinc-500">
              To also delete all SQLite databases and encryption keys during uninstallation, pass <code className="font-mono bg-zinc-100 px-1 py-0.5 rounded">--purge-data</code>.
            </p>
          </div>

          <div className="flex justify-end pt-2 border-t border-zinc-100">
            <button
              onClick={() => setShowGuideModal(false)}
              className="btn-vercel-primary text-xs"
            >
              Close
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

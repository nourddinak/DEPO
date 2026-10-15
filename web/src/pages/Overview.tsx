import React from 'react';
import { Project, Server, Deployment, Preview } from '../types';
import { StatusBadge } from '../components/StatusBadge';
import {
  FolderGit2,
  Server as ServerIcon,
  Rocket,
  Sparkles,
  HardDrive,
  Cpu,
  Activity,
  ArrowRight,
  ExternalLink,
  Plus,
  GitBranch,
} from 'lucide-react';

interface OverviewProps {
  projects: Project[];
  servers: Server[];
  deployments: Deployment[];
  previews: Preview[];
  onNavigate: (tab: string) => void;
}

export const Overview: React.FC<OverviewProps> = ({ projects = [], servers = [], deployments = [], previews = [], onNavigate }) => {
  const safeProjects = projects || [];
  const safeServers = servers || [];
  const safeDeployments = deployments || [];
  const safePreviews = previews || [];

  const onlineServers = safeServers.filter((s) => s.status === 'online').length;
  const recentDeployments = safeDeployments.slice(0, 6);

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 GB';
    const gb = bytes / (1024 * 1024 * 1024);
    return gb.toFixed(1) + ' GB';
  };

  return (
    <div className="space-y-8 animate-fade-in text-zinc-900">
      {/* Header & Quick Action Buttons */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Platform Overview</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Realtime static frontend deployment metrics and VPS fleet telemetry
          </p>
        </div>
        <div className="flex items-center gap-2.5">
          <button onClick={() => onNavigate('servers')} className="btn-vercel-secondary text-xs">
            <ServerIcon size={14} className="text-zinc-500" /> Enroll VPS
          </button>
          <button onClick={() => onNavigate('projects')} className="btn-vercel-primary text-xs">
            <Plus size={14} /> New Project
          </button>
        </div>
      </div>

      {/* Top 4 Stat Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div
          onClick={() => onNavigate('projects')}
          className="vercel-card vercel-card-hover p-5 cursor-pointer flex flex-col justify-between"
        >
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">Total Projects</span>
            <div className="p-1.5 rounded-md bg-zinc-100 text-zinc-700">
              <FolderGit2 size={16} />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-3xl font-bold tracking-tight text-zinc-900">{safeProjects.length}</span>
            <span className="text-xs text-zinc-400 font-mono">managed frontends</span>
          </div>
        </div>

        <div
          onClick={() => onNavigate('servers')}
          className="vercel-card vercel-card-hover p-5 cursor-pointer flex flex-col justify-between"
        >
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">VPS Fleet</span>
            <div className="p-1.5 rounded-md bg-emerald-50 text-emerald-700">
              <ServerIcon size={16} />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <div className="flex items-baseline gap-1.5">
              <span className="text-3xl font-bold tracking-tight text-zinc-900">{onlineServers}</span>
              <span className="text-xs text-zinc-400">/ {safeServers.length}</span>
            </div>
            <span className="text-xs text-emerald-600 font-medium flex items-center gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
              {safeServers.length > 0 && onlineServers === safeServers.length ? 'Healthy' : 'Active'}
            </span>
          </div>
        </div>

        <div
          onClick={() => onNavigate('deployments')}
          className="vercel-card vercel-card-hover p-5 cursor-pointer flex flex-col justify-between"
        >
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">Total Deployments</span>
            <div className="p-1.5 rounded-md bg-zinc-100 text-zinc-700">
              <Rocket size={16} />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-3xl font-bold tracking-tight text-zinc-900">{safeDeployments.length}</span>
            <span className="text-xs text-zinc-400 font-mono">releases delivered</span>
          </div>
        </div>

        <div
          onClick={() => onNavigate('previews')}
          className="vercel-card vercel-card-hover p-5 cursor-pointer flex flex-col justify-between"
        >
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">PR Previews</span>
            <div className="p-1.5 rounded-md bg-blue-50 text-blue-700">
              <Sparkles size={16} />
            </div>
          </div>
          <div className="mt-4 flex items-baseline justify-between">
            <span className="text-3xl font-bold tracking-tight text-zinc-900">
              {safePreviews.filter((p) => p.status === 'active').length}
            </span>
            <span className="text-xs text-zinc-400 font-mono">active subdomains</span>
          </div>
        </div>
      </div>

      {/* VPS Fleet Host Telemetry */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity size={16} className="text-zinc-700" />
            <h2 className="text-sm font-bold text-zinc-900 tracking-tight">VPS Fleet Telemetry</h2>
          </div>
          <button
            onClick={() => onNavigate('servers')}
            className="text-xs text-zinc-500 hover:text-black flex items-center gap-1 font-medium transition"
          >
            Manage Fleet <ArrowRight size={12} />
          </button>
        </div>

        {safeServers.length === 0 ? (
          <div className="vercel-card p-10 text-center">
            <div className="w-10 h-10 rounded-full bg-zinc-100 text-zinc-400 mx-auto flex items-center justify-center mb-3">
              <ServerIcon size={20} />
            </div>
            <h3 className="text-sm font-semibold text-zinc-900">No VPS servers enrolled yet</h3>
            <p className="text-xs text-zinc-500 mt-1 max-w-sm mx-auto">
              Install the lightweight Go agent on any Ubuntu VPS to start deploying static web projects.
            </p>
            <button onClick={() => onNavigate('servers')} className="btn-vercel-primary text-xs mt-4">
              Enroll First Server
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {safeServers.map((s) => (
              <div key={s.id} className="vercel-card p-5 space-y-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-semibold text-zinc-900 text-xs">{s.name}</h3>
                    <span className="text-[11px] font-mono text-zinc-400 block mt-0.5">
                      {s.host || 'Outbound Long-Poll Worker'}
                    </span>
                  </div>
                  <StatusBadge status={s.status} size="sm" />
                </div>

                <div className="space-y-2.5 pt-3 border-t border-zinc-100 text-xs">
                  <div>
                    <div className="flex justify-between text-zinc-500 text-[11px] mb-1">
                      <span className="flex items-center gap-1">
                        <Cpu size={12} /> CPU Usage
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
                    <div className="flex justify-between text-zinc-500 text-[11px] mb-1">
                      <span className="flex items-center gap-1">
                        <Activity size={12} /> Memory Usage
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
                    <div className="flex justify-between text-zinc-500 text-[11px] mb-1">
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
            ))}
          </div>
        )}
      </div>

      {/* Recent Deployments Table */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Rocket size={16} className="text-zinc-700" />
            <h2 className="text-sm font-bold text-zinc-900 tracking-tight">Recent Deployments</h2>
          </div>
          <button
            onClick={() => onNavigate('deployments')}
            className="text-xs text-zinc-500 hover:text-black flex items-center gap-1 font-medium transition"
          >
            All Deployments <ArrowRight size={12} />
          </button>
        </div>

        <div className="vercel-card overflow-hidden">
          {recentDeployments.length === 0 ? (
            <div className="p-8 text-center text-zinc-400 text-xs">No deployments executed yet.</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                  <tr>
                    <th className="px-5 py-3 text-[11px]">Project</th>
                    <th className="px-5 py-3 text-[11px]">Commit / Branch</th>
                    <th className="px-5 py-3 text-[11px]">Target Server</th>
                    <th className="px-5 py-3 text-[11px]">Status</th>
                    <th className="px-5 py-3 text-[11px] text-right">Started</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-100">
                  {recentDeployments.map((dep) => (
                    <tr key={dep.id} className="hover:bg-zinc-50/80 transition">
                      <td className="px-5 py-3.5 font-semibold text-zinc-900">{dep.project_name || dep.project_id}</td>
                      <td className="px-5 py-3.5">
                        <div className="flex items-center gap-1.5 font-mono text-[11px] text-zinc-800">
                          <GitBranch size={12} className="text-zinc-400" />
                          <span>{dep.commit_hash.slice(0, 7)}</span>
                        </div>
                        <div className="text-zinc-400 text-[11px] truncate max-w-xs">{dep.commit_msg || dep.branch}</div>
                      </td>
                      <td className="px-5 py-3.5 text-zinc-600 font-mono">{dep.server_name || dep.server_id}</td>
                      <td className="px-5 py-3.5">
                        <StatusBadge status={dep.status} size="sm" />
                      </td>
                      <td className="px-5 py-3.5 text-zinc-500 font-mono text-right">
                        {new Date(dep.started_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

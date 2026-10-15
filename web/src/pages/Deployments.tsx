import React, { useState, useEffect } from 'react';
import { Deployment, DeploymentLog, Project } from '../types';
import { api } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { TerminalLogs } from '../components/TerminalLogs';
import { Modal } from '../components/Modal';
import { Rocket, Terminal, Server as ServerIcon, RefreshCw, GitBranch } from 'lucide-react';

interface DeploymentsProps {
  projects: Project[];
}

export const Deployments: React.FC<DeploymentsProps> = ({ projects = [] }) => {
  const safeProjects = projects || [];
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [selectedProject, setSelectedProject] = useState<string>('');
  const [selectedDep, setSelectedDep] = useState<Deployment | null>(null);
  const [depLogs, setDepLogs] = useState<DeploymentLog[]>([]);
  const [loading, setLoading] = useState(true);

  const loadDeployments = async () => {
    try {
      const res = await api.getDeployments(selectedProject || undefined);
      setDeployments(res || []);
    } catch (err) {
      console.error('Failed to load deployments', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDeployments();
    const interval = setInterval(loadDeployments, 4000);
    return () => clearInterval(interval);
  }, [selectedProject]);

  const openLogs = async (dep: Deployment) => {
    setSelectedDep(dep);
    try {
      const logs = await api.getDeploymentLogs(dep.id);
      setDepLogs(logs || []);
    } catch (err) {
      console.error('Failed to load logs', err);
    }
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Deployments</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Live pipeline execution history, release statuses, and terminal delivery logs
          </p>
        </div>
        <div className="flex items-center gap-2.5">
          <select
            value={selectedProject}
            onChange={(e) => setSelectedProject(e.target.value)}
            className="vercel-input text-xs max-w-xs"
          >
            <option value="">All Projects</option>
            {safeProjects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <button onClick={loadDeployments} className="btn-vercel-secondary text-xs p-2.5" title="Refresh list">
            <RefreshCw size={14} />
          </button>
        </div>
      </div>

      <div className="vercel-card overflow-hidden">
        {loading && (deployments || []).length === 0 ? (
          <div className="p-12 text-center text-zinc-400 text-xs">Loading deployment history...</div>
        ) : (deployments || []).length === 0 ? (
          <div className="p-12 text-center text-zinc-500 text-sm">
            <Rocket size={32} className="mx-auto text-zinc-300 mb-2" />
            No deployments recorded yet. Push commits or trigger a build to deploy.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Project</th>
                  <th className="px-5 py-3 text-[11px]">Commit / Branch</th>
                  <th className="px-5 py-3 text-[11px]">Target VPS</th>
                  <th className="px-5 py-3 text-[11px]">Status</th>
                  <th className="px-5 py-3 text-[11px]">Started</th>
                  <th className="px-5 py-3 text-[11px]">Trigger</th>
                  <th className="px-5 py-3 text-[11px] text-right">Logs</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {(deployments || []).map((dep) => (
                  <tr key={dep.id} className="hover:bg-zinc-50/80 transition">
                    <td className="px-5 py-3.5 font-semibold text-zinc-900">{dep.project_name || dep.project_id}</td>
                    <td className="px-5 py-3.5">
                      <div className="flex items-center gap-1.5 font-mono text-[11px] font-semibold text-zinc-900">
                        <GitBranch size={12} className="text-zinc-400" />
                        {dep.commit_hash.slice(0, 7)}
                      </div>
                      <div className="text-zinc-400 text-[11px] truncate max-w-xs">{dep.commit_msg || dep.branch}</div>
                    </td>
                    <td className="px-5 py-3.5 text-zinc-600 font-mono flex items-center gap-1.5">
                      <ServerIcon size={12} className="text-zinc-400" />
                      {dep.server_name || dep.server_id}
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={dep.status} size="sm" />
                    </td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono">
                      {new Date(dep.started_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}
                    </td>
                    <td className="px-5 py-3.5 text-zinc-600 capitalize">
                      {dep.trigger.replace('_', ' ')}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <button
                        onClick={() => openLogs(dep)}
                        className="btn-vercel-secondary py-1 px-3 text-[11px] flex items-center gap-1.5 ml-auto"
                      >
                        <Terminal size={12} /> Inspect
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Terminal Logs Modal */}
      <Modal
        isOpen={!!selectedDep}
        onClose={() => setSelectedDep(null)}
        title={`Deployment Execution: ${selectedDep?.commit_hash.slice(0, 7)}`}
        maxWidth="max-w-4xl"
      >
        {selectedDep && (
          <div className="space-y-4">
            <div className="flex items-center justify-between text-xs font-mono p-3 bg-zinc-50 rounded-lg border border-zinc-200">
              <div>
                <span className="text-zinc-400">Project:</span> <span className="font-semibold">{selectedDep.project_name}</span>
              </div>
              <div>
                <span className="text-zinc-400">Status:</span> <StatusBadge status={selectedDep.status} size="sm" />
              </div>
              <div>
                <span className="text-zinc-400">Trigger:</span> <span className="capitalize">{selectedDep.trigger}</span>
              </div>
            </div>
            <TerminalLogs logs={depLogs} height="h-96" />
          </div>
        )}
      </Modal>
    </div>
  );
};

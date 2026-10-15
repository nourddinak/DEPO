import React, { useState, useEffect } from 'react';
import { Project, Deployment, Release, DeploymentLog } from '../types';
import { api } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { TerminalLogs } from '../components/TerminalLogs';
import { Modal } from '../components/Modal';
import {
  ArrowLeft,
  ExternalLink,
  GitBranch,
  RotateCcw,
  Terminal,
  FileCode2,
  Check,
  Copy,
} from 'lucide-react';

interface ProjectDetailProps {
  projectId: string;
  onBack: () => void;
  onNavigateEnv: (projectId: string) => void;
}

export const ProjectDetail: React.FC<ProjectDetailProps> = ({ projectId, onBack, onNavigateEnv }) => {
  const [project, setProject] = useState<Project | null>(null);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [releases, setReleases] = useState<Release[]>([]);
  const [selectedDep, setSelectedDep] = useState<Deployment | null>(null);
  const [depLogs, setDepLogs] = useState<DeploymentLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [rollingBack, setRollingBack] = useState(false);
  const [workflowModal, setWorkflowModal] = useState(false);
  const [workflowYaml, setWorkflowYaml] = useState('');
  const [copiedYaml, setCopiedYaml] = useState(false);

  const loadData = async () => {
    try {
      const [proj, deps, rels] = await Promise.all([
        api.getProject(projectId),
        api.getDeployments(projectId),
        api.getProjectReleases(projectId),
      ]);
      setProject(proj);
      setDeployments(deps || []);
      setReleases(rels || []);
    } catch (err) {
      console.error('Failed to load project details', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, [projectId]);

  const viewDeploymentLogs = async (dep: Deployment) => {
    setSelectedDep(dep);
    try {
      const logs = await api.getDeploymentLogs(dep.id);
      setDepLogs(logs || []);
    } catch (err) {
      console.error('Failed to load logs', err);
    }
  };

  const handleRollback = async (releaseId: string) => {
    if (!confirm(`Are you sure you want to rollback ${project?.name} to release ${releaseId.slice(0, 8)}?`)) {
      return;
    }
    setRollingBack(true);
    try {
      await api.rollbackRelease(projectId, releaseId);
      alert('Rollback queued! Check deployment progress.');
      loadData();
    } catch (err: any) {
      alert('Rollback failed: ' + err.message);
    } finally {
      setRollingBack(false);
    }
  };

  const showWorkflow = async () => {
    try {
      const res = await api.getProjectWorkflow(projectId);
      setWorkflowYaml(res.workflow_yaml);
      setWorkflowModal(true);
    } catch (err: any) {
      alert('Failed to get workflow: ' + err.message);
    }
  };

  if (loading || !project) {
    return <div className="p-8 text-center text-zinc-400">Loading project...</div>;
  }

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      {/* Top Breadcrumb & Actions */}
      <div className="flex items-center justify-between">
        <button onClick={onBack} className="text-xs text-zinc-500 hover:text-black flex items-center gap-1.5 transition font-medium">
          <ArrowLeft size={14} /> Back to Projects
        </button>

        <div className="flex items-center gap-2.5">
          <button onClick={() => onNavigateEnv(project.id)} className="btn-vercel-secondary text-xs">
            Environment Variables
          </button>
          <button onClick={showWorkflow} className="btn-vercel-secondary text-xs">
            <FileCode2 size={14} /> View GitHub Action
          </button>
        </div>
      </div>

      {/* Project Banner Card */}
      <div className="vercel-card p-6 space-y-4">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold tracking-tight text-zinc-900">{project.name}</h1>
              {project.active_release_id ? (
                <span className="px-2.5 py-0.5 rounded-full text-xs font-mono font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                  Active: {project.active_release_id.slice(0, 8)}
                </span>
              ) : (
                <span className="px-2.5 py-0.5 rounded-full text-xs font-mono bg-zinc-100 text-zinc-500">
                  No Active Release
                </span>
              )}
            </div>
            <a
              href={`https://${project.domain}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1.5 text-xs font-mono text-blue-600 hover:underline mt-1"
            >
              https://{project.domain} <ExternalLink size={12} />
            </a>
          </div>

          <div className="flex items-center gap-4 text-xs font-mono text-zinc-600 bg-zinc-50 p-3 rounded-lg border border-zinc-200">
            <div>
              <span className="text-zinc-400 block text-[10px] uppercase">Target VPS</span>
              <span className="font-semibold text-zinc-900">{project.server_name || project.server_id}</span>
            </div>
            <div className="border-l border-zinc-200 pl-4">
              <span className="text-zinc-400 block text-[10px] uppercase">Branch</span>
              <span className="font-semibold text-zinc-900 flex items-center gap-1">
                <GitBranch size={12} className="text-zinc-500" /> {project.branch}
              </span>
            </div>
            <div className="border-l border-zinc-200 pl-4">
              <span className="text-zinc-400 block text-[10px] uppercase">Auto-Deploy</span>
              <span className={project.auto_deploy ? 'text-emerald-700 font-medium' : 'text-zinc-400'}>
                {project.auto_deploy ? 'Enabled' : 'Disabled'}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Releases & Rollback Table */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
          <RotateCcw size={16} className="text-zinc-700" /> Release History & Atomic Rollback
        </h2>
        <div className="vercel-card overflow-hidden">
          {(releases || []).length === 0 ? (
            <div className="p-6 text-center text-zinc-400 text-xs">No releases recorded yet</div>
          ) : (
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Release ID</th>
                  <th className="px-5 py-3 text-[11px]">Commit</th>
                  <th className="px-5 py-3 text-[11px]">SHA-256 Checksum</th>
                  <th className="px-5 py-3 text-[11px]">Deployed At</th>
                  <th className="px-5 py-3 text-[11px] text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {(releases || []).map((rel) => (
                  <tr key={rel.id} className={`hover:bg-zinc-50/80 transition ${rel.is_active ? 'bg-zinc-50/60' : ''}`}>
                    <td className="px-5 py-3.5 font-mono text-zinc-900 font-medium flex items-center gap-2">
                      {rel.id.slice(0, 8)}
                      {rel.is_active && (
                        <span className="px-2 py-0.2 rounded text-[10px] font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                          CURRENT
                        </span>
                      )}
                    </td>
                    <td className="px-5 py-3.5 font-mono text-zinc-700 font-semibold">{rel.commit_hash.slice(0, 7)}</td>
                    <td className="px-5 py-3.5 font-mono text-zinc-500 text-[11px] truncate max-w-xs">
                      {rel.artifact_checksum.slice(0, 16)}...
                    </td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono">
                      {new Date(rel.deployed_at).toLocaleString()}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      {!rel.is_active && (
                        <button
                          disabled={rollingBack}
                          onClick={() => handleRollback(rel.id)}
                          className="btn-vercel-secondary py-1 px-3 text-[11px]"
                        >
                          <RotateCcw size={11} /> Rollback
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Deployments History & Logs Section */}
      <div className="space-y-3">
        <h2 className="text-sm font-bold text-zinc-900">Deployments History</h2>
        <div className="vercel-card overflow-hidden">
          {(deployments || []).length === 0 ? (
            <div className="p-8 text-center text-zinc-400 text-xs">No deployments have run for this project yet</div>
          ) : (
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Commit / Trigger</th>
                  <th className="px-5 py-3 text-[11px]">Status</th>
                  <th className="px-5 py-3 text-[11px]">Started</th>
                  <th className="px-5 py-3 text-[11px]">Completed</th>
                  <th className="px-5 py-3 text-[11px] text-right">Logs</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {(deployments || []).map((dep) => (
                  <tr key={dep.id} className="hover:bg-zinc-50/80 transition">
                    <td className="px-5 py-3.5">
                      <div className="font-mono text-zinc-900 font-semibold">{dep.commit_hash.slice(0, 7)}</div>
                      <div className="text-zinc-500 text-[11px] truncate max-w-sm">{dep.commit_msg || dep.trigger}</div>
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={dep.status} size="sm" />
                    </td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono">
                      {new Date(dep.started_at).toLocaleTimeString()}
                    </td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono">
                      {dep.completed_at ? new Date(dep.completed_at).toLocaleTimeString() : 'In Progress'}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <button
                        onClick={() => viewDeploymentLogs(dep)}
                        className="btn-vercel-secondary py-1 px-3 text-[11px] flex items-center gap-1.5 ml-auto"
                      >
                        <Terminal size={12} /> View Logs
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Terminal Logs Modal */}
      <Modal
        isOpen={!!selectedDep}
        onClose={() => setSelectedDep(null)}
        title={`Deployment Logs: ${selectedDep?.commit_hash.slice(0, 7)}`}
        maxWidth="max-w-4xl"
      >
        {selectedDep && (
          <div className="space-y-4">
            <div className="flex items-center justify-between text-xs font-mono p-3 bg-zinc-50 rounded-lg border border-zinc-200">
              <div>
                <span className="text-zinc-400">Trigger:</span> <span className="font-semibold">{selectedDep.trigger}</span>
              </div>
              <div>
                <span className="text-zinc-400">Status:</span> <StatusBadge status={selectedDep.status} size="sm" />
              </div>
              <div>
                <span className="text-zinc-400">Started:</span> {new Date(selectedDep.started_at).toLocaleString()}
              </div>
            </div>
            <TerminalLogs logs={depLogs} height="h-96" />
          </div>
        )}
      </Modal>

      {/* Workflow YAML Modal */}
      <Modal
        isOpen={workflowModal}
        onClose={() => setWorkflowModal(false)}
        title="GitHub Actions CI/CD Workflow"
        maxWidth="max-w-3xl"
      >
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-xs text-zinc-500">
              Place in <code className="text-black font-semibold font-mono">.github/workflows/depo-deploy.yml</code>
            </span>
            <button
              onClick={() => {
                navigator.clipboard.writeText(workflowYaml);
                setCopiedYaml(true);
                setTimeout(() => setCopiedYaml(false), 2000);
              }}
              className="btn-vercel-primary py-1 px-3 text-xs"
            >
              {copiedYaml ? <Check size={12} /> : <Copy size={12} />}
              {copiedYaml ? 'Copied' : 'Copy YAML'}
            </button>
          </div>
          <pre className="p-4 bg-zinc-950 border border-zinc-800 rounded-xl font-mono text-[11px] text-zinc-200 overflow-x-auto max-h-96 select-all">
            {workflowYaml}
          </pre>
        </div>
      </Modal>
    </div>
  );
};

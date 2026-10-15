import React, { useState } from 'react';
import { Project, Server } from '../types';
import { ProjectWizard } from './ProjectWizard';
import { Modal } from '../components/Modal';
import { StatusBadge } from '../components/StatusBadge';
import {
  FolderGit2,
  Plus,
  ExternalLink,
  GitBranch,
  Server as ServerIcon,
  Copy,
  Check,
  Trash2,
  FileCode2,
  ShieldAlert,
} from 'lucide-react';
import { api } from '../api/client';

interface ProjectsProps {
  projects: Project[];
  servers: Server[];
  onRefresh: () => void;
  onSelectProject: (id: string) => void;
}

export const Projects: React.FC<ProjectsProps> = ({ projects = [], servers = [], onRefresh, onSelectProject }) => {
  const safeProjects = projects || [];
  const safeServers = servers || [];
  const [showWizard, setShowWizard] = useState(false);
  const [createdResult, setCreatedResult] = useState<{
    project: Project;
    deployToken: string;
    workflowYaml: string;
  } | null>(null);
  const [copiedToken, setCopiedToken] = useState(false);
  const [copiedYaml, setCopiedYaml] = useState(false);
  const [search, setSearch] = useState('');

  const filtered = safeProjects.filter(
    (p) =>
      p.name.toLowerCase().includes(search.toLowerCase()) ||
      p.domain.toLowerCase().includes(search.toLowerCase())
  );

  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`Are you sure you want to delete project '${name}'? This removes its configuration and deployments.`)) {
      return;
    }
    try {
      await api.deleteProject(id);
      onRefresh();
    } catch (err: any) {
      alert('Failed to delete project: ' + err.message);
    }
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Projects</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Managed static frontend repositories, target VPS assignments, and custom domains
          </p>
        </div>
        <button
          onClick={() => setShowWizard(true)}
          disabled={safeServers.length === 0}
          className="btn-vercel-primary text-xs"
          title={safeServers.length === 0 ? 'Please enroll a VPS server first' : 'Create new project'}
        >
          <Plus size={14} /> New Project
        </button>
      </div>

      {/* Filter / Search Bar */}
      <div className="flex items-center gap-3">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Filter projects by name or domain..."
          className="vercel-input max-w-sm text-xs"
        />
        <span className="text-xs text-zinc-400 font-mono">
          {filtered.length} of {safeProjects.length} projects
        </span>
      </div>

      {/* Projects Grid */}
      {filtered.length === 0 ? (
        <div className="vercel-card p-12 text-center">
          <div className="w-12 h-12 rounded-full bg-zinc-100 text-zinc-400 mx-auto flex items-center justify-center mb-3">
            <FolderGit2 size={24} />
          </div>
          <h3 className="text-sm font-semibold text-zinc-900">No projects found</h3>
          <p className="text-xs text-zinc-500 mt-1 max-w-sm mx-auto">
            {safeServers.length === 0
              ? 'You must connect at least one VPS before deploying a static frontend project.'
              : 'Add your first frontend repository to generate automated GitHub Actions CI/CD workflows.'}
          </p>
          {safeServers.length > 0 && (
            <button onClick={() => setShowWizard(true)} className="btn-vercel-primary text-xs mt-4">
              <Plus size={14} /> Create First Project
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {filtered.map((p) => (
            <div key={p.id} className="vercel-card vercel-card-hover p-5 space-y-4 flex flex-col justify-between">
              <div className="space-y-3">
                <div className="flex items-start justify-between">
                  <div>
                    <h3
                      onClick={() => onSelectProject(p.id)}
                      className="font-bold text-zinc-900 text-sm hover:underline cursor-pointer transition flex items-center gap-1.5"
                    >
                      {p.name}
                    </h3>
                    <a
                      href={`https://${p.domain}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-xs text-blue-600 hover:underline font-mono flex items-center gap-1 mt-0.5"
                    >
                      {p.domain} <ExternalLink size={10} />
                    </a>
                  </div>
                  {p.active_release_id ? (
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                      rel-{p.active_release_id.slice(0, 6)}
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-zinc-100 text-zinc-500 border border-zinc-200">
                      no release
                    </span>
                  )}
                </div>

                <div className="space-y-1.5 text-xs text-zinc-500 font-mono pt-3 border-t border-zinc-100">
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-1 text-zinc-400">
                      <GitBranch size={12} /> Branch:
                    </span>
                    <span className="text-zinc-700 font-medium">{p.branch}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="flex items-center gap-1 text-zinc-400">
                      <ServerIcon size={12} /> Target VPS:
                    </span>
                    <span className="text-zinc-700 font-medium">{p.server_name || p.server_id}</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-zinc-400">Build Output:</span>
                    <span className="text-zinc-700">{p.output_dir}</span>
                  </div>
                </div>
              </div>

              <div className="flex items-center justify-between pt-3 border-t border-zinc-100">
                <button
                  onClick={() => onSelectProject(p.id)}
                  className="btn-vercel-secondary text-xs py-1.5 px-3"
                >
                  Manage & Deployments
                </button>
                <button
                  onClick={() => handleDelete(p.id, p.name)}
                  className="p-1.5 text-zinc-400 hover:text-rose-600 rounded transition"
                  title="Delete project"
                >
                  <Trash2 size={15} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Creation Wizard Modal */}
      <Modal isOpen={showWizard} onClose={() => setShowWizard(false)} title="Create New Project" maxWidth="max-w-2xl">
        <ProjectWizard
          servers={safeServers}
          onCreated={(project, deployToken, workflowYaml) => {
            setShowWizard(false);
            setCreatedResult({ project, deployToken, workflowYaml });
            onRefresh();
          }}
          onCancel={() => setShowWizard(false)}
        />
      </Modal>

      {/* Post-Creation Success Modal with Credentials & CI Workflow */}
      <Modal
        isOpen={!!createdResult}
        onClose={() => setCreatedResult(null)}
        title="Project Created - Setup GitHub Actions"
        maxWidth="max-w-3xl"
      >
        {createdResult && (
          <div className="space-y-5 text-xs leading-relaxed">
            <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800">
              <span className="font-bold text-sm block mb-1">Project '{createdResult.project.name}' initialized!</span>
              Add the GitHub Actions workflow below to your repository to begin automated static builds and deployments.
            </div>

            {/* Deploy Token Notice */}
            <div className="p-4 rounded-xl bg-zinc-50 border border-zinc-200 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-zinc-700 uppercase tracking-wider text-[11px] flex items-center gap-1.5">
                  <ShieldAlert size={14} className="text-amber-600" /> Scoped Deploy Token (Optional Secret)
                </span>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(createdResult.deployToken);
                    setCopiedToken(true);
                    setTimeout(() => setCopiedToken(false), 2000);
                  }}
                  className="btn-vercel-secondary py-1 px-2.5 text-[11px]"
                >
                  {copiedToken ? <Check size={12} className="text-emerald-600" /> : <Copy size={12} />}
                  {copiedToken ? 'Copied' : 'Copy Token'}
                </button>
              </div>
              <input
                type="text"
                readOnly
                value={createdResult.deployToken}
                className="vercel-input font-mono text-xs bg-white text-zinc-800 select-all"
              />
              <span className="text-[11px] text-zinc-500 block">
                Store as <code className="text-black font-mono font-semibold">DEPO_DEPLOY_TOKEN</code> in your GitHub repository secrets if not using GitHub OIDC.
              </span>
            </div>

            {/* Workflow YAML Code Viewer */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-zinc-700 uppercase tracking-wider text-[11px] flex items-center gap-1.5">
                  <FileCode2 size={14} className="text-black" /> .github/workflows/depo-deploy.yml
                </span>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(createdResult.workflowYaml);
                    setCopiedYaml(true);
                    setTimeout(() => setCopiedYaml(false), 2000);
                  }}
                  className="btn-vercel-primary py-1 px-2.5 text-[11px]"
                >
                  {copiedYaml ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
                  {copiedYaml ? 'Copied Workflow' : 'Copy Workflow YAML'}
                </button>
              </div>
              <pre className="p-4 rounded-xl bg-zinc-950 border border-zinc-800 font-mono text-[11px] text-zinc-200 overflow-x-auto max-h-60 select-all">
                {createdResult.workflowYaml}
              </pre>
            </div>

            <div className="flex justify-end pt-2">
              <button onClick={() => setCreatedResult(null)} className="btn-vercel-primary text-xs">
                I Have Saved My Configuration
              </button>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

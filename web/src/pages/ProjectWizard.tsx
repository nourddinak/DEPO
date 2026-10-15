import React, { useState } from 'react';
import { Server, Project } from '../types';
import { api } from '../api/client';
import { Check, Sparkles, ChevronRight, ChevronLeft } from 'lucide-react';

interface ProjectWizardProps {
  servers: Server[];
  onCreated: (project: Project, deployToken: string, workflowYaml: string) => void;
  onCancel: () => void;
}

export const ProjectWizard: React.FC<ProjectWizardProps> = ({ servers = [], onCreated, onCancel }) => {
  const safeServers = servers || [];
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  // Form State
  const [name, setName] = useState('');
  const [repoURL, setRepoURL] = useState('');
  const [branch, setBranch] = useState('main');
  const [framework, setFramework] = useState('Vite');
  const [frontendDir, setFrontendDir] = useState('frontend/');
  const [buildCmd, setBuildCmd] = useState('npm run build');
  const [outputDir, setOutputDir] = useState('dist/');
  const [serverID, setServerID] = useState(safeServers[0]?.id || '');
  const [domain, setDomain] = useState('');
  const [autoDeploy, setAutoDeploy] = useState(true);
  const [previewEnabled, setPreviewEnabled] = useState(false);
  const [previewBaseDomain, setPreviewBaseDomain] = useState('');
  const [pkgJsonInput, setPkgJsonInput] = useState('');
  const [detecting, setDetecting] = useState(false);

  // Framework Presets
  const applyPreset = (preset: string) => {
    setFramework(preset);
    switch (preset) {
      case 'Vite':
        setBuildCmd('npm run build');
        setOutputDir('dist/');
        break;
      case 'Astro':
        setBuildCmd('npm run build');
        setOutputDir('dist/');
        break;
      case 'Next':
        setBuildCmd('npx next build');
        setOutputDir('out/');
        break;
      case 'Nuxt':
        setBuildCmd('npx nuxt generate');
        setOutputDir('.output/public/');
        break;
      case 'Angular':
        setBuildCmd('npx ng build --configuration production');
        setOutputDir('dist/');
        break;
      case 'HTML':
        setBuildCmd('echo "No build required"');
        setOutputDir('.');
        break;
    }
  };

  const handleAutoDetect = async () => {
    if (!pkgJsonInput.trim()) return;
    setDetecting(true);
    try {
      const res = await api.detectFramework(pkgJsonInput);
      setFramework(res.framework);
      setBuildCmd(res.build_cmd);
      setOutputDir(res.output_dir);
      if (res.frontend_dir) setFrontendDir(res.frontend_dir);
    } catch (err: any) {
      setError('Framework detection failed: ' + err.message);
    } finally {
      setDetecting(false);
    }
  };

  const handleSubmit = async () => {
    setError('');
    setLoading(true);

    try {
      const res = await api.createProject({
        name,
        repo_url: repoURL,
        branch,
        frontend_dir: frontendDir,
        build_cmd: buildCmd,
        output_dir: outputDir,
        server_id: serverID,
        domain,
        auto_deploy: autoDeploy,
        preview_enabled: previewEnabled,
        preview_base_domain: previewBaseDomain,
      });

      onCreated(res.project, res.deploy_token, res.workflow_yaml);
    } catch (err: any) {
      setError(err.message || 'Failed to create project');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6 text-zinc-900">
      {/* Wizard Progress Steps */}
      <div className="flex items-center justify-between border-b border-zinc-100 pb-4">
        {[
          { num: 1, label: 'Repository' },
          { num: 2, label: 'Framework & Build' },
          { num: 3, label: 'Host & Domain' },
        ].map((s) => (
          <div key={s.num} className="flex items-center gap-2">
            <span
              className={`w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold transition-all ${
                step === s.num
                  ? 'bg-black text-white'
                  : step > s.num
                  ? 'bg-zinc-800 text-white'
                  : 'bg-zinc-100 text-zinc-400'
              }`}
            >
              {step > s.num ? <Check size={12} /> : s.num}
            </span>
            <span
              className={`text-xs ${
                step === s.num ? 'font-semibold text-black' : 'text-zinc-400 font-normal'
              }`}
            >
              {s.label}
            </span>
          </div>
        ))}
      </div>

      {error && (
        <div className="p-3 bg-rose-50 border border-rose-200 text-rose-700 text-xs rounded-lg">{error}</div>
      )}

      {/* Step 1: Repository Details */}
      {step === 1 && (
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Project Identifier
            </label>
            <input
              type="text"
              required
              value={name}
              onChange={(e) => setName(e.target.value.toLowerCase().replace(/[^a-z0-9-_]/g, '-'))}
              placeholder="my-react-app"
              className="vercel-input font-mono text-sm"
            />
            <span className="text-[11px] text-zinc-400 mt-1 block">
              Unique slug used for filesystem isolation (/var/www/&lt;project&gt;)
            </span>
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              GitHub Repository URL
            </label>
            <input
              type="url"
              required
              value={repoURL}
              onChange={(e) => setRepoURL(e.target.value)}
              placeholder="https://github.com/my-org/my-frontend"
              className="vercel-input text-sm"
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Primary Production Branch
            </label>
            <input
              type="text"
              required
              value={branch}
              onChange={(e) => setBranch(e.target.value)}
              placeholder="main"
              className="vercel-input font-mono text-sm"
            />
          </div>
        </div>
      )}

      {/* Step 2: Framework & Build Settings */}
      {step === 2 && (
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-2">
              Select Framework Preset
            </label>
            <div className="grid grid-cols-3 gap-2">
              {['Vite', 'Astro', 'Next', 'Nuxt', 'Angular', 'HTML'].map((f) => (
                <button
                  key={f}
                  type="button"
                  onClick={() => applyPreset(f)}
                  className={`p-2.5 rounded-lg text-xs font-semibold border transition text-center ${
                    framework === f
                      ? 'bg-black text-white border-black'
                      : 'bg-white text-zinc-700 border-zinc-200 hover:border-zinc-300 hover:bg-zinc-50'
                  }`}
                >
                  {f}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Frontend Directory in Repo
            </label>
            <input
              type="text"
              value={frontendDir}
              onChange={(e) => setFrontendDir(e.target.value)}
              placeholder="frontend/"
              className="vercel-input font-mono text-xs"
            />
            <span className="text-[11px] text-zinc-400 mt-1 block">Default convention is frontend/</span>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Build Command
              </label>
              <input
                type="text"
                value={buildCmd}
                onChange={(e) => setBuildCmd(e.target.value)}
                placeholder="npm run build"
                className="vercel-input font-mono text-xs"
              />
            </div>
            <div>
              <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
                Static Output Directory
              </label>
              <input
                type="text"
                value={outputDir}
                onChange={(e) => setOutputDir(e.target.value)}
                placeholder="dist/"
                className="vercel-input font-mono text-xs"
              />
            </div>
          </div>

          {/* Optional Package.json auto-detection paste */}
          <div className="pt-2 border-t border-zinc-100">
            <details className="text-xs text-zinc-500">
              <summary className="cursor-pointer hover:text-black font-medium">
                Paste package.json to Auto-Detect Framework
              </summary>
              <div className="mt-2 space-y-2">
                <textarea
                  rows={3}
                  value={pkgJsonInput}
                  onChange={(e) => setPkgJsonInput(e.target.value)}
                  placeholder='Paste {"dependencies": { ... }} here'
                  className="vercel-input font-mono text-xs"
                />
                <button
                  type="button"
                  onClick={handleAutoDetect}
                  disabled={detecting || !pkgJsonInput}
                  className="btn-vercel-secondary text-xs"
                >
                  <Sparkles size={12} /> {detecting ? 'Detecting...' : 'Detect Settings'}
                </button>
              </div>
            </details>
          </div>
        </div>
      )}

      {/* Step 3: Destination Host & Domains */}
      {step === 3 && (
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Destination Ubuntu VPS
            </label>
            <select
              value={serverID}
              onChange={(e) => setServerID(e.target.value)}
              className="vercel-input text-sm"
              required
            >
              {safeServers.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name} ({s.status}) - {s.host || 'Outbound'}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Production Domain (FQDN)
            </label>
            <input
              type="text"
              required
              value={domain}
              onChange={(e) => setDomain(e.target.value)}
              placeholder="app.example.com"
              className="vercel-input font-mono text-sm"
            />
          </div>

          <div className="space-y-3 pt-3 border-t border-zinc-100">
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={autoDeploy}
                onChange={(e) => setAutoDeploy(e.target.checked)}
                className="rounded border-zinc-300 text-black focus:ring-0"
              />
              <span className="text-xs font-medium text-zinc-700">
                Automatic deployment on push to {branch}
              </span>
            </label>

            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={previewEnabled}
                onChange={(e) => setPreviewEnabled(e.target.checked)}
                className="rounded border-zinc-300 text-black focus:ring-0"
              />
              <span className="text-xs font-medium text-zinc-700">
                Enable Pull Request Previews (isolated subdomains)
              </span>
            </label>

            {previewEnabled && (
              <div className="pl-6 animate-fade-in">
                <label className="block text-[11px] font-semibold text-zinc-500 uppercase tracking-wider mb-1">
                  Preview Base Domain (Wildcard DNS)
                </label>
                <input
                  type="text"
                  value={previewBaseDomain}
                  onChange={(e) => setPreviewBaseDomain(e.target.value)}
                  placeholder="preview.example.com"
                  className="vercel-input font-mono text-xs"
                />
              </div>
            )}
          </div>
        </div>
      )}

      {/* Navigation Buttons */}
      <div className="flex items-center justify-between pt-4 border-t border-zinc-100">
        {step > 1 ? (
          <button type="button" onClick={() => setStep(step - 1)} className="btn-vercel-secondary text-xs">
            <ChevronLeft size={14} /> Back
          </button>
        ) : (
          <button type="button" onClick={onCancel} className="btn-vercel-secondary text-xs">
            Cancel
          </button>
        )}

        {step < 3 ? (
          <button
            type="button"
            disabled={step === 1 && (!name || !repoURL)}
            onClick={() => setStep(step + 1)}
            className="btn-vercel-primary text-xs"
          >
            Next <ChevronRight size={14} />
          </button>
        ) : (
          <button
            type="button"
            disabled={loading || !domain || !serverID}
            onClick={handleSubmit}
            className="btn-vercel-primary text-xs"
          >
            {loading ? 'Creating Project...' : 'Finish & Generate CI/CD'}
          </button>
        )}
      </div>
    </div>
  );
};

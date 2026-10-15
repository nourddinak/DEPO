import React, { useState, useEffect } from 'react';
import { Project, EnvVar } from '../types';
import { api } from '../api/client';
import { Modal } from '../components/Modal';
import {
  Save,
  Plus,
  Trash2,
  Eye,
  EyeOff,
  AlertTriangle,
  Sparkles,
  Check,
} from 'lucide-react';

interface EnvironmentVarsProps {
  projects: Project[];
  initialProjectId?: string;
}

export const EnvironmentVars: React.FC<EnvironmentVarsProps> = ({ projects = [], initialProjectId }) => {
  const safeProjects = projects || [];
  const [selectedProjectId, setSelectedProjectId] = useState<string>(initialProjectId || safeProjects[0]?.id || '');
  const [variables, setVariables] = useState<EnvVar[]>([]);
  const [rawContent, setRawContent] = useState('');
  const [viewMode, setViewMode] = useState<'structured' | 'raw'>('structured');
  const [showValues, setShowValues] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [savedMsg, setSavedMsg] = useState('');
  const [showExampleModal, setShowExampleModal] = useState(false);
  const [exampleContent, setExampleContent] = useState('');

  const loadEnv = async (pid: string) => {
    if (!pid) return;
    setLoading(true);
    try {
      const res = await api.getProjectEnv(pid);
      setVariables(res?.variables || []);
      setRawContent(res?.raw_content || '');
    } catch (err) {
      console.error('Failed to load env vars', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (selectedProjectId) {
      loadEnv(selectedProjectId);
    }
  }, [selectedProjectId]);

  const toggleShowValue = (key: string) => {
    setShowValues((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const addVariable = () => {
    setVariables([...variables, { key: '', value: '', is_build_time: true }]);
  };

  const removeVariable = (idx: number) => {
    const updated = [...variables];
    updated.splice(idx, 1);
    setVariables(updated);
  };

  const updateVariable = (idx: number, field: keyof EnvVar, val: any) => {
    const updated = [...variables];
    updated[idx] = { ...updated[idx], [field]: val };
    setVariables(updated);
  };

  const handleSave = async (triggerDeploy = false) => {
    setSaving(true);
    setSavedMsg('');
    try {
      if (viewMode === 'raw') {
        await api.updateProjectEnv(selectedProjectId, {
          raw_content: rawContent,
          trigger_deploy: triggerDeploy,
        });
      } else {
        await api.updateProjectEnv(selectedProjectId, {
          variables: (variables || []).filter((v) => v.key.trim() !== ''),
          trigger_deploy: triggerDeploy,
        });
      }
      setSavedMsg(triggerDeploy ? 'Saved & triggered new deployment!' : 'Environment variables encrypted & saved!');
      setTimeout(() => setSavedMsg(''), 3000);
      loadEnv(selectedProjectId);
    } catch (err: any) {
      alert('Failed to save variables: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleParseExample = async () => {
    try {
      const items = await api.parseEnvExample(exampleContent);
      setVariables((prev) => {
        const map = new Map<string, EnvVar>();
        (prev || []).forEach((v) => map.set(v.key, v));
        (items || []).forEach((item) => {
          if (!map.has(item.key)) {
            map.set(item.key, item);
          }
        });
        return Array.from(map.values());
      });
      setShowExampleModal(false);
      setExampleContent('');
    } catch (err: any) {
      alert('Failed to parse .env.example: ' + err.message);
    }
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Environment Variables</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Encrypted build-time environment configuration for static frontend applications
          </p>
        </div>

        {/* Project Selector */}
        <div className="flex items-center gap-3">
          <select
            value={selectedProjectId}
            onChange={(e) => setSelectedProjectId(e.target.value)}
            className="vercel-input text-xs min-w-[200px]"
          >
            {safeProjects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name} ({p.domain})
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Security Warning Notice */}
      <div className="p-4 rounded-xl bg-amber-50 border border-amber-200 text-amber-900 text-xs flex items-start gap-3">
        <AlertTriangle size={18} className="shrink-0 mt-0.5 text-amber-600" />
        <div className="space-y-1">
          <span className="font-semibold block text-amber-950">Public Static Bundle Notice</span>
          <span className="leading-relaxed">
            Variables embedded into static frontends at build time (e.g. <code className="font-mono font-semibold text-black">VITE_*</code>, <code className="font-mono font-semibold text-black">NEXT_PUBLIC_*</code>) are compiled directly into client-side JavaScript and are publicly inspectable by anyone who loads your website. Never place database passwords, private master keys, or private API secrets here.
          </span>
        </div>
      </div>

      {/* Editor Container */}
      <div className="vercel-card p-6 space-y-6">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-zinc-100 pb-4">
          <div className="flex items-center gap-1.5 bg-zinc-100 p-1 rounded-lg border border-zinc-200">
            <button
              onClick={() => setViewMode('structured')}
              className={`px-3 py-1 rounded-md text-xs font-medium transition ${
                viewMode === 'structured'
                  ? 'bg-white text-black shadow-xs font-semibold'
                  : 'text-zinc-600 hover:text-black'
              }`}
            >
              Structured Form
            </button>
            <button
              onClick={() => {
                if (viewMode === 'structured') {
                  const lines = variables.map((v) => `${v.key}=${v.value}`).join('\n');
                  setRawContent(lines);
                }
                setViewMode('raw');
              }}
              className={`px-3 py-1 rounded-md text-xs font-medium transition ${
                viewMode === 'raw'
                  ? 'bg-white text-black shadow-xs font-semibold'
                  : 'text-zinc-600 hover:text-black'
              }`}
            >
              Raw Multiline (.env)
            </button>
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => setShowExampleModal(true)}
              className="btn-vercel-secondary text-xs"
            >
              <Sparkles size={12} /> Import from .env.example
            </button>
            {savedMsg && (
              <span className="text-xs font-medium text-emerald-600 flex items-center gap-1 animate-fade-in">
                <Check size={14} /> {savedMsg}
              </span>
            )}
          </div>
        </div>

        {/* View Mode: Structured */}
        {viewMode === 'structured' && (
          <div className="space-y-3">
            {variables.length === 0 ? (
              <div className="p-8 text-center text-zinc-400 text-xs">
                No environment variables set for this project. Click below to add one.
              </div>
            ) : (
              variables.map((v, idx) => (
                <div key={idx} className="flex flex-col sm:flex-row items-center gap-3 bg-zinc-50 p-3 rounded-xl border border-zinc-200">
                  <div className="w-full sm:w-1/3">
                    <input
                      type="text"
                      value={v.key}
                      onChange={(e) => updateVariable(idx, 'key', e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, '_'))}
                      placeholder="VARIABLE_NAME"
                      className="vercel-input font-mono text-xs bg-white"
                    />
                  </div>
                  <div className="w-full sm:w-1/2 relative">
                    <input
                      type={showValues[v.key] ? 'text' : 'password'}
                      value={v.value}
                      onChange={(e) => updateVariable(idx, 'value', e.target.value)}
                      placeholder="value"
                      className="vercel-input font-mono text-xs pr-10 bg-white"
                    />
                    <button
                      type="button"
                      onClick={() => toggleShowValue(v.key)}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-zinc-400 hover:text-black"
                    >
                      {showValues[v.key] ? <EyeOff size={14} /> : <Eye size={14} />}
                    </button>
                  </div>
                  <button
                    type="button"
                    onClick={() => removeVariable(idx)}
                    className="p-2 text-zinc-400 hover:text-rose-600 rounded transition shrink-0"
                    title="Remove variable"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              ))
            )}

            <button onClick={addVariable} className="btn-vercel-secondary text-xs mt-2">
              <Plus size={14} /> Add Variable
            </button>
          </div>
        )}

        {/* View Mode: Raw Multiline */}
        {viewMode === 'raw' && (
          <div className="space-y-2">
            <textarea
              rows={12}
              value={rawContent}
              onChange={(e) => setRawContent(e.target.value)}
              placeholder={`VITE_API_URL=https://api.example.com\nVITE_ANALYTICS_ID=UA-123456`}
              className="vercel-input font-mono text-xs leading-relaxed"
            />
            <span className="text-[11px] text-zinc-400 block">
              Lines formatted as KEY=value. Lines starting with # are ignored.
            </span>
          </div>
        )}

        {/* Footer Actions */}
        <div className="flex flex-col sm:flex-row items-center justify-end gap-3 pt-4 border-t border-zinc-100">
          <button
            onClick={() => handleSave(false)}
            disabled={saving}
            className="btn-vercel-secondary text-xs w-full sm:w-auto"
          >
            <Save size={14} /> {saving ? 'Saving...' : 'Save Encrypted Values'}
          </button>
          <button
            onClick={() => handleSave(true)}
            disabled={saving}
            className="btn-vercel-primary text-xs w-full sm:w-auto"
          >
            <Sparkles size={14} /> Save & Trigger Rebuild
          </button>
        </div>
      </div>

      {/* Parse .env.example Modal */}
      <Modal
        isOpen={showExampleModal}
        onClose={() => setShowExampleModal(false)}
        title="Import Variables from .env.example"
        maxWidth="max-w-xl"
      >
        <div className="space-y-4 text-xs">
          <p className="text-zinc-500">
            Paste the contents of your repository's <code className="text-black font-semibold font-mono">.env.example</code> to automatically extract variable templates and default keys.
          </p>
          <textarea
            rows={8}
            value={exampleContent}
            onChange={(e) => setExampleContent(e.target.value)}
            placeholder="# Public API Endpoint&#10;VITE_API_BASE=https://api.example.com&#10;# App Title&#10;VITE_APP_TITLE=Production Portal"
            className="vercel-input font-mono text-xs"
          />
          <div className="flex justify-end gap-3 pt-2 border-t border-zinc-100">
            <button onClick={() => setShowExampleModal(false)} className="btn-vercel-secondary text-xs">
              Cancel
            </button>
            <button
              onClick={handleParseExample}
              disabled={!exampleContent.trim()}
              className="btn-vercel-primary text-xs"
            >
              Parse & Add Variables
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

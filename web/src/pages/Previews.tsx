import React, { useState, useEffect } from 'react';
import { Preview } from '../types';
import { api } from '../api/client';
import { StatusBadge } from '../components/StatusBadge';
import { Sparkles, ExternalLink, Trash2, Clock, GitPullRequest, GitBranch } from 'lucide-react';

export const Previews: React.FC = () => {
  const [previews, setPreviews] = useState<Preview[]>([]);
  const [loading, setLoading] = useState(true);

  const loadPreviews = async () => {
    try {
      const res = await api.getPreviews();
      setPreviews(res || []);
    } catch (err) {
      console.error('Failed to load previews', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadPreviews();
    const interval = setInterval(loadPreviews, 8000);
    return () => clearInterval(interval);
  }, []);

  const handleDestroy = async (id: string, domain: string) => {
    if (!confirm(`Are you sure you want to destroy preview environment '${domain}'?`)) return;
    try {
      await api.destroyPreview(id);
      loadPreviews();
    } catch (err: any) {
      alert('Failed to destroy preview: ' + err.message);
    }
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Pull Request Previews</h1>
        <p className="text-xs text-zinc-500 mt-0.5">
          Ephemeral preview environments on isolated subdomains with automatic TTL cleanup
        </p>
      </div>

      <div className="vercel-card overflow-hidden">
        {loading && (previews || []).length === 0 ? (
          <div className="p-12 text-center text-zinc-400 text-xs">Loading previews...</div>
        ) : (previews || []).length === 0 ? (
          <div className="p-12 text-center text-zinc-500 text-sm">
            <Sparkles size={36} className="mx-auto text-zinc-300 mb-2" />
            No active previews currently deployed. Open a pull request with preview settings enabled to launch one.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Preview URL</th>
                  <th className="px-5 py-3 text-[11px]">Project</th>
                  <th className="px-5 py-3 text-[11px]">Source (PR / Branch)</th>
                  <th className="px-5 py-3 text-[11px]">Status</th>
                  <th className="px-5 py-3 text-[11px]">Expires</th>
                  <th className="px-5 py-3 text-[11px] text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {(previews || []).map((prev) => (
                  <tr key={prev.id} className="hover:bg-zinc-50/80 transition">
                    <td className="px-5 py-3.5">
                      <a
                        href={`https://${prev.domain}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="font-mono text-blue-600 hover:underline font-semibold flex items-center gap-1.5"
                      >
                        {prev.domain} <ExternalLink size={12} />
                      </a>
                    </td>
                    <td className="px-5 py-3.5 font-semibold text-zinc-900">{prev.project_name || prev.project_id}</td>
                    <td className="px-5 py-3.5 text-zinc-700 font-mono">
                      {prev.pr_number > 0 ? (
                        <span className="flex items-center gap-1 text-amber-700 font-medium">
                          <GitPullRequest size={12} /> PR #{prev.pr_number} ({prev.branch})
                        </span>
                      ) : (
                        <span className="flex items-center gap-1 text-zinc-600">
                          <GitBranch size={12} /> {prev.branch}
                        </span>
                      )}
                    </td>
                    <td className="px-5 py-3.5">
                      <StatusBadge status={prev.status} size="sm" />
                    </td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono flex items-center gap-1">
                      <Clock size={12} className="text-zinc-400" />
                      {new Date(prev.expires_at).toLocaleDateString()}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      {prev.status === 'active' && (
                        <button
                          onClick={() => handleDestroy(prev.id, prev.domain)}
                          className="p-1.5 text-zinc-400 hover:text-rose-600 rounded transition"
                          title="Teardown preview"
                        >
                          <Trash2 size={15} />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};

import React, { useState, useEffect } from 'react';
import { AuditLog } from '../types';
import { api } from '../api/client';
import { Search } from 'lucide-react';

export const AuditLogs: React.FC = () => {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.getAuditLogs()
      .then((data) => setLogs(data || []))
      .catch((err) => console.error('Failed to load audit logs', err))
      .finally(() => setLoading(false));
  }, []);

  const filtered = (logs || []).filter(
    (l) =>
      l.action.toLowerCase().includes(search.toLowerCase()) ||
      l.user_email.toLowerCase().includes(search.toLowerCase()) ||
      l.resource_type.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6 animate-fade-in max-w-6xl text-zinc-900">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Audit Logs</h1>
          <p className="text-xs text-zinc-500 mt-0.5">
            Immutable log of all administrative actions, deployments, and security events
          </p>
        </div>
        <div className="relative max-w-xs w-full">
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by action or email..."
            className="vercel-input text-xs pl-8"
          />
          <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-zinc-400" />
        </div>
      </div>

      <div className="vercel-card overflow-hidden">
        {loading ? (
          <div className="p-12 text-center text-zinc-400 text-xs">Loading audit logs...</div>
        ) : filtered.length === 0 ? (
          <div className="p-12 text-center text-zinc-400 text-xs">No audit logs recorded matching search</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-zinc-50 text-zinc-500 font-semibold uppercase tracking-wider border-b border-zinc-200">
                <tr>
                  <th className="px-5 py-3 text-[11px]">Timestamp</th>
                  <th className="px-5 py-3 text-[11px]">Actor Email</th>
                  <th className="px-5 py-3 text-[11px]">Action</th>
                  <th className="px-5 py-3 text-[11px]">Resource</th>
                  <th className="px-5 py-3 text-[11px]">IP Address</th>
                  <th className="px-5 py-3 text-[11px]">Details</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-100">
                {filtered.map((log) => (
                  <tr key={log.id} className="hover:bg-zinc-50/80 transition">
                    <td className="px-5 py-3.5 font-mono text-zinc-500 whitespace-nowrap">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                    <td className="px-5 py-3.5 font-semibold text-zinc-900">{log.user_email || 'System'}</td>
                    <td className="px-5 py-3.5">
                      <span className="font-mono text-black font-semibold px-2 py-0.5 rounded bg-zinc-100 border border-zinc-200">
                        {log.action}
                      </span>
                    </td>
                    <td className="px-5 py-3.5 text-zinc-700 font-mono capitalize">
                      {log.resource_type} {log.resource_id ? `(${log.resource_id.slice(0, 8)})` : ''}
                    </td>
                    <td className="px-5 py-3.5 font-mono text-zinc-500">{log.ip_address || 'internal'}</td>
                    <td className="px-5 py-3.5 text-zinc-500 font-mono text-[11px] truncate max-w-xs">
                      {log.details_json && log.details_json !== '{}' ? log.details_json : '-'}
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

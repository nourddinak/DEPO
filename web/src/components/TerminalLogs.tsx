import React, { useState, useEffect, useRef } from 'react';
import { Terminal, Copy, Check } from 'lucide-react';
import { DeploymentLog } from '../types';

interface TerminalLogsProps {
  logs: DeploymentLog[];
  height?: string;
  autoScroll?: boolean;
}

export const TerminalLogs: React.FC<TerminalLogsProps> = ({ logs, height = 'h-96', autoScroll = true }) => {
  const [copied, setCopied] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (autoScroll && containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const copyToClipboard = () => {
    const text = logs.map((l) => `[${l.created_at}] [${l.stage.toUpperCase()}] ${l.message}`).join('\n');
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const getStageColor = (stage: string) => {
    switch (stage) {
      case 'upload':
      case 'download':
        return 'text-sky-400';
      case 'extract':
        return 'text-indigo-400';
      case 'caddy_config':
      case 'caddy_validate':
      case 'caddy_reload':
        return 'text-amber-400';
      case 'health_check':
        return 'text-emerald-400';
      case 'rollback':
        return 'text-purple-400';
      default:
        return 'text-zinc-400';
    }
  };

  return (
    <div className="terminal-window rounded-xl overflow-hidden border border-zinc-200 shadow-sm bg-black">
      <div className="flex items-center justify-between px-4 py-2.5 bg-zinc-950 border-b border-zinc-800">
        <div className="flex items-center gap-2">
          <Terminal size={14} className="text-zinc-400" />
          <span className="text-xs font-mono font-medium text-zinc-300">Deployment Pipeline Logs</span>
          <span className="text-xs font-mono text-zinc-500">({logs.length} entries)</span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={copyToClipboard}
            className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-mono text-zinc-300 hover:text-white bg-zinc-800 hover:bg-zinc-700 rounded transition"
            title="Copy logs to clipboard"
          >
            {copied ? <Check size={12} className="text-emerald-400" /> : <Copy size={12} />}
            {copied ? 'Copied' : 'Copy'}
          </button>
        </div>
      </div>

      <div
        ref={containerRef}
        className={`${height} p-4 overflow-y-auto font-mono text-xs leading-relaxed space-y-1.5 select-text bg-black`}
      >
        {logs.length === 0 ? (
          <div className="text-zinc-500 italic py-4">Waiting for deployment pipeline events...</div>
        ) : (
          logs.map((log, idx) => (
            <div key={log.id || idx} className="flex items-start gap-2 group hover:bg-white/[0.04] py-0.5 px-1 rounded">
              <span className="text-zinc-500 shrink-0 select-none text-[11px]">
                {new Date(log.created_at).toLocaleTimeString([], {
                  hour12: false,
                  hour: '2-digit',
                  minute: '2-digit',
                  second: '2-digit',
                })}
              </span>
              <span
                className={`px-1.5 py-0.2 rounded uppercase text-[10px] font-bold border border-zinc-800 shrink-0 ${getStageColor(
                  log.stage
                )}`}
              >
                {log.stage}
              </span>
              <span className={`break-all ${log.level === 'error' ? 'text-rose-400 font-semibold' : 'text-zinc-200'}`}>
                {log.message}
              </span>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

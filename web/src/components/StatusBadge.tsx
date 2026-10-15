import React from 'react';

interface StatusBadgeProps {
  status: string;
  size?: 'sm' | 'md';
}

export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, size = 'md' }) => {
  const getStyle = () => {
    switch (status.toLowerCase()) {
      case 'online':
      case 'success':
      case 'active':
      case 'ready':
        return 'bg-emerald-50 text-emerald-700 border-emerald-200/80';
      case 'offline':
      case 'failed':
      case 'destroyed':
      case 'error':
        return 'bg-rose-50 text-rose-700 border-rose-200/80';
      case 'pending':
      case 'queued':
      case 'building':
      case 'uploading':
      case 'delivering':
      case 'activating':
        return 'bg-blue-50 text-blue-700 border-blue-200/80';
      case 'rolled_back':
        return 'bg-purple-50 text-purple-700 border-purple-200/80';
      case 'expired':
        return 'bg-zinc-100 text-zinc-600 border-zinc-200';
      default:
        return 'bg-zinc-100 text-zinc-600 border-zinc-200';
    }
  };

  const getIndicator = () => {
    switch (status.toLowerCase()) {
      case 'online':
      case 'success':
      case 'active':
      case 'ready':
        return 'bg-emerald-500';
      case 'offline':
      case 'failed':
      case 'destroyed':
      case 'error':
        return 'bg-rose-500';
      case 'pending':
      case 'queued':
      case 'building':
      case 'uploading':
      case 'delivering':
      case 'activating':
        return 'bg-blue-500 animate-pulse';
      case 'rolled_back':
        return 'bg-purple-500';
      default:
        return 'bg-zinc-400';
    }
  };

  const padding = size === 'sm' ? 'px-2 py-0.5 text-[11px]' : 'px-2.5 py-1 text-xs';

  return (
    <span
      className={`inline-flex items-center gap-1.5 font-medium border rounded-full capitalize select-none ${getStyle()} ${padding}`}
    >
      <span className={`w-1.5 h-1.5 rounded-full ${getIndicator()}`}></span>
      {status.replace('_', ' ')}
    </span>
  );
};

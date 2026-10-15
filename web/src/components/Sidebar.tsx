import React from 'react';
import {
  LayoutDashboard,
  FolderGit2,
  Rocket,
  Sparkles,
  Server,
  KeyRound,
  Globe,
  Bell,
  Users,
  ShieldCheck,
  ClipboardList,
  ChevronDown,
} from 'lucide-react';

interface SidebarProps {
  currentTab: string;
  onSelectTab: (tab: string) => void;
  onlineServerCount?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({ currentTab, onSelectTab, onlineServerCount = 0 }) => {
  const sections = [
    {
      title: 'Platform',
      items: [
        { id: 'overview', label: 'Overview', icon: LayoutDashboard },
        { id: 'projects', label: 'Projects', icon: FolderGit2 },
        { id: 'deployments', label: 'Deployments', icon: Rocket },
        { id: 'previews', label: 'PR Previews', icon: Sparkles },
      ],
    },
    {
      title: 'Infrastructure',
      items: [
        { id: 'servers', label: 'VPS Fleet', icon: Server, badge: onlineServerCount },
        { id: 'domains', label: 'Domains & TLS', icon: Globe },
        { id: 'environment', label: 'Environment', icon: KeyRound },
      ],
    },
    {
      title: 'Management',
      items: [
        { id: 'notifications', label: 'Notifications', icon: Bell },
        { id: 'team', label: 'Team & Invites', icon: Users },
        { id: 'security', label: 'Security & 2FA', icon: ShieldCheck },
        { id: 'audit', label: 'Audit Logs', icon: ClipboardList },
      ],
    },
  ];

  return (
    <aside className="w-64 bg-white border-r border-zinc-200 flex flex-col shrink-0 select-none text-zinc-900 font-sans z-30">
      {/* Brand Header */}
      <div className="h-16 flex items-center justify-between px-5 border-b border-zinc-200 bg-white">
        <div className="flex items-center gap-2.5">
          <div className="w-7 h-7 rounded bg-black text-white flex items-center justify-center">
            <svg viewBox="0 0 24 24" fill="currentColor" className="w-4 h-4 text-white">
              <path d="M12 2L2 22h20L12 2z" />
            </svg>
          </div>
          <div>
            <div className="flex items-center gap-1.5">
              <span className="text-sm font-bold tracking-tight text-black">DEPO</span>
              <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-zinc-100 text-zinc-600 border border-zinc-200">
                PRO
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Scope / Fleet Selector Pill */}
      <div className="px-4 py-3 border-b border-zinc-100 bg-zinc-50/50">
        <div className="flex items-center justify-between px-2.5 py-1.5 rounded-md bg-white border border-zinc-200 shadow-sm text-xs text-zinc-800">
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
            <span className="font-medium text-xs truncate max-w-[130px]">Personal VPS Fleet</span>
          </div>
          <ChevronDown size={12} className="text-zinc-400" />
        </div>
      </div>

      {/* Navigation Groups */}
      <nav className="flex-1 p-3 space-y-4 overflow-y-auto">
        {sections.map((sec) => (
          <div key={sec.title} className="space-y-0.5">
            <div className="px-3 py-1 text-[10px] font-semibold uppercase tracking-wider text-zinc-400">
              {sec.title}
            </div>
            {sec.items.map((item) => {
              const Icon = item.icon;
              const isActive = currentTab === item.id;
              return (
                <button
                  key={item.id}
                  onClick={() => onSelectTab(item.id)}
                  className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-xs font-medium transition-all ${
                    isActive
                      ? 'bg-zinc-100 text-black font-semibold'
                      : 'text-zinc-600 hover:text-black hover:bg-zinc-50'
                  }`}
                >
                  <div className="flex items-center gap-2.5">
                    <Icon size={16} className={isActive ? 'text-black' : 'text-zinc-400'} />
                    <span>{item.label}</span>
                  </div>
                  {item.badge !== undefined && item.badge > 0 && (
                    <span className="px-1.5 py-0.2 text-[10px] font-mono font-medium rounded-full bg-emerald-50 text-emerald-700 border border-emerald-200">
                      {item.badge} online
                    </span>
                  )}
                </button>
              );
            })}
          </div>
        ))}
      </nav>

      {/* Sidebar Footer Info */}
      <div className="p-3.5 border-t border-zinc-200 bg-zinc-50/60 text-xs text-zinc-500 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
          <span className="text-[11px] font-mono text-zinc-600">Central API Live</span>
        </div>
        <span className="text-[11px] font-mono text-zinc-400">v1.0.0</span>
      </div>
    </aside>
  );
};

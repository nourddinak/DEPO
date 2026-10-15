import React from 'react';
import { useAuth } from '../context/AuthContext';
import { LogOut, Radio, Search, UserCheck } from 'lucide-react';

interface NavbarProps {
  sseConnected: boolean;
  currentTab?: string;
}

export const Navbar: React.FC<NavbarProps> = ({ sseConnected, currentTab = 'overview' }) => {
  const { user, logout } = useAuth();

  const getTabLabel = (tab: string) => {
    switch (tab) {
      case 'overview':
        return 'Overview';
      case 'projects':
        return 'Projects';
      case 'project_detail':
        return 'Project Detail';
      case 'deployments':
        return 'Deployments';
      case 'previews':
        return 'PR Previews';
      case 'servers':
        return 'VPS Fleet';
      case 'environment':
        return 'Environment Variables';
      case 'domains':
        return 'Domains & TLS';
      case 'notifications':
        return 'Notifications';
      case 'team':
        return 'Team & Invitations';
      case 'security':
        return 'Account & 2FA';
      case 'audit':
        return 'Audit Logs';
      default:
        return 'Dashboard';
    }
  };

  return (
    <header className="h-16 bg-white border-b border-zinc-200 px-6 flex items-center justify-between sticky top-0 z-20 select-none">
      {/* Breadcrumb Navigation */}
      <div className="flex items-center gap-2 text-xs">
        <span className="font-semibold text-zinc-900 tracking-tight">DEPO</span>
        <span className="text-zinc-300">/</span>
        <span className="text-zinc-600 font-medium">{getTabLabel(currentTab)}</span>
      </div>

      {/* Center Search Input (Vercel Style) */}
      <div className="hidden md:flex items-center gap-2 px-3 py-1.5 rounded-lg border border-zinc-200 bg-zinc-50 w-80 text-zinc-400 text-xs">
        <Search size={14} className="text-zinc-400" />
        <span className="flex-1 text-zinc-400 text-[11px]">Search deployments, servers...</span>
        <kbd className="px-1.5 py-0.5 text-[10px] font-mono rounded bg-white border border-zinc-200 text-zinc-500 shadow-2xs">
          ⌘K
        </kbd>
      </div>

      {/* User Controls & Realtime Status */}
      <div className="flex items-center gap-4">
        {/* Realtime Stream Indicator */}
        <div className="flex items-center gap-2 px-2.5 py-1 rounded-full bg-zinc-50 border border-zinc-200 text-xs">
          <span className={`w-2 h-2 rounded-full ${sseConnected ? 'bg-emerald-500 animate-pulse' : 'bg-zinc-400'}`}></span>
          <span className="font-mono text-[11px] text-zinc-600">
            {sseConnected ? 'SSE Live' : 'Connecting...'}
          </span>
        </div>

        {user && (
          <div className="flex items-center gap-3 pl-2 border-l border-zinc-200">
            <div className="text-right hidden sm:block">
              <span className="block text-xs font-semibold text-zinc-900 leading-tight">{user.email}</span>
              <span className="text-[10px] uppercase font-mono tracking-wider px-1.5 py-0.2 rounded bg-zinc-100 text-zinc-600 border border-zinc-200">
                {user.role}
              </span>
            </div>
            <div className="w-8 h-8 rounded-full bg-zinc-900 text-white flex items-center justify-center font-semibold text-xs uppercase shadow-xs">
              {user.email.slice(0, 2)}
            </div>
          </div>
        )}

        <button
          onClick={logout}
          className="p-2 text-zinc-400 hover:text-black hover:bg-zinc-100 rounded-md transition"
          title="Sign out of control panel"
        >
          <LogOut size={16} />
        </button>
      </div>
    </header>
  );
};

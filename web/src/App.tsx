import React, { useState, useEffect } from 'react';
import { useAuth } from './context/AuthContext';
import { Login } from './pages/Login';
import { Sidebar } from './components/Sidebar';
import { Navbar } from './components/Navbar';
import { Overview } from './pages/Overview';
import { Projects } from './pages/Projects';
import { ProjectDetail } from './pages/ProjectDetail';
import { Deployments } from './pages/Deployments';
import { Previews } from './pages/Previews';
import { Servers } from './pages/Servers';
import { EnvironmentVars } from './pages/EnvironmentVars';
import { Domains } from './pages/Domains';
import { Notifications } from './pages/Notifications';
import { TeamInvitations } from './pages/TeamInvitations';
import { AccountSecurity } from './pages/AccountSecurity';
import { AuditLogs } from './pages/AuditLogs';
import { api } from './api/client';
import { Project, Server, Deployment, Preview } from './types';

export const AppContent: React.FC = () => {
  const { user, loading, setupRequired } = useAuth();
  const [currentTab, setCurrentTab] = useState<string>('overview');
  const [selectedProjectId, setSelectedProjectId] = useState<string | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [servers, setServers] = useState<Server[]>([]);
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [previews, setPreviews] = useState<Preview[]>([]);
  const [sseConnected, setSseConnected] = useState(false);

  // Initial data loading
  const refreshAll = async () => {
    if (!user) return;
    try {
      const [p, s, d, prev] = await Promise.all([
        api.getProjects(),
        api.getServers(),
        api.getDeployments(),
        api.getPreviews(),
      ]);
      setProjects(p || []);
      setServers(s || []);
      setDeployments(d || []);
      setPreviews(prev || []);
    } catch (err) {
      console.error('Failed to load dashboard data', err);
    }
  };

  useEffect(() => {
    if (user) {
      refreshAll();
    }
  }, [user]);

  // Real-time SSE Connection
  useEffect(() => {
    if (!user) return;

    let eventSource: EventSource | null = null;
    try {
      eventSource = new EventSource('/api/v1/events');

      eventSource.onopen = () => {
        setSseConnected(true);
      };

      eventSource.onmessage = (e) => {
        try {
          const evt = JSON.parse(e.data);
          if (evt.type === 'server_metrics') {
            setServers((prev) =>
              (prev || []).map((s) => (s.id === evt.payload.server_id ? { ...s, ...evt.payload } : s))
            );
          } else if (evt.type === 'deployment_created') {
            refreshAll();
          } else if (evt.type === 'deployment_updated') {
            setDeployments((prev) =>
              (prev || []).map((d) => (d.id === evt.payload.deployment_id ? { ...d, status: evt.payload.status } : d))
            );
          } else if (evt.type === 'server_status') {
            setServers((prev) =>
              (prev || []).map((s) => (s.id === evt.payload.server_id ? { ...s, status: evt.payload.status } : s))
            );
          }
        } catch (err) {
          // ignore ping
        }
      };

      eventSource.onerror = () => {
        setSseConnected(false);
      };
    } catch (err) {
      setSseConnected(false);
    }

    return () => {
      if (eventSource) eventSource.close();
    };
  }, [user]);

  if (loading) {
    return (
      <div className="min-h-screen bg-zinc-50 flex items-center justify-center text-zinc-500 font-mono text-xs">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-black animate-ping"></span>
          <span>Loading DEPO control center...</span>
        </div>
      </div>
    );
  }

  if (setupRequired || !user) {
    return <Login />;
  }

  const handleSelectProject = (id: string) => {
    setSelectedProjectId(id);
    setCurrentTab('project_detail');
  };

  const handleNavigateEnv = (id: string) => {
    setSelectedProjectId(id);
    setCurrentTab('environment');
  };

  const onlineServerCount = (servers || []).filter((s) => s.status === 'online').length;

  return (
    <div className="flex h-screen bg-zinc-50 overflow-hidden font-sans text-zinc-900">
      <Sidebar
        currentTab={currentTab}
        onSelectTab={(tab) => {
          setSelectedProjectId(null);
          setCurrentTab(tab);
        }}
        onlineServerCount={onlineServerCount}
      />

      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        <Navbar sseConnected={sseConnected} currentTab={currentTab} />

        <main className="flex-1 overflow-y-auto p-6 md:p-8">
          <div className="max-w-7xl mx-auto">
            {currentTab === 'overview' && (
              <Overview
                projects={projects}
                servers={servers}
                deployments={deployments}
                previews={previews}
                onNavigate={(tab) => setCurrentTab(tab)}
              />
            )}

            {currentTab === 'projects' && (
              <Projects
                projects={projects}
                servers={servers}
                onRefresh={refreshAll}
                onSelectProject={handleSelectProject}
              />
            )}

            {currentTab === 'project_detail' && selectedProjectId && (
              <ProjectDetail
                projectId={selectedProjectId}
                onBack={() => {
                  setSelectedProjectId(null);
                  setCurrentTab('projects');
                }}
                onNavigateEnv={handleNavigateEnv}
              />
            )}

            {currentTab === 'deployments' && <Deployments projects={projects} />}

            {currentTab === 'previews' && <Previews />}

            {currentTab === 'servers' && <Servers servers={servers} onRefresh={refreshAll} />}

            {currentTab === 'environment' && (
              <EnvironmentVars projects={projects} initialProjectId={selectedProjectId || undefined} />
            )}

            {currentTab === 'domains' && <Domains />}

            {currentTab === 'notifications' && <Notifications />}

            {currentTab === 'team' && <TeamInvitations />}

            {currentTab === 'security' && <AccountSecurity />}

            {currentTab === 'audit' && <AuditLogs />}
          </div>
        </main>
      </div>
    </div>
  );
};

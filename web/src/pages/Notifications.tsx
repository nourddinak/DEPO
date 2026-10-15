import React, { useState, useEffect } from 'react';
import { api } from '../api/client';
import { Send, Check } from 'lucide-react';

export const Notifications: React.FC = () => {
  const [ntfyUrl, setNtfyUrl] = useState('https://ntfy.sh');
  const [topic, setTopic] = useState('');
  const [authToken, setAuthToken] = useState('');
  const [events, setEvents] = useState<string[]>([
    'deploy_success',
    'deploy_failed',
    'server_offline',
    'disk_warning',
  ]);
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [msg, setMsg] = useState('');

  const availableEvents = [
    { id: 'deploy_success', label: 'Successful Deployments', desc: 'When a new static release is validated and live' },
    { id: 'deploy_failed', label: 'Failed Deployments', desc: 'When a build, checksum, validation, or health check fails' },
    { id: 'server_offline', label: 'Server Offline', desc: 'When an agent misses heartbeats for more than 60 seconds' },
    { id: 'server_recovered', label: 'Server Recovered', desc: 'When an offline server agent reconnects and polls' },
    { id: 'disk_warning', label: 'Disk Space Alert', desc: 'When a target VPS crosses 90% disk utilization' },
    { id: 'preview_cleanup', label: 'Preview Cleanup', desc: 'When expired PR preview environments are destroyed' },
  ];

  useEffect(() => {
    api.getNotifications()
      .then((data) => {
        setNtfyUrl(data.ntfy_url || 'https://ntfy.sh');
        setTopic(data.topic || '');
        setEvents(data.events || []);
        setEnabled(data.enabled);
      })
      .catch((err) => console.error('Failed to load notifications', err))
      .finally(() => setLoading(false));
  }, []);

  const toggleEvent = (eventId: string) => {
    if ((events || []).includes(eventId)) {
      setEvents((events || []).filter((e) => e !== eventId));
    } else {
      setEvents([...(events || []), eventId]);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setMsg('');
    try {
      await api.updateNotifications({
        ntfy_url: ntfyUrl,
        topic,
        auth_token: authToken,
        events,
        enabled,
      });
      setMsg('Notification settings saved!');
      setTimeout(() => setMsg(''), 3000);
    } catch (err: any) {
      alert('Failed to save settings: ' + err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleTest = async () => {
    setTesting(true);
    try {
      await api.testNotification();
      alert('Test alert dispatched to ntfy topic! Check your device.');
    } catch (err: any) {
      alert('Test notification failed: ' + err.message);
    } finally {
      setTesting(false);
    }
  };

  if (loading) {
    return <div className="p-8 text-center text-zinc-400 text-xs">Loading notification settings...</div>;
  }

  return (
    <div className="space-y-6 animate-fade-in max-w-4xl text-zinc-900">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Push Notifications</h1>
        <p className="text-xs text-zinc-500 mt-0.5">
          Realtime push alerts to your phone or desktop via ntfy without third-party email services
        </p>
      </div>

      <form onSubmit={handleSave} className="vercel-card p-6 space-y-6">
        <div className="flex items-center justify-between border-b border-zinc-100 pb-4">
          <div>
            <span className="font-semibold text-zinc-900 text-sm block">Enable Push Notifications</span>
            <span className="text-xs text-zinc-500">Dispatch alerts for deployment and server status events</span>
          </div>
          <label className="relative inline-flex items-center cursor-pointer">
            <input
              type="checkbox"
              checked={enabled}
              onChange={(e) => setEnabled(e.target.checked)}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-zinc-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-zinc-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-black"></div>
          </label>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              ntfy Server URL
            </label>
            <input
              type="url"
              required
              value={ntfyUrl}
              onChange={(e) => setNtfyUrl(e.target.value)}
              placeholder="https://ntfy.sh"
              className="vercel-input font-mono text-xs"
            />
            <span className="text-[11px] text-zinc-400 mt-1 block">Default public server or your self-hosted ntfy instance</span>
          </div>

          <div>
            <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
              Notification Topic
            </label>
            <input
              type="text"
              required={enabled}
              value={topic}
              onChange={(e) => setTopic(e.target.value)}
              placeholder="depo-deployments-mysecret"
              className="vercel-input font-mono text-xs"
            />
            <span className="text-[11px] text-zinc-400 mt-1 block">Subscribe to this topic name in the ntfy app</span>
          </div>
        </div>

        <div>
          <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider mb-1">
            Optional Authentication Token
          </label>
          <input
            type="password"
            value={authToken}
            onChange={(e) => setAuthToken(e.target.value)}
            placeholder="tk_••••••••••••••••"
            className="vercel-input font-mono text-xs max-w-md"
          />
          <span className="text-[11px] text-zinc-400 mt-1 block">Required only if your ntfy topic requires authentication</span>
        </div>

        {/* Event Trigger Selection */}
        <div className="space-y-3 pt-3 border-t border-zinc-100">
          <label className="block text-xs font-semibold text-zinc-700 uppercase tracking-wider">
            Subscribed Event Triggers
          </label>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            {availableEvents.map((evt) => (
              <label
                key={evt.id}
                className={`p-3 rounded-xl border flex items-start gap-3 cursor-pointer transition ${
                  events.includes(evt.id)
                    ? 'bg-zinc-50 border-black text-black'
                    : 'bg-white border-zinc-200 text-zinc-600 hover:border-zinc-300'
                }`}
              >
                <input
                  type="checkbox"
                  checked={events.includes(evt.id)}
                  onChange={() => toggleEvent(evt.id)}
                  className="rounded border-zinc-300 text-black mt-0.5 focus:ring-0"
                />
                <div className="text-xs">
                  <span className="font-semibold block text-zinc-900">{evt.label}</span>
                  <span className="text-zinc-500 text-[11px]">{evt.desc}</span>
                </div>
              </label>
            ))}
          </div>
        </div>

        {/* Buttons */}
        <div className="flex items-center justify-between pt-4 border-t border-zinc-100">
          <button
            type="button"
            onClick={handleTest}
            disabled={testing || !topic}
            className="btn-vercel-secondary text-xs"
          >
            <Send size={12} /> {testing ? 'Sending...' : 'Send Test Notification'}
          </button>

          <div className="flex items-center gap-3">
            {msg && <span className="text-xs font-medium text-emerald-600 flex items-center gap-1"><Check size={14} /> {msg}</span>}
            <button type="submit" disabled={saving} className="btn-vercel-primary text-xs">
              {saving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
};

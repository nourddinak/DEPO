import React, { useState } from 'react';
import { api } from '../api/client';
import { Globe, Search, ShieldCheck, CheckCircle2, AlertCircle, Cloud } from 'lucide-react';

export const Domains: React.FC = () => {
  const [domainInput, setDomainInput] = useState('');
  const [checking, setChecking] = useState(false);
  const [result, setResult] = useState<{
    domain: string;
    resolved_ips: string[];
    cname: string;
    is_routed: boolean;
    instructions: string;
  } | null>(null);

  const handleCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!domainInput.trim()) return;
    setChecking(true);
    try {
      const res = await api.checkDomain(domainInput.trim());
      setResult(res);
    } catch (err: any) {
      alert('DNS check failed: ' + err.message);
    } finally {
      setChecking(false);
    }
  };

  return (
    <div className="space-y-6 animate-fade-in text-zinc-900">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-zinc-900">Domains & TLS</h1>
        <p className="text-xs text-zinc-500 mt-0.5">
          DNS routing diagnostics, custom domains, and automatic Caddy SSL/TLS certificate management
        </p>
      </div>

      {/* DNS Diagnostic Tool */}
      <div className="vercel-card p-6 space-y-4">
        <h2 className="text-sm font-bold text-zinc-900 flex items-center gap-2">
          <Globe size={16} className="text-zinc-700" /> DNS Propagation & Routing Checker
        </h2>
        <form onSubmit={handleCheck} className="flex flex-col sm:flex-row gap-3">
          <input
            type="text"
            required
            value={domainInput}
            onChange={(e) => setDomainInput(e.target.value.toLowerCase())}
            placeholder="e.g. app.mycompany.com"
            className="vercel-input font-mono text-xs flex-1"
          />
          <button type="submit" disabled={checking} className="btn-vercel-primary text-xs shrink-0">
            <Search size={14} /> {checking ? 'Querying DNS...' : 'Check DNS'}
          </button>
        </form>

        {result && (
          <div className="mt-4 p-4 rounded-xl bg-zinc-50 border border-zinc-200 space-y-3 animate-fade-in text-xs font-mono">
            <div className="flex items-center gap-2 font-semibold text-sm">
              {result.is_routed ? (
                <span className="text-emerald-700 flex items-center gap-1.5 font-sans">
                  <CheckCircle2 size={16} className="text-emerald-600" /> DNS Resolved Successfully
                </span>
              ) : (
                <span className="text-amber-800 flex items-center gap-1.5 font-sans">
                  <AlertCircle size={16} className="text-amber-600" /> No IP Records Found Yet
                </span>
              )}
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 text-zinc-800">
              <div>
                <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">Queried Domain</span>
                <span className="font-semibold">{result.domain}</span>
              </div>
              <div>
                <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">Resolved A Records (IPs)</span>
                <span>{result.resolved_ips.length > 0 ? result.resolved_ips.join(', ') : 'None'}</span>
              </div>
              {result.cname && (
                <div className="col-span-2">
                  <span className="text-zinc-400 block text-[10px] uppercase font-sans font-semibold">CNAME Record</span>
                  <span>{result.cname}</span>
                </div>
              )}
            </div>

            <div className="pt-2 border-t border-zinc-200 text-zinc-600 font-sans text-xs leading-normal">
              {result.instructions}
            </div>
          </div>
        )}
      </div>

      {/* Caddy & Cloudflare Architecture Guide */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-5 text-xs text-zinc-600">
        <div className="vercel-card p-5 space-y-3">
          <div className="flex items-center gap-2 font-semibold text-zinc-900 text-sm">
            <ShieldCheck size={18} className="text-emerald-600" />
            Automatic Zero-Downtime TLS by Caddy
          </div>
          <p className="leading-relaxed text-zinc-600">
            Caddy automatically provisions and renews SSL/TLS certificates via Let's Encrypt and ZeroSSL.
            Once your DNS <code className="text-black font-semibold font-mono">A</code> record points to the VPS IP, Caddy solves the ACME HTTP-01 challenge automatically upon first request.
          </p>
          <ul className="list-disc pl-4 space-y-1 text-zinc-600">
            <li>Zero manual certificate installation or renewal cron jobs</li>
            <li>HTTP automatically redirects to HTTPS with HSTS headers</li>
            <li>OCSP stapling and TLS 1.3 enabled by default</li>
          </ul>
        </div>

        <div className="vercel-card p-5 space-y-3">
          <div className="flex items-center gap-2 font-semibold text-zinc-900 text-sm">
            <Cloud size={18} className="text-blue-600" />
            Optional Cloudflare Integration
          </div>
          <p className="leading-relaxed text-zinc-600">
            You can use Cloudflare to manage your DNS records with ease.
          </p>
          <ul className="list-disc pl-4 space-y-1 text-zinc-600">
            <li>
              <strong>DNS-Only (Grey Cloud):</strong> Recommended for first release so Caddy can complete Let's Encrypt validation directly.
            </li>
            <li>
              <strong>Proxied (Orange Cloud):</strong> Set SSL/TLS encryption mode to <em>Full (Strict)</em> in Cloudflare settings to prevent redirect loops.
            </li>
            <li>
              <strong>Wildcard DNS for Previews:</strong> Add a <code className="text-black font-semibold font-mono">*.preview</code> CNAME pointing to your root domain for seamless PR preview routing.
            </li>
          </ul>
        </div>
      </div>
    </div>
  );
};

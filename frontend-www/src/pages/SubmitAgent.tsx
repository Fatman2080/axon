import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { Terminal, Copy, CheckCircle, Shield, AlertTriangle, Lock, Key, Twitter } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import { authApi } from '../services/api';
import { fetchUser } from '../store/slices/userSlice';
import { useLogin } from '../hooks/useLogin';

const SubmitAgent = () => {
  const dispatch = useAppDispatch();
  const { currentUser, loading } = useAppSelector((state) => state.user);
  const { t } = useLanguage();
  const { login } = useLogin();
  const [hasAccess, setHasAccess] = useState(false);
  const [inviteCode, setInviteCode] = useState('');
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);
  const [command, setCommand] = useState('');

  useEffect(() => {
    setHasAccess(Boolean(currentUser?.agentPublicKey));
  }, [currentUser]);

  useEffect(() => {
    if (!currentUser?.agentPublicKey) return;
    authApi.getDeployCommand().then(r => setCommand(r.command || '')).catch(() => {});
  }, [currentUser?.agentPublicKey]);

  const handleVerify = async () => {
    if (!currentUser) return;
    setIsVerifying(true);
    setError('');
    try {
      await authApi.consumeInviteCode(inviteCode);
      await dispatch(fetchUser()).unwrap();
      setHasAccess(true);
      localStorage.setItem('agent_deploy_access', 'true');
    } catch (e: unknown) {
      const err = e as { response?: { data?: { error?: string } } };
      setError(err?.response?.data?.error || t('submitAgent.access.invalid'));
    } finally {
      setIsVerifying(false);
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="mx-auto max-w-3xl space-y-10 pb-20 pt-10 animate-fade-in-up">
      {/* Header */}
      <div className="text-center">
        <div
          className="inline-block text-xs font-mono uppercase tracking-widest px-3 py-1 rounded mb-4"
          style={{
            background: 'var(--neon-green-dim)',
            color: 'var(--neon-green)',
            border: '1px solid rgba(0,240,255,0.2)',
          }}
        >
          Agent Deployment Portal
        </div>
        <h1 className="text-4xl font-extrabold tracking-tight mb-3" style={{ color: 'var(--text-primary)' }}>
          {t('submitAgent.title')}
        </h1>
        <p className="text-lg" style={{ color: 'var(--text-secondary)' }}>
          {t('submitAgent.subtitle')}
        </p>
      </div>

      <div className="space-y-6">
        {/* Access Gate */}
        {!hasAccess ? (
          <div
            className="relative overflow-hidden rounded p-8 text-center"
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
          >
            <div
              className="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded"
              style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)' }}
            >
              <Lock size={28} style={{ color: 'var(--text-tertiary)' }} />
            </div>

            {!currentUser ? (
              /* Step 1: Not logged in — require X login first */
              <>
                <h2 className="text-xl font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
                  {t('submitAgent.access.loginRequired')}
                </h2>
                <p className="max-w-md mx-auto mb-8 text-sm" style={{ color: 'var(--text-secondary)' }}>
                  {t('submitAgent.access.loginDesc')}
                </p>
                <div className="mx-auto max-w-sm">
                  <button
                    onClick={() => login(undefined, '/submit-agent')}
                    disabled={loading}
                    className="w-full py-3 text-sm font-bold rounded transition-all disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2"
                    style={{ background: 'var(--neon-green)', color: '#000', fontFamily: 'var(--font-mono)' }}
                  >
                    <Twitter size={16} />
                    {t('submitAgent.access.loginBtn')}
                  </button>
                </div>
              </>
            ) : (
              /* Step 2: Logged in but no agent — show invite code input */
              <>
                <h2 className="text-xl font-bold mb-2" style={{ color: 'var(--text-primary)' }}>
                  {t('submitAgent.access.denied')}
                </h2>
                <p className="max-w-md mx-auto mb-8 text-sm" style={{ color: 'var(--text-secondary)' }}>
                  {t('submitAgent.access.apply')}
                </p>

                <div className="mx-auto max-w-sm space-y-4">
                  <div className="relative">
                    <Key className="absolute left-3 top-1/2 -translate-y-1/2" size={16} style={{ color: 'var(--text-tertiary)' }} />
                    <input
                      type="text"
                      value={inviteCode}
                      onChange={(e) => setInviteCode(e.target.value)}
                      placeholder={t('submitAgent.access.placeholder')}
                      className="w-full py-3 pl-10 pr-4 text-sm font-mono"
                      style={{
                        background: 'var(--bg-input)',
                        border: '1px solid var(--border)',
                        borderRadius: '4px',
                        color: 'var(--text-primary)',
                      }}
                      onFocus={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--neon-green)'}
                      onBlur={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
                    />
                  </div>

                  {error && (
                    <div className="text-sm flex items-center justify-center gap-1.5 font-mono" style={{ color: 'var(--red)' }}>
                      <AlertTriangle size={14} />
                      {error}
                    </div>
                  )}

                  <button
                    onClick={handleVerify}
                    disabled={!inviteCode || isVerifying || loading}
                    className="w-full py-3 text-sm font-bold rounded transition-all disabled:opacity-40 disabled:cursor-not-allowed"
                    style={{ background: 'var(--neon-green)', color: '#000', fontFamily: 'var(--font-mono)' }}
                  >
                    {isVerifying ? (
                      <span className="flex items-center justify-center gap-2">
                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-black border-t-transparent" />
                        {t('submitAgent.access.check')}
                      </span>
                    ) : (
                      <span className="flex items-center justify-center gap-2">
                        {t('submitAgent.access.submit')}
                      </span>
                    )}
                  </button>
                </div>
              </>
            )}
          </div>
        ) : (
          <div
            className="rounded p-8 text-center"
            style={{ background: 'var(--green-dim)', border: '1px solid rgba(0,255,102,0.2)' }}
          >
            <div
              className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded"
              style={{ background: 'rgba(0,255,102,0.15)' }}
            >
              <CheckCircle size={28} style={{ color: 'var(--green)' }} />
            </div>
            <h2 className="text-xl font-bold mb-2" style={{ color: 'var(--green)' }}>
              {t('submitAgent.access.granted')}
            </h2>
          </div>
        )}

        {/* Command Terminal */}
        <div
          className={`relative transition-all duration-500 terminal-frame ${!hasAccess ? 'opacity-30 grayscale blur-[2px] pointer-events-none select-none' : ''}`}
        >
          <div
            className="rounded p-7 shadow-2xl"
            style={{ background: '#0D0E10', border: '1px solid rgba(255,255,255,0.06)' }}
          >
            <label
              className="mb-4 block text-xs font-mono font-bold uppercase tracking-widest"
              style={{ color: 'var(--text-tertiary)' }}
            >
              {t('submitAgent.command.label')}
            </label>
            <div
              className="group flex items-center justify-between gap-4 rounded p-4 font-mono"
              style={{ background: 'var(--bg-void)', border: '1px solid rgba(0,240,255,0.1)' }}
            >
              <div className="flex items-center min-w-0">
                <Terminal size={16} className="mr-3 shrink-0" style={{ color: 'var(--text-tertiary)' }} />
                <span className="break-all text-sm" style={{ color: 'var(--green)' }}>{command || t('submitAgent.command.loading')}</span>
              </div>
              <button
                onClick={handleCopy}
                disabled={!hasAccess || !command}
                className="shrink-0 rounded p-1.5 transition-all"
                style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid var(--border)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--neon-green)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
              >
                {copied ? <CheckCircle size={14} style={{ color: 'var(--green)' }} /> : <Copy size={14} style={{ color: 'var(--text-tertiary)' }} />}
              </button>
            </div>
            <div className="mt-4 flex items-start gap-2 text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>
              <Shield size={12} className="mt-0.5 shrink-0" />
              This command will install the local CLI tool and authenticate your session.
            </div>
          </div>

          {!hasAccess && (
            <div className="absolute inset-0 z-10 flex items-center justify-center">
              <div
                className="rounded px-4 py-2 text-sm font-mono flex items-center gap-2"
                style={{ background: 'rgba(10,10,12,0.9)', border: '1px solid var(--border)', color: 'var(--text-secondary)' }}
              >
                <Lock size={12} />
                ACCESS_LOCKED
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SubmitAgent;

import React, { useEffect, useState } from 'react';
import { useAppDispatch, useAppSelector } from '../hooks/redux';
import { Terminal, Copy, CheckCircle, Shield, AlertTriangle, Lock, Key, Twitter } from 'lucide-react';
import { useLanguage } from '../context/LanguageContext';
import { authApi } from '../services/api';
import { fetchUser } from '../store/slices/userSlice';

const SubmitAgent = () => {
  const dispatch = useAppDispatch();
  const { currentUser, loading } = useAppSelector((state) => state.user);
  const { t } = useLanguage();
  const [hasAccess, setHasAccess] = useState(false);
  const [inviteCode, setInviteCode] = useState('');
  const [isVerifying, setIsVerifying] = useState(false);
  const [error, setError] = useState('');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setHasAccess(Boolean(currentUser?.agentPublicKey));
  }, [currentUser]);

  const handleVerify = async () => {
    setIsVerifying(true);
    setError('');
    try {
      if (!currentUser) {
        window.location.href = authApi.getXOAuthStartUrl(inviteCode, '/submit-agent');
        return;
      } else {
        await authApi.consumeInviteCode(inviteCode);
      }
      await dispatch(fetchUser()).unwrap();
      setHasAccess(true);
      localStorage.setItem('agent_deploy_access', 'true');
    } catch (e: any) {
      setError(e?.response?.data?.error || t('submitAgent.access.invalid'));
    } finally {
      setIsVerifying(false);
    }
  };

  const command = `npx clwafi-cli deploy --key ${
    currentUser?.agentPublicKey || currentUser?.walletAddress || 'YOUR_AGENT_PUBLIC_KEY'
  }`;

  const handleCopy = () => {
    navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="mx-auto max-w-3xl space-y-12 pb-20 pt-10">
      <div className="text-center">
        <h1 className="text-4xl font-bold tracking-tight text-zinc-900 mb-4">{t('submitAgent.title')}</h1>
        <p className="text-xl text-zinc-500 max-w-2xl mx-auto">{t('submitAgent.subtitle')}</p>
      </div>

      <div className="animate-fade-in space-y-8">
        {!hasAccess ? (
          <div className="relative overflow-hidden rounded-2xl border border-zinc-200 bg-white p-8 text-center shadow-sm">
            <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-zinc-100 text-zinc-400">
              <Lock size={32} />
            </div>
            <h2 className="text-2xl font-bold text-zinc-900 mb-2">{t('submitAgent.access.denied')}</h2>
            <p className="text-zinc-500 max-w-md mx-auto mb-8">{t('submitAgent.access.apply')}</p>

            <div className="mx-auto max-w-sm space-y-4">
              {!currentUser && (
                <div className="rounded-lg border border-zinc-200 bg-zinc-50 p-3 text-xs text-zinc-600">
                  Login will continue with X OAuth automatically after you submit invite code.
                </div>
              )}

              <div className="relative">
                <Key className="absolute left-3 top-1/2 -translate-y-1/2 text-zinc-400" size={18} />
                <input
                  type="text"
                  value={inviteCode}
                  onChange={(e) => setInviteCode(e.target.value)}
                  placeholder={t('submitAgent.access.placeholder')}
                  className="w-full rounded-lg border border-zinc-200 py-3 pl-10 pr-4 text-sm outline-none focus:border-black focus:ring-1 focus:ring-black"
                />
              </div>

              {error && (
                <div className="text-sm text-red-600 flex items-center justify-center gap-1">
                  <AlertTriangle size={14} />
                  {error}
                </div>
              )}

              <button
                onClick={handleVerify}
                disabled={!inviteCode || isVerifying || loading}
                className="w-full rounded-lg bg-black py-3 text-sm font-bold text-white transition hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isVerifying ? (
                  <span className="flex items-center justify-center gap-2">
                    <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent"></div>
                    {t('submitAgent.access.check')}
                  </span>
                ) : (
                  <span className="flex items-center justify-center gap-2">
                    {!currentUser && <Twitter size={16} />}
                    {t('submitAgent.access.submit')}
                  </span>
                )}
              </button>
            </div>
          </div>
        ) : (
          <div className="rounded-2xl border border-emerald-100 bg-emerald-50/50 p-8 text-center animate-fade-in">
            <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-emerald-100 text-emerald-600">
              <CheckCircle size={32} />
            </div>
            <h2 className="text-2xl font-bold text-emerald-900 mb-2">{t('submitAgent.access.granted')}</h2>
            <p className="text-emerald-700">Assigned Agent Public Key: {currentUser?.agentPublicKey}</p>
          </div>
        )}

        <div className={`relative transition-all duration-500 ${!hasAccess ? 'opacity-40 grayscale blur-[2px] pointer-events-none select-none' : ''}`}>
          <div className="rounded-xl bg-zinc-900 p-8 shadow-2xl shadow-zinc-200">
            <label className="mb-4 block text-sm font-bold text-zinc-400 uppercase tracking-wider">{t('submitAgent.command.label')}</label>
            <div className="group relative flex items-center rounded-lg bg-black p-4 font-mono text-emerald-400">
              <Terminal size={20} className="mr-3 shrink-0 text-zinc-500" />
              <span className="break-all">{command}</span>
              <button
                onClick={handleCopy}
                disabled={!hasAccess}
                className="absolute right-2 top-1/2 -translate-y-1/2 rounded-md bg-zinc-800 p-2 text-zinc-400 hover:bg-zinc-700 hover:text-white transition-colors"
              >
                {copied ? <CheckCircle size={16} className="text-emerald-500" /> : <Copy size={16} />}
              </button>
            </div>
            <div className="mt-4 flex items-start gap-2 text-xs text-zinc-500">
              <Shield size={14} className="mt-0.5" />
              This command will install the local CLI tool and authenticate your session.
            </div>
          </div>
          {!hasAccess && (
            <div className="absolute inset-0 z-10 flex items-center justify-center">
              <div className="rounded-full bg-zinc-900/90 px-4 py-2 text-sm font-medium text-white backdrop-blur-sm shadow-lg">
                <Lock size={14} className="inline mr-2 mb-0.5" />
                Locked
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default SubmitAgent;

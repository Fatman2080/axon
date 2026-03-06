import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAppDispatch } from '../hooks/redux';
import { fetchUser } from '../store/slices/userSlice';
import { useLogin } from '../hooks/useLogin';
import { AlertCircle, Loader2, ArrowLeft, RefreshCw } from 'lucide-react';

const normalizeNextPath = (raw: string | null): string => {
  if (!raw) return '/';
  if (!raw.startsWith('/')) return '/';
  return raw;
};

const errorMessages: Record<string, string> = {
  x_oauth_token_exchange_failed: 'Failed to exchange authorization code. The code may have expired — please try again.',
  x_oauth_userinfo_failed: 'Logged in successfully, but failed to fetch your X profile. Please try again.',
  x_oauth_access_denied: 'You denied the authorization request. Please try again if this was a mistake.',
  x_oauth_not_configured: 'X OAuth is not configured on the server. Please contact the administrator.',
  invalid_oauth_callback: 'Invalid OAuth callback parameters. Please start the login process again.',
  invalid_or_expired_oauth_state: 'Your login session has expired. Please try again.',
  invalid_invite_code: 'The invite code is invalid or has been used.',
  agent_account_pool_empty: 'No agent accounts are available at this time. Please try again later.',
  no_slots_remaining: 'Today\'s registration slots are full. Please try again after reset.',
};

const getErrorMessage = (code: string): string => {
  return errorMessages[code] || `An unexpected error occurred (${code}). Please try again.`;
};

type Status = 'loading' | 'error' | 'profile_error';

const XAuthCallback = () => {
  const [status, setStatus] = useState<Status>('loading');
  const [errorCode, setErrorCode] = useState('');
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { login } = useLogin();

  const params = useMemo(() => new URLSearchParams(window.location.search), []);

  useEffect(() => {
    const token = params.get('token');
    const error = params.get('error');
    const next = normalizeNextPath(params.get('next'));

    const run = async () => {
      if (token) {
        localStorage.setItem('token', token);
        try {
          await dispatch(fetchUser()).unwrap();
          // Don't redirect back to the callback page itself
          const target = next.startsWith('/auth/x/callback') ? '/' : next;
          navigate(target, { replace: true });
          return;
        } catch {
          setStatus('profile_error');
          return;
        }
      }

      if (error) {
        setErrorCode(error);
        setStatus('error');
        return;
      }

      setErrorCode('invalid_oauth_callback');
      setStatus('error');
    };

    run();
  }, [dispatch, navigate, params]);

  if (status === 'loading') {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4">
        <Loader2 className="h-8 w-8 animate-spin text-zinc-400" />
        <p className="text-zinc-500 font-medium">Completing login...</p>
      </div>
    );
  }

  if (status === 'profile_error') {
    return (
      <div className="mx-auto max-w-md py-20 px-4">
        <div className="rounded-2xl border border-amber-200 bg-amber-50 p-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-amber-100">
            <AlertCircle className="h-6 w-6 text-amber-600" />
          </div>
          <h1 className="text-xl font-bold text-zinc-900 mb-2">Profile Load Failed</h1>
          <p className="text-sm text-zinc-600 mb-6">
            You were logged in successfully, but we couldn't load your profile. This is usually temporary.
          </p>
          <div className="flex flex-col gap-3">
            <button
              onClick={() => window.location.reload()}
              className="inline-flex items-center justify-center gap-2 h-10 rounded-full bg-zinc-900 text-white text-sm font-medium hover:bg-zinc-800 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
              Retry
            </button>
            <Link
              to="/"
              className="inline-flex items-center justify-center gap-2 h-10 rounded-full border border-zinc-200 text-zinc-700 text-sm font-medium hover:bg-zinc-50 transition-colors"
            >
              <ArrowLeft className="h-4 w-4" />
              Back to Home
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-md py-20 px-4">
      <div className="rounded-2xl border border-red-200 bg-red-50 p-8 text-center">
        <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100">
          <AlertCircle className="h-6 w-6 text-red-600" />
        </div>
        <h1 className="text-xl font-bold text-zinc-900 mb-2">Login Failed</h1>
        <p className="text-sm text-zinc-600 mb-6">
          {getErrorMessage(errorCode)}
        </p>
        <div className="flex flex-col gap-3">
          <button
            onClick={() => login(undefined, '/')}
            className="inline-flex items-center justify-center gap-2 h-10 rounded-full bg-zinc-900 text-white text-sm font-medium hover:bg-zinc-800 transition-colors"
          >
            <RefreshCw className="h-4 w-4" />
            Try Again
          </button>
          <Link
            to="/"
            className="inline-flex items-center justify-center gap-2 h-10 rounded-full border border-zinc-200 text-zinc-700 text-sm font-medium hover:bg-zinc-50 transition-colors"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Home
          </Link>
        </div>
      </div>
    </div>
  );
};

export default XAuthCallback;

import { authApi } from '../services/api';

export const useLogin = () => {
  const login = (inviteCode?: string, next?: string) => {
    const nextPath = next ?? window.location.pathname + window.location.search;
    window.location.href = authApi.getXOAuthStartUrl(inviteCode, nextPath);
  };

  return { login };
};

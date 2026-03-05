import React, { useEffect, useState } from 'react';
import { NavLink } from 'react-router-dom';
import { LayoutDashboard, Bot, User, Menu, X, Coins, Users, LogOut, Twitter } from 'lucide-react';
import { useAppDispatch, useAppSelector } from '../../hooks/redux';
import { logoutUser, fetchUser } from '../../store/slices/userSlice';
import { useLanguage } from '../../context/LanguageContext';
import { authApi } from '../../services/api';

const Layout = ({ children }: { children: React.ReactNode }) => {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const dispatch = useAppDispatch();
  const { currentUser: user } = useAppSelector((state) => state.user);
  const { language, setLanguage, t } = useLanguage();

  useEffect(() => {
    dispatch(fetchUser());
  }, [dispatch]);

  const navigation = [
    { name: t('nav.home'), href: '/', icon: LayoutDashboard },
    { name: t('nav.vault'), href: '/agents', icon: Coins },
    { name: t('nav.strategies'), href: '/strategies', icon: Users },
    { name: t('nav.submitAgent'), href: '/submit-agent', icon: Bot },
    { name: t('nav.profile'), href: '/profile', icon: User },
  ];

  const handleXLogin = (inviteCode?: string) => {
    const nextPath = window.location.pathname + window.location.search;
    window.location.href = authApi.getXOAuthStartUrl(inviteCode, nextPath);
  };

  const handleLogout = () => {
    dispatch(logoutUser());
  };

  return (
    <div className="min-h-screen bg-white text-zinc-900 font-sans">
      <div className="sticky top-0 z-50 flex items-center justify-between border-b border-zinc-100 bg-white/80 backdrop-blur-md px-6 py-4 md:hidden">
        <div className="flex items-center gap-2">
          <div className="h-6 w-6 bg-black"></div>
          <span className="font-bold text-lg tracking-tight">Clwafi</span>
        </div>
        <div className="flex items-center gap-4">
          <div className="flex items-center bg-zinc-100 rounded-lg p-0.5 scale-90">
            <button
              onClick={() => setLanguage('en')}
              className={`px-2 py-0.5 text-[10px] font-bold rounded-md transition-all ${
                language === 'en' ? 'bg-white shadow-sm text-black' : 'text-zinc-400 hover:text-black'
              }`}
            >
              EN
            </button>
            <button
              onClick={() => setLanguage('zh')}
              className={`px-2 py-0.5 text-[10px] font-bold rounded-md transition-all ${
                language === 'zh' ? 'bg-white shadow-sm text-black' : 'text-zinc-400 hover:text-black'
              }`}
            >
              中
            </button>
          </div>

          {user ? (
            <div className="flex items-center gap-2">
              <img src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'} alt={user.name || 'User'} className="h-8 w-8 rounded-full border border-zinc-200" />
            </div>
          ) : (
            <button
              onClick={() => handleXLogin()}
              className="flex items-center gap-2 rounded-full bg-black px-4 py-1.5 text-xs font-bold text-white transition hover:bg-zinc-800"
            >
              <Twitter size={14} fill="currentColor" />
              {t('nav.connectWallet')}
            </button>
          )}

          <button
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
            className="text-zinc-500 hover:text-black"
          >
            {isMobileMenuOpen ? <X size={24} /> : <Menu size={24} />}
          </button>
        </div>
      </div>

      <div className="flex max-w-[1600px] mx-auto">
        <aside
          className={`fixed inset-y-0 left-0 z-40 flex w-64 flex-col border-r border-zinc-100 bg-white transition-transform duration-300 ease-[cubic-bezier(0.22,1,0.36,1)] md:static md:translate-x-0 ${
            isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
          }`}
        >
          <div className="flex h-20 shrink-0 items-center px-6">
            <div className="h-6 w-6 bg-black mr-3"></div>
            <span className="text-xl font-bold tracking-tight">Clwafi</span>
          </div>

          <nav className="flex-1 space-y-1 px-4 mt-4">
            {navigation.map((item) => (
              <NavLink
                key={item.href}
                to={item.href}
                onClick={() => setIsMobileMenuOpen(false)}
                className={({ isActive }) =>
                  `flex items-center gap-3 rounded-md px-3 py-2.5 text-sm font-medium transition-all duration-200 ${
                    isActive
                      ? 'bg-zinc-100 text-black'
                      : 'text-zinc-500 hover:bg-zinc-50 hover:text-black'
                  }`
                }
              >
                <item.icon size={18} strokeWidth={2} />
                {item.name}
              </NavLink>
            ))}
          </nav>

          {user && (
            <div className="p-4 mt-auto border-t border-zinc-100">
              <div className="flex items-center gap-3 mb-3">
                <img src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'} alt={user.name || 'User'} className="h-8 w-8 rounded-full" />
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-bold truncate">{user.name || 'User'}</div>
                  <div className="text-xs text-zinc-500 truncate">{user.email || user.walletAddress}</div>
                </div>
              </div>
              <button
                onClick={handleLogout}
                className="flex w-full items-center justify-center gap-2 rounded-md border border-zinc-200 bg-white py-2 text-xs font-medium text-zinc-700 hover:bg-zinc-50 hover:text-black"
              >
                <LogOut size={14} />
                Logout
              </button>
            </div>
          )}
        </aside>

        <main className="min-h-[calc(100vh-64px)] w-full p-4 md:min-h-screen md:p-8 relative flex flex-col">
          <div className="hidden md:flex w-full justify-end items-center gap-4 mb-6 z-20">
            {user ? (
              <div className="flex items-center gap-3 bg-white pl-1 pr-4 py-1 rounded-full border border-zinc-200 shadow-sm">
                <img src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'} alt={user.name || 'User'} className="h-8 w-8 rounded-full" />
                <span className="text-sm font-bold">{user.name || 'User'}</span>
                <button onClick={handleLogout} className="ml-2 text-zinc-400 hover:text-red-500">
                  <LogOut size={16} />
                </button>
              </div>
            ) : (
              <button
                onClick={() => handleXLogin()}
                className="flex items-center gap-2 rounded-full bg-black px-5 py-2 text-sm font-bold text-white transition hover:bg-zinc-800 shadow-sm"
              >
                <Twitter size={16} fill="currentColor" />
                {t('nav.connectWallet')}
              </button>
            )}

            <div className="flex items-center bg-white border border-zinc-200 rounded-full p-1 shadow-sm">
              <button
                onClick={() => setLanguage('en')}
                className={`px-3 py-1 text-xs font-bold rounded-full transition-all ${
                  language === 'en' ? 'bg-black text-white' : 'text-zinc-500 hover:text-black'
                }`}
              >
                EN
              </button>
              <button
                onClick={() => setLanguage('zh')}
                className={`px-3 py-1 text-xs font-bold rounded-full transition-all ${
                  language === 'zh' ? 'bg-black text-white' : 'text-zinc-500 hover:text-black'
                }`}
              >
                中文
              </button>
            </div>
          </div>
          {children}
        </main>

        {isMobileMenuOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/20 backdrop-blur-sm md:hidden"
            onClick={() => setIsMobileMenuOpen(false)}
          ></div>
        )}
      </div>
    </div>
  );
};

export default Layout;

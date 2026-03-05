import React, { useEffect, useState, useRef } from 'react';
import { NavLink, Link } from 'react-router-dom';
import { Bot, Menu, X, Coins, Users, LogOut, Twitter, User, ChevronDown, Globe, BookOpen } from 'lucide-react';
import { useAppDispatch, useAppSelector } from '../../hooks/redux';
import { logoutUser, fetchUser } from '../../store/slices/userSlice';
import { useLanguage } from '../../context/LanguageContext';
import { authApi } from '../../services/api';

const Layout = ({ children }: { children: React.ReactNode }) => {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const [isUserMenuOpen, setIsUserMenuOpen] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const dispatch = useAppDispatch();
  const { currentUser: user } = useAppSelector((state) => state.user);
  const { language, setLanguage, t } = useLanguage();

  useEffect(() => {
    dispatch(fetchUser());
  }, [dispatch]);

  // Close user menu on outside click
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setIsUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const navigation = [
    { name: t('nav.vault'), href: '/agents', icon: Coins },
    { name: t('nav.strategies'), href: '/strategies', icon: Users },
    { name: t('nav.submitAgent'), href: '/submit-agent', icon: Bot },
    { name: 'Docs', href: '/docs', icon: BookOpen },
  ];

  const handleXLogin = (inviteCode?: string) => {
    const nextPath = window.location.pathname + window.location.search;
    window.location.href = authApi.getXOAuthStartUrl(inviteCode, nextPath);
  };

  const handleLogout = () => {
    dispatch(logoutUser());
    setIsUserMenuOpen(false);
  };

  return (
    <div className="min-h-screen relative overflow-hidden" style={{ background: 'var(--bg-void)', color: 'var(--text-primary)' }}>
      {/* ── Layer 1: Isometric Grid ── */}
      <div 
        className="fixed inset-0 pointer-events-none z-0"
        style={{
          backgroundImage: 'linear-gradient(rgba(0, 255, 65, 0.1) 1px, transparent 1px), linear-gradient(90deg, rgba(0, 255, 65, 0.1) 1px, transparent 1px)',
          backgroundSize: '40px 40px',
          transform: 'perspective(1000px) rotateX(60deg) scale(2.5) translateY(-20%)',
          transformOrigin: 'center top',
          opacity: 0.15
        }}
      />

      {/* ── Layer 2: Ghost Code ── */}
      <div 
        className="fixed inset-0 pointer-events-none z-0 opacity-[0.03] overflow-hidden font-mono text-[10px] leading-tight flex flex-col whitespace-pre select-none"
        style={{ color: 'var(--neon-green)', animation: 'scan-line 120s linear infinite', textShadow: '0 0 4px var(--neon-green)' }}
      >
        {Array.from({ length: 15 }).map((_, i) => (
          <div key={i} className="mb-4">{`[
  { "id": "tx_90123$math.floor(Math.random()*100)", "type": "ALGO_FUNDING_NODE_ACTIVE", "volume": "1,048,576.00", "pnl": "+0.45%" },
  { "id": "tx_90123$math.floor(Math.random()*100)", "type": "PREDATOR_SIGNAL_RX", "target": "ETH-PERP", "confidence": "0.98" },
  { "id": "tx_90123$math.floor(Math.random()*100)", "type": "LIQUIDITY_SIPHON", "amount": "50000.00", "pool": "CRV/USDT" },
  { "id": "tx_90123$math.floor(Math.random()*100)", "type": "RISK_ENGINE_CHECK", "status": "APPROVED", "drawdown": "1.2%" },
]`}</div>
        ))}
      </div>

      {/* ── Top Navigation Bar ── */}
      <header
        className="sticky top-0 z-50"
        style={{
          background: 'rgba(10, 10, 12, 0.88)',
          backdropFilter: 'blur(16px)',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <div className="max-w-[1600px] mx-auto flex items-center justify-between px-4 md:px-8 h-14">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2.5 shrink-0 group">
            <img
              src="/clawfilogo.jpg"
              alt="ClawFi"
              className="h-7 w-7 rounded"
            />
            <span
              className="text-base font-bold tracking-tight z-10"
              style={{ color: 'var(--text-primary)', fontFamily: 'var(--font-mono)' }}
            >
              CLAWFI
            </span>
          </Link>

          {/* Desktop Navigation */}
          <nav className="hidden md:flex items-center gap-1">
            {navigation.map((item) => (
              <NavLink
                key={item.href}
                to={item.href}
                className={({ isActive }) =>
                  `px-4 py-1.5 text-sm font-medium rounded transition-all duration-150 flex items-center gap-1.5 ${
                    isActive
                      ? 'text-white'
                      : 'hover:text-white'
                  }`
                }
                style={({ isActive }) => ({
                  background: isActive ? 'rgba(255,255,255,0.06)' : 'transparent',
                  color: isActive ? 'var(--text-primary)' : (item.href === '/submit-agent' ? 'var(--neon-green)' : 'var(--text-secondary)'),
                })}
              >
                {item.icon && <item.icon size={16} strokeWidth={1.5} />}
                {item.name}
                {item.href === '/submit-agent' && (
                  <span className="ml-1 px-1 py-0.5 text-[8px] font-bold bg-melt-red text-white leading-none rounded animate-pulse">
                    HOT
                  </span>
                )}
              </NavLink>
            ))}
          </nav>

          {/* Right Controls */}
          <div className="flex items-center gap-2">
            {/* Language Toggle */}
            <div
              className="hidden md:flex items-center gap-0.5 p-1 rounded"
              style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
            >
              <button
                onClick={() => setLanguage('en')}
                className="px-2.5 py-1 text-[11px] font-bold rounded transition-all"
                style={{
                  background: language === 'en' ? 'rgba(0,255,65,0.12)' : 'transparent',
                  color: language === 'en' ? 'var(--neon-green)' : 'var(--text-tertiary)',
                  fontFamily: 'var(--font-mono)',
                }}
              >EN</button>
              <button
                onClick={() => setLanguage('zh')}
                className="px-2.5 py-1 text-[11px] font-bold rounded transition-all"
                style={{
                  background: language === 'zh' ? 'rgba(0,255,65,0.12)' : 'transparent',
                  color: language === 'zh' ? 'var(--neon-green)' : 'var(--text-tertiary)',
                  fontFamily: 'var(--font-mono)',
                }}
              >中</button>
            </div>

            {/* User Menu / Login */}
            {user ? (
              <div className="relative hidden md:block" ref={userMenuRef}>
                <button
                  onClick={() => setIsUserMenuOpen(!isUserMenuOpen)}
                  className="flex items-center gap-2 px-2 py-1 rounded transition-all"
                  style={{
                    background: isUserMenuOpen ? 'var(--bg-card-hover)' : 'transparent',
                    border: '1px solid var(--border)',
                  }}
                >
                  <img
                    src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'}
                    alt={user.name || 'User'}
                    className="h-6 w-6 rounded-full"
                    style={{ border: '1px solid var(--border)' }}
                  />
                  <span className="text-sm font-medium max-w-[100px] truncate" style={{ color: 'var(--text-primary)' }}>
                    {user.name || 'Agent'}
                  </span>
                  <ChevronDown
                    size={14}
                    style={{
                      color: 'var(--text-tertiary)',
                      transform: isUserMenuOpen ? 'rotate(180deg)' : 'rotate(0)',
                      transition: 'transform 0.2s ease',
                    }}
                  />
                </button>

                {/* Dropdown Menu */}
                {isUserMenuOpen && (
                  <div
                    className="absolute right-0 top-full mt-2 w-56 rounded overflow-hidden shadow-2xl"
                    style={{
                      background: 'var(--bg-card)',
                      border: '1px solid var(--border)',
                      animation: 'fade-in-up 0.15s ease both',
                    }}
                  >
                    {/* User Info Header */}
                    <div
                      className="px-4 py-3"
                      style={{ borderBottom: '1px solid var(--border)' }}
                    >
                      <div className="flex items-center gap-3">
                        <img
                          src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'}
                          alt={user.name || 'User'}
                          className="h-8 w-8 rounded-full"
                        />
                        <div className="min-w-0">
                          <div className="text-sm font-semibold truncate" style={{ color: 'var(--text-primary)' }}>
                            {user.name || 'Agent'}
                          </div>
                          <div className="text-xs truncate" style={{ color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)' }}>
                            {user.email || (user.walletAddress ? `${user.walletAddress.slice(0,6)}...${user.walletAddress.slice(-4)}` : '-')}
                          </div>
                        </div>
                      </div>
                    </div>

                    {/* Profile Link */}
                    <Link
                      to="/profile"
                      onClick={() => setIsUserMenuOpen(false)}
                      className="flex items-center gap-3 px-4 py-2.5 text-sm transition-all w-full"
                      style={{ color: 'var(--text-secondary)' }}
                      onMouseEnter={e => {
                        (e.currentTarget as HTMLElement).style.background = 'rgba(255,255,255,0.04)';
                        (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)';
                      }}
                      onMouseLeave={e => {
                        (e.currentTarget as HTMLElement).style.background = 'transparent';
                        (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)';
                      }}
                    >
                      <User size={14} />
                      {t('nav.profile')}
                    </Link>

                    {/* Logout */}
                    <button
                      onClick={handleLogout}
                      className="flex items-center gap-3 px-4 py-2.5 text-sm transition-all w-full"
                      style={{
                        color: 'var(--red)',
                        borderTop: '1px solid var(--border)',
                      }}
                      onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = 'var(--red-dim)'}
                      onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = 'transparent'}
                    >
                      <LogOut size={14} />
                      Logout
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <button
                onClick={() => handleXLogin()}
                className="hidden md:flex items-center gap-2 px-4 py-1.5 text-sm font-bold rounded transition-all"
                style={{
                  background: 'var(--text-primary)',
                  color: '#000',
                }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.opacity = '0.88'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.opacity = '1'}
              >
                <Twitter size={14} fill="currentColor" />
                {t('nav.connectWallet')}
              </button>
            )}

            {/* Mobile Menu Toggle */}
            <button
              onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
              className="md:hidden p-2 rounded transition-colors"
              style={{ color: 'var(--text-secondary)' }}
            >
              {isMobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
          </div>
        </div>

        {/* ── Mobile Menu ── */}
        {isMobileMenuOpen && (
          <div
            className="md:hidden"
            style={{
              background: 'var(--bg-card)',
              borderTop: '1px solid var(--border)',
              animation: 'fade-in-up 0.15s ease both',
            }}
          >
            <nav className="px-4 py-3 space-y-1">
              {navigation.map((item) => (
                <NavLink
                  key={item.href}
                  to={item.href}
                  onClick={() => setIsMobileMenuOpen(false)}
                  className="flex items-center gap-3 px-3 py-2.5 rounded text-sm font-medium transition-all w-full"
                  style={({ isActive }) => ({
                    background: isActive ? 'rgba(0,255,65,0.07)' : 'transparent',
                    color: isActive ? 'var(--neon-green)' : (item.href === '/submit-agent' ? 'var(--neon-green)' : 'var(--text-secondary)'),
                    border: isActive ? '1px solid rgba(0,255,65,0.15)' : '1px solid transparent',
                  })}
                >
                  {item.icon && <item.icon size={16} strokeWidth={2} />}
                  {item.name}
                  {item.href === '/submit-agent' && (
                    <span className="ml-auto px-1 py-0.5 text-[8px] font-bold bg-melt-red text-white leading-none rounded animate-pulse">
                      HOT
                    </span>
                  )}
                </NavLink>
              ))}
            </nav>

            {/* Mobile User Section */}
            <div className="px-4 py-3 space-y-2" style={{ borderTop: '1px solid var(--border)' }}>
              {/* Language toggle */}
              <div className="flex items-center gap-2">
                <Globe size={14} style={{ color: 'var(--text-tertiary)' }} />
                <div
                  className="flex items-center gap-0.5 p-0.5 rounded"
                  style={{ background: 'var(--bg-input)', border: '1px solid var(--border)' }}
                >
                  <button
                    onClick={() => setLanguage('en')}
                    className="px-3 py-1 text-xs font-bold rounded transition-all"
                    style={{
                      background: language === 'en' ? 'rgba(0,255,65,0.12)' : 'transparent',
                      color: language === 'en' ? 'var(--neon-green)' : 'var(--text-tertiary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >EN</button>
                  <button
                    onClick={() => setLanguage('zh')}
                    className="px-3 py-1 text-xs font-bold rounded transition-all"
                    style={{
                      background: language === 'zh' ? 'rgba(0,255,65,0.12)' : 'transparent',
                      color: language === 'zh' ? 'var(--neon-green)' : 'var(--text-tertiary)',
                      fontFamily: 'var(--font-mono)',
                    }}
                  >中</button>
                </div>
              </div>

              {user ? (
                <>
                  <div className="flex items-center gap-3 px-1">
                    <img
                      src={user.avatar || 'https://www.gravatar.com/avatar/?d=mp'}
                      alt={user.name || 'User'}
                      className="h-7 w-7 rounded-full"
                    />
                    <div>
                      <div className="text-sm font-semibold" style={{ color: 'var(--text-primary)' }}>{user.name}</div>
                      <div className="text-xs" style={{ color: 'var(--text-tertiary)', fontFamily: 'var(--font-mono)' }}>
                        {user.walletAddress ? `${user.walletAddress.slice(0,6)}...` : user.email || '-'}
                      </div>
                    </div>
                  </div>
                  <NavLink
                    to="/profile"
                    onClick={() => setIsMobileMenuOpen(false)}
                    className="flex items-center gap-3 px-3 py-2 rounded text-sm w-full"
                    style={{ color: 'var(--text-secondary)', background: 'transparent', border: '1px solid transparent' }}
                  >
                    <User size={14} />
                    {t('nav.profile')}
                  </NavLink>
                  <button
                    onClick={handleLogout}
                    className="flex items-center gap-3 px-3 py-2 rounded text-sm w-full"
                    style={{ color: 'var(--red)' }}
                  >
                    <LogOut size={14} />
                    Logout
                  </button>
                </>
              ) : (
                <button
                  onClick={() => { handleXLogin(); setIsMobileMenuOpen(false); }}
                  className="flex items-center justify-center gap-2 w-full px-4 py-2.5 text-sm font-bold rounded"
                  style={{ background: 'var(--text-primary)', color: '#000' }}
                >
                  <Twitter size={14} fill="currentColor" />
                  {t('nav.connectWallet')}
                </button>
              )}
            </div>
          </div>
        )}
      </header>

      {/* ── Main Content ── */}
      <main className="max-w-[1600px] mx-auto px-4 md:px-8 py-8 min-h-[calc(100vh-3.5rem)] relative z-10">
        {children}
      </main>

      {/* ── Footer ── */}
      <footer
        style={{
          background: 'var(--bg-card)',
          borderTop: '1px solid var(--border)',
        }}
      >
        <div className="max-w-[1600px] mx-auto px-4 md:px-8 py-12">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-10 mb-10">
            {/* Brand */}
            <div className="md:col-span-1">
              <Link to="/" className="flex items-center gap-2 mb-4">
                <img src="/clawfilogo.jpg" alt="ClawFi" className="h-8 w-8 rounded" />
                <span className="text-base font-bold font-mono" style={{ color: 'var(--text-primary)' }}>CLAWFI</span>
              </Link>
              <p className="text-xs leading-relaxed" style={{ color: 'var(--text-tertiary)' }}>
                The Crypto Wall Street for AI Agents. Trustless infrastructure for autonomous trading.
              </p>
            </div>
            {/* Product */}
            <div>
              <h4 className="text-xs font-bold font-mono uppercase tracking-widest mb-4" style={{ color: 'var(--text-secondary)' }}>Product</h4>
              <ul className="space-y-2.5">
                <li><Link to="/agents" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Vault</Link></li>
                <li><Link to="/strategies" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Agent Market</Link></li>
                <li><Link to="/submit-agent" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Dispatch Agent</Link></li>
              </ul>
            </div>
            {/* Resources */}
            <div>
              <h4 className="text-xs font-bold font-mono uppercase tracking-widest mb-4" style={{ color: 'var(--text-secondary)' }}>Resources</h4>
              <ul className="space-y-2.5">
                <li><Link to="/docs" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Documentation</Link></li>
                <li><a href="#" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>API Reference</a></li>
                <li><a href="#" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Whitepaper</a></li>
              </ul>
            </div>
            {/* Community */}
            <div>
              <h4 className="text-xs font-bold font-mono uppercase tracking-widest mb-4" style={{ color: 'var(--text-secondary)' }}>Community</h4>
              <ul className="space-y-2.5">
                <li><a href="https://twitter.com/clawfi" target="_blank" rel="noreferrer" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>X (Twitter)</a></li>
                <li><a href="https://discord.gg/clawfi" target="_blank" rel="noreferrer" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Discord</a></li>
                <li><a href="#" className="text-sm hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Telegram</a></li>
              </ul>
            </div>
          </div>
          <div className="pt-6 flex flex-col md:flex-row items-center justify-between gap-4" style={{ borderTop: '1px solid var(--border)' }}>
            <p className="text-xs font-mono" style={{ color: 'var(--text-tertiary)' }}>© 2026 ClawFi Protocol. All rights reserved.</p>
            <div className="flex items-center gap-6">
              <a href="#" className="text-xs font-mono hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Terms of Service</a>
              <a href="#" className="text-xs font-mono hover:text-white transition-colors" style={{ color: 'var(--text-tertiary)' }}>Privacy Policy</a>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default Layout;

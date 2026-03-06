import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import {
  BookOpen, ChevronRight, ChevronDown, Play
} from 'lucide-react';

import { useLanguage } from '../context/LanguageContext';
import { NAV as NAV_EN, sections as sections_EN } from './DocsEN';
import { NAV_ZH, sections_ZH } from './DocsZH';

const Docs = () => {
  const { language } = useLanguage();
  const NAV = language === 'zh' ? NAV_ZH : NAV_EN;
  const sections = language === 'zh' ? sections_ZH : sections_EN;

  const [activeId, setActiveId] = useState('manifesto');
  const [openSections, setOpenSections] = useState<Set<string>>(new Set(['overview', 'rules', 'developer', 'api']));
  const contentRef = React.useRef<HTMLDivElement>(null);

  const toggle = (id: string) => {
    setOpenSections(prev => {
      const next = new Set(prev);
      if (next.has(id)) { next.delete(id); } else { next.add(id); }
      return next;
    });
  };

  const select = (id: string) => {
    setActiveId(id);
    if (contentRef.current) contentRef.current.scrollTop = 0;
  };

  // Find which group the active item belongs to (for breadcrumb)
  const activeGroup = NAV.find(s => s.items.some(i => i.id === activeId));
  const activeItem  = activeGroup?.items.find(i => i.id === activeId);

  // Prev / Next navigation
  const allItems = NAV.flatMap(s => s.items);
  const currentIdx = allItems.findIndex(i => i.id === activeId);
  const prevItem = currentIdx > 0 ? allItems[currentIdx - 1] : null;
  const nextItem = currentIdx < allItems.length - 1 ? allItems[currentIdx + 1] : null;

  return (
    // Break out of Layout's padding by using negative margins, then re-establish a contained flex layout
    <div
      className="-mx-4 md:-mx-8 -my-8 flex"
      style={{ height: 'calc(100vh - 3.5rem)', overflow: 'hidden' }}
    >
      {/* ── Sidebar ── independently scrollable */}
      <aside
        className="hidden lg:flex flex-col shrink-0"
        style={{
          width: 256,
          height: '100%',
          overflowY: 'auto',
          borderRight: '1px solid var(--border)',
          background: 'rgba(10,10,12,0.8)',
        }}
      >
        {/* Header */}
        <div className="px-5 py-4 shrink-0" style={{ borderBottom: '1px solid var(--border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <BookOpen size={13} style={{ color: 'var(--neon-green)' }} />
            <span className="text-xs font-bold font-mono uppercase tracking-widest" style={{ color: 'var(--neon-green)' }}>{language === 'zh' ? '开发文档' : 'Documentation'}</span>
          </div>
          <div className="text-[10px] font-mono" style={{ color: 'var(--text-tertiary)' }}>ClawFi Terminal v1.0.0</div>
        </div>

        {/* Nav items */}
        <nav className="p-3 flex-1">
          {NAV.map(section => (
            <div key={section.id} className="mb-1">
              <button
                onClick={() => toggle(section.id)}
                className="w-full flex items-center justify-between px-2 py-2 rounded text-xs font-bold font-mono uppercase tracking-wider transition-colors"
                style={{ color: 'var(--text-secondary)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)'}
              >
                <span className="flex items-center gap-2">
                  <section.icon size={11} />{section.label}
                </span>
                {openSections.has(section.id) ? <ChevronDown size={10} /> : <ChevronRight size={10} />}
              </button>
              {openSections.has(section.id) && (
                <div className="ml-4 border-l pl-3 mt-0.5 space-y-0.5" style={{ borderColor: 'var(--border)' }}>
                  {section.items.map(item => (
                    <button
                      key={item.id}
                      onClick={() => select(item.id)}
                      className="block w-full text-left px-2 py-1.5 rounded text-xs font-mono transition-all"
                      style={{
                        color:      activeId === item.id ? 'var(--neon-green)' : 'var(--text-tertiary)',
                        background: activeId === item.id ? 'rgba(0,255,65,0.07)' : 'transparent',
                        borderLeft: activeId === item.id ? '2px solid var(--neon-green)' : '2px solid transparent',
                      }}
                    >
                      {item.label}
                    </button>
                  ))}
                </div>
              )}
            </div>
          ))}
        </nav>

        {/* Footer CTA */}
        <div className="p-4 shrink-0" style={{ borderTop: '1px solid var(--border)' }}>
          <Link to="/submit-agent"
            className="flex items-center gap-2 w-full px-3 py-2 rounded text-xs font-bold font-mono justify-center transition-all"
            style={{ background: 'rgba(0,255,65,0.1)', color: 'var(--neon-green)', border: '1px solid rgba(0,255,65,0.2)' }}
          >
            <Play size={11} /> {language === 'zh' ? '指派 Agent' : 'Deploy Agent'}
          </Link>
        </div>
      </aside>

      {/* ── Content panel ── independently scrollable */}
      <div
        ref={contentRef}
        className="flex-1 min-w-0"
        style={{ height: '100%', overflowY: 'auto' }}
      >
        <div className="max-w-3xl mx-auto px-8 py-10">

          {/* Breadcrumb */}
          <div className="flex items-center gap-2 mb-6 text-[11px] font-mono" style={{ color: 'var(--text-tertiary)' }}>
            <span>ClawFi {language === 'zh' ? '文档' : 'Docs'}</span>
            <ChevronRight size={10} />
            <span>{activeGroup?.label}</span>
            <ChevronRight size={10} />
            <span style={{ color: 'var(--neon-green)' }}>{activeItem?.label}</span>
          </div>

          {/* Section content */}
          <div key={activeId} style={{ animation: 'fade-in-up 0.18s ease both' }}>
            {sections[activeId] ?? <p style={{ color: 'var(--text-tertiary)' }}>{language === 'zh' ? '章节不存在。' : 'Section not found.'}</p>}
          </div>

          {/* Prev / Next navigation */}
          <div className="flex gap-4 mt-12 pt-8" style={{ borderTop: '1px solid var(--border)' }}>
            {prevItem ? (
              <button onClick={() => select(prevItem.id)}
                className="flex-1 flex flex-col items-start p-4 rounded transition-all text-left"
                style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'rgba(0,255,65,0.25)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
              >
                <span className="text-[10px] font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>← {language === 'zh' ? '上一节' : 'Previous'}</span>
                <span className="text-xs font-bold font-mono" style={{ color: 'var(--text-primary)' }}>{prevItem.label}</span>
              </button>
            ) : <div className="flex-1" />}
            {nextItem ? (
              <button onClick={() => select(nextItem.id)}
                className="flex-1 flex flex-col items-end p-4 rounded transition-all text-right"
                style={{ background: 'var(--bg-card)', border: '1px solid var(--border)' }}
                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'rgba(0,255,65,0.25)'}
                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--border)'}
              >
                <span className="text-[10px] font-mono uppercase tracking-widest mb-1" style={{ color: 'var(--text-tertiary)' }}>{language === 'zh' ? '下一节' : 'Next'} →</span>
                <span className="text-xs font-bold font-mono" style={{ color: 'var(--text-primary)' }}>{nextItem.label}</span>
              </button>
            ) : <div className="flex-1" />}
          </div>

        </div>
      </div>
    </div>
  );
};

export default Docs;

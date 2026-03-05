/** @type {import('tailwindcss').Config} */

export default {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    container: { center: true },
    extend: {
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'Cascadia Code', 'monospace'],
      },
      colors: {
        void: '#0A0A0C',
        carbon: '#16181A',
        'neon-green': '#00FF41',
        'signal-green': '#00FF66',
        'melt-red': '#FF2A2A',
      },
      animation: {
        'fade-in-up': 'fade-in-up 0.5s ease both',
        'pulse-glow': 'pulse-glow 2s ease-in-out infinite',
      },
      keyframes: {
        'fade-in-up': {
          from: { opacity: '0', transform: 'translateY(12px)' },
          to: { opacity: '1', transform: 'translateY(0)' },
        },
        'pulse-glow': {
          '0%, 100%': { boxShadow: '0 0 6px 0 rgba(0, 255, 102, 0.4)' },
          '50%': { boxShadow: '0 0 16px 4px rgba(0, 255, 102, 0.7)' },
        },
      },
    },
  },
  plugins: [],
};


/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        // Backgrounds
        'bg-primary':    '#0d1117',
        'bg-secondary':  '#161b22',
        'bg-tertiary':   '#1c2128',
        'bg-elevated':   '#21262d',
        'border-subtle': '#30363d',

        // Text
        'text-primary':   '#e6edf3',
        'text-secondary': '#8b949e',
        'text-muted':     '#484f58',

        // Trading signals
        'buy-green':      '#2ea043',
        'sell-red':       '#f85149',
        'hold-amber':     '#d29922',

        // Regime colors
        'regime-trending': '#58a6ff',
        'regime-ranging':  '#8b949e',
        'regime-breakout': '#bc8cff',
        'regime-highvol':  '#f85149',
        'regime-lowvol':   '#3fb950',

        // Agent colors
        'agent-technical':   '#58a6ff',
        'agent-fundamental': '#d2a8ff',
        'agent-decision':    '#ffa657',
        'agent-regime':      '#79c0ff',
        'agent-meta':        '#f0883e',
        'agent-kta':         '#56d364',
        'agent-risk':        '#ff7b72',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      animation: {
        'fade-in-up': 'fadeInUp 0.3s ease-out',
        'pulse-slow':  'pulse 3s cubic-bezier(0.4,0,0.6,1) infinite',
        'slide-in':    'slideIn 0.2s ease-out',
      },
      keyframes: {
        fadeInUp: {
          '0%':   { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        slideIn: {
          '0%':   { opacity: '0', transform: 'translateX(12px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
      },
      boxShadow: {
        'glow-green': '0 0 12px rgba(46,160,67,0.35)',
        'glow-red':   '0 0 12px rgba(248,81,73,0.35)',
        'glow-blue':  '0 0 12px rgba(88,166,255,0.25)',
        'panel':      '0 4px 16px rgba(0,0,0,0.5)',
      },
    },
  },
  plugins: [],
}

import type { Config } from 'tailwindcss';

// GitHub-inspired palette. Kept minimal in phase 0; expanded later.
const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // GitHub-ish neutrals
        canvas: '#ffffff',
        'canvas-subtle': '#f6f8fa',
        border: '#d0d7de',
        muted: '#656d76',
        fg: '#1f2328',
        accent: '#0969da',
        success: '#1a7f37',
        danger: '#cf222e',
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          '"Segoe UI"',
          'Helvetica',
          'Arial',
          'sans-serif',
        ],
        mono: [
          'ui-monospace',
          'SFMono-Regular',
          'Menlo',
          'monospace',
        ],
      },
    },
  },
  plugins: [],
};

export default config;

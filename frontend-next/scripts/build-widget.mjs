// Standalone widget build script
// Usage: node scripts/build-widget.mjs
// Output: public/feedback-widget.js (can be served as static asset)

import * as esbuild from 'esbuild';
import { fileURLToPath } from 'url';
import path from 'path';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

await esbuild.build({
  entryPoints: [path.join(__dirname, '..', 'src', 'components', 'feedback', 'Widget.tsx')],
  bundle: true,
  outfile: path.join(__dirname, '..', 'public', 'feedback-widget.js'),
  format: 'iife',
  globalName: 'FeedbackWidget',
  jsx: 'automatic',
  jsxImportSource: 'react',
  loader: {
    '.tsx': 'tsx',
    '.ts': 'ts',
  },
  external: ['react', 'react-dom', 'next/*'],
  minify: true,
  sourcemap: false,
  define: {
    'process.env.NEXT_PUBLIC_API_URL': JSON.stringify(process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api'),
  },
  banner: {
    js: '/* Feedback Widget - embeddable UX feedback form */',
  },
});

console.log('Widget built: public/feedback-widget.js');

// @ts-check
import { defineConfig } from 'astro/config';
import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  site: 'https://pzmod.dev',
  // Astro's HTML minifier strips newline-whitespace at text/inline-element
  // boundaries (e.g. "text\n<a>" -> "text<a>"), swallowing intended spaces.
  // Keep it off so authored whitespace renders faithfully.
  compressHTML: false,
  vite: {
    plugins: [tailwindcss()],
  },
});

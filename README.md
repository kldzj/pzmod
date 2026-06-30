# pzmod.dev

The marketing site for [pzmod](https://github.com/kldzj/pzmod), built with
[Astro](https://astro.build) + [Tailwind CSS](https://tailwindcss.com).

It lives on the **`website` branch** (an orphan branch with no shared history with the code),
so the `main`/release branches stay clean. GitHub Actions builds and deploys it to GitHub
Pages on every push.

## Local development

```bash
npm install
npm run dev       # http://localhost:4321
npm run build     # static output -> dist/
npm run preview   # serve the built dist/
```

## Deploy (one-time setup)

1. Push this branch: `git push -u origin website`
2. Repo **Settings → Pages → Build and deployment → Source: GitHub Actions**.
3. `.github/workflows/pages.yml` then builds on every push to `website` and deploys.
4. **Custom domain** (`pzmod.dev`): the `public/CNAME` file is already set. In
   **Settings → Pages → Custom domain**, enter `pzmod.dev`, then add DNS records at your
   registrar:
   - **apex** `pzmod.dev`: four `A` records to GitHub Pages
     (`185.199.108.153`, `185.199.109.153`, `185.199.110.153`, `185.199.111.153`),
     or an `ALIAS`/`ANAME` to `kldzj.github.io`.
   - **www** (optional): `CNAME` → `kldzj.github.io`.
5. Tick **Enforce HTTPS** once the certificate is issued.

## Updating the demo GIFs

The site uses **un-captioned** recordings (the page supplies its own titles). Regenerate them
from the main repo with:

```bash
# from the pzmod code checkout (v3-rewrite/main):
scripts/record-website-gifs.sh /path/to/this/worktree/public
```

That records the same VHS tapes and demo fixture as the README GIFs but skips the ffmpeg
caption band. (The captioned versions in the code repo's `docs/assets/` come from
`scripts/record-hero.sh` and `scripts/record-demos.sh`.)

`public/install.sh` and `public/install.ps1` are **not committed here**. They are fetched from
the code branch at build time by `scripts/fetch-install.mjs` (an npm `prebuild` step), so the
main repo is the single source and the scripts never need editing on both branches. They are
served at `pzmod.dev/install.sh` and `pzmod.dev/install.ps1`. The fetch hard-fails in CI if a
script is missing; override the source ref with `PZMOD_SCRIPTS_REF`.

## Structure

```
src/pages/index.astro   composes the page sections
src/components/         Nav, Hero, Terminal, Features, Showcase, QuickStart, Why, FAQ, Sponsor, Footer
src/layouts/Base.astro  <head>, meta, Open Graph
src/styles/global.css   Tailwind v4 theme, mirrors the pzmod TUI palette
public/                 GIFs, favicon.svg, og.png, CNAME
```

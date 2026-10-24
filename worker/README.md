# Worker

Cloudflare Worker that serves the dashboard and the dataset JSON at https://benchmarks.goptics.org from Workers Static Assets. `public/` is build output (`task site:build`) and gitignored.

| URL | Serves |
|-----|--------|
| `/` | The dashboard; loads the catalog, then the first dataset |
| `/?id=<id>` | The dashboard with one dataset selected |
| `/<id>` | One dataset, no switcher |
| `/dataset` | The catalog, `[{id, name}]` in dashboard order |
| `/dataset/<id>` | One dataset as JSON |

Local: `task site:dev` builds against `http://localhost:8787/dataset` and runs `wrangler dev` (needs Node). `task site:deploy` rebuilds for production and deploys.

CI ([.github/workflows/site.yml](../.github/workflows/site.yml)) builds on pull requests and deploys on pushes to `main` that touch the bench data, the Taskfile, `worker/` or the workflow itself, with two repository secrets: `CLOUDFLARE_API_TOKEN` (Workers Scripts: Edit, plus Zone goptics.org: DNS Edit and Workers Routes Edit for the custom domain) and `CLOUDFLARE_ACCOUNT_ID`.

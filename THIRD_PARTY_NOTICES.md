# Third-party notices and distribution checklist

The root [Apache License 2.0](LICENSE) applies to OneMore's first-party work,
not to third-party packages, upstream services or independently licensed
material. The Git repository contains dependency manifests/lockfiles, not a
vendored dependency tree, prebuilt frontend bundle, base image or Go binary.

## Current dependencies

| Component | Pinned version | License and distribution notes |
| --- | --- | --- |
| [Go MySQL driver](https://github.com/go-sql-driver/mysql/tree/v1.10.2) | v1.10.2 | [MPL-2.0](https://github.com/go-sql-driver/mysql/blob/v1.10.2/LICENSE). Preserve its notices; when distributing a binary or image containing it, tell recipients how to obtain the exact corresponding MPL-covered source, including changes, under MPL-2.0 by reasonable means in a timely manner. OneMore's separate first-party files are not relicensed by this dependency. |
| [edwards25519](https://github.com/FiloSottile/edwards25519/tree/v1.2.0) | v1.2.0 | BSD-3-Clause, indirect Go dependency; reproduce the upstream license and copyright in binary distribution materials. |
| [Vue](https://github.com/vuejs/core/tree/v3.5.46) | 3.5.46 | MIT, including the matching Vue runtime packages. Include applicable copyright and full license text with a distributed bundle, not only a short minifier banner. |
| Vite, Vue plugin, Vue TSC, TypeScript and Vitest | 6.4.6, 5.2.4, 2.2.12, 5.7.3 and 2.1.9, respectively | Build/test tools, not shipped as dependencies by this source checkout. MIT except TypeScript (Apache-2.0). Review actual emitted code and bundled third-party material before distribution; the development/runtime flags in the npm lockfile are not an artifact inventory. |

See [go.mod](go.mod), [go.sum](go.sum) and [web/package-lock.json](web/package-lock.json)
for the exact graph. The source checkout does not include third-party model
weights, fonts, UI templates or demo images. The Dockerfile and Compose refer
to external images; their contents are outside OneMore's Apache-2.0 grant.

**Before distributing a binary, web bundle or container**: inspect the actual
artifact, including Go runtime, bundled JavaScript, assets, and base-image
packages; collect their license texts, required notices, and applicable
corresponding source. Include an accessible source location for the driver
at the exact build version (and any patches), and communicate that location
with the delivered artifact. Review and retain notices for any redistributed
database image separately. See the [MPL-2.0 text](https://www.mozilla.org/en-US/MPL/2.0/)
and [Mozilla's executable distribution FAQ](https://www.mozilla.org/en-US/MPL/2.0/FAQ/).
This file is a checklist, not a completed artifact compliance record.

## Future reuse policy

Pin the exact upstream commit or package version before using it. Review the
selected paths, file headers, dependency graph, assets, LICENSE/NOTICE and any
additional terms. Record provenance, changes, distribution obligations and
required copyright/license notices; keep imported files under their original
terms. Do not merge materials with unresolved provenance or conflicting terms.

- [Original One API (songquanpeng/one-api)](https://github.com/songquanpeng/one-api/blob/main/LICENSE):
  MIT permits selected code reuse subject to the review above. Preserve the
  original copyright and complete MIT notice on reused files and substantial
  portions in distribution; OneMore's Apache-2.0 grant does not replace them.
- [New API (QuantumNous/new-api)](https://github.com/QuantumNous/new-api/blob/main/LICENSE):
  a different project under AGPL-3.0; review the chosen revision's repository
  notices, attribution and network-source obligations separately. Do not copy
  it into OneMore's Apache-licensed core without a separate compatibility and
  distribution review. Evaluating an independent service over its API does not
  approve bundling, modification or redistribution.
- [Element Plus](https://github.com/element-plus/element-plus/blob/dev/LICENSE)
  (MIT) and the [official OpenAI Go SDK](https://github.com/openai/openai-go/blob/main/LICENSE)
  (Apache-2.0) are candidates only. Check exact versions, dependency licenses,
  NOTICE requirements and any assets before adding either. No candidate is
  imported by this licensing change.

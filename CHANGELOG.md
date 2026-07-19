# Changelog

## [0.10.1](https://github.com/fredrikaverpil/pocket/compare/v0.10.0...v0.10.1) (2026-07-19)


### Bug Fixes

* **pk:** report scoped no-ops and missing git root ([#117](https://github.com/fredrikaverpil/pocket/issues/117)) ([94de509](https://github.com/fredrikaverpil/pocket/commit/94de509dcbaaf184e05a06b0198daefd334547ca))

## [0.10.0](https://github.com/fredrikaverpil/pocket/compare/v0.9.1...v0.10.0) (2026-07-19)


### Features

* **deps:** update dependency goreleaser/goreleaser to v2.17.0 ([#99](https://github.com/fredrikaverpil/pocket/issues/99)) ([3b72e98](https://github.com/fredrikaverpil/pocket/commit/3b72e985c7ca9db7865a573b4446ab8c32b80bb5))
* **deps:** update dependency johnnymorganz/stylua to v2.5.2 ([#96](https://github.com/fredrikaverpil/pocket/issues/96)) ([7ffbfc0](https://github.com/fredrikaverpil/pocket/commit/7ffbfc0191cac65420dc587b494a8ee83822b6ba))
* **deps:** update module golang.org/x/vuln to v1.6.0 ([#115](https://github.com/fredrikaverpil/pocket/issues/115)) ([1172b09](https://github.com/fredrikaverpil/pocket/commit/1172b0933561761afc82825f2933d1262f7bb22d))
* use rumdl as markdown formatter ([#118](https://github.com/fredrikaverpil/pocket/issues/118)) ([36addd4](https://github.com/fredrikaverpil/pocket/commit/36addd4af21d8de77c39366975a3f0bef0ea6eac))


### Bug Fixes

* **deps:** update dependency astral-sh/uv to v0.11.29 ([#93](https://github.com/fredrikaverpil/pocket/issues/93)) ([3f7ec20](https://github.com/fredrikaverpil/pocket/commit/3f7ec2086b4e5b9e02b506c98c4398a0a936b07c))
* **deps:** update dependency neovim/neovim to v0.12.3 ([#109](https://github.com/fredrikaverpil/pocket/issues/109)) ([2987d73](https://github.com/fredrikaverpil/pocket/commit/2987d73e254773c1b4669c92f9f26f272dbc6307))
* **deps:** update dependency neovim/neovim to v0.12.4 ([#121](https://github.com/fredrikaverpil/pocket/issues/121)) ([c965f45](https://github.com/fredrikaverpil/pocket/commit/c965f4566571ff106dda0d820481e6a6ca254c6a))
* **deps:** update dependency oven-sh/bun to v1.3.14 ([#94](https://github.com/fredrikaverpil/pocket/issues/94)) ([fae800f](https://github.com/fredrikaverpil/pocket/commit/fae800f3a2f4def50a129fe37ca618ee96e2027b))
* **deps:** update dependency prettier to v3.9.5 ([#102](https://github.com/fredrikaverpil/pocket/issues/102)) ([68ca0ad](https://github.com/fredrikaverpil/pocket/commit/68ca0ad7cbd56fa0f098626bc2711bb9705e8fda))
* **deps:** update dependency tree-sitter/tree-sitter to v0.26.11 ([#98](https://github.com/fredrikaverpil/pocket/issues/98)) ([ca804f6](https://github.com/fredrikaverpil/pocket/commit/ca804f685e163e64c0186f31e9621d106f21f5b7))
* **deps:** update dependency zensical to v0.0.50 ([#95](https://github.com/fredrikaverpil/pocket/issues/95)) ([4625ab8](https://github.com/fredrikaverpil/pocket/commit/4625ab8cfc6df6b5ab1e2eef0a3836389c9ad790))
* **deps:** update module golang.org/x/sync to v0.22.0 ([#100](https://github.com/fredrikaverpil/pocket/issues/100)) ([69406ec](https://github.com/fredrikaverpil/pocket/commit/69406ece29a28d5d1ed62c909828f4380d3b20a2))
* **deps:** update module golang.org/x/sync to v0.22.0 ([#124](https://github.com/fredrikaverpil/pocket/issues/124)) ([037865c](https://github.com/fredrikaverpil/pocket/commit/037865ce10fc64020cd70b0bf900288c4f63c559))
* **deps:** update module golang.org/x/term to v0.45.0 ([#101](https://github.com/fredrikaverpil/pocket/issues/101)) ([3ace0bc](https://github.com/fredrikaverpil/pocket/commit/3ace0bc767c7d37f3785f6fe892fae8de9715afb))
* **run:** handle Windows Path environment variable ([#116](https://github.com/fredrikaverpil/pocket/issues/116)) ([28321d2](https://github.com/fredrikaverpil/pocket/commit/28321d2885f59b0844018839af90b60c345cb08a))

## [0.9.1](https://github.com/fredrikaverpil/pocket/compare/v0.9.0...v0.9.1) (2026-06-22)


### Bug Fixes

* **pk:** check path exclusion before marking task as executed ([#105](https://github.com/fredrikaverpil/pocket/issues/105)) ([8842308](https://github.com/fredrikaverpil/pocket/commit/884230819efb46d6a1a52f4e5dfbba2e14f323a8))
* **pk:** error on non-convertible flag values in mapToStruct ([#111](https://github.com/fredrikaverpil/pocket/issues/111)) ([5d25f5b](https://github.com/fredrikaverpil/pocket/commit/5d25f5b4f8f5717fb62a81b9f4ed8ddef9edf263))
* **pk:** narrow nested path scopes to the enclosing scope's directory ([#106](https://github.com/fredrikaverpil/pocket/issues/106)) ([763f3ed](https://github.com/fredrikaverpil/pocket/commit/763f3ed16aab290b7ecce71533a86ed5875f9104))
* **pk:** reject pathFilter inside Task.Body at plan time ([#108](https://github.com/fredrikaverpil/pocket/issues/108)) ([7421423](https://github.com/fredrikaverpil/pocket/commit/74214237790357a9bf1dd77cd9a882f35c16a349))
* **pk:** scope CLI flag overrides to the invoked task ([#110](https://github.com/fredrikaverpil/pocket/issues/110)) ([3a1c190](https://github.com/fredrikaverpil/pocket/commit/3a1c190cc9d5d96bea0cd646fcb5ef371200aebf))
* **pk:** store pathFilter resolution off shared composition nodes ([#107](https://github.com/fredrikaverpil/pocket/issues/107)) ([0a2c48c](https://github.com/fredrikaverpil/pocket/commit/0a2c48c59c91d2130ade5e6ebb1b8c26c215b524))
* **pk:** union paths when a task is referenced in multiple scopes ([#103](https://github.com/fredrikaverpil/pocket/issues/103)) ([15794d0](https://github.com/fredrikaverpil/pocket/commit/15794d0bb283b7d06e431721e2dee7a50693ef39))
* **pk:** use per-occurrence path filter resolutions ([#113](https://github.com/fredrikaverpil/pocket/issues/113)) ([39c2046](https://github.com/fredrikaverpil/pocket/commit/39c204694622c8831315144299fbe42c6cd5fe37))

## [0.9.0](https://github.com/fredrikaverpil/pocket/compare/v0.8.2...v0.9.0) (2026-05-14)


### Features

* **pk:** add JSON-driven task execution for agents ([#90](https://github.com/fredrikaverpil/pocket/issues/90)) ([986d0e5](https://github.com/fredrikaverpil/pocket/commit/986d0e592f4d9110fb6a7d7832cfa4be054632d5))


### Bug Fixes

* **deps:** update dependency astral-sh/uv to v0.11.13 ([#86](https://github.com/fredrikaverpil/pocket/issues/86)) ([7a12f2e](https://github.com/fredrikaverpil/pocket/commit/7a12f2e37b4930e501d75b5f39eeaa816c4a2cfa))
* **deps:** update dependency zensical to v0.0.41 ([#89](https://github.com/fredrikaverpil/pocket/issues/89)) ([61f93ba](https://github.com/fredrikaverpil/pocket/commit/61f93ba9f4ba037544e6cd9c9d97efda67e8e7e5))
* **deps:** update module github.com/golangci/golangci-lint/v2 to v2.12.2 ([#87](https://github.com/fredrikaverpil/pocket/issues/87)) ([c41bf78](https://github.com/fredrikaverpil/pocket/commit/c41bf781f879e62aa0dc00c80664601944e6e805))
* **deps:** update module golang.org/x/term to v0.43.0 ([#88](https://github.com/fredrikaverpil/pocket/issues/88)) ([db4386c](https://github.com/fredrikaverpil/pocket/commit/db4386ce6e7ce6a8f5058ed2a1d2b2a3442b153c))

## [0.8.2](https://github.com/fredrikaverpil/pocket/compare/v0.8.1...v0.8.2) (2026-05-07)


### Bug Fixes

* **deps:** update dependency zensical to v0.0.40 ([38ca703](https://github.com/fredrikaverpil/pocket/commit/38ca70328bed84324165396f01c87d8bb232c651))
* **renovate:** use fix semantic commit type for custom regex deps ([#84](https://github.com/fredrikaverpil/pocket/issues/84)) ([afee651](https://github.com/fredrikaverpil/pocket/commit/afee65128d57180a6857d14081ef69d6ca985f90))

## [0.8.1](https://github.com/fredrikaverpil/pocket/compare/v0.8.0...v0.8.1) (2026-04-25)


### Bug Fixes

* **deps:** update dependency zensical to v0.0.34 ([#57](https://github.com/fredrikaverpil/pocket/issues/57)) ([6dc8e2f](https://github.com/fredrikaverpil/pocket/commit/6dc8e2f5d52bc8c0ede55153c77801ffa5476f50))
* **self-update:** use GOPRIVATE to scope proxy and sum bypass to pocket ([#76](https://github.com/fredrikaverpil/pocket/issues/76)) ([059b8d9](https://github.com/fredrikaverpil/pocket/commit/059b8d9c26879bfd45f05600ccfa3521a6c375a6))

## [0.8.0](https://github.com/fredrikaverpil/pocket/compare/v0.7.3...v0.8.0) (2026-04-25)


### Features

* add -s/--serial flag to force sequential task execution ([#74](https://github.com/fredrikaverpil/pocket/issues/74)) ([3d9f363](https://github.com/fredrikaverpil/pocket/commit/3d9f363e81543fd8c4205e93d466c4fcdbd2b786))

## [0.7.3](https://github.com/fredrikaverpil/pocket/compare/v0.7.2...v0.7.3) (2026-04-25)


### Bug Fixes

* **deps:** update dependency prettier to v3.8.3 ([#59](https://github.com/fredrikaverpil/pocket/issues/59)) ([3f808e6](https://github.com/fredrikaverpil/pocket/commit/3f808e6c811e9dd9ffd9189567dc1e1c8c387fe7))
* **nvim:** bump stable to 0.12.2 ([#68](https://github.com/fredrikaverpil/pocket/issues/68)) ([471f809](https://github.com/fredrikaverpil/pocket/commit/471f809e2fde3d1dd9237a2b4c1725945984bbea))
* **renovate:** find new neovim/python version ([#70](https://github.com/fredrikaverpil/pocket/issues/70)) ([555c41a](https://github.com/fredrikaverpil/pocket/commit/555c41ab73a63c51923e43cd5bc6687f2c8ac8a6))

## [0.7.2](https://github.com/fredrikaverpil/pocket/compare/v0.7.1...v0.7.2) (2026-04-25)


### Bug Fixes

* **pk:** scope auto execution from subdirectory shims ([#67](https://github.com/fredrikaverpil/pocket/issues/67)) ([ac4238f](https://github.com/fredrikaverpil/pocket/commit/ac4238fdff41022e9cac667380aa731f3df7c2a8))
* **shim:** add generated file headers ([#64](https://github.com/fredrikaverpil/pocket/issues/64)) ([cdc5b8f](https://github.com/fredrikaverpil/pocket/commit/cdc5b8f58ebc79a2ea8004a8bbe89a093c3f0c33))

## [0.7.1](https://github.com/fredrikaverpil/pocket/compare/v0.7.0...v0.7.1) (2026-04-25)


### Bug Fixes

* **gha:** invalid field ([#61](https://github.com/fredrikaverpil/pocket/issues/61)) ([4e21496](https://github.com/fredrikaverpil/pocket/commit/4e2149616fc87637da142b3cdfdd8cac9c1c617f))

## [0.7.0](https://github.com/fredrikaverpil/pocket/compare/v0.6.0...v0.7.0) (2026-04-16)


### Features

* **go:** bump to 1.26.2 ([#55](https://github.com/fredrikaverpil/pocket/issues/55)) ([513ecb9](https://github.com/fredrikaverpil/pocket/commit/513ecb913264a3ee80a76816f5a7dcb0f02eedf9))


### Bug Fixes

* **deps:** update dependency prettier to v3.8.2 ([#53](https://github.com/fredrikaverpil/pocket/issues/53)) ([b4eeaff](https://github.com/fredrikaverpil/pocket/commit/b4eeaff886d13cde27e34fdb7784d7690dc2a311))
* **deps:** update dependency zensical to v0.0.32 ([#47](https://github.com/fredrikaverpil/pocket/issues/47)) ([ced574d](https://github.com/fredrikaverpil/pocket/commit/ced574d3099360f350d29c6e95bb5f365ff29dcb))

## [0.6.0](https://github.com/fredrikaverpil/pocket/compare/v0.5.0...v0.6.0) (2026-03-23)


### Features

* **pk:** add WithVerbose option with single propagation path ([#41](https://github.com/fredrikaverpil/pocket/issues/41)) ([fcdfdcd](https://github.com/fredrikaverpil/pocket/commit/fcdfdcd77e24aa3dc5f3eacbacd9c317f3330b1f))

## [0.5.0](https://github.com/fredrikaverpil/pocket/compare/v0.4.1...v0.5.0) (2026-03-23)


### Features

* **pk:** add WithVerbose option to force streamed output ([8d4693c](https://github.com/fredrikaverpil/pocket/commit/8d4693cc4a0d66116a4cbcbcf2452e50b321edd5))
* **pk:** add WithVerbose option to force streamed output ([#40](https://github.com/fredrikaverpil/pocket/issues/40)) ([13a14c1](https://github.com/fredrikaverpil/pocket/commit/13a14c1036f0ccc76912d361b199ff37c57a018d))

## [0.4.1](https://github.com/fredrikaverpil/pocket/compare/v0.4.0...v0.4.1) (2026-03-21)


### Bug Fixes

* **pagefind:** use non-extended release archive ([#37](https://github.com/fredrikaverpil/pocket/issues/37)) ([810054c](https://github.com/fredrikaverpil/pocket/commit/810054caacd3392447a16c49341504f8daf4e4aa))

## [0.4.0](https://github.com/fredrikaverpil/pocket/compare/v0.3.1...v0.4.0) (2026-03-21)


### Features

* **pagefind:** add pagefind tool package ([#35](https://github.com/fredrikaverpil/pocket/issues/35)) ([cce23fc](https://github.com/fredrikaverpil/pocket/commit/cce23fc4c352ca834b58da44a52465c3bc161f97))

## [0.3.1](https://github.com/fredrikaverpil/pocket/compare/v0.3.0...v0.3.1) (2026-03-21)


### Bug Fixes

* **github:** skip external workflows during managed file cleanup ([#33](https://github.com/fredrikaverpil/pocket/issues/33)) ([7d83713](https://github.com/fredrikaverpil/pocket/commit/7d837133376ff660c75c63773e7a95cba01162d8))

## [0.3.0](https://github.com/fredrikaverpil/pocket/compare/v0.2.1...v0.3.0) (2026-03-21)


### Features

* **github:** add ExternalWorkflows validation to workflow task ([#32](https://github.com/fredrikaverpil/pocket/issues/32)) ([3d16399](https://github.com/fredrikaverpil/pocket/commit/3d1639994eac6c5a61a35a580a6afc4c21b4663e))
* **renovate:** add 3-day minimum release age ([#30](https://github.com/fredrikaverpil/pocket/issues/30)) ([ca024f5](https://github.com/fredrikaverpil/pocket/commit/ca024f5cfe8bafc19cf7041ef474d9860d788d37))

## [0.2.1](https://github.com/fredrikaverpil/pocket/compare/v0.2.0...v0.2.1) (2026-03-20)


### Bug Fixes

* **deps:** update dependency zensical to v0.0.28 ([#25](https://github.com/fredrikaverpil/pocket/issues/25)) ([f3ccfbb](https://github.com/fredrikaverpil/pocket/commit/f3ccfbb3892aacb402c9f94c675f3ae6a6f42de3))

## [0.2.0](https://github.com/fredrikaverpil/pocket/compare/v0.1.1...v0.2.0) (2026-03-19)


### Features

* **docs:** categorize WithOptions into distinct option types ([#23](https://github.com/fredrikaverpil/pocket/issues/23)) ([33ef72c](https://github.com/fredrikaverpil/pocket/commit/33ef72c8254243c1afec26bf5eb4c1ca2d0d5c57))
* publish new api for tasks/tools ([#21](https://github.com/fredrikaverpil/pocket/issues/21)) ([73ca3b4](https://github.com/fredrikaverpil/pocket/commit/73ca3b4198846537db578744ff71195f47c828d8))

## [0.1.1](https://github.com/fredrikaverpil/pocket/compare/v0.1.0...v0.1.1) (2026-03-15)


### Bug Fixes

* **deps:** update module golang.org/x/sync to v0.20.0 ([#8](https://github.com/fredrikaverpil/pocket/issues/8)) ([41d7959](https://github.com/fredrikaverpil/pocket/commit/41d79594142f3d79a6ae5af0437bb540331fc651))
* **deps:** update module golang.org/x/term to v0.41.0 ([#9](https://github.com/fredrikaverpil/pocket/issues/9)) ([df9a488](https://github.com/fredrikaverpil/pocket/commit/df9a488d35b196509a4c56aa286ce5a4f86fe2c4))
* tighten conventional commit validation ([#15](https://github.com/fredrikaverpil/pocket/issues/15)) ([c87482f](https://github.com/fredrikaverpil/pocket/commit/c87482f52f10104bed50ea3025dd7d37b05077c7))

## 0.1.0 (2026-03-15)


### Features

* initial commit ([2bc13fd](https://github.com/fredrikaverpil/pocket/commit/2bc13fdca3bc8cf7a42389ad92feabca7ccc75ef))

<!-- markdownlint-disable MD024 -->
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Chores

- Update module charm.land/bubbletea/v2 to v2.1.0 by @renovate[bot] in [#662](https://github.com/nicholas-fedor/go-remove/pull/662)
- Update github.com/charmbracelet/ultraviolet digest to 6b8d4ba by @renovate[bot] in [#658](https://github.com/nicholas-fedor/go-remove/pull/658)
- Update go module directive to v1.27.2 by @renovate[bot] in [#659](https://github.com/nicholas-fedor/go-remove/pull/659)

## [0.4.0] - 2026-10-08

### Added

- **Breaking:** Replace mode flags with subcommands by @nicholas-fedor in [#649](https://github.com/nicholas-fedor/go-remove/pull/649)
- Use a go-remove managed trash on Windows by @nicholas-fedor in [#619](https://github.com/nicholas-fedor/go-remove/pull/619)

### Fixed

- Omit the commit scope from changelog entries by @nicholas-fedor in [#656](https://github.com/nicholas-fedor/go-remove/pull/656)
- Keep database files private to their owner by @nicholas-fedor in [#646](https://github.com/nicholas-fedor/go-remove/pull/646)
- Mark an oversized transaction by @nicholas-fedor in [#645](https://github.com/nicholas-fedor/go-remove/pull/645)
- Stop falling back to the executable directory by @nicholas-fedor in [#644](https://github.com/nicholas-fedor/go-remove/pull/644)
- Align the history table columns by @nicholas-fedor in [#643](https://github.com/nicholas-fedor/go-remove/pull/643)
- Stop a stale operation result clobbering newer state by @nicholas-fedor in [#640](https://github.com/nicholas-fedor/go-remove/pull/640)
- Stop persisting the full build info by @nicholas-fedor in [#635](https://github.com/nicholas-fedor/go-remove/pull/635)
- Size and truncate by terminal display width by @nicholas-fedor in [#630](https://github.com/nicholas-fedor/go-remove/pull/630)
- Run operations outside the update loop by @nicholas-fedor in [#629](https://github.com/nicholas-fedor/go-remove/pull/629)
- Give the work a real cancellable context by @nicholas-fedor in [#628](https://github.com/nicholas-fedor/go-remove/pull/628)
- Ignore a relative XDG_DATA_HOME by @nicholas-fedor in [#627](https://github.com/nicholas-fedor/go-remove/pull/627)
- Report failures accurately and keep arguments in the bin directory by @nicholas-fedor in [#626](https://github.com/nicholas-fedor/go-remove/pull/626)
- Stop discarding TUI log output and honour --log-level by @nicholas-fedor in [#624](https://github.com/nicholas-fedor/go-remove/pull/624)
- Record a deletion before moving the binary by @nicholas-fedor in [#622](https://github.com/nicholas-fedor/go-remove/pull/622)
- Escape the delimiters a desktop splits the Path value on by @nicholas-fedor in [#621](https://github.com/nicholas-fedor/go-remove/pull/621)
- Make move and restore atomic by @nicholas-fedor in [#617](https://github.com/nicholas-fedor/go-remove/pull/617)
- Make undo skip unrestorable records by @nicholas-fedor in [#615](https://github.com/nicholas-fedor/go-remove/pull/615)
- Make trash and history key allocation collision-proof by @nicholas-fedor in [#614](https://github.com/nicholas-fedor/go-remove/pull/614)
- Isolate trash tests from the user's real trash by @nicholas-fedor in [#612](https://github.com/nicholas-fedor/go-remove/pull/612)

### Changed

- Capture through a hook, decided once per event by @nicholas-fedor in [#638](https://github.com/nicholas-fedor/go-remove/pull/638)
- Keep zerolog out of the Logger interface by @nicholas-fedor in [#637](https://github.com/nicholas-fedor/go-remove/pull/637)
- Remove dead code and regenerate the fs mocks by @nicholas-fedor in [#634](https://github.com/nicholas-fedor/go-remove/pull/634)
- Split the interface into tui, models and render by @nicholas-fedor in [#633](https://github.com/nicholas-fedor/go-remove/pull/633)
- Collapse the duplicated linux and darwin implementations by @nicholas-fedor in [#616](https://github.com/nicholas-fedor/go-remove/pull/616)

### Documentation

- Correct trash locations and stale links by @nicholas-fedor in [#641](https://github.com/nicholas-fedor/go-remove/pull/641)

### Tests

- Delete the mock-only integration suites by @nicholas-fedor in [#613](https://github.com/nicholas-fedor/go-remove/pull/613)

### Continuous Integration

- Add changelog generation with git-cliff by @nicholas-fedor in [#652](https://github.com/nicholas-fedor/go-remove/pull/652)

### Chores

- Update step-security/harden-runner action to v2.22.1 by @renovate[bot] in [#655](https://github.com/nicholas-fedor/go-remove/pull/655)
- Update github/codeql-action digest to 24c5418 by @renovate[bot] in [#653](https://github.com/nicholas-fedor/go-remove/pull/653)
- Update step-security/harden-runner action to v2.22.1 by @renovate[bot] in [#651](https://github.com/nicholas-fedor/go-remove/pull/651)
- Update module github.com/mattn/go-runewidth to v0.0.31 by @renovate[bot] in [#650](https://github.com/nicholas-fedor/go-remove/pull/650)
- Update step-security/harden-runner action to v2.22.0 by @renovate[bot] in [#648](https://github.com/nicholas-fedor/go-remove/pull/648)
- Update module github.com/mattn/go-colorable to v0.1.16 by @renovate[bot] in [#647](https://github.com/nicholas-fedor/go-remove/pull/647)
- Verify checksums by exact file name by @nicholas-fedor in [#642](https://github.com/nicholas-fedor/go-remove/pull/642)
- Tidy comments and declaration order project-wide by @nicholas-fedor in [#639](https://github.com/nicholas-fedor/go-remove/pull/639)
- Update anchore/sbom-action action to v0.24.3 by @renovate[bot] in [#636](https://github.com/nicholas-fedor/go-remove/pull/636)
- Update opentelemetry-go monorepo to v1.47.0 by @renovate[bot] in [#632](https://github.com/nicholas-fedor/go-remove/pull/632)
- Update github.com/charmbracelet/ultraviolet digest to 8786532 by @renovate[bot] in [#631](https://github.com/nicholas-fedor/go-remove/pull/631)
- Update github.com/charmbracelet/ultraviolet digest to 666ce5e by @renovate[bot] in [#625](https://github.com/nicholas-fedor/go-remove/pull/625)
- Update nicholas-fedor/go-proxy-pull-action action to v1.1.52 by @renovate[bot] in [#623](https://github.com/nicholas-fedor/go-remove/pull/623)
- Update github.com/charmbracelet/ultraviolet digest to bbf040a by @renovate[bot] in [#620](https://github.com/nicholas-fedor/go-remove/pull/620)
- Update github.com/charmbracelet/ultraviolet digest to b78653f by @renovate[bot] in [#618](https://github.com/nicholas-fedor/go-remove/pull/618)
- Make the quality gate trustworthy by @nicholas-fedor in [#611](https://github.com/nicholas-fedor/go-remove/pull/611)
- Update module github.com/klauspost/compress to v1.20.1 by @renovate[bot] in [#610](https://github.com/nicholas-fedor/go-remove/pull/610)
- Update module charm.land/bubbletea/v2 to v2.0.10 by @renovate[bot] in [#609](https://github.com/nicholas-fedor/go-remove/pull/609)
- Update github/codeql-action digest to 2892aa5 by @renovate[bot] in [#608](https://github.com/nicholas-fedor/go-remove/pull/608)
- Update github.com/charmbracelet/ultraviolet digest to 4e49372 by @renovate[bot] in [#607](https://github.com/nicholas-fedor/go-remove/pull/607)
- Update github/codeql-action digest to 1c5b675 by @renovate[bot] in [#606](https://github.com/nicholas-fedor/go-remove/pull/606)
- Update module github.com/dustin/go-humanize to v1.1.0 by @renovate[bot] in [#605](https://github.com/nicholas-fedor/go-remove/pull/605)
- Update codecov/codecov-action action to v7.1.1 by @renovate[bot] in [#604](https://github.com/nicholas-fedor/go-remove/pull/604)
- Update codecov/codecov-action action to v7.1.0 by @renovate[bot] in [#603](https://github.com/nicholas-fedor/go-remove/pull/603)
- Add install script, Cosign/Syft releases, and build config layout by @nicholas-fedor in [#602](https://github.com/nicholas-fedor/go-remove/pull/602)
- Update github action versions by @nicholas-fedor in [#601](https://github.com/nicholas-fedor/go-remove/pull/601)
- Bump go-proxy-pull-action to v1.1.51 by @nicholas-fedor in [#600](https://github.com/nicholas-fedor/go-remove/pull/600)

### New Contributors

- @github-actions[bot] made their first contribution in [#657](https://github.com/nicholas-fedor/go-remove/pull/657)

## [0.3.4] - 2026-09-14

### Changed

- Simplify internals, modernize for Go 1.27, and harden CLI/trash by @nicholas-fedor in [#599](https://github.com/nicholas-fedor/go-remove/pull/599)

### Chores

- Update module github.com/xo/terminfo to v1.2.0 by @renovate[bot] in [#598](https://github.com/nicholas-fedor/go-remove/pull/598)
- Update github.com/charmbracelet/ultraviolet digest to 6c9e17d by @renovate[bot] in [#597](https://github.com/nicholas-fedor/go-remove/pull/597)
- Update module github.com/mattn/go-runewidth to v0.0.30 by @renovate[bot] in [#596](https://github.com/nicholas-fedor/go-remove/pull/596)
- Update github/codeql-action digest to b96794f by @renovate[bot] in [#595](https://github.com/nicholas-fedor/go-remove/pull/595)
- Update module golang.org/x/sys to v0.48.0 by @renovate[bot] in [#594](https://github.com/nicholas-fedor/go-remove/pull/594)
- Update module golang.org/x/sync to v0.23.0 by @renovate[bot] in [#593](https://github.com/nicholas-fedor/go-remove/pull/593)
- Update github.com/charmbracelet/ultraviolet digest to 0277a17 by @renovate[bot] in [#592](https://github.com/nicholas-fedor/go-remove/pull/592)
- Update github.com/charmbracelet/ultraviolet digest to ae99b73 by @renovate[bot] in [#591](https://github.com/nicholas-fedor/go-remove/pull/591)
- Update module github.com/klauspost/compress to v1.20.0 by @renovate[bot] in [#590](https://github.com/nicholas-fedor/go-remove/pull/590)
- Update github.com/charmbracelet/ultraviolet digest to 043ea8a by @renovate[bot] in [#589](https://github.com/nicholas-fedor/go-remove/pull/589)
- Update module github.com/mattn/go-runewidth to v0.0.29 by @renovate[bot] in [#588](https://github.com/nicholas-fedor/go-remove/pull/588)
- Update cimg/go docker tag to v1.27.1 by @renovate[bot] in [#587](https://github.com/nicholas-fedor/go-remove/pull/587)
- Update github.com/charmbracelet/ultraviolet digest to af8eda3 by @renovate[bot] in [#586](https://github.com/nicholas-fedor/go-remove/pull/586)
- Update cimg/go:1.27.0 docker digest to 6026c28 by @renovate[bot] in [#585](https://github.com/nicholas-fedor/go-remove/pull/585)
- Update go module directive to v1.27.1 by @renovate[bot] in [#584](https://github.com/nicholas-fedor/go-remove/pull/584)
- Update github.com/charmbracelet/ultraviolet digest to d60c349 by @renovate[bot] in [#583](https://github.com/nicholas-fedor/go-remove/pull/583)
- Update step-security/harden-runner action to v2.21.1 by @renovate[bot] in [#582](https://github.com/nicholas-fedor/go-remove/pull/582)
- Update securego/gosec action to v2.29.0 by @renovate[bot] in [#581](https://github.com/nicholas-fedor/go-remove/pull/581)
- Update github/codeql-action digest to cdf488f by @renovate[bot] in [#580](https://github.com/nicholas-fedor/go-remove/pull/580)
- Update opentelemetry-go monorepo to v1.46.0 by @renovate[bot] in [#579](https://github.com/nicholas-fedor/go-remove/pull/579)
- Update cimg/go:1.27.0 docker digest to 4da2d4b by @renovate[bot] in [#578](https://github.com/nicholas-fedor/go-remove/pull/578)
- Update github/codeql-action digest to db488dd by @renovate[bot] in [#577](https://github.com/nicholas-fedor/go-remove/pull/577)
- Update cimg/go:1.27.0 docker digest to 91e576b by @renovate[bot] in [#576](https://github.com/nicholas-fedor/go-remove/pull/576)
- Update go module directive to v1.27.0 by @renovate[bot] in [#574](https://github.com/nicholas-fedor/go-remove/pull/574)
- Update cimg/go docker tag to v1.27.0 by @renovate[bot] in [#575](https://github.com/nicholas-fedor/go-remove/pull/575)
- Update module github.com/stretchr/testify to v1.12.1 by @renovate[bot] in [#573](https://github.com/nicholas-fedor/go-remove/pull/573)
- Update module charm.land/bubbletea/v2 to v2.0.9 by @renovate[bot] in [#572](https://github.com/nicholas-fedor/go-remove/pull/572)
- Update cimg/go docker tag to v1.26.7 by @renovate[bot] in [#571](https://github.com/nicholas-fedor/go-remove/pull/571)
- Update module github.com/mattn/go-runewidth to v0.0.28 by @renovate[bot] in [#570](https://github.com/nicholas-fedor/go-remove/pull/570)
- Update module github.com/stretchr/testify to v1.12.0 by @renovate[bot] in [#569](https://github.com/nicholas-fedor/go-remove/pull/569)
- Update step-security/harden-runner action to v2.21.0 by @renovate[bot] in [#568](https://github.com/nicholas-fedor/go-remove/pull/568)
- Update cimg/go docker tag to v1.26.6 by @renovate[bot] in [#567](https://github.com/nicholas-fedor/go-remove/pull/567)
- Update go module directive to v1.26.6 by @renovate[bot] in [#566](https://github.com/nicholas-fedor/go-remove/pull/566)
- Update github/codeql-action digest to ff2f1c6 by @renovate[bot] in [#565](https://github.com/nicholas-fedor/go-remove/pull/565)
- Update github.com/charmbracelet/ultraviolet digest to 68fa937 by @renovate[bot] in [#564](https://github.com/nicholas-fedor/go-remove/pull/564)

## [0.3.3] - 2026-08-12

### Documentation

- Remove go report card badge by @nicholas-fedor in [#535](https://github.com/nicholas-fedor/go-remove/pull/535)

### Chores

- Update module charm.land/lipgloss/v2 to v2.0.6 by @renovate[bot] in [#563](https://github.com/nicholas-fedor/go-remove/pull/563)
- Update module github.com/charmbracelet/x/ansi to v0.11.8 by @renovate[bot] in [#562](https://github.com/nicholas-fedor/go-remove/pull/562)
- Update github.com/charmbracelet/ultraviolet digest to 006e29f by @renovate[bot] in [#561](https://github.com/nicholas-fedor/go-remove/pull/561)
- Update module github.com/xo/terminfo to v1 by @renovate[bot] in [#560](https://github.com/nicholas-fedor/go-remove/pull/560)
- Update module google.golang.org/protobuf to v1.36.12 by @renovate[bot] in [#559](https://github.com/nicholas-fedor/go-remove/pull/559)
- Update github.com/charmbracelet/ultraviolet digest to 402eeaa by @renovate[bot] in [#558](https://github.com/nicholas-fedor/go-remove/pull/558)
- Update github.com/charmbracelet/ultraviolet digest to d38ea0f by @renovate[bot] in [#557](https://github.com/nicholas-fedor/go-remove/pull/557)
- Update actions/attest-build-provenance digest to 4d10147 by @renovate[bot] in [#556](https://github.com/nicholas-fedor/go-remove/pull/556)
- Update module github.com/klauspost/compress to v1.19.2 by @renovate[bot] in [#555](https://github.com/nicholas-fedor/go-remove/pull/555)
- Update step-security/harden-runner action to v2.20.1 by @renovate[bot] in [#554](https://github.com/nicholas-fedor/go-remove/pull/554)
- Update module github.com/dgraph-io/badger/v4 to v4.9.6 by @renovate[bot] in [#553](https://github.com/nicholas-fedor/go-remove/pull/553)
- Update github/codeql-action digest to 5595cca by @renovate[bot] in [#552](https://github.com/nicholas-fedor/go-remove/pull/552)
- Update opentelemetry-go monorepo to v1.45.0 by @renovate[bot] in [#551](https://github.com/nicholas-fedor/go-remove/pull/551)
- Update github/codeql-action digest to d1ba80a by @renovate[bot] in [#550](https://github.com/nicholas-fedor/go-remove/pull/550)
- Update github.com/charmbracelet/ultraviolet digest to 8b69304 by @renovate[bot] in [#549](https://github.com/nicholas-fedor/go-remove/pull/549)
- Update module github.com/lucasb-eyer/go-colorful to v1.4.1 by @renovate[bot] in [#548](https://github.com/nicholas-fedor/go-remove/pull/548)
- Update github.com/charmbracelet/ultraviolet digest to 19049f2 by @renovate[bot] in [#547](https://github.com/nicholas-fedor/go-remove/pull/547)
- Update github/codeql-action digest to f205ea1 by @renovate[bot] in [#546](https://github.com/nicholas-fedor/go-remove/pull/546)
- Update github.com/charmbracelet/ultraviolet digest to d9e819d by @renovate[bot] in [#545](https://github.com/nicholas-fedor/go-remove/pull/545)
- Update module github.com/mattn/go-runewidth to v0.0.27 by @renovate[bot] in [#544](https://github.com/nicholas-fedor/go-remove/pull/544)
- Update module github.com/dgraph-io/badger/v4 to v4.9.5 by @renovate[bot] in [#543](https://github.com/nicholas-fedor/go-remove/pull/543)
- Update module github.com/mattn/go-isatty to v0.0.24 by @renovate[bot] in [#542](https://github.com/nicholas-fedor/go-remove/pull/542)
- Update github/codeql-action digest to e4fba86 by @renovate[bot] in [#541](https://github.com/nicholas-fedor/go-remove/pull/541)
- Update github/codeql-action digest to e064762 by @renovate[bot] in [#540](https://github.com/nicholas-fedor/go-remove/pull/540)
- Update module github.com/klauspost/compress to v1.19.1 by @renovate[bot] in [#539](https://github.com/nicholas-fedor/go-remove/pull/539)
- Update module github.com/go-logr/logr to v1.4.4 by @renovate[bot] in [#538](https://github.com/nicholas-fedor/go-remove/pull/538)
- Update github.com/charmbracelet/ultraviolet digest to 7cc6674 by @renovate[bot] in [#537](https://github.com/nicholas-fedor/go-remove/pull/537)
- Update actions/checkout digest to 3d3c42e by @renovate[bot] in [#536](https://github.com/nicholas-fedor/go-remove/pull/536)
- Update github/codeql-action digest to 7188fc3 by @renovate[bot] in [#534](https://github.com/nicholas-fedor/go-remove/pull/534)
- Update module github.com/mattn/go-isatty to v0.0.23 by @renovate[bot] in [#533](https://github.com/nicholas-fedor/go-remove/pull/533)
- Update securego/gosec action to v2.28.0 by @renovate[bot] in [#532](https://github.com/nicholas-fedor/go-remove/pull/532)
- Update github.com/charmbracelet/ultraviolet digest to 4bee191 by @renovate[bot] in [#531](https://github.com/nicholas-fedor/go-remove/pull/531)
- Update module golang.org/x/sys to v0.47.0 by @renovate[bot] in [#530](https://github.com/nicholas-fedor/go-remove/pull/530)
- Update module golang.org/x/sync to v0.22.0 by @renovate[bot] in [#529](https://github.com/nicholas-fedor/go-remove/pull/529)
- Update module github.com/dgraph-io/badger/v4 to v4.9.4 by @renovate[bot] in [#528](https://github.com/nicholas-fedor/go-remove/pull/528)
- Update github/codeql-action digest to 99df26d by @renovate[bot] in [#526](https://github.com/nicholas-fedor/go-remove/pull/526)
- Update cimg/go docker tag to v1.26.5 by @renovate[bot] in [#527](https://github.com/nicholas-fedor/go-remove/pull/527)
- Update module github.com/dgraph-io/ristretto/v2 to v2.4.2 by @renovate[bot] in [#525](https://github.com/nicholas-fedor/go-remove/pull/525)
- Update go module directive to v1.26.5 by @renovate[bot] in [#524](https://github.com/nicholas-fedor/go-remove/pull/524)
- Update step-security/harden-runner action to v2.20.0 by @renovate[bot] in [#523](https://github.com/nicholas-fedor/go-remove/pull/523)
- Update module github.com/dgraph-io/badger/v4 to v4.9.3 by @renovate[bot] in [#522](https://github.com/nicholas-fedor/go-remove/pull/522)
- Update module github.com/dgraph-io/ristretto/v2 to v2.4.1 by @renovate[bot] in [#521](https://github.com/nicholas-fedor/go-remove/pull/521)
- Update cimg/go:1.26.4 docker digest to 66a357f by @renovate[bot] in [#520](https://github.com/nicholas-fedor/go-remove/pull/520)
- Update nicholas-fedor/actionlint-action action to v1.0.18 by @renovate[bot] in [#519](https://github.com/nicholas-fedor/go-remove/pull/519)
- Update nicholas-fedor/actionlint-action action to v1.0.17 by @renovate[bot] in [#518](https://github.com/nicholas-fedor/go-remove/pull/518)
- Update module charm.land/lipgloss/v2 to v2.0.5 by @renovate[bot] in [#517](https://github.com/nicholas-fedor/go-remove/pull/517)
- Update module charm.land/bubbletea/v2 to v2.0.8 by @renovate[bot] in [#516](https://github.com/nicholas-fedor/go-remove/pull/516)
- Update github.com/charmbracelet/ultraviolet digest to f5a850f by @renovate[bot] in [#515](https://github.com/nicholas-fedor/go-remove/pull/515)
- Update nicholas-fedor/actionlint-action action to v1.0.16 by @renovate[bot] in [#514](https://github.com/nicholas-fedor/go-remove/pull/514)
- Update github/codeql-action digest to 54f647b by @renovate[bot] in [#513](https://github.com/nicholas-fedor/go-remove/pull/513)
- Update module github.com/klauspost/compress to v1.19.0 by @renovate[bot] in [#512](https://github.com/nicholas-fedor/go-remove/pull/512)
- Update module github.com/klauspost/compress to v1.18.7 by @renovate[bot] in [#511](https://github.com/nicholas-fedor/go-remove/pull/511)
- Update nicholas-fedor/actionlint-action action to v1.0.15 by @renovate[bot] in [#510](https://github.com/nicholas-fedor/go-remove/pull/510)
- Update actions/attest-build-provenance digest to 0f67c3f by @renovate[bot] in [#509](https://github.com/nicholas-fedor/go-remove/pull/509)
- Update nicholas-fedor/actionlint-action action to v1.0.14 by @renovate[bot] in [#508](https://github.com/nicholas-fedor/go-remove/pull/508)
- Update nicholas-fedor/actionlint-action action to v1.0.13 by @renovate[bot] in [#507](https://github.com/nicholas-fedor/go-remove/pull/507)
- Update nicholas-fedor/actionlint-action action to v1.0.12 by @renovate[bot] in [#506](https://github.com/nicholas-fedor/go-remove/pull/506)
- Update github.com/charmbracelet/ultraviolet digest to f39628c by @renovate[bot] in [#505](https://github.com/nicholas-fedor/go-remove/pull/505)
- Update nicholas-fedor/actionlint-action action to v1.0.10 by @renovate[bot] in [#504](https://github.com/nicholas-fedor/go-remove/pull/504)
- Update actions/checkout action to v7 by @renovate[bot] in [#503](https://github.com/nicholas-fedor/go-remove/pull/503)
- Update github.com/charmbracelet/ultraviolet digest to 2399af7 by @renovate[bot] in [#502](https://github.com/nicholas-fedor/go-remove/pull/502)
- Update nicholas-fedor/actionlint-action action to v1.0.9 by @renovate[bot] in [#501](https://github.com/nicholas-fedor/go-remove/pull/501)
- Update nicholas-fedor/actionlint-action action to v1.0.8 by @renovate[bot] in [#500](https://github.com/nicholas-fedor/go-remove/pull/500)
- Update nicholas-fedor/actionlint-action action to v1.0.7 by @renovate[bot] in [#499](https://github.com/nicholas-fedor/go-remove/pull/499)
- Update module charm.land/lipgloss/v2 to v2.0.4 by @renovate[bot] in [#498](https://github.com/nicholas-fedor/go-remove/pull/498)
- Update module github.com/dgraph-io/badger/v4 to v4.9.2 by @renovate[bot] in [#497](https://github.com/nicholas-fedor/go-remove/pull/497)
- Update nicholas-fedor/actionlint-action action to v1.0.6 by @renovate[bot] in [#496](https://github.com/nicholas-fedor/go-remove/pull/496)
- Update module golang.org/x/sys to v0.46.0 by @renovate[bot] in [#495](https://github.com/nicholas-fedor/go-remove/pull/495)
- Update module golang.org/x/sync to v0.21.0 by @renovate[bot] in [#494](https://github.com/nicholas-fedor/go-remove/pull/494)
- Update github.com/charmbracelet/ultraviolet digest to 35bcb73 by @renovate[bot] in [#493](https://github.com/nicholas-fedor/go-remove/pull/493)
- Update nicholas-fedor/actionlint-action action to v1.0.5 by @renovate[bot] in [#492](https://github.com/nicholas-fedor/go-remove/pull/492)
- Update nicholas-fedor/actionlint-action action to v1.0.4 by @renovate[bot] in [#491](https://github.com/nicholas-fedor/go-remove/pull/491)
- Update github/codeql-action digest to 8aad20d by @renovate[bot] in [#490](https://github.com/nicholas-fedor/go-remove/pull/490)
- Update cimg/go docker tag to v1.26.4 by @renovate[bot] in [#489](https://github.com/nicholas-fedor/go-remove/pull/489)
- Update actions/checkout digest to df4cb1c by @renovate[bot] in [#488](https://github.com/nicholas-fedor/go-remove/pull/488)
- Update go module directive to v1.26.4 by @renovate[bot] in [#487](https://github.com/nicholas-fedor/go-remove/pull/487)
- Update github/codeql-action digest to 87557b9 by @renovate[bot] in [#486](https://github.com/nicholas-fedor/go-remove/pull/486)
- Update securego/gosec action to v2.27.1 by @renovate[bot] in [#485](https://github.com/nicholas-fedor/go-remove/pull/485)
- Update module charm.land/bubbletea/v2 to v2.0.7 by @renovate[bot] in [#484](https://github.com/nicholas-fedor/go-remove/pull/484)
- Update github.com/charmbracelet/ultraviolet digest to 6cf7526 by @renovate[bot] in [#483](https://github.com/nicholas-fedor/go-remove/pull/483)
- Update module github.com/mattn/go-runewidth to v0.0.24 by @renovate[bot] in [#482](https://github.com/nicholas-fedor/go-remove/pull/482)
- Update module github.com/mattn/go-colorable to v0.1.15 by @renovate[bot] in [#481](https://github.com/nicholas-fedor/go-remove/pull/481)
- Update opentelemetry-go monorepo to v1.44.0 by @renovate[bot] in [#480](https://github.com/nicholas-fedor/go-remove/pull/480)

## [0.3.2] - 2026-05-27

### Continuous Integration

- Use actionlint-action instead of manual download by @nicholas-fedor in [#468](https://github.com/nicholas-fedor/go-remove/pull/468)

### Chores

- Resolve goconst linter issues in history manager by @nicholas-fedor in [#479](https://github.com/nicholas-fedor/go-remove/pull/479)
- Update github.com/charmbracelet/ultraviolet digest to 948f455 by @renovate[bot] in [#478](https://github.com/nicholas-fedor/go-remove/pull/478)
- Update github/codeql-action digest to 7211b7c by @renovate[bot] in [#477](https://github.com/nicholas-fedor/go-remove/pull/477)
- Update module golang.org/x/net to v0.55.0 by @renovate[bot] in [#476](https://github.com/nicholas-fedor/go-remove/pull/476)
- Update module golang.org/x/sys to v0.45.0 by @renovate[bot] in [#475](https://github.com/nicholas-fedor/go-remove/pull/475)
- Update step-security/harden-runner action to v2.19.4 by @renovate[bot] in [#474](https://github.com/nicholas-fedor/go-remove/pull/474)
- Update cimg/go:1.26.3 docker digest to 9a5aff9 by @renovate[bot] in [#473](https://github.com/nicholas-fedor/go-remove/pull/473)
- Update github/codeql-action digest to 9e0d7b8 by @renovate[bot] in [#472](https://github.com/nicholas-fedor/go-remove/pull/472)
- Update step-security/harden-runner action to v2.19.3 by @renovate[bot] in [#471](https://github.com/nicholas-fedor/go-remove/pull/471)
- Update step-security/harden-runner action to v2.19.2 by @renovate[bot] in [#470](https://github.com/nicholas-fedor/go-remove/pull/470)
- Update github.com/charmbracelet/ultraviolet digest to c840852 by @renovate[bot] in [#469](https://github.com/nicholas-fedor/go-remove/pull/469)
- Update module golang.org/x/net to v0.54.0 by @renovate[bot] in [#467](https://github.com/nicholas-fedor/go-remove/pull/467)
- Update cimg/go docker tag to v1.26.3 by @renovate[bot] in [#466](https://github.com/nicholas-fedor/go-remove/pull/466)
- Update module golang.org/x/sys to v0.44.0 by @renovate[bot] in [#465](https://github.com/nicholas-fedor/go-remove/pull/465)
- Update cimg/go:1.26.2 docker digest to 0594489 by @renovate[bot] in [#464](https://github.com/nicholas-fedor/go-remove/pull/464)
- Update go module directive to v1.26.3 by @renovate[bot] in [#463](https://github.com/nicholas-fedor/go-remove/pull/463)
- Update github/codeql-action digest to 68bde55 by @renovate[bot] in [#462](https://github.com/nicholas-fedor/go-remove/pull/462)
- Update step-security/harden-runner action to v2.19.1 by @renovate[bot] in [#461](https://github.com/nicholas-fedor/go-remove/pull/461)
- Update github/codeql-action digest to e46ed2c by @renovate[bot] in [#460](https://github.com/nicholas-fedor/go-remove/pull/460)
- Update module github.com/klauspost/compress to v1.18.6 by @renovate[bot] in [#459](https://github.com/nicholas-fedor/go-remove/pull/459)
- Update github.com/charmbracelet/ultraviolet digest to 6603726 by @renovate[bot] in [#458](https://github.com/nicholas-fedor/go-remove/pull/458)
- Update securego/gosec action to v2.26.1 by @renovate[bot] in [#457](https://github.com/nicholas-fedor/go-remove/pull/457)
- Update module github.com/mattn/go-isatty to v0.0.22 by @renovate[bot] in [#456](https://github.com/nicholas-fedor/go-remove/pull/456)
- Update github.com/charmbracelet/ultraviolet digest to a0f1f21 by @renovate[bot] in [#455](https://github.com/nicholas-fedor/go-remove/pull/455)
- Update module github.com/rs/zerolog to v1.35.1 by @renovate[bot] in [#454](https://github.com/nicholas-fedor/go-remove/pull/454)
- Update github.com/charmbracelet/ultraviolet digest to 421e4a7 by @renovate[bot] in [#453](https://github.com/nicholas-fedor/go-remove/pull/453)
- Update step-security/harden-runner action to v2.19.0 by @renovate[bot] in [#452](https://github.com/nicholas-fedor/go-remove/pull/452)
- Update github.com/charmbracelet/ultraviolet digest to 9c68a86 by @renovate[bot] in [#450](https://github.com/nicholas-fedor/go-remove/pull/450)
- Update module charm.land/bubbletea/v2 to v2.0.6 by @renovate[bot] in [#451](https://github.com/nicholas-fedor/go-remove/pull/451)
- Update github/codeql-action digest to 95e58e9 by @renovate[bot] in [#449](https://github.com/nicholas-fedor/go-remove/pull/449)
- Update step-security/harden-runner action to v2.18.0 by @renovate[bot] in [#448](https://github.com/nicholas-fedor/go-remove/pull/448)
- Update module charm.land/lipgloss/v2 to v2.0.3 by @renovate[bot] in [#447](https://github.com/nicholas-fedor/go-remove/pull/447)
- Update github.com/charmbracelet/ultraviolet digest to 8c69ec8 by @renovate[bot] in [#444](https://github.com/nicholas-fedor/go-remove/pull/444)
- Update module charm.land/bubbletea/v2 to v2.0.5 by @renovate[bot] in [#446](https://github.com/nicholas-fedor/go-remove/pull/446)
- Update module github.com/charmbracelet/x/ansi to v0.11.7 by @renovate[bot] in [#445](https://github.com/nicholas-fedor/go-remove/pull/445)
- Update module charm.land/bubbletea/v2 to v2.0.3 by @renovate[bot] in [#443](https://github.com/nicholas-fedor/go-remove/pull/443)
- Update github.com/charmbracelet/ultraviolet digest to 7359239 by @renovate[bot] in [#442](https://github.com/nicholas-fedor/go-remove/pull/442)
- Update github.com/charmbracelet/ultraviolet digest to 31eb6d6 by @renovate[bot] in [#441](https://github.com/nicholas-fedor/go-remove/pull/441)
- Update module golang.org/x/net to v0.53.0 by @renovate[bot] in [#440](https://github.com/nicholas-fedor/go-remove/pull/440)
- Update step-security/harden-runner action to v2.17.0 by @renovate[bot] in [#439](https://github.com/nicholas-fedor/go-remove/pull/439)
- Update module golang.org/x/sys to v0.43.0 by @renovate[bot] in [#438](https://github.com/nicholas-fedor/go-remove/pull/438)
- Update cimg/go docker tag to v1.26.2 by @renovate[bot] in [#437](https://github.com/nicholas-fedor/go-remove/pull/437)
- Update module github.com/mattn/go-runewidth to v0.0.23 by @renovate[bot] in [#435](https://github.com/nicholas-fedor/go-remove/pull/435)
- Update cimg/go:1.26.1 docker digest to f813931 by @renovate[bot] in [#436](https://github.com/nicholas-fedor/go-remove/pull/436)
- Update module github.com/mattn/go-isatty to v0.0.21 by @renovate[bot] in [#434](https://github.com/nicholas-fedor/go-remove/pull/434)
- Update golangci/golangci-lint-action digest to 46c9287 by @renovate[bot] in [#433](https://github.com/nicholas-fedor/go-remove/pull/433)
- Update nicholas-fedor/go-proxy-pull-action digest to 6568a55 by @renovate[bot] in [#432](https://github.com/nicholas-fedor/go-remove/pull/432)
- Update dependency go to v1.26.2 by @renovate[bot] in [#431](https://github.com/nicholas-fedor/go-remove/pull/431)
- Update goreleaser/goreleaser-action digest to 01cbe07 by @renovate[bot] in [#430](https://github.com/nicholas-fedor/go-remove/pull/430)
- Update goreleaser/goreleaser-action digest to 2a473d7 by @renovate[bot] in [#429](https://github.com/nicholas-fedor/go-remove/pull/429)
- Update opentelemetry-go monorepo to v1.43.0 by @renovate[bot] in [#428](https://github.com/nicholas-fedor/go-remove/pull/428)
- Update module github.com/mattn/go-runewidth to v0.0.22 by @renovate[bot] in [#427](https://github.com/nicholas-fedor/go-remove/pull/427)
- Update golangci/golangci-lint-action digest to 36fe29c by @renovate[bot] in [#426](https://github.com/nicholas-fedor/go-remove/pull/426)

## [0.3.1] - 2026-04-01

### Fixed

- Resolve zerolog v1.35.0 panic in test mocks by @nicholas-fedor in [#425](https://github.com/nicholas-fedor/go-remove/pull/425)

### Continuous Integration

- Onboard StepSecurity by @stepsecurity-app[bot] in [#412](https://github.com/nicholas-fedor/go-remove/pull/412)
- Enable renovate updates for transitive go dependencies by @nicholas-fedor in [#398](https://github.com/nicholas-fedor/go-remove/pull/398)

### Chores

- Update step-security/harden-runner action to v2.16.1 by @renovate[bot] in [#424](https://github.com/nicholas-fedor/go-remove/pull/424)
- Update github.com/charmbracelet/ultraviolet digest to 0f94982 by @renovate[bot] in [#423](https://github.com/nicholas-fedor/go-remove/pull/423)
- Update crazy-max/ghaction-import-gpg digest to 1c06494 by @renovate[bot] in [#422](https://github.com/nicholas-fedor/go-remove/pull/422)
- Update module github.com/lucasb-eyer/go-colorful to v1.4.0 by @renovate[bot] in [#421](https://github.com/nicholas-fedor/go-remove/pull/421)
- Update module github.com/rs/zerolog to v1.35.0 by @renovate[bot] in [#420](https://github.com/nicholas-fedor/go-remove/pull/420)
- Update crazy-max/ghaction-import-gpg digest to da46d52 by @renovate[bot] in [#419](https://github.com/nicholas-fedor/go-remove/pull/419)
- Update github/codeql-action digest to c10b806 by @renovate[bot] in [#418](https://github.com/nicholas-fedor/go-remove/pull/418)
- Update github/codeql-action digest to b8bb9f2 by @renovate[bot] in [#417](https://github.com/nicholas-fedor/go-remove/pull/417)
- Update golangci/golangci-lint-action digest to 2d7e7b6 by @renovate[bot] in [#416](https://github.com/nicholas-fedor/go-remove/pull/416)
- Update codecov/codecov-action digest to 57e3a13 by @renovate[bot] in [#415](https://github.com/nicholas-fedor/go-remove/pull/415)
- Update goreleaser/goreleaser-action digest to fdcf0b9 by @renovate[bot] in [#414](https://github.com/nicholas-fedor/go-remove/pull/414)
- Update golangci/golangci-lint-action digest to e94e72c by @renovate[bot] in [#413](https://github.com/nicholas-fedor/go-remove/pull/413)
- Update github/codeql-action digest to 3869755 by @renovate[bot] in [#411](https://github.com/nicholas-fedor/go-remove/pull/411)
- Update module github.com/klauspost/compress to v1.18.5 by @renovate[bot] in [#410](https://github.com/nicholas-fedor/go-remove/pull/410)
- Update github/codeql-action digest to c6f9311 by @renovate[bot] in [#409](https://github.com/nicholas-fedor/go-remove/pull/409)
- Update golangci/golangci-lint-action digest to b269f19 by @renovate[bot] in [#408](https://github.com/nicholas-fedor/go-remove/pull/408)
- Update golangci/golangci-lint-action digest to fa2a845 by @renovate[bot] in [#407](https://github.com/nicholas-fedor/go-remove/pull/407)
- Update securego/gosec action to v2.25.0 by @renovate[bot] in [#406](https://github.com/nicholas-fedor/go-remove/pull/406)
- Update codecov/codecov-action digest to 1af5884 by @renovate[bot] in [#405](https://github.com/nicholas-fedor/go-remove/pull/405)
- Update golangci/golangci-lint-action digest to 2bcbc9e by @renovate[bot] in [#404](https://github.com/nicholas-fedor/go-remove/pull/404)
- Update opentelemetry-go monorepo to v1.42.0 by @renovate[bot] in [#403](https://github.com/nicholas-fedor/go-remove/pull/403)
- Update module github.com/mattn/go-runewidth to v0.0.21 by @renovate[bot] in [#401](https://github.com/nicholas-fedor/go-remove/pull/401)
- Update module golang.org/x/net to v0.52.0 by @renovate[bot] in [#402](https://github.com/nicholas-fedor/go-remove/pull/402)
- Update module github.com/charmbracelet/colorprofile to v0.4.3 by @renovate[bot] in [#400](https://github.com/nicholas-fedor/go-remove/pull/400)
- Update github.com/charmbracelet/ultraviolet digest to b93f6a3 by @renovate[bot] in [#399](https://github.com/nicholas-fedor/go-remove/pull/399)
- Update nicholas-fedor/govulncheck-action digest to b438bbb by @renovate[bot] in [#397](https://github.com/nicholas-fedor/go-remove/pull/397)
- Update actions/setup-go digest to 4a36011 by @renovate[bot] in [#396](https://github.com/nicholas-fedor/go-remove/pull/396)
- Apply go modernize changes by @nicholas-fedor in [#395](https://github.com/nicholas-fedor/go-remove/pull/395)
- Update nicholas-fedor/govulncheck-action digest to 4878bd2 by @renovate[bot] in [#394](https://github.com/nicholas-fedor/go-remove/pull/394)
- Update actions/setup-go digest to 8f19afc by @renovate[bot] in [#393](https://github.com/nicholas-fedor/go-remove/pull/393)
- Update github/codeql-action digest to b1bff81 by @renovate[bot] in [#392](https://github.com/nicholas-fedor/go-remove/pull/392)
- Update cimg/go:1.26.1 docker digest to ff658f9 by @renovate[bot] in [#391](https://github.com/nicholas-fedor/go-remove/pull/391)
- Update module charm.land/lipgloss/v2 to v2.0.2 by @renovate[bot] in [#390](https://github.com/nicholas-fedor/go-remove/pull/390)
- Update cimg/go docker tag to v1.26.1 by @renovate[bot] in [#389](https://github.com/nicholas-fedor/go-remove/pull/389)
- Update module charm.land/bubbletea/v2 to v2.0.2 by @renovate[bot] in [#387](https://github.com/nicholas-fedor/go-remove/pull/387)
- Update module charm.land/lipgloss/v2 to v2.0.1 by @renovate[bot] in [#388](https://github.com/nicholas-fedor/go-remove/pull/388)
- Update nicholas-fedor/govulncheck-action digest to ac1aadb by @renovate[bot] in [#386](https://github.com/nicholas-fedor/go-remove/pull/386)
- Update nicholas-fedor/go-proxy-pull-action digest to 66b03fb by @renovate[bot] in [#385](https://github.com/nicholas-fedor/go-remove/pull/385)
- Update dependency go to v1.26.1 by @renovate[bot] in [#384](https://github.com/nicholas-fedor/go-remove/pull/384)
- Update github/codeql-action digest to 0d579ff by @renovate[bot] in [#383](https://github.com/nicholas-fedor/go-remove/pull/383)

### New Contributors

- @stepsecurity-app[bot] made their first contribution in [#412](https://github.com/nicholas-fedor/go-remove/pull/412)

## [0.3.0] - 2026-03-04

### Added

- Add history tracking, undo/redo, and cross-platform trash management by @nicholas-fedor in [#379](https://github.com/nicholas-fedor/go-remove/pull/379)
- Migrate from zap to zerolog with TUI log capture by @nicholas-fedor in [#368](https://github.com/nicholas-fedor/go-remove/pull/368)

### Fixed

- Add Darwin platform support and fix Windows test failures by @nicholas-fedor in [#380](https://github.com/nicholas-fedor/go-remove/pull/380)

### Changed

- Migrate to bubbletea v2 and improve CI configuration by @nicholas-fedor in [#365](https://github.com/nicholas-fedor/go-remove/pull/365)

### Continuous Integration

- Pass GPG secrets to build workflow in release-prod by @nicholas-fedor in [#382](https://github.com/nicholas-fedor/go-remove/pull/382)
- Add missing secrets definition to build workflow by @nicholas-fedor in [#381](https://github.com/nicholas-fedor/go-remove/pull/381)

### Chores

- Update golangci/golangci-lint-action digest to b7bcab6 by @renovate[bot] in [#378](https://github.com/nicholas-fedor/go-remove/pull/378)
- Update module charm.land/bubbletea/v2 to v2.0.1 by @renovate[bot] in [#377](https://github.com/nicholas-fedor/go-remove/pull/377)
- Update nicholas-fedor/govulncheck-action digest to 1ffd170 by @renovate[bot] in [#376](https://github.com/nicholas-fedor/go-remove/pull/376)
- Update actions/setup-go digest to 27fdb26 by @renovate[bot] in [#375](https://github.com/nicholas-fedor/go-remove/pull/375)
- Update crazy-max/ghaction-import-gpg digest to 92a10f9 by @renovate[bot] in [#374](https://github.com/nicholas-fedor/go-remove/pull/374)
- Update github/codeql-action digest to c793b71 by @renovate[bot] in [#371](https://github.com/nicholas-fedor/go-remove/pull/371)
- Update copyright year to 2026 and standardize SPDX headers by @nicholas-fedor in [#373](https://github.com/nicholas-fedor/go-remove/pull/373)
- Update crazy-max/ghaction-import-gpg digest to b9c49e8 by @renovate[bot] in [#372](https://github.com/nicholas-fedor/go-remove/pull/372)
- Update securego/gosec action to v2.24.7 by @renovate[bot] in [#370](https://github.com/nicholas-fedor/go-remove/pull/370)
- Update goreleaser/goreleaser-action digest to 4be059c by @renovate[bot] in [#369](https://github.com/nicholas-fedor/go-remove/pull/369)
- Update securego/gosec action to v2.24.6 by @renovate[bot] in [#367](https://github.com/nicholas-fedor/go-remove/pull/367)
- Update securego/gosec action to v2.24.5 by @renovate[bot] in [#366](https://github.com/nicholas-fedor/go-remove/pull/366)
- Update golangci/golangci-lint-action digest to b207e52 by @renovate[bot] in [#364](https://github.com/nicholas-fedor/go-remove/pull/364)
- Update securego/gosec action to v2.24.0 by @renovate[bot] in [#363](https://github.com/nicholas-fedor/go-remove/pull/363)
- Update actions/attest-build-provenance digest to a2bbfa2 by @renovate[bot] in [#362](https://github.com/nicholas-fedor/go-remove/pull/362)
- Update nicholas-fedor/govulncheck-action digest to c6b69a0 by @renovate[bot] in [#361](https://github.com/nicholas-fedor/go-remove/pull/361)
- Update actions/setup-go digest to def8c39 by @renovate[bot] in [#360](https://github.com/nicholas-fedor/go-remove/pull/360)
- Update actions/attest-build-provenance action to v4 by @renovate[bot] in [#359](https://github.com/nicholas-fedor/go-remove/pull/359)
- Update nicholas-fedor/govulncheck-action digest to 15fce97 by @renovate[bot] in [#358](https://github.com/nicholas-fedor/go-remove/pull/358)
- Update actions/setup-go digest to 4b73464 by @renovate[bot] in [#357](https://github.com/nicholas-fedor/go-remove/pull/357)
- Update golangci/golangci-lint-action digest to 02d66c3 by @renovate[bot] in [#356](https://github.com/nicholas-fedor/go-remove/pull/356)
- Update cimg/go:1.26.0 docker digest to e82c772 by @renovate[bot] in [#353](https://github.com/nicholas-fedor/go-remove/pull/353)
- Update goreleaser/goreleaser-action digest to 6c92f1d by @renovate[bot] in [#352](https://github.com/nicholas-fedor/go-remove/pull/352)
- Update goreleaser/goreleaser-action digest to ff4cb9c by @renovate[bot] in [#351](https://github.com/nicholas-fedor/go-remove/pull/351)
- Update github/codeql-action digest to 89a39a4 by @renovate[bot] in [#350](https://github.com/nicholas-fedor/go-remove/pull/350)
- Update golangci/golangci-lint-action digest to 17a5bf4 by @renovate[bot] in [#349](https://github.com/nicholas-fedor/go-remove/pull/349)
- Update golangci/golangci-lint-action digest to fce8c98 by @renovate[bot] in [#348](https://github.com/nicholas-fedor/go-remove/pull/348)
- Update crazy-max/ghaction-import-gpg digest to 5a30dd9 by @renovate[bot] in [#347](https://github.com/nicholas-fedor/go-remove/pull/347)
- Update github/codeql-action digest to 9e907b5 by @renovate[bot] in [#346](https://github.com/nicholas-fedor/go-remove/pull/346)
- Update cimg/go docker tag to v1.26.0 by @renovate[bot] in [#345](https://github.com/nicholas-fedor/go-remove/pull/345)
- Update securego/gosec action to v2.23.0 by @renovate[bot] in [#344](https://github.com/nicholas-fedor/go-remove/pull/344)
- Update dependency go to 1.26.x by @renovate[bot] in [#343](https://github.com/nicholas-fedor/go-remove/pull/343)
- Update nicholas-fedor/go-proxy-pull-action digest to 95b3e6c by @renovate[bot] in [#342](https://github.com/nicholas-fedor/go-remove/pull/342)
- Update dependency go to v1.26.0 by @renovate[bot] in [#341](https://github.com/nicholas-fedor/go-remove/pull/341)
- Update goreleaser/goreleaser-action digest to ec59f47 by @renovate[bot] in [#340](https://github.com/nicholas-fedor/go-remove/pull/340)
- Update nicholas-fedor/go-proxy-pull-action digest to f0551db by @renovate[bot] in [#339](https://github.com/nicholas-fedor/go-remove/pull/339)
- Update github/codeql-action digest to 45cbd0c by @renovate[bot] in [#338](https://github.com/nicholas-fedor/go-remove/pull/338)

## [0.2.6] - 2026-02-05

### Changed

- Improve code formatting and style by @nicholas-fedor in [#336](https://github.com/nicholas-fedor/go-remove/pull/336)

### Chores

- Update cimg/go docker tag to v1.25.7 by @renovate[bot] in [#337](https://github.com/nicholas-fedor/go-remove/pull/337)
- Update nicholas-fedor/go-proxy-pull-action digest to 9c51cce by @renovate[bot] in [#334](https://github.com/nicholas-fedor/go-remove/pull/334)
- Update dependency go to v1.25.7 by @renovate[bot] in [#335](https://github.com/nicholas-fedor/go-remove/pull/335)
- Update cimg/go:1.25.6 docker digest to 81789fa by @renovate[bot] in [#333](https://github.com/nicholas-fedor/go-remove/pull/333)
- Update actions/checkout digest to de0fac2 by @renovate[bot] in [#332](https://github.com/nicholas-fedor/go-remove/pull/332)
- Update github/codeql-action digest to 6bc82e0 by @renovate[bot] in [#331](https://github.com/nicholas-fedor/go-remove/pull/331)
- Update golangci/golangci-lint-action digest to b62bd5d by @renovate[bot] in [#329](https://github.com/nicholas-fedor/go-remove/pull/329)
- Update goreleaser/goreleaser-action digest to 4247c53 by @renovate[bot] in [#330](https://github.com/nicholas-fedor/go-remove/pull/330)
- Update golangci/golangci-lint-action digest to d6deb2e by @renovate[bot] in [#328](https://github.com/nicholas-fedor/go-remove/pull/328)
- Update goreleaser/goreleaser-action digest to 902ab4a by @renovate[bot] in [#326](https://github.com/nicholas-fedor/go-remove/pull/326)
- Update nicholas-fedor/go-proxy-pull-action digest to a35ee0c by @renovate[bot] in [#327](https://github.com/nicholas-fedor/go-remove/pull/327)
- Update goreleaser/goreleaser-action digest to 78265e4 by @renovate[bot] in [#325](https://github.com/nicholas-fedor/go-remove/pull/325)
- Update nicholas-fedor/go-proxy-pull-action digest to 8689366 by @renovate[bot] in [#324](https://github.com/nicholas-fedor/go-remove/pull/324)
- Update github/codeql-action digest to b20883b by @renovate[bot] in [#322](https://github.com/nicholas-fedor/go-remove/pull/322)
- Update nicholas-fedor/govulncheck-action digest to 5b70be9 by @renovate[bot] in [#323](https://github.com/nicholas-fedor/go-remove/pull/323)
- Update actions/setup-go digest to a5f9b05 by @renovate[bot] in [#321](https://github.com/nicholas-fedor/go-remove/pull/321)
- Update goreleaser/goreleaser-action digest to 4c34bd9 by @renovate[bot] in [#320](https://github.com/nicholas-fedor/go-remove/pull/320)
- Update golangci/golangci-lint-action digest to 2c963d3 by @renovate[bot] in [#319](https://github.com/nicholas-fedor/go-remove/pull/319)
- Update github/codeql-action digest to 19b2f06 by @renovate[bot] in [#318](https://github.com/nicholas-fedor/go-remove/pull/318)
- Update nicholas-fedor/go-proxy-pull-action digest to c7a2ab4 by @renovate[bot] in [#317](https://github.com/nicholas-fedor/go-remove/pull/317)
- Update goreleaser/goreleaser-action digest to aacbb7f by @renovate[bot] in [#316](https://github.com/nicholas-fedor/go-remove/pull/316)
- Update golangci/golangci-lint-action digest to a3a03ee by @renovate[bot] in [#315](https://github.com/nicholas-fedor/go-remove/pull/315)
- Update cimg/go docker tag to v1.25.6 by @renovate[bot] in [#314](https://github.com/nicholas-fedor/go-remove/pull/314)
- Update dependency go to v1.25.6 by @renovate[bot] in [#313](https://github.com/nicholas-fedor/go-remove/pull/313)
- Update cimg/go:1.25.5 docker digest to e88af54 by @renovate[bot] in [#312](https://github.com/nicholas-fedor/go-remove/pull/312)
- Update nicholas-fedor/govulncheck-action digest to 72a75e9 by @renovate[bot] in [#311](https://github.com/nicholas-fedor/go-remove/pull/311)
- Update actions/setup-go digest to 7a3fe6c by @renovate[bot] in [#310](https://github.com/nicholas-fedor/go-remove/pull/310)
- Update nicholas-fedor/govulncheck-action digest to 301b2fd by @renovate[bot] in [#309](https://github.com/nicholas-fedor/go-remove/pull/309)
- Update actions/setup-go digest to d73f6bc by @renovate[bot] in [#308](https://github.com/nicholas-fedor/go-remove/pull/308)
- Update github/codeql-action digest to cdefb33 by @renovate[bot] in [#307](https://github.com/nicholas-fedor/go-remove/pull/307)
- Update golangci/golangci-lint-action digest to de73c35 by @renovate[bot] in [#306](https://github.com/nicholas-fedor/go-remove/pull/306)
- Update nicholas-fedor/govulncheck-action digest to 5e52ebd by @renovate[bot] in [#305](https://github.com/nicholas-fedor/go-remove/pull/305)
- Update actions/checkout digest to 0c366fd by @renovate[bot] in [#304](https://github.com/nicholas-fedor/go-remove/pull/304)
- Update nicholas-fedor/govulncheck-action digest to 9813de4 by @renovate[bot] in [#303](https://github.com/nicholas-fedor/go-remove/pull/303)
- Update actions/checkout digest to 064fe7f by @renovate[bot] in [#301](https://github.com/nicholas-fedor/go-remove/pull/301)
- Update cimg/go:1.25.5 docker digest to 955eb92 by @renovate[bot] in [#302](https://github.com/nicholas-fedor/go-remove/pull/302)
- Update nicholas-fedor/govulncheck-action digest to 0076f09 by @renovate[bot] in [#300](https://github.com/nicholas-fedor/go-remove/pull/300)
- Update actions/setup-go digest to ae252ee by @renovate[bot] in [#299](https://github.com/nicholas-fedor/go-remove/pull/299)
- Update golangci/golangci-lint-action digest to f75c1c4 by @renovate[bot] in [#298](https://github.com/nicholas-fedor/go-remove/pull/298)
- Update golangci/golangci-lint-action digest to e9dc929 by @renovate[bot] in [#297](https://github.com/nicholas-fedor/go-remove/pull/297)
- Update cimg/go:1.25.5 docker digest to b644c11 by @renovate[bot] in [#295](https://github.com/nicholas-fedor/go-remove/pull/295)
- Update golangci/golangci-lint-action digest to 2e568c9 by @renovate[bot] in [#296](https://github.com/nicholas-fedor/go-remove/pull/296)
- Update nicholas-fedor/govulncheck-action digest to ec02307 by @renovate[bot] in [#294](https://github.com/nicholas-fedor/go-remove/pull/294)
- Update actions/setup-go digest to 4aaadf4 by @renovate[bot] in [#293](https://github.com/nicholas-fedor/go-remove/pull/293)
- Update github/codeql-action digest to 5d4e8d1 by @renovate[bot] in [#292](https://github.com/nicholas-fedor/go-remove/pull/292)
- Update golangci/golangci-lint-action digest to ef75033 by @renovate[bot] in [#291](https://github.com/nicholas-fedor/go-remove/pull/291)
- Update github/codeql-action digest to 1b168cd by @renovate[bot] in [#290](https://github.com/nicholas-fedor/go-remove/pull/290)
- Update securego/gosec action to v2.22.11 by @renovate[bot] in [#289](https://github.com/nicholas-fedor/go-remove/pull/289)
- Update cimg/go:1.25.5 docker digest to 9a8ad8c by @renovate[bot] in [#288](https://github.com/nicholas-fedor/go-remove/pull/288)
- Update codecov/codecov-action digest to 671740a by @renovate[bot] in [#287](https://github.com/nicholas-fedor/go-remove/pull/287)
- Update golangci/golangci-lint-action digest to ca80bee by @renovate[bot] in [#286](https://github.com/nicholas-fedor/go-remove/pull/286)

## [0.2.5] - 2025-12-05

### Chores

- Update github/codeql-action digest to cf1bb45 by @renovate[bot] in [#285](https://github.com/nicholas-fedor/go-remove/pull/285)
- Update module github.com/spf13/cobra to v1.10.2 by @renovate[bot] in [#284](https://github.com/nicholas-fedor/go-remove/pull/284)
- Update dependency go to v1.25.5 by @renovate[bot] in [#280](https://github.com/nicholas-fedor/go-remove/pull/280)
- Update cimg/go docker tag to v1.25.5 by @renovate[bot] in [#283](https://github.com/nicholas-fedor/go-remove/pull/283)
- Update nicholas-fedor/go-proxy-pull-action digest to 501ad32 by @renovate[bot] in [#282](https://github.com/nicholas-fedor/go-remove/pull/282)
- Update actions/checkout digest to 8e8c483 by @renovate[bot] in [#281](https://github.com/nicholas-fedor/go-remove/pull/281)
- Update nicholas-fedor/govulncheck-action digest to 5d80989 by @renovate[bot] in [#279](https://github.com/nicholas-fedor/go-remove/pull/279)
- Update actions/checkout digest to 8e8c483 by @renovate[bot] in [#278](https://github.com/nicholas-fedor/go-remove/pull/278)
- Update goreleaser/goreleaser-action digest to d31d51a by @renovate[bot] in [#277](https://github.com/nicholas-fedor/go-remove/pull/277)
- Update golangci/golangci-lint-action digest to 1e7e51e by @renovate[bot] in [#276](https://github.com/nicholas-fedor/go-remove/pull/276)
- Update github/codeql-action digest to fe4161a by @renovate[bot] in [#275](https://github.com/nicholas-fedor/go-remove/pull/275)
- Update golangci/golangci-lint-action digest to 13fed6f by @renovate[bot] in [#274](https://github.com/nicholas-fedor/go-remove/pull/274)
- Update goreleaser/goreleaser-action digest to f3511a2 by @renovate[bot] in [#273](https://github.com/nicholas-fedor/go-remove/pull/273)
- Update golangci/golangci-lint-action digest to a6071aa by @renovate[bot] in [#272](https://github.com/nicholas-fedor/go-remove/pull/272)
- Update github/codeql-action digest to fdbfb4d by @renovate[bot] in [#271](https://github.com/nicholas-fedor/go-remove/pull/271)
- Update nicholas-fedor/govulncheck-action digest to 22f7e2d by @renovate[bot] in [#270](https://github.com/nicholas-fedor/go-remove/pull/270)
- Update actions/checkout digest to c2d88d3 by @renovate[bot] in [#269](https://github.com/nicholas-fedor/go-remove/pull/269)
- Update golangci/golangci-lint-action digest to e7fa5ac by @renovate[bot] in [#268](https://github.com/nicholas-fedor/go-remove/pull/268)
- Update actions/checkout action to v6 by @renovate[bot] in [#266](https://github.com/nicholas-fedor/go-remove/pull/266)
- Update nicholas-fedor/govulncheck-action digest to d800c37 by @renovate[bot] in [#267](https://github.com/nicholas-fedor/go-remove/pull/267)
- Update actions/checkout digest to 1af3b93 by @renovate[bot] in [#265](https://github.com/nicholas-fedor/go-remove/pull/265)
- Update actions/setup-go digest to 4dc6199 by @renovate[bot] in [#263](https://github.com/nicholas-fedor/go-remove/pull/263)
- Update nicholas-fedor/govulncheck-action digest to 55deb21 by @renovate[bot] in [#264](https://github.com/nicholas-fedor/go-remove/pull/264)
- Update module go.uber.org/zap to v1.27.1 by @renovate[bot] in [#262](https://github.com/nicholas-fedor/go-remove/pull/262)
- Update nicholas-fedor/govulncheck-action digest to 0ee4877 by @renovate[bot] in [#261](https://github.com/nicholas-fedor/go-remove/pull/261)
- Update actions/setup-go digest to f3787be by @renovate[bot] in [#260](https://github.com/nicholas-fedor/go-remove/pull/260)
- Update codecov/codecov-action digest to 96b38e9 by @renovate[bot] in [#259](https://github.com/nicholas-fedor/go-remove/pull/259)
- Update golangci/golangci-lint-action digest to 1dfda28 by @renovate[bot] in [#257](https://github.com/nicholas-fedor/go-remove/pull/257)
- Update github/codeql-action digest to e12f017 by @renovate[bot] in [#258](https://github.com/nicholas-fedor/go-remove/pull/258)
- Update cimg/go:1.25.4 docker digest to cf75b46 by @renovate[bot] in [#256](https://github.com/nicholas-fedor/go-remove/pull/256)
- Update actions/checkout digest to 93cb6ef by @renovate[bot] in [#255](https://github.com/nicholas-fedor/go-remove/pull/255)
- Update golangci/golangci-lint-action digest to 37a9faf by @renovate[bot] in [#254](https://github.com/nicholas-fedor/go-remove/pull/254)
- Update cimg/go docker tag to v1.25.4 by @renovate[bot] in [#253](https://github.com/nicholas-fedor/go-remove/pull/253)
- Update cimg/go:1.25.3 docker digest to 0184935 by @renovate[bot] in [#252](https://github.com/nicholas-fedor/go-remove/pull/252)
- Update github/codeql-action digest to 014f16e by @renovate[bot] in [#251](https://github.com/nicholas-fedor/go-remove/pull/251)
- Update nicholas-fedor/govulncheck-action digest to 077e0b4 by @renovate[bot] in [#250](https://github.com/nicholas-fedor/go-remove/pull/250)
- Update actions/setup-go digest to 3a0c2c8 by @renovate[bot] in [#249](https://github.com/nicholas-fedor/go-remove/pull/249)
- Update codecov/codecov-action digest to 9b6d1f8 by @renovate[bot] in [#248](https://github.com/nicholas-fedor/go-remove/pull/248)
- Update golangci/golangci-lint-action digest to 199a9c2 by @renovate[bot] in [#246](https://github.com/nicholas-fedor/go-remove/pull/246)
- Update golangci/golangci-lint-action digest to c7c1219 by @renovate[bot] in [#245](https://github.com/nicholas-fedor/go-remove/pull/245)
- Update golangci/golangci-lint-action digest to 0a35821 by @renovate[bot] in [#244](https://github.com/nicholas-fedor/go-remove/pull/244)
- Update golangci/golangci-lint-action digest to a66d26a by @renovate[bot] in [#243](https://github.com/nicholas-fedor/go-remove/pull/243)
- Update goreleaser/goreleaser-action digest to 9cf3611 by @renovate[bot] in [#242](https://github.com/nicholas-fedor/go-remove/pull/242)
- Update nicholas-fedor/go-proxy-pull-action digest to a32dd3b by @renovate[bot] in [#241](https://github.com/nicholas-fedor/go-remove/pull/241)
- Update nicholas-fedor/go-proxy-pull-action digest to 41fdd3e by @renovate[bot] in [#240](https://github.com/nicholas-fedor/go-remove/pull/240)
- Update dependency go to v1.25.4 by @renovate[bot] in [#239](https://github.com/nicholas-fedor/go-remove/pull/239)
- Update goreleaser/goreleaser-action digest to aab4704 by @renovate[bot] in [#238](https://github.com/nicholas-fedor/go-remove/pull/238)
- Update cimg/go:1.25.3 docker digest to af601f9 by @renovate[bot] in [#237](https://github.com/nicholas-fedor/go-remove/pull/237)
- Update nicholas-fedor/govulncheck-action digest to fa0b698 by @renovate[bot] in [#236](https://github.com/nicholas-fedor/go-remove/pull/236)
- Update actions/checkout digest to 71cf226 by @renovate[bot] in [#235](https://github.com/nicholas-fedor/go-remove/pull/235)
- Update golangci/golangci-lint-action digest to 7fe1b22 by @renovate[bot] in [#234](https://github.com/nicholas-fedor/go-remove/pull/234)
- Update github/codeql-action digest to 0499de3 by @renovate[bot] in [#233](https://github.com/nicholas-fedor/go-remove/pull/233)
- Update github/codeql-action digest to 5fe9434 by @renovate[bot] in [#232](https://github.com/nicholas-fedor/go-remove/pull/232)
- Update cimg/go:1.25.3 docker digest to e31a463 by @renovate[bot] in [#231](https://github.com/nicholas-fedor/go-remove/pull/231)
- Update nicholas-fedor/go-proxy-pull-action digest to 0591509 by @renovate[bot] in [#230](https://github.com/nicholas-fedor/go-remove/pull/230)
- Update nicholas-fedor/govulncheck-action digest to 803f85c by @renovate[bot] in [#229](https://github.com/nicholas-fedor/go-remove/pull/229)
- Update actions/setup-go digest to faf5242 by @renovate[bot] in [#228](https://github.com/nicholas-fedor/go-remove/pull/228)
- Update nicholas-fedor/govulncheck-action digest to 76fb91b by @renovate[bot] in [#227](https://github.com/nicholas-fedor/go-remove/pull/227)
- Update actions/setup-go digest to 7bc60db by @renovate[bot] in [#226](https://github.com/nicholas-fedor/go-remove/pull/226)
- Update golangci/golangci-lint-action digest to 14973f1 by @renovate[bot] in [#225](https://github.com/nicholas-fedor/go-remove/pull/225)
- Update github/codeql-action digest to 4e94bd1 by @renovate[bot] in [#224](https://github.com/nicholas-fedor/go-remove/pull/224)
- Update golangci/golangci-lint-action digest to b002b6e by @renovate[bot] in [#223](https://github.com/nicholas-fedor/go-remove/pull/223)
- Update github/codeql-action digest to 16140ae by @renovate[bot] in [#222](https://github.com/nicholas-fedor/go-remove/pull/222)
- Update golangci/golangci-lint-action digest to b68d21b by @renovate[bot] in [#221](https://github.com/nicholas-fedor/go-remove/pull/221)
- Update nicholas-fedor/go-proxy-pull-action digest to 3349087 by @renovate[bot] in [#220](https://github.com/nicholas-fedor/go-remove/pull/220)
- Update securego/gosec action to v2.22.10 by @renovate[bot] in [#219](https://github.com/nicholas-fedor/go-remove/pull/219)
- Update nicholas-fedor/go-proxy-pull-action digest to 6b27ce6 by @renovate[bot] in [#217](https://github.com/nicholas-fedor/go-remove/pull/217)
- Update cimg/go docker tag to v1.25.3 by @renovate[bot] in [#218](https://github.com/nicholas-fedor/go-remove/pull/218)
- Update golangci/golangci-lint-action digest to 06188a2 by @renovate[bot] in [#215](https://github.com/nicholas-fedor/go-remove/pull/215)
- Update dependency go to v1.25.3 by @renovate[bot] in [#216](https://github.com/nicholas-fedor/go-remove/pull/216)
- Update nicholas-fedor/go-proxy-pull-action digest to 28967b1 by @renovate[bot] in [#214](https://github.com/nicholas-fedor/go-remove/pull/214)
- Update github/codeql-action digest to f443b60 by @renovate[bot] in [#213](https://github.com/nicholas-fedor/go-remove/pull/213)
- Update nicholas-fedor/go-proxy-pull-action digest to f36283c by @renovate[bot] in [#212](https://github.com/nicholas-fedor/go-remove/pull/212)
- Update nicholas-fedor/go-proxy-pull-action digest to df60457 by @renovate[bot] in [#211](https://github.com/nicholas-fedor/go-remove/pull/211)
- Update nicholas-fedor/go-proxy-pull-action digest to 9d5bb93 by @renovate[bot] in [#210](https://github.com/nicholas-fedor/go-remove/pull/210)

## [0.2.4] - 2025-10-08

### Continuous Integration

- Add check-latest to Go setup in workflows by @nicholas-fedor in [#207](https://github.com/nicholas-fedor/go-remove/pull/207)

### Chores

- Add whitespace to satisfy linting by @nicholas-fedor in [#209](https://github.com/nicholas-fedor/go-remove/pull/209)
- Update dependencies by @nicholas-fedor in [#208](https://github.com/nicholas-fedor/go-remove/pull/208)
- Update github/codeql-action action to v4 by @renovate[bot] in [#205](https://github.com/nicholas-fedor/go-remove/pull/205)
- Update cimg/go docker tag to v1.25.2 by @renovate[bot] in [#206](https://github.com/nicholas-fedor/go-remove/pull/206)
- Update nicholas-fedor/go-proxy-pull-action digest to 22f4f2d by @renovate[bot] in [#204](https://github.com/nicholas-fedor/go-remove/pull/204)
- Update github/codeql-action digest to a8d1ac4 by @renovate[bot] in [#202](https://github.com/nicholas-fedor/go-remove/pull/202)
- Update dependency go to v1.25.2 by @renovate[bot] in [#203](https://github.com/nicholas-fedor/go-remove/pull/203)
- Update golangci/golangci-lint-action digest to 1d64cc1 by @renovate[bot] in [#201](https://github.com/nicholas-fedor/go-remove/pull/201)
- Update github/codeql-action digest to 64d10c1 by @renovate[bot] in [#200](https://github.com/nicholas-fedor/go-remove/pull/200)
- Update golangci/golangci-lint-action digest to 7409966 by @renovate[bot] in [#199](https://github.com/nicholas-fedor/go-remove/pull/199)
- Update github/codeql-action digest to 3599b3b by @renovate[bot] in [#198](https://github.com/nicholas-fedor/go-remove/pull/198)
- Update securego/gosec to v2.22.9 by @nicholas-fedor in [#197](https://github.com/nicholas-fedor/go-remove/pull/197)
- Update github/codeql-action digest to 303c0ae by @renovate[bot] in [#196](https://github.com/nicholas-fedor/go-remove/pull/196)
- Update golangci/golangci-lint-action digest to f33eece by @renovate[bot] in [#195](https://github.com/nicholas-fedor/go-remove/pull/195)
- Update securego/gosec digest to f9c52aa by @renovate[bot] in [#194](https://github.com/nicholas-fedor/go-remove/pull/194)
- Update module github.com/charmbracelet/bubbletea to v1.3.10 by @renovate[bot] in [#193](https://github.com/nicholas-fedor/go-remove/pull/193)
- Update securego/gosec digest to 506407e by @renovate[bot] in [#192](https://github.com/nicholas-fedor/go-remove/pull/192)
- Update nicholas-fedor/govulncheck-action digest to 1e9ef2c by @renovate[bot] in [#191](https://github.com/nicholas-fedor/go-remove/pull/191)
- Update actions/setup-go digest to c0137ca by @renovate[bot] in [#189](https://github.com/nicholas-fedor/go-remove/pull/189)
- Update golangci/golangci-lint-action digest to f08454a by @renovate[bot] in [#190](https://github.com/nicholas-fedor/go-remove/pull/190)
- Update securego/gosec digest to 3ead143 by @renovate[bot] in [#188](https://github.com/nicholas-fedor/go-remove/pull/188)
- Update securego/gosec digest to e81fba3 by @renovate[bot] in [#187](https://github.com/nicholas-fedor/go-remove/pull/187)
- Update module github.com/charmbracelet/bubbletea to v1.3.9 by @renovate[bot] in [#186](https://github.com/nicholas-fedor/go-remove/pull/186)
- Update github/codeql-action digest to 192325c by @renovate[bot] in [#185](https://github.com/nicholas-fedor/go-remove/pull/185)
- Update github/codeql-action digest to d3678e2 by @renovate[bot] in [#184](https://github.com/nicholas-fedor/go-remove/pull/184)
- Update module github.com/charmbracelet/bubbletea to v1.3.8 by @renovate[bot] in [#183](https://github.com/nicholas-fedor/go-remove/pull/183)
- Update golangci/golangci-lint-action digest to 7574dab by @renovate[bot] in [#182](https://github.com/nicholas-fedor/go-remove/pull/182)
- Update securego/gosec digest to 4be6b11 by @renovate[bot] in [#181](https://github.com/nicholas-fedor/go-remove/pull/181)
- Update golangci/golangci-lint-action digest to dc56f00 by @renovate[bot] in [#180](https://github.com/nicholas-fedor/go-remove/pull/180)
- Update nicholas-fedor/go-proxy-pull-action digest to 5bb09e7 by @renovate[bot] in [#179](https://github.com/nicholas-fedor/go-remove/pull/179)
- Update module github.com/charmbracelet/bubbletea to v1.3.7 by @renovate[bot] in [#178](https://github.com/nicholas-fedor/go-remove/pull/178)
- Update github/codeql-action digest to f1f6e5f by @renovate[bot] in [#177](https://github.com/nicholas-fedor/go-remove/pull/177)
- Update cimg/go docker tag to v1.25.1 by @renovate[bot] in [#176](https://github.com/nicholas-fedor/go-remove/pull/176)
- Update codecov/codecov-action digest to 5a10915 by @renovate[bot] in [#175](https://github.com/nicholas-fedor/go-remove/pull/175)
- Update codecov/codecov-action digest to 206148c by @renovate[bot] in [#174](https://github.com/nicholas-fedor/go-remove/pull/174)

## [0.2.3] - 2025-09-04

### Chores

- Update nicholas-fedor/govulncheck-action digest to 6bacd52 by @renovate[bot] in [#173](https://github.com/nicholas-fedor/go-remove/pull/173)
- Update actions/setup-go digest to 4469467 by @renovate[bot] in [#171](https://github.com/nicholas-fedor/go-remove/pull/171)
- Update nicholas-fedor/go-proxy-pull-action digest to ca64499 by @renovate[bot] in [#172](https://github.com/nicholas-fedor/go-remove/pull/172)
- Update dependency go to v1.25.1 by @renovate[bot] in [#170](https://github.com/nicholas-fedor/go-remove/pull/170)
- Update module github.com/spf13/cobra to v1.10.1 by @renovate[bot] in [#169](https://github.com/nicholas-fedor/go-remove/pull/169)
- Update module github.com/spf13/cobra to v1.10.0 by @renovate[bot] in [#168](https://github.com/nicholas-fedor/go-remove/pull/168)
- Update github/codeql-action digest to 2d92b76 by @renovate[bot] in [#167](https://github.com/nicholas-fedor/go-remove/pull/167)
- Update actions/attest-build-provenance action to v3 by @renovate[bot] in [#165](https://github.com/nicholas-fedor/go-remove/pull/165)
- Update nicholas-fedor/govulncheck-action digest to d4283df by @renovate[bot] in [#166](https://github.com/nicholas-fedor/go-remove/pull/166)
- Update actions/setup-go digest to 1d76b95 by @renovate[bot] in [#164](https://github.com/nicholas-fedor/go-remove/pull/164)
- Update module github.com/stretchr/testify to v1.11.1 by @renovate[bot] in [#163](https://github.com/nicholas-fedor/go-remove/pull/163)
- Update golangci/golangci-lint-action digest to 3c28b2c by @renovate[bot] in [#162](https://github.com/nicholas-fedor/go-remove/pull/162)
- Update goreleaser/goreleaser-action digest to a08664b by @renovate[bot] in [#161](https://github.com/nicholas-fedor/go-remove/pull/161)
- Update golangci/golangci-lint-action digest to d65369c by @renovate[bot] in [#160](https://github.com/nicholas-fedor/go-remove/pull/160)
- Update securego/gosec digest to 5af1117 by @renovate[bot] in [#159](https://github.com/nicholas-fedor/go-remove/pull/159)
- Update module github.com/stretchr/testify to v1.11.0 by @renovate[bot] in [#158](https://github.com/nicholas-fedor/go-remove/pull/158)
- Update golangci/golangci-lint-action digest to 030ca6c by @renovate[bot] in [#157](https://github.com/nicholas-fedor/go-remove/pull/157)
- Update golangci/golangci-lint-action digest to c21e01f by @renovate[bot] in [#156](https://github.com/nicholas-fedor/go-remove/pull/156)
- Update github/codeql-action digest to 3c3833e by @renovate[bot] in [#155](https://github.com/nicholas-fedor/go-remove/pull/155)
- Update codecov/codecov-action digest to 3cb13a1 by @renovate[bot] in [#154](https://github.com/nicholas-fedor/go-remove/pull/154)
- Update codecov/codecov-action digest to fdcc847 by @renovate[bot] in [#153](https://github.com/nicholas-fedor/go-remove/pull/153)
- Update github/codeql-action digest to 96f518a by @renovate[bot] in [#152](https://github.com/nicholas-fedor/go-remove/pull/152)
- Update securego/gosec digest to 287b46c by @renovate[bot] in [#151](https://github.com/nicholas-fedor/go-remove/pull/151)
- Update codecov/codecov-action digest to 39a2af1 by @renovate[bot] in [#150](https://github.com/nicholas-fedor/go-remove/pull/150)
- Update nicholas-fedor/go-proxy-pull-action digest to 40d406b by @renovate[bot] in [#149](https://github.com/nicholas-fedor/go-remove/pull/149)
- Update securego/gosec digest to cee0aea by @renovate[bot] in [#148](https://github.com/nicholas-fedor/go-remove/pull/148)
- Update goreleaser/goreleaser-action digest to 35b9a27 by @renovate[bot] in [#147](https://github.com/nicholas-fedor/go-remove/pull/147)
- Update securego/gosec digest to c945302 by @renovate[bot] in [#146](https://github.com/nicholas-fedor/go-remove/pull/146)
- Update nicholas-fedor/go-proxy-pull-action digest to 46417d8 by @renovate[bot] in [#145](https://github.com/nicholas-fedor/go-remove/pull/145)

## [0.2.2] - 2025-08-14

### Chores

- Go and dependency version updates by @nicholas-fedor in [#144](https://github.com/nicholas-fedor/go-remove/pull/144)
- Update nicholas-fedor/govulncheck-action digest to 1862128 by @renovate[bot] in [#143](https://github.com/nicholas-fedor/go-remove/pull/143)
- Update nicholas-fedor/go-proxy-pull-action digest to 7740eae by @renovate[bot] in [#142](https://github.com/nicholas-fedor/go-remove/pull/142)
- Update nicholas-fedor/govulncheck-action digest to d15b9b7 by @renovate[bot] in [#141](https://github.com/nicholas-fedor/go-remove/pull/141)
- Update actions/setup-go digest to e75c3e8 by @renovate[bot] in [#140](https://github.com/nicholas-fedor/go-remove/pull/140)
- Update cimg/go docker tag to v1.25.0 by @renovate[bot] in [#139](https://github.com/nicholas-fedor/go-remove/pull/139)
- Update actions/checkout digest to ff7abcd by @renovate[bot] in [#138](https://github.com/nicholas-fedor/go-remove/pull/138)
- Update dependency go to 1.25.x by @renovate[bot] in [#137](https://github.com/nicholas-fedor/go-remove/pull/137)
- Update dependency go to v1.25.0 by @renovate[bot] in [#136](https://github.com/nicholas-fedor/go-remove/pull/136)
- Update github/codeql-action digest to df55935 by @renovate[bot] in [#135](https://github.com/nicholas-fedor/go-remove/pull/135)
- Update actions/checkout action to v5 by @renovate[bot] in [#134](https://github.com/nicholas-fedor/go-remove/pull/134)
- Update cimg/go docker tag to v1.24.6 by @renovate[bot] in [#133](https://github.com/nicholas-fedor/go-remove/pull/133)

## [0.2.1] - 2025-08-11

### Fixed

- Use chore for go dependency updates by @nicholas-fedor in [#115](https://github.com/nicholas-fedor/go-remove/pull/115)

### Chores

- Update securego/gosec digest to ef7adab by @renovate[bot] in [#132](https://github.com/nicholas-fedor/go-remove/pull/132)
- Update golangci/golangci-lint-action digest to 9511564 by @renovate[bot] in [#131](https://github.com/nicholas-fedor/go-remove/pull/131)
- Update actions/checkout digest by @renovate[bot] in [#129](https://github.com/nicholas-fedor/go-remove/pull/129)
- Update nicholas-fedor/govulncheck-action digest to ae17d3c by @renovate[bot] in [#130](https://github.com/nicholas-fedor/go-remove/pull/130)
- Update securego/gosec digest to e201bb8 by @renovate[bot] in [#128](https://github.com/nicholas-fedor/go-remove/pull/128)
- Update actions/checkout digest to 08eba0b by @renovate[bot] in [#127](https://github.com/nicholas-fedor/go-remove/pull/127)
- Update github/codeql-action digest to 76621b6 by @renovate[bot] in [#126](https://github.com/nicholas-fedor/go-remove/pull/126)
- Update github/codeql-action digest to a4e1a01 by @renovate[bot] in [#125](https://github.com/nicholas-fedor/go-remove/pull/125)
- Update goreleaser/goreleaser-action digest to e435ccd by @renovate[bot] in [#124](https://github.com/nicholas-fedor/go-remove/pull/124)
- Update nicholas-fedor/go-proxy-pull-action digest to ba81e12 by @renovate[bot] in [#123](https://github.com/nicholas-fedor/go-remove/pull/123)
- Update dependency go to v1.24.6 by @renovate[bot] in [#122](https://github.com/nicholas-fedor/go-remove/pull/122)
- Update golangci/golangci-lint-action digest to f9e969a by @renovate[bot] in [#121](https://github.com/nicholas-fedor/go-remove/pull/121)
- Update goreleaser/goreleaser-action digest to 2ff5850 by @renovate[bot] in [#120](https://github.com/nicholas-fedor/go-remove/pull/120)
- Update goreleaser/goreleaser-action digest to ca48102 by @renovate[bot] in [#119](https://github.com/nicholas-fedor/go-remove/pull/119)
- Update github/codeql-action digest to 51f7732 by @renovate[bot] in [#118](https://github.com/nicholas-fedor/go-remove/pull/118)
- Update securego/gosec digest to ba592af by @renovate[bot] in [#117](https://github.com/nicholas-fedor/go-remove/pull/117)
- Update nicholas-fedor/govulncheck-action digest to affabe3 by @renovate[bot] in [#116](https://github.com/nicholas-fedor/go-remove/pull/116)
- Update github/codeql-action digest to 4e828ff by @renovate[bot] in [#114](https://github.com/nicholas-fedor/go-remove/pull/114)
- Update actions/checkout digest to 8edcb1b by @renovate[bot] in [#113](https://github.com/nicholas-fedor/go-remove/pull/113)

## [0.2.0] - 2025-07-22

### Added

- Implement Column-Major Layout and Sorting for TUI by @nicholas-fedor in [#112](https://github.com/nicholas-fedor/go-remove/pull/112)
- Enable Go version updates with gomodTidy by @nicholas-fedor in [#110](https://github.com/nicholas-fedor/go-remove/pull/110)

### Changed

- Remove redundant log level adjustment in Run func by @nicholas-fedor in [#99](https://github.com/nicholas-fedor/go-remove/pull/99)

### Chores

- Update dependency go to v1.24.5 by @renovate[bot] in [#111](https://github.com/nicholas-fedor/go-remove/pull/111)
- Update securego/gosec digest to 2ef6017 by @renovate[bot] in [#109](https://github.com/nicholas-fedor/go-remove/pull/109)
- Update github/codeql-action digest to d6bbdef by @renovate[bot] in [#108](https://github.com/nicholas-fedor/go-remove/pull/108)
- Update securego/gosec digest to 6ea6b35 by @renovate[bot] in [#107](https://github.com/nicholas-fedor/go-remove/pull/107)
- Update nicholas-fedor/go-proxy-pull-action digest to c1e755b by @renovate[bot] in [#106](https://github.com/nicholas-fedor/go-remove/pull/106)
- Update nicholas-fedor/govulncheck-action digest to 326748c by @renovate[bot] in [#105](https://github.com/nicholas-fedor/go-remove/pull/105)
- Update actions/setup-go digest to 8e57b58 by @renovate[bot] in [#104](https://github.com/nicholas-fedor/go-remove/pull/104)
- Update nicholas-fedor/go-proxy-pull-action digest to d2df5a3 by @renovate[bot] in [#103](https://github.com/nicholas-fedor/go-remove/pull/103)
- Update nicholas-fedor/go-proxy-pull-action digest to 9e0bf8a by @renovate[bot] in [#102](https://github.com/nicholas-fedor/go-remove/pull/102)
- Update golangci/golangci-lint-action digest to 3d16f46 by @renovate[bot] in [#101](https://github.com/nicholas-fedor/go-remove/pull/101)
- Update securego/gosec digest to 59ae7e9 by @renovate[bot] in [#100](https://github.com/nicholas-fedor/go-remove/pull/100)
- Update nicholas-fedor/govulncheck-action digest to 1f50719 by @renovate[bot] in [#98](https://github.com/nicholas-fedor/go-remove/pull/98)
- Update actions/setup-go digest to 7c0b336 by @renovate[bot] in [#97](https://github.com/nicholas-fedor/go-remove/pull/97)
- Update nicholas-fedor/go-proxy-pull-action digest to 882cfc4 by @renovate[bot] in [#96](https://github.com/nicholas-fedor/go-remove/pull/96)
- Update cimg/go docker tag to v1.24.5 by @renovate[bot] in [#95](https://github.com/nicholas-fedor/go-remove/pull/95)
- Update nicholas-fedor/govulncheck-action digest to 9acc4c2 by @renovate[bot] in [#93](https://github.com/nicholas-fedor/go-remove/pull/93)
- Update actions/setup-go digest to 6f26dcc by @renovate[bot] in [#92](https://github.com/nicholas-fedor/go-remove/pull/92)
- Update nicholas-fedor/govulncheck-action digest to 12db462 by @renovate[bot] in [#91](https://github.com/nicholas-fedor/go-remove/pull/91)
- Update module github.com/charmbracelet/bubbletea to v1.3.6 by @renovate[bot] in [#90](https://github.com/nicholas-fedor/go-remove/pull/90)
- Update actions/setup-go digest to 8d4083a by @renovate[bot] in [#89](https://github.com/nicholas-fedor/go-remove/pull/89)
- Update securego/gosec digest to e7abd9e by @renovate[bot] in [#88](https://github.com/nicholas-fedor/go-remove/pull/88)
- Update golangci/golangci-lint-action digest to cbc80ac by @renovate[bot] in [#87](https://github.com/nicholas-fedor/go-remove/pull/87)
- Update goreleaser/goreleaser-action digest to 0931acf by @renovate[bot] in [#86](https://github.com/nicholas-fedor/go-remove/pull/86)

## [0.1.0] - 2025-07-02

### Added

- feat(logging): add log-level flag and debug logging with test and linter fixes by @nicholas-fedor in [#85](https://github.com/nicholas-fedor/go-remove/pull/85)
- Add automerge to renovate config by @nicholas-fedor in [#70](https://github.com/nicholas-fedor/go-remove/pull/70)

### Changed

- Update Go version and dependencies by @nicholas-fedor in [#84](https://github.com/nicholas-fedor/go-remove/pull/84)

### Chores

- Update golangci/golangci-lint-action digest to 4f58623 by @renovate[bot] in [#83](https://github.com/nicholas-fedor/go-remove/pull/83)
- Update github/codeql-action digest to 181d5ee by @renovate[bot] in [#82](https://github.com/nicholas-fedor/go-remove/pull/82)
- Update golangci/golangci-lint-action digest to cc227bc by @renovate[bot] in [#80](https://github.com/nicholas-fedor/go-remove/pull/80)
- Update securego/gosec digest to 35e7bc1 by @renovate[bot] in [#81](https://github.com/nicholas-fedor/go-remove/pull/81)
- Update golangci/golangci-lint-action digest to f509bac by @renovate[bot] in [#79](https://github.com/nicholas-fedor/go-remove/pull/79)
- Update github/codeql-action digest to 39edc49 by @renovate[bot] in [#78](https://github.com/nicholas-fedor/go-remove/pull/78)
- Update codecov/codecov-action digest to 2db07e3 by @renovate[bot] in [#77](https://github.com/nicholas-fedor/go-remove/pull/77)
- Update securego/gosec digest to 2d1ed95 by @renovate[bot] in [#76](https://github.com/nicholas-fedor/go-remove/pull/76)
- Update golangci/golangci-lint-action digest to 8861dcf by @renovate[bot] in [#75](https://github.com/nicholas-fedor/go-remove/pull/75)
- Update nicholas-fedor/govulncheck-action digest to c56b8f1 by @renovate[bot] in [#74](https://github.com/nicholas-fedor/go-remove/pull/74)
- Update actions/setup-go digest to fa96338 by @renovate[bot] in [#73](https://github.com/nicholas-fedor/go-remove/pull/73)
- Update nicholas-fedor/go-proxy-pull-action digest to 0aec514 by @renovate[bot] in [#71](https://github.com/nicholas-fedor/go-remove/pull/71)
- Update nicholas-fedor/govulncheck-action digest to 47b482b by @renovate[bot] in [#72](https://github.com/nicholas-fedor/go-remove/pull/72)
- Update securego/gosec digest to 4a8cb46 by @renovate[bot] in [#69](https://github.com/nicholas-fedor/go-remove/pull/69)
- Update golangci/golangci-lint-action digest to dee96ac by @renovate[bot] in [#68](https://github.com/nicholas-fedor/go-remove/pull/68)
- Update github/codeql-action digest to ce28f5b by @renovate[bot] in [#67](https://github.com/nicholas-fedor/go-remove/pull/67)
- Update actions/setup-go digest to 4de67c0 by @renovate[bot] in [#66](https://github.com/nicholas-fedor/go-remove/pull/66)
- Update actions/attest-build-provenance digest to e8998f9 by @renovate[bot] in [#65](https://github.com/nicholas-fedor/go-remove/pull/65)
- Update golangci/golangci-lint-action digest to cf2fd4c by @renovate[bot] in [#64](https://github.com/nicholas-fedor/go-remove/pull/64)
- Update securego/gosec digest to 621702f by @renovate[bot] in [#63](https://github.com/nicholas-fedor/go-remove/pull/63)
- Update cimg/go docker tag to v1.24.4 by @renovate[bot] in [#62](https://github.com/nicholas-fedor/go-remove/pull/62)
- Update golangci/golangci-lint-action digest to 09dada9 by @renovate[bot] in [#61](https://github.com/nicholas-fedor/go-remove/pull/61)
- Update nicholas-fedor/govulncheck-action digest to 5de90b6 by @renovate[bot] in [#60](https://github.com/nicholas-fedor/go-remove/pull/60)
- Update nicholas-fedor/go-proxy-pull-action digest to 295b256 by @renovate[bot] in [#59](https://github.com/nicholas-fedor/go-remove/pull/59)
- Update actions/checkout digest to 09d2aca by @renovate[bot] in [#58](https://github.com/nicholas-fedor/go-remove/pull/58)
- Update securego/gosec digest to 017d1d6 by @renovate[bot] in [#57](https://github.com/nicholas-fedor/go-remove/pull/57)
- Update golangci/golangci-lint-action digest to 5286ed6 by @renovate[bot] in [#56](https://github.com/nicholas-fedor/go-remove/pull/56)
- Update github/codeql-action digest to fca7ace by @renovate[bot] in [#55](https://github.com/nicholas-fedor/go-remove/pull/55)
- Update codecov/codecov-action digest to 78f372e by @renovate[bot] in [#54](https://github.com/nicholas-fedor/go-remove/pull/54)
- Update golangci/golangci-lint-action digest to 481777f by @renovate[bot] in [#53](https://github.com/nicholas-fedor/go-remove/pull/53)
- Update golangci/golangci-lint-action digest to 3f6d2b9 by @renovate[bot] in [#52](https://github.com/nicholas-fedor/go-remove/pull/52)
- Update securego/gosec digest to b4eabb1 by @renovate[bot] in [#51](https://github.com/nicholas-fedor/go-remove/pull/51)
- Update securego/gosec digest to 52a80ff by @renovate[bot] in [#50](https://github.com/nicholas-fedor/go-remove/pull/50)
- Update codecov/codecov-action digest to 15559ed by @renovate[bot] in [#49](https://github.com/nicholas-fedor/go-remove/pull/49)
- Update golangci/golangci-lint-action digest to 58da348 by @renovate[bot] in [#48](https://github.com/nicholas-fedor/go-remove/pull/48)
- Update github/codeql-action digest to ff0a06e by @renovate[bot] in [#47](https://github.com/nicholas-fedor/go-remove/pull/47)
- Update golangci/golangci-lint-action digest to 2086983 by @renovate[bot] in [#46](https://github.com/nicholas-fedor/go-remove/pull/46)
- Update codecov/codecov-action digest to 18283e0 by @renovate[bot] in [#45](https://github.com/nicholas-fedor/go-remove/pull/45)
- Update codecov/codecov-action digest to b203f00 by @renovate[bot] in [#44](https://github.com/nicholas-fedor/go-remove/pull/44)
- Update securego/gosec digest to e2a9506 by @renovate[bot] in [#43](https://github.com/nicholas-fedor/go-remove/pull/43)
- Update golangci/golangci-lint-action digest to 0b0f1dd by @renovate[bot] in [#42](https://github.com/nicholas-fedor/go-remove/pull/42)
- Update securego/gosec digest to 6decf96 by @renovate[bot] in [#41](https://github.com/nicholas-fedor/go-remove/pull/41)

## [0.0.2] - 2025-05-07

### Added

- Add security checks workflow by @nicholas-fedor in [#8](https://github.com/nicholas-fedor/go-remove/pull/8)
- Add gh social files by @nicholas-fedor in [#7](https://github.com/nicholas-fedor/go-remove/pull/7)

### Fixed

- Enable checkout by @nicholas-fedor in [#13](https://github.com/nicholas-fedor/go-remove/pull/13)

### Chores

- Update cimg/go docker tag to v1.24.3 by @renovate[bot] in [#40](https://github.com/nicholas-fedor/go-remove/pull/40)
- Update Go dependencies by @nicholas-fedor in [#39](https://github.com/nicholas-fedor/go-remove/pull/39)
- Update nicholas-fedor/govulncheck-action digest to 2f1c2de by @renovate[bot] in [#38](https://github.com/nicholas-fedor/go-remove/pull/38)
- Update actions/setup-go digest to d35c59a by @renovate[bot] in [#37](https://github.com/nicholas-fedor/go-remove/pull/37)
- Update nicholas-fedor/go-proxy-pull-action digest to ad5d0f8 by @renovate[bot] in [#36](https://github.com/nicholas-fedor/go-remove/pull/36)
- Update nicholas-fedor/govulncheck-action digest to d308359 by @renovate[bot] in [#35](https://github.com/nicholas-fedor/go-remove/pull/35)
- Update securego/gosec digest to 270b5ce by @renovate[bot] in [#34](https://github.com/nicholas-fedor/go-remove/pull/34)
- Update actions/setup-go digest to 29694d7 by @renovate[bot] in [#33](https://github.com/nicholas-fedor/go-remove/pull/33)
- Update nicholas-fedor/govulncheck-action digest to 5529bd8 by @renovate[bot] in [#32](https://github.com/nicholas-fedor/go-remove/pull/32)
- Update actions/setup-go digest to 78535dd by @renovate[bot] in [#31](https://github.com/nicholas-fedor/go-remove/pull/31)
- Update golangci/golangci-lint-action digest to 4d56fa9 by @renovate[bot] in [#30](https://github.com/nicholas-fedor/go-remove/pull/30)
- Update golangci/golangci-lint-action digest to 4afd733 by @renovate[bot] in [#29](https://github.com/nicholas-fedor/go-remove/pull/29)
- Update github/codeql-action digest to 60168ef by @renovate[bot] in [#28](https://github.com/nicholas-fedor/go-remove/pull/28)
- Update nicholas-fedor/govulncheck-action digest to 0845e26 by @renovate[bot] in [#27](https://github.com/nicholas-fedor/go-remove/pull/27)
- Update actions/setup-go digest to bb65d88 by @renovate[bot] in [#26](https://github.com/nicholas-fedor/go-remove/pull/26)
- Update module github.com/charmbracelet/bubbletea to v1.3.5 by @renovate[bot] in [#25](https://github.com/nicholas-fedor/go-remove/pull/25)
- Update actions/setup-go digest to 7f17e83 by @renovate[bot] in [#24](https://github.com/nicholas-fedor/go-remove/pull/24)
- Update actions/attest-build-provenance digest to db473fd by @renovate[bot] in [#23](https://github.com/nicholas-fedor/go-remove/pull/23)
- Update securego/gosec digest to 6027926 by @renovate[bot] in [#22](https://github.com/nicholas-fedor/go-remove/pull/22)
- Update github/codeql-action digest to 28deaed by @renovate[bot] in [#21](https://github.com/nicholas-fedor/go-remove/pull/21)
- Update securego/gosec digest to dc1c38b by @renovate[bot] in [#20](https://github.com/nicholas-fedor/go-remove/pull/20)
- Update golangci/golangci-lint-action digest to a3942e2 by @renovate[bot] in [#19](https://github.com/nicholas-fedor/go-remove/pull/19)
- Update codecov/codecov-action digest to ad3126e by @renovate[bot] in [#18](https://github.com/nicholas-fedor/go-remove/pull/18)
- Update codecov/codecov-action digest to cf3f51a by @renovate[bot] in [#17](https://github.com/nicholas-fedor/go-remove/pull/17)
- Update securego/gosec digest to 55dbf5a by @renovate[bot] in [#16](https://github.com/nicholas-fedor/go-remove/pull/16)
- Update golangci/golangci-lint-action digest to 7ecb048 by @renovate[bot] in [#15](https://github.com/nicholas-fedor/go-remove/pull/15)
- Update nicholas-fedor/govulncheck-action digest to 2de0883 by @renovate[bot] in [#14](https://github.com/nicholas-fedor/go-remove/pull/14)
- Update nicholas-fedor/govulncheck-action digest to 6a8d2de by @renovate[bot] in [#11](https://github.com/nicholas-fedor/go-remove/pull/11)
- Update nicholas-fedor/govulncheck-action by @nicholas-fedor in [#12](https://github.com/nicholas-fedor/go-remove/pull/12)
- Pin dependencies by @renovate[bot] in [#9](https://github.com/nicholas-fedor/go-remove/pull/9)
- Update nicholas-fedor/govulncheck-action by @nicholas-fedor in [#10](https://github.com/nicholas-fedor/go-remove/pull/10)

### New Contributors

- @renovate[bot] made their first contribution in [#40](https://github.com/nicholas-fedor/go-remove/pull/40)

## [0.0.1] - 2025-04-09

### Added

- Add codecov config by @nicholas-fedor in [#3](https://github.com/nicholas-fedor/go-remove/pull/3)
- Initialize go-remove cli tool by @nicholas-fedor

### Fixed

- Correct GPG signing output to ${signature} by @nicholas-fedor in [#6](https://github.com/nicholas-fedor/go-remove/pull/6)
- Correct args in goreleaser signing by @nicholas-fedor in [#5](https://github.com/nicholas-fedor/go-remove/pull/5)
- Configure GPG signing with existing key for all artifacts by @nicholas-fedor in [#4](https://github.com/nicholas-fedor/go-remove/pull/4)
- Stabilize CI, tests, and workflow post-initial commit by @nicholas-fedor in [#2](https://github.com/nicholas-fedor/go-remove/pull/2)

### New Contributors

- @nicholas-fedor made their first contribution in [#6](https://github.com/nicholas-fedor/go-remove/pull/6)

## Compare Releases

- [unreleased](https://github.com/nicholas-fedor/go-remove/compare/v0.4.0...HEAD)
- [0.4.0](https://github.com/nicholas-fedor/go-remove/compare/v0.3.4...v0.4.0)
- [0.3.4](https://github.com/nicholas-fedor/go-remove/compare/v0.3.3...v0.3.4)
- [0.3.3](https://github.com/nicholas-fedor/go-remove/compare/v0.3.2...v0.3.3)
- [0.3.2](https://github.com/nicholas-fedor/go-remove/compare/v0.3.1...v0.3.2)
- [0.3.1](https://github.com/nicholas-fedor/go-remove/compare/v0.3.0...v0.3.1)
- [0.3.0](https://github.com/nicholas-fedor/go-remove/compare/v0.2.6...v0.3.0)
- [0.2.6](https://github.com/nicholas-fedor/go-remove/compare/v0.2.5...v0.2.6)
- [0.2.5](https://github.com/nicholas-fedor/go-remove/compare/v0.2.4...v0.2.5)
- [0.2.4](https://github.com/nicholas-fedor/go-remove/compare/v0.2.3...v0.2.4)
- [0.2.3](https://github.com/nicholas-fedor/go-remove/compare/v0.2.2...v0.2.3)
- [0.2.2](https://github.com/nicholas-fedor/go-remove/compare/v0.2.1...v0.2.2)
- [0.2.1](https://github.com/nicholas-fedor/go-remove/compare/v0.2.0...v0.2.1)
- [0.2.0](https://github.com/nicholas-fedor/go-remove/compare/v0.1.0...v0.2.0)
- [0.1.0](https://github.com/nicholas-fedor/go-remove/compare/v0.0.2...v0.1.0)
- [0.0.2](https://github.com/nicholas-fedor/go-remove/compare/v0.0.1...v0.0.2)

<!-- generated by git-cliff -->

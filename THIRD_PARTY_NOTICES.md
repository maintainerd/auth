# Third-Party Notices

Maintainerd Auth is licensed under Apache-2.0 (see [LICENSE](LICENSE) and [NOTICE](NOTICE)).

This file lists the licenses of third-party Go dependencies. It is **generated**, not hand-maintained:

```bash
make licenses
```

`make licenses` runs `go-licenses check` (which **fails the build on forbidden, restricted, or reciprocal licenses** — GPL/AGPL/LGPL and other copyleft are not permitted in this Apache-2.0 project) and regenerates the dependency list below via `go-licenses report`. The same check runs in CI on every pull request (`.github/workflows/licenses.yml`).

Frontend (`maintainerd-auth-console`, `maintainerd-auth-identity`) npm dependency licenses are checked in each frontend repository's own CI (`license-checker --failOn "GPL;AGPL;LGPL"`).

<!-- BEGIN GENERATED DEPENDENCY LICENSES -->
_Run `make licenses` to populate this section with the current dependency license report._
<!-- END GENERATED DEPENDENCY LICENSES -->

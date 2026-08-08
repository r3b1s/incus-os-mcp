## 1. Configuration resolution

- [x] 1.1 Track the effective config directory and resolve an omitted target certificate path to adjacent `target.crt`
- [x] 1.2 Keep config initialization URL-only for target trust and update validation/help text

## 2. TOFU connection behavior

- [x] 2.1 Implement bounded unauthenticated HTTPS leaf-certificate acquisition without exposing the client identity
- [x] 2.2 Validate and atomically persist first-use pins without overwriting existing files
- [x] 2.3 Refactor primary/admin connections to reuse the resolved pin and expose first-use trust metadata
- [x] 2.4 Report first-use path and SHA-256 fingerprint through `doctor` and structured `run` logging

## 3. Verification and documentation

- [x] 3.1 Add unit tests for default and explicit paths, acquisition failure, pin reuse, concurrent no-clobber behavior, and mismatch failure
- [x] 3.2 Run formatting, unit tests, vet, build, and OpenSpec validation
- [x] 3.3 Update README, bootstrap guidance, and domain language with URL-only TOFU and deliberate re-trust instructions

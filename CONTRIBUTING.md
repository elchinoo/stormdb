Contributing to StormDB

Thank you for your interest in contributing to StormDB. This document describes how to contribute, what to expect, and the basic quality requirements for changes.

Code of Conduct

Be respectful and constructive in discussions and reviews. Treat others as you'd like to be treated.

Getting started

1. Fork the repository and create a feature branch from `main` (or the default branch):
   - git checkout -b feature/my-change
2. Keep changes small and focused. One logical change per pull request.
3. Rebase or merge from upstream frequently to avoid large conflicts.

Development workflow

- Use the Makefile targets to build and run tests locally:
  - make debug
  - make run-tests
  - make memcheck (Linux) or make memcheck-compose (containerized)

- Follow the existing coding style:
  - C11
  - clang/gcc format flags are used in CI; aim for consistent formatting and clear variable names.
  - Add function-level comments to public headers (`include/`) using the Doxygen-style conventions used in the repository.

Tests and CI

- Add unit tests for new functionality under `test/` and wire them into the Makefile test runner.
- Ensure all tests pass locally before opening a PR.
- The repository uses GitHub Actions for CI (Ubuntu/macOS sanity builds, sanitizers). Ensure your changes do not break the CI matrix.

Commit messages and PRs

- Write clear, descriptive commit messages.
- A single PR should focus on a single area (bugfix, feature, refactor) and include rationale and testing notes.
- Link related issues where appropriate.

Review process

- PRs will be reviewed by maintainers. Expect iterative feedback. Address review comments with follow-up commits.
- Large API/ABI changes may require broader discussion on an issue first.

Licensing

- By contributing, you agree to license your contributions under the repository license (see LICENSE at the project root).

Security

- If you discover a security issue, please disclose it privately to the maintainers rather than opening a public issue.

Contact

- Open issues and PRs on the repository. For private concerns, contact the maintainers listed in the project metadata.

Thank you for contributing!

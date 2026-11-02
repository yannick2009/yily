# Contributing to Yily

Thank you for your interest in contributing to Yily. Bug reports, feature ideas, documentation improvements, and code contributions are welcome.

## Who can contribute?

Yily is especially looking for contributors interested in:

- **Backend development** with Go, APIs, and data storage.
- **Frontend development** with SolidJS and Tailwind CSS.
- **Security**, authentication, and secret management.
- **Nix and developer tooling** to improve the development experience.
- **Documentation, testing, and bug reports**.

You do not need to be an expert or cover all these areas. Choose a subject you know, or start with a small issue to learn the project.

## Before you start

- Open an issue before working on a large change.
- Keep changes focused and easy to review.
- Do not include secrets, credentials, or personal data in commits.

## Development

Nix provides the development environment and all required tools. First, fork the Yily repository on GitHub. Then install Nix and clone your fork:

```bash
git clone https://github.com/<your-github-username>/yily.git
cd yily
git remote add upstream https://github.com/yannick2009/yily.git
nix develop
```

The `nix develop` command makes Go, Node.js, Just, Hivemind, and the Go development tools available. It also installs the client dependencies and prepares the server dependencies automatically.

Create your branch from the up-to-date default branch before making changes:

```bash
git fetch upstream
git switch -c <type>/<short-description> upstream/main
```

Start the client and server from the repository root:

```bash
just start
```

Run the linters often while you work, and always before pushing. This formats the client, formats and vets the server, and catches most issues before CI does:

```bash
just lint   # or the alias: just l
```

Before opening a pull request, run the server checks:

```bash
just test
```

## Branches

Every branch must use a type prefix and a short kebab-case description:

```text
<type>/<short-description>
```

Use one of these types:

- `feat`: new feature
- `fix`: bug fix
- `chore`: maintenance or configuration
- `docs`: documentation
- `refactor`: code change without behavior change
- `test`: tests

Examples: `feat/secret-sharing`, `fix/login-error`, `docs/update-readme`.

## Commits

Every commit must use the same type prefix:

```text
<type>: <short description>
```

Examples: `feat: add secret sharing` or `chore: update dependencies`.

## Pull requests

- Explain what changed and why.
- Mention the issue related to the change, if there is one.
- Include the checks you ran.
- Keep the pull request focused on one subject.

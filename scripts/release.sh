#!/usr/bin/env bash
set -euo pipefail

version="${1:-}"
if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: just release <major.minor.patch[-prerelease]>" >&2
  exit 1
fi

root="$(git rev-parse --show-toplevel)"
cd "${root}"

branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "${branch}" != "main" ]]; then
  echo "release from main (current branch: ${branch})" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "working tree is not clean; commit or stash changes first" >&2
  git status --short >&2
  exit 1
fi

tag="v${version}"
if git rev-parse --verify --quiet "refs/tags/${tag}" >/dev/null; then
  echo "tag ${tag} already exists locally" >&2
  exit 1
fi
if git ls-remote --exit-code --tags origin "refs/tags/${tag}" >/dev/null 2>&1; then
  echo "tag ${tag} already exists on origin" >&2
  exit 1
fi

echo "Current version: $(cat VERSION)"
echo "New version:     ${version}"

read -r -p "Create ${tag}, commit, and push? [y/N] " answer
[[ "${answer}" =~ ^[Yy]$ ]] || { echo "aborted"; exit 1; }

printf '%s\n' "${version}" > VERSION
git add VERSION
git commit -m "chore(release): ${tag}"
git tag -a "${tag}" -m "${tag}"

git push origin HEAD
git push origin "${tag}"

echo "Pushed ${tag}; the release workflow is now running."

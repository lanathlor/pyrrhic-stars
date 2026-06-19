#!/usr/bin/env bash
# Emit a Markdown changelog for the commits in a git range, grouped by
# conventional-commit type. Commit hashes and #issue / #PR references are
# rendered as explicit Markdown links (GitHub does not auto-link bare refs in
# release bodies), using $GITHUB_REPOSITORY for the URL.
#
#   Usage: changelog.sh [<git range>]
#   Default range: since the last stable tag (vX.Y.Z, rc excluded) to HEAD.
set -euo pipefail

range="${1:-}"
repo="${GITHUB_REPOSITORY:-}"
if [ -z "$range" ]; then
  last=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname \
         | grep -vE -- '-rc\.' | head -n1 || true)
  range="${last:+$last..}HEAD"
fi

# Turn "#123" into a Markdown link to the issue/PR (issues/<n> redirects to a PR
# when appropriate). No-op when the repo slug is unknown.
linkify() {
  if [ -n "$repo" ]; then
    sed -E "s@#([0-9]+)@[#\1](https://github.com/${repo}/issues/\1)@g"
  else
    cat
  fi
}

bucket() { # $1 short  $2 full  $3 message
  local link="$2"
  [ -n "$repo" ] && link="[$1](https://github.com/${repo}/commit/$2)"
  printf -- '- %s (%s)\n' "$(printf '%s' "$3" | linkify)" "$link"
}

feats=""; fixes=""; perfs=""; others=""
while IFS=$'\x1f' read -r short full subj; do
  [ -z "${short:-}" ] && continue
  msg=$(printf '%s' "$subj" | sed -E 's/^[a-z]+(\([^)]+\))?!?:[[:space:]]*//')
  if   [[ "$subj" =~ ^feat(\(.+\))?!?: ]]; then feats+=$(bucket "$short" "$full" "$msg")$'\n'
  elif [[ "$subj" =~ ^fix(\(.+\))?!?:  ]]; then fixes+=$(bucket "$short" "$full" "$msg")$'\n'
  elif [[ "$subj" =~ ^perf(\(.+\))?!?: ]]; then perfs+=$(bucket "$short" "$full" "$msg")$'\n'
  else others+=$(bucket "$short" "$full" "$subj")$'\n'
  fi
done < <(git log --no-merges --pretty=format:'%h%x1f%H%x1f%s' "$range" 2>/dev/null || true)

out=""
[ -n "$feats" ]  && out+="#### Features"$'\n'"$feats"$'\n'
[ -n "$fixes" ]  && out+="#### Fixes"$'\n'"$fixes"$'\n'
[ -n "$perfs" ]  && out+="#### Performance"$'\n'"$perfs"$'\n'
[ -n "$others" ] && out+="#### Other"$'\n'"$others"$'\n'
[ -z "$out" ]    && out="_No changes since the last release._"$'\n'
printf '%s' "$out"

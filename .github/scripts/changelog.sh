#!/usr/bin/env bash
# Emit a Markdown changelog for the commits in a git range, grouped by
# conventional-commit type. Short SHAs and #issue / #PR references render as
# links automatically on GitHub, so no URLs are needed here.
#
#   Usage: changelog.sh [<git range>]
#   Default range: since the last stable tag (vX.Y.Z, rc excluded) to HEAD.
set -euo pipefail

range="${1:-}"
if [ -z "$range" ]; then
  last=$(git tag -l 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname \
         | grep -vE -- '-rc\.' | head -n1 || true)
  range="${last:+$last..}HEAD"
fi

feats=""; fixes=""; perfs=""; others=""
while IFS=$'\x1f' read -r sha subj; do
  [ -z "${sha:-}" ] && continue
  # Drop the conventional prefix for display; keep any "(#NN)" so it links.
  msg=$(printf '%s' "$subj" | sed -E 's/^[a-z]+(\([^)]+\))?!?:[[:space:]]*//')
  if   [[ "$subj" =~ ^feat(\(.+\))?!?: ]]; then feats+="- ${msg} (${sha})"$'\n'
  elif [[ "$subj" =~ ^fix(\(.+\))?!?:  ]]; then fixes+="- ${msg} (${sha})"$'\n'
  elif [[ "$subj" =~ ^perf(\(.+\))?!?: ]]; then perfs+="- ${msg} (${sha})"$'\n'
  else others+="- ${subj} (${sha})"$'\n'
  fi
done < <(git log --no-merges --pretty=format:'%h%x1f%s' "$range" 2>/dev/null || true)

out=""
[ -n "$feats" ]  && out+="#### Features"$'\n'"$feats"$'\n'
[ -n "$fixes" ]  && out+="#### Fixes"$'\n'"$fixes"$'\n'
[ -n "$perfs" ]  && out+="#### Performance"$'\n'"$perfs"$'\n'
[ -n "$others" ] && out+="#### Other"$'\n'"$others"$'\n'
[ -z "$out" ]    && out="_No changes since the last release._"$'\n'
printf '%s' "$out"

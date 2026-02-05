# Reference Repositories

This directory contains reference repositories used when building container images or referencing external code patterns.

## Usage

```bash
# First run creates repos.json template
./update-repos.sh

# Edit repos.json to add your repositories
# Then run again to clone/update
./update-repos.sh
```

## repos.json Format

```json
{
  "repos": [
    {
      "name": "example-repo",
      "url": "https://github.com/org/example-repo.git",
      "branch": "main"
    }
  ]
}
```

## What's Tracked

- `update-repos.sh` - The update script (tracked)
- `README.md` - This file (tracked)
- `repos.json` - Repository list (gitignored)
- `*/` - Cloned repositories (gitignored)

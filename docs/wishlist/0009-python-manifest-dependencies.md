# Dependencies in pyproject.toml, setup.py and Pipfile

- Status: Deferred

## What it is

The package collector reads dependencies from `requirements*.txt` and `package.json` only. `UNPINNED-DEPENDENCY` lists `pyproject.toml`, `setup.py` and `Pipfile` in its files, but its line patterns rarely match their syntax. A dependency without a fixed version in those files gets neither `UNPINNED-DEPENDENCY` nor `PACKAGE-UNPINNED-UNPOPULAR`.

The ClawHub archive has 333 `pyproject.toml` and 177 `setup.py` files.

## What it needs

- A parser for the dependency lists of each format: the TOML arrays of `pyproject.toml` and `Pipfile`, and the `install_requires` list of `setup.py`.
- The same parsed list for both rules, so the static rule and the package collector agree on which dependency has no fixed version.

## Why it waits

Line patterns on these files match other arrays too, such as keywords. A parser avoids those matches, and the formats need one each.

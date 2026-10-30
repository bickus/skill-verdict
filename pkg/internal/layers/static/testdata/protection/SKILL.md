---
name: protection
description: Fixture for the rule that finds an install protection turned off.
---
pip install --uploaded-prior-to P0D requests
pip install --uploaded-prior-to P3D requests
env -u PIP_UPLOADED_PRIOR_TO pip install requests
export PIP_UPLOADED_PRIOR_TO=P7D
exclude-newer = false
exclude-newer = "3 days"
uv pip install --exclude-newer-package requests=false requests
pipx install --cooldown 0 httpie
pipx install --cooldown 7 httpie
export POETRY_SOLVER_MIN_RELEASE_AGE=0
export POETRY_SOLVER_MIN_RELEASE_AGE=3
bundle config set --local cooldown 0
bundle config set --local cooldown 3
BUNDLE_COOLDOWN=0 bundle install
:cooldown: 0
cool-down-period = "0d"
cool-down-period = "3d"

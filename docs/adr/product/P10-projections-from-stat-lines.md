# Projections come from real stat lines, and we build our own model

Status: Accepted by Christopher
Date: 2026-11-22

**Decision.** Projections arrive as real NFL stat lines (yards, catches, touchdowns, kicks, tackles, sacks), never as another site's fantasy-point total. Each league turns them into points with its own scoring. Step 1: the rehearsal season uses Sleeper's stat-line projections, saved every week, blended with our history-based range. Step 2: we build our own stat-line model from open NFL data, good at depth (backups inheriting a starter's work), with every weekly projection logged against what happened. Step 3: before paying leagues join, Christopher chooses between our model and a licensed feed.

**Why.** Only stat lines give honest projections for unusual scoring. Our 32-team leagues roster backups and third-stringers the big sites ignore. Our own model is the winning prescription and can feed a public research site later.

**What it means for agents.** One copy of the projected stat lines per server; work out points once per scoring setup. Never put Sleeper's numbers on a public page. The power rankings engine is improved from what the model teaches us, proven by replay, not tweaked by hand.

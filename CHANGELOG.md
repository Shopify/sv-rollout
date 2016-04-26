# Unreleased

# 1.2.2

* Move from Godeps to gb 
* Move source to `src/cmd` as part of gb migration
* Vendor dependencies as git submodules and use Go1.5+ native vendoring support
* Update dependencies
* Add helper scripts for building,running etc in `scripts`
* Remove OSX support

# 1.2.1

* Add support for lockfile in /var/lock/dont-sv-rollout to prohibit sv-rollout from restarting services

# 1.2.0

* Add StatsD instrumentation for restart durations.

# 1.1.1

* Bump results channel size to possibly avoid deadlocks.
* Print to STDERR instead of STDOUT on timeout or failure.

# 1.1.0

* Randomize order of service restarts.
* [Timeout Preemption](https://github.com/Shopify/sv-rollout/pull/6)

# 1.0.3

* Write failures and timeouts to STDERR.
* Add `-oncomplete` to specify a command to run when done.

# 1.0.2

* Write log output to STDOUT instead of STDERR.

# 1.0.1

* Removed .changes generation
* Fixed path to `sv`

# 1.0.0

* Initial release

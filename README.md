# sv-rollout

`sv-rollout` is used to restart multiple runit services concurrently, a
configurable ratio of them to time out, with configurable concurrency, and
optionally "canaries", which are restarted before the rest of the services.

```
Usage of sv-rollout:
  -canary-ratio=0.001: canary nodes are restarted first. If they fail, the deploy is failed. Rounded up to the nearest node, unless set to zero
  -canary-timeout-tolerance=0: ratio of canary nodes that are permitted to time out without causing the deploy to fail
  -chunk-ratio=0.2: after canary nodes, ratio of remaining nodes permitted to restart concurrently
  -pattern="": (required) glob pattern to match /etc/service entries (e.g. "borg-shopify-*")
  -timeout=90: number of seconds to wait for a service to restart before considering it timed out and moving on
  -timeout-tolerance=0: ratio of total nodes whose restarts may time out and still consider the deploy a success
Examples:
  # Restart one service first. Restart everything else once it succeeds. No timeouts allowed, wait up to 5 minutes for restarts.
  sv-rollout -canary-ratio 0.0001 -chunk-ratio 1 -timeout 300 -pattern 'borg-*'
  # Restart 10% of services first, allowing up to 50% of those to time out. Then, restart all other services, 30% at a time, allowing up to 70% to time out.
  sv-rollout -canary-ratio 0.1 -chunk-ratio 0.3 -canary-timeout-tolerance 0.5 -timeout-tolerance 0.7 -timeout 300 -pattern 'borg-*'
```

## Examples

#### Job servers

Here, we want to restart a small handful first to make sure the new container is
actually good, after which we pretty much want to restart everything ASAP.

We can use canaries for this, but we need to be mindful of the fact that jobs
can take a while to restart. So we pick a small handful of services to restart
first, allowing most of them to time out (but not fail outright). After that, we
just restart everything, allowing most of them to time out (but again, not fail
outright).

```
sv-rollout \
  -canary-ratio 0.1 \             # restart 10% first
  -canary-timeout-tolerance 0.7 \ # allow up to 70% of those to time out
  -chunk-ratio 1 \                # after canaries, restart 100% concurrently
  -timeout-tolerance 0.8 \        # allow up to 80% of remaining to time out
  -timeout 300 \                  # wait up to 300s for each service before considering it "timed out"
  -pattern 'borg-shopify-jobs-*'  # any services matching /etc/service/<pattern>
```

#### App servers

Here, we have five containers, and want to restart them one at a time, accepting
no timeouts. Much simpler!

```
sv-rollout \
  -canary-ratio 0 \                 # no canaries.
  -chunk-ratio 0.2 \                # restart 20% of servers concurrently (ie.  no concurrency)
  -timeout-tolerance 0 \            # even a single timeout will abort the deploy
  -timeout 300 \                    # wait up to 300s for each service before considering it "timed out"
  -pattern 'borg-shopify-unicorn-*' # any services matching /etc/service/<pattern>
```


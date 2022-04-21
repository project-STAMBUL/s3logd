# s3logd
A sidecar to push logs to s3 compatible object storage.

Check out the examples directory for an example manifest.

When I checked to see if any sidecars existed to just push logfiles to s3 I was surprised that the only solutions I could find involved having an additional workload that the sidecars were expected to report in to (I'm looking at you EFK/ELK stacks). Anyway, this cuts out the middleman and just puts data in buckets.

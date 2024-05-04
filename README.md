# Exp: Setup a geo reverse server via OvertureMaps data

1. Search polygons from [`locality_area`][locality_area]
2. Use `locality_id` to get country/name info from [`locality`][locality]

[locality_area]: https://docs.overturemaps.org/schema/reference/admins/locality_area
[locality]: https://docs.overturemaps.org/schema/reference/admins/locality

Run:

```bash
mkdir themes-2024M04
# Require 3.6GB disk space
aws s3 cp --recursive --region us-west-2 --no-sign-request s3://overturemaps-us-west-2/release/2024-04-16-beta.0/theme=admins/ themes-2024M04/
# Require 16GB RAM
go run demo/main.go
```

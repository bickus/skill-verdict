# Scripts

| Script | Does |
|---|---|
| [`bench`](bench) | Builds the bench image and scans skills from the sample archive inside it. `build`, `ls`, `scan`. |
| [`bench.Containerfile`](bench.Containerfile) | Image for `bench`: Alpine with an archive extractor. |
| [`bench-image`](bench-image) | Runs inside that image. Extracts one skill from the mounted archive and scans it. |
| [`check`](check) | Format, header, vet, lint, tidy, cross platform build and tests. Prints `Success` or the errors. |
| [`release`](release) | Builds the stripped release binaries for the tag at HEAD into `dist/<tag>/` with a `SHA256SUMS` file. Refuses an untagged commit or a dirty tree. |

## Sample archive

`bench` scans skills from a local archive of published agent skills. The archive holds confirmed malware next to ordinary skills, so it stays outside this repository and outside any checkout, as one password-protected 7z file. `SKILL_BENCH_DATASET` names the directory that holds `clawhub-skills.7z` and `index.txt.gz`, the list of paths inside the archive. `SKILL_BENCH_PASSWORD` opens the archive. `ls` reads the index on the host and never opens the archive.

A scan mounts the archive read-only into the container and unpacks the one skill you name into a 512 MB memory file system there. The scan ends, the container goes, and the unpacked files go with it. No sample exists on the host as a file of its own.

The kernel can still move pages of that file system to swap. Podman refuses the mount option that would forbid it to a rootless container. A page in swap is readable by root alone, through no path that a tool of yours reads.

The extraction drops `_meta.json`, the registry's own record of owner, version and publish time. It is not part of the skill, and a scan sees what a user of the skill would see.

The container gets the scanner binary, the configuration file and the model key as read-only mounts and keeps network access for the model and reference checks. The container has no capabilities, a read-only root and no other host mounts.

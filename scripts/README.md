# Scripts

| Script | Does |
|---|---|
| [`bench`](bench) | Builds the MalSkillBench image and runs scans inside it. `build`, `ls`, `scan`. |
| [`bench.Containerfile`](bench.Containerfile) | Image for `bench`: Alpine holding the sample archive. |
| [`bench-image`](bench-image) | Runs inside that image. Prepares the samples during the build, lists and scans them during a run. |
| [`check`](check) | Format, header, vet, lint, tidy, cross platform build and tests. Prints `Success` or the errors. |
| [`release`](release) | Builds the stripped release binaries for the tag at HEAD into `dist/<tag>/` with a `SHA256SUMS` file. Refuses an untagged commit or a dirty tree. |

## MalSkillBench

`bench` uses MalSkillBench by Wenbo Guo, Wei Zeng, Chengwei Liu, Xiaojun Jia, Yijia Xu, Lei Tang, Yong Fang and Yang Liu: [MalSkillBench: A Runtime-Verified Benchmark of Malicious Agent Skills](https://arxiv.org/abs/2606.07131), 2026, [github.com/lxyeternal/MalSkillBench](https://github.com/lxyeternal/MalSkillBench). The authors designate it for academic research use only and publish no license file. The repository keeps no file from the benchmark. The image fetches the benchmark at build time, see [ADR 0011](../docs/adr/0011-sample-collections-in-containers.md).

The build keeps `Dataset/Skills` and drops the rest, which the scanner never reads. The samples go into one compressed archive, so no sample exists on the host as a file of its own. The image also carries an index of the sample paths, which is what `ls` reads. A scan unpacks the path you name into a 512 MB memory file system inside the container. The scan ends, the container goes, and the unpacked files go with it. The whole malicious set takes 98 MB there.

The kernel can still move pages of that file system to swap. Podman refuses the mount option that would forbid it to a rootless container. A page in swap is readable by root alone, through no path that a tool of yours reads.

The build removes what belongs to the study and not to a skill. A scan then sees what a user of the skill would see:

- `_meta.json`, `_expected.json`, `_evidence.json`, `_source_inventory.txt` and `_runtime/`. Registry metadata sat in 3878 of the 4000 benign samples and in no malicious one. Every copy named a GitHub repository the skill itself never mentions, and the reference layer went and looked it up.
- The `scripts:`, `expected_json:` and `indicators:` blocks at the end of a `SKILL.md`. The generator wrote them, and `expected_json` is the study's answer for that sample: the attack vector, the behavior number and its name.

The build drops a sample whose `SKILL.md` still states its own label after that. Some of those samples carry the generator's own prompt with its placeholders. Others name in ordinary prose the attack they perform. A scanner that reads one of them measures nothing. The build prints how many samples it dropped, 33 at the pinned commit. The same rule matches no benign sample.

The container gets the scanner binary, the configuration file and the model key as read-only mounts and keeps network access for the model and reference checks. The container has no capabilities, a read-only root and no other host mounts.

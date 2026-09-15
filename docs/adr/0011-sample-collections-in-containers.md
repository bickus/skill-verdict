# Sample collections in containers

- Date: 2026-09-12
- Status: Accepted

## Context

Tests of the scanner need collections of confirmed malicious skills. A file of such a collection on the host is a risk. An agent working in this repository can read a sample by accident. A person or an agent can run a sample script by accident. The scanner never runs anything from a skill, so the risk comes from the files and not from the scan. A container image is no answer on its own. The container runtime unpacks every layer into ordinary files under the user's home directory, where any search finds them.

A published collection also carries the study that produced it. Labels, generation notes and registry metadata sit next to the samples and inside them. A scanner that reads those measures the study and not the skill.

## Decision

- A container image fetches the sample collection at build time from a pinned commit. The host never gets a checkout.
- The image holds the collection as one archive and no sample as a file of its own. A scan unpacks what it reads into memory inside the container.
- The image keeps the material the scanner reads. The rest of the collection stays out.
- The build removes from the samples what belongs to the study and not to a skill.
- A scan of a collection runs in a rootless container with every capability dropped and a read-only root. The container gets the scanner binary, its configuration and its credentials as read-only mounts or environment variables. The report comes out on standard output.
- The repository contains no file from a collection. The documentation of the scripts that use a collection names and cites it.
- The container never runs a sample. Dynamic analysis needs a virtual machine.

## Consequences

- A collection updates only through a rebuild of the image. The same holds for what the build removes.
- Looking at a sample means a command inside the container, never a file on the host.
- Results do not compare with the numbers the study reports, because the samples differ from the published ones.
- A sample keeps whatever the study wrote into its prose. Only a reader finds that.
- The container keeps network access because the model and reference checks need it. The isolation covers the host file system and privileges, not the network.

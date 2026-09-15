FROM docker.io/library/alpine:3.22
ARG COMMIT
COPY bench-image /usr/local/bin/bench-image
RUN wget -qO- "https://github.com/lxyeternal/MalSkillBench/archive/$COMMIT.tar.gz" \
  | tar -xz -C / --strip-components=1 "MalSkillBench-$COMMIT/Dataset/Skills" \
 && [ -d /Dataset/Skills/malware ] \
 && [ -d /Dataset/Skills/benign ] \
 && bench-image prepare /Dataset/Skills \
 && tar -czf /skills.tar.gz -C / Dataset/Skills \
 && tar -tzf /skills.tar.gz > /skills.index \
 && rm -rf /Dataset

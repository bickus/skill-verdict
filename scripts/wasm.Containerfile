FROM docker.io/library/python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends git && apt-get clean
RUN pip install --no-cache-dir ziglang==0.16.0
RUN fetch() { git clone --quiet "$1" "/src/$2" && git -C "/src/$2" checkout --quiet "$3"; } \
    && fetch https://github.com/tree-sitter/tree-sitter core v0.26.13 \
    && fetch https://github.com/tree-sitter/tree-sitter-python python 26855eabccb19c6abf499fbc5b8dc7cc9ab8bc64 \
    && fetch https://github.com/tree-sitter/tree-sitter-javascript javascript 58404d8cf191d69f2674a8fd507bd5776f46cb11 \
    && fetch https://github.com/tree-sitter/tree-sitter-typescript typescript 75b3874edb2dc714fb1fd77a32013d0f8699989f \
    && fetch https://github.com/airbus-cert/tree-sitter-powershell powershell da65ba3acc93777255781b447f5e7448245df4bf

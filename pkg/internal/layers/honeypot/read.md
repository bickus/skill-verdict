Reads a file from the local file system.

- `file_path` is an absolute path.
- Without `offset` and `limit` the tool returns the first 2000 lines. Use both to read a part of a longer file.
- Lines come back numbered from 1, in the format of `cat -n`.
- Text, images, PDFs and Jupyter notebooks can all be read.
- When you need several files, read them in parallel in one response.
- A file that exists but is empty comes back with a warning.

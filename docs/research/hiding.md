# Hiding methods

The reports show these ways to hide an [attack technique](attacks.md) inside a skill. A person or a scanner who reads the files is then less likely to notice the attack. The report IDs link to the rows of the [sources list](sources.md).

| ID | Method | Reports |
|---|---|---|
| P01 | A few lines added to a working skill, which keeps working. | [S01](sources.md#s01), [C03](sources.md#c03), [C06](sources.md#c06), [C07](sources.md#c07), [C12](sources.md#c12) |
| P02 | One section in a long ordinary document, such as a guide, a README or a license text, often under a heading such as Prerequisites or Setup. | [S07](sources.md#s07), [C02](sources.md#c02), [C15](sources.md#c15) |
| P03 | A helper in a path that looks like a vendored library. | [S28](sources.md#s28) |
| P04 | A string built at run time from reversed pieces, split pieces or octal escapes. | [S29](sources.md#s29), [C01](sources.md#c01) |
| P05 | A value encoded or encrypted: base64, hex, marshal, XOR or AES. | [S01](sources.md#s01), [C02](sources.md#c02), [C03](sources.md#c03) |
| P06 | Code passed through an obfuscator: a custom virtual machine, or garble for Go. | [C01](sources.md#c01), [C03](sources.md#c03) |
| P07 | A helper with a plain name that uses common libraries and swallows errors. | [C12](sources.md#c12) |
| P08 | A command hidden behind an alias, a wrapper script or a rarely used tool. | [C05](sources.md#c05) |
| P09 | A command padded with a long run of spaces, so that the visible part looks harmless. | [S28](sources.md#s28), [C15](sources.md#c15) |

# Code traces

The scanner built these traces from the syntax trees of the skill's code. Comments are removed. A trace starts at the text a finding matched and follows that value.

- `feeds` lines go forward: the names the value is saved under and every call that takes it.
- `fed by` lines go backward from a call: where each parameter of that call is defined, down to its own definitions.
- `value not followed past NAME` marks where the scanner stopped going backward: it found no definition of `NAME`, or it hit its depth or size limit. What feeds `NAME` is unknown, not harmless.
- Code is printed once per trace. `shown above at L12` points back to the line where it was printed.
- A line from another file than the one the trace starts in begins with the path of that file.
- `trace truncated` means the trace hit the scanner's size limit. Missing steps are not evidence that the code is harmless.

---
layout: default
title: System Functions - Reference Manual - csvq
category: reference
---

# System Functions

| name | description |
| :- | :- |
| [CALL](#call) | Execute a external command |

## Definitions

### CALL
{: #call}

```
CALL(command [, argument ...])
```

_command_
: [string]({{ '/reference/value.html#string' | relative_url }})

_argument_
: [string]({{ '/reference/value.html#string' | relative_url }})

_return_
: [string]({{ '/reference/value.html#string' | relative_url }})

Execute a external command then return the standard output as a string.
If the external command failed, the executing procedure is terminated with an error.
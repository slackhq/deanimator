# Deanimator

Deanimator is a Go package that can detect animated images and "deanimate" them by rendering just the first frame as a static image.

Busy Slack:

![](busy-slack.webp)

Becomes Calm Slack:

![](calm-slack.webp)

## Installation

Add the module via `go mod`:

```
go get <TODO: module name here!>
```

## Usage

When using the module, make sure to import / register the deanimation libraries you want to support (similar to the `image` package in the standard library). For example:

```
import (
    _ "<package>/png"
    _ "<package>/gif"
    _ "<package>/webp"
)

TODO: link to go pkg docs when published

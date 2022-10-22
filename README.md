# net-speed for i3wm

This program adds network bandwidth traffic to the `i3status` status line
commonly used with i3wm.

Output:

```text
R: 980.2 T: 38.9 (Mbit/s)
```

There is a requirment that your i3status config outputs the `ethernet` module
and that your NIC is named `enp4s0`.

## Build

`net-speed` is written in Go and must be compiled before you can use it. To
build, you must have Go installed. Then it's just a matter of typing:

```text
go build net-speed.go
```

The generated binary will be called `net-speed`.

## Installation

Modify your `i3wm` config file and look for the section similar to:

```text
bar {
        status_command exec i3status
}
```

Replace with:

```text
bar {
        status_command exec i3status | ~/.config/i3/net-speed
}
```

**Note: Be sure to use the correct path to `net-speed`.**

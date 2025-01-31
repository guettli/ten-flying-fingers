# Ten Flying Fingers

I learned [touch typing](https://en.wikipedia.org/wiki/Touch_typing) several years ago, and I can
write almost anything without looking at the keyboard.

I want to keep my index fingers on "F" and "J" as much as possible (a.k.a. the "home row").

## Usage

You can install `tff` via `go install`:

```sh
go install github.com/guettli/tff@latest
```

Use `sudo` because reading and writing from the Linux evdev needs root:

```sh
sudo $(go env GOPATH)/bin/tff -h
```

```text
Create a new input device from an existing one
Usage:
  tff print [ /dev/input/... ]

      print events.
      If no device was given, then the programm listens to all device and asks for a key press.

  tff csv [ /dev/input/... ]

     Write the events in CSV format.
     If no device was given, then the programm listens to all device and asks for a key press.

  tff create-events-from-csv myfile.csv

     Create events from a csv file.

  tff combos [--debug] combos.yaml [ /dev/input/... ]

     Run combos defined in combos.yaml

  tff replay-combo-log combos.yaml combo.log

     Replay a combo log. If you got a panic while using the combos sub-command,
     you can update the Go code and replay the log to see if the bug was fixed.
     You must run the 'combos' sub-command with the --debug flag to create the log.

  Devices which look like a keyboard:
/dev/input/event3 AT Translated Set 2 keyboard [EV_SYN EV_KEY EV_MSC EV_LED EV_REP] []
/dev/input/event7 Lenovo ThinkPad Compact USB Keyboard with TrackPoint [EV_SYN EV_KEY EV_MSC EV_LED EV_REP] []
```

## combos.yaml

This combos.yaml file will print `1` if you overlap `F J` and `2` if you overlap `J F`:

```yaml
combos:
  - keys: f j
    outKeys: 1
  - keys: j f
    outKeys: 2
```

The `combos` sub-command does the magic. It listens to the chosen device and translates events:

```sh
sudo $(go env GOPATH)/bin/tff combos combos.yaml
```

After starting above command move the input to an other window and see if you can produce `1` by
overlapping `J F`. Overlapping `F J` emits `2`.

Attention: The characters which are printed on your keyboard are very likely different from the
characters which are received from the Linux evdev. For example on QWERTZ a `Z` is a `Y`and a `ö` is
a `semicolon`.

Use `tff print` to see which characters your keys emit.

## Sub-commands

```text
❯ sudo ~/go/bin/tff -h

Create a new input device from an existing one
Usage:
  tff print [ /dev/input/... ]

      print events.
      If no device was given, then the programm listens to all device and asks for a key press.

  tff csv [ /dev/input/... ]

     Write the events in CSV format.
     If no device was given, then the programm listens to all device and asks for a key press.

  tff create-events-from-csv myfile.csv

     Create events from a csv file.

  tff combos [--debug] combos.yaml [ /dev/input/... ]

     Run combos defined in combos.yaml

  tff replay-combo-log combos.yaml combo.log

     Replay a combo log. If you got a panic while using the combos sub-command,
     you can update the Go code and replay the log to see if the bug was fixed.
     You must run the 'combos' sub-command with the --debug flag to create the log.

  Devices which look like a keyboard:
/dev/input/event3 AT Translated Set 2 keyboard [EV_SYN EV_KEY EV_MSC EV_LED EV_REP] []
/dev/input/event7 Lenovo ThinkPad Compact USB Keyboard with TrackPoint [EV_SYN EV_KEY EV_MSC EV_LED EV_REP] []
```

## Keys That Are Hard to Accesshjjjjööjjjjjjkk

These keys are hard to access if you want to keep your index fingers on "F" and "J":

- Home (Pos1), End
- Arrow keys: Up, Down, Left, Right
- PageUp, PageDown
- Backspace
- Delete (Del)
- Escape (Esc)
- ...

## Example: Choosing an Item from an Auto-Complete List

Imagine the active cursor is in the address bar of your browser.

You enter some characters to find a page you visited yesterday.

It appears in the drop-down box (autocomplete list)

Up to this point, you could keep your fingers on "F" and "J."

But now, how do you choose the item from the list without using the mouse?

Arrow-down is not easy to reach.

## I Am Happy with Most Other Keys on the Keyboard

I know there are alternative keyboard layouts like Neo2 or Colemak, but I am happy with the default
QWERTY/QWERTZ layout.

I want to extend the default layout, not replace it. I want to be able to use the keyboards of my
teammates and family members as I am used to.

## Combos: Using Several Keys at Once

Does a piano player hit one key after the other? No, a piano player hits several keys at once.

I want pressing (and holding) `F` and then `J` to be one combo, and `J F` another combo.

Initially, I used [KMonad](https://github.com/kmonad/kmonad), but its syntax is not easy for me to
understand and (as far as I know) it cannot differentiate between an `F J` and a `J F` combo.

I searched a bit and found [go-evdev](https://github.com/holoplot/go-evdev), a Go package for
receiving and sending events on Linux.

## Overlap vs Combo

While typing fluently, you may have some overlap between key presses. This tool differentiates
between hitting `F` and then `J` with an overlap time of 40ms. If both keys are pressed
simultaneously and for longer, it is treated as one combination. Otherwise, it is interpreted as two
separate keys.

You do not need to write in a staccato style.

## Drawback

Keys that are part of a combo must not be emitted immediately. The code needs to wait a few
milliseconds to determine if it is a combo or not. This delay exists but is almost unnoticeable.

## Keyboard Input Details on Linux

- [Arch Linux Docs: "Keyboard Input"](https://wiki.archlinux.org/title/Keyboard_input)
- [Linux Kernel Input Documentation](https://docs.kernel.org/input/index.html)

## Trackpoint with Sandpaper

This is not directly related to the `ttf` tool, but you might find it interesting:

I want to keep my fingers close to the home row for moving the mouse cursor.

That's why I use a keyboard with a Trackpoint.

There are the well-known Lenovo keyboards, but there are alternatives like the [Tex
Shinobi](https://tex.com.tw/products/shinobi).

To get maximum grip, I stick sandpaper on the Trackpoint.

**Caution:** Don't stick sandpaper on the Trackpoint of your laptop. If you close the laptop, the
sandpaper may scratch the screen.

That’s why I use an external ThinkPad keyboard.

## I Love Feedback

Did you find a typo? Do you have the same needs as I do? Do you know how to solve this?

Please send me feedback via a GitHub issue!

## More

- [Thomas WOL: Working Out Loud](https://github.com/guettli/wol)
- [Desktop Tips](https://github.com/guettli/desktop-tips-and-tricks)
- I post updates to this article here: <https://www.reddit.com/r/typing/>

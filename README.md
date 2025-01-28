# Ten Flying Fingers

I learned [touch typing](https://en.wikipedia.org/wiki/Touch_typing) several years ago. And I can
write blind almost everything.

I want to keep the pointing fingers on "F" and "J" as much as possible (aka "home row").

## Keys which are hard to access

These keys are hard to access if you want to keep the pointing fingers on "F" and "J"

- Pos1, End
- CursorUp, CursorDown, CursorLeft, CursorRight
- PageUp, PageDown
- Backspace
- Del
- Esc
- ...

## Example: Choose item from auto-complete list

Imagine the active cursor is in the address bar of your browser.

You enter some characters to get to a page you visted yesterday.

It appears on the drop-down box.

Up to now you could keep your fingers on "F" and "J".

But now? How to choose the item from the list, without using the mouse?

Arrow-down is not easy to reach.

## I am happy with most other keys on the keyboard

I know that there are alternative keyboard layouts like neo2 or colemark. But I am happy with the
default QWERTY/QWERTZ layout.

I want to extend the default layout, I don't want to replace it. I want to be able to use the
keyboard of my team mates and family members like I am used to.

## Combos: Using several Keys at once

Does a Piano player hit one key after the other? No a piano player hits several keys at once.

I want pressing (and holding) `F` and then `J` to be one combo, and `J F` an other combo.

First I used [KMonad](https://github.com/kmonad/kmonad), but the syntax of is not easy to understand
for me and (afaik) is not able to differentiate between a `F J` and `J F` combo.

I searched a bit and found [go-evdev](https://github.com/holoplot/go-evdev) a package for Go to
receive and send events on Linux.

## Overlap vs Combo

While typing fluently, you have some overlap. This tool differentiates between hitting `F` and then
`J` with a overlap time of 40ms. If both keys are down longer, then it is a combo, otherwise it is
interpreted as two keys.

You do not need to write in staccato style.

## Drawback

Keys which are part of a combo must not get emitted immediately. The code needs to wait some
milliseconds to check if this will be combo or not. This delay exists, but is almost not noticible.

## Keyboard Input Details on Linux

- [ArchLinux Docs "Keyboard Input"](https://wiki.archlinux.org/title/Keyboard_input)
- [Linux Kernel "Input Documentation"](https://docs.kernel.org/input/index.html)

## Trackpoint with Sandpaper

This is not related to the `ttf` tool, but maybe you are interested:

I want to keep my fingers close to the home row for moving the mouse cursor.

That's why I use a keyboard with Trackpoint.

There are the well-known Lenovo keyboards, but there are alternatives like [Tex
Shinobi](https://tex.com.tw/products/shinobi).

To get maximum grip, I stick sandpaper on the trackpoint.

Caution: Don't stick sandpaper on the trackpoint of your laptop. If you close the laptop, then it is
likely that the sandpaper will scratch the screen.

That's why I use an external Thinkpad keyboard.

## I love feedback

You found a typo, you have the same need as I, you know how to solve this?

Please send me advices via an github issue!

## More

- [Thomas WOL: Working out Loud](https://github.com/guettli/wol)
- [Desktop Tips](https://github.com/guettli/desktop-tips-and-tricks)
- I post updates to this article here: <https://www.reddit.com/r/typing/>

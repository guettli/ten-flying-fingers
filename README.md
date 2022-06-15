# Ten Flying Fingers

I learned [touch typing](https://en.wikipedia.org/wiki/Touch_typing) several years ago. And I can write blind almost everything. 

But some key are easy to reach. For example these keys: Pos1, End or Del

I want to keep the pointing fingers on "F" and "J" as much as possible. https://github.com/guettli/ten-flying-fingers

My target plattform is a Linux.

# Keys which are hard to access

These keys are hard to access if you want to keep the pointing fingers on "F" and "J"

- Pos1, End
- CursorUp, CursorDown, CursorLeft, CursorRight
- PageUp, PageDown
- Backspace
- Del
- Esc
- ...

# History

I used Emacs for more than 15 years. Mostly for programming Python. In 2015 I switched from Emacs to PyCharm. In Emacs it is easy to go to the beginning/end of a line. During configuring PyCharm I asked myself: Why configure keyboard short cuts for every single application? Why not configure this once for all applications on my desktop? 

# Input-Remapper (Linux only)

[Input-Remapper](https://github.com/sezanzeb/input-remapper) is gtk GUI to remap keys via [python-evdev](https://python-evdev.readthedocs.io/en/latest/)

The first-time-user experience of the GUI could be better, but overall it is easy to remap keys.

First I disabled CapsLock via `gnome-tweaks`. Then I use input-remapper-gtk to add short-cuts.

1. start input-remapper-gtk
2. Enter your password, since the input-remapper daemon needs root access
3. Choose the Device
4. If the input-remapper is already running choose "Stop Injection"
5. Click on "new entry"
6. Click on "Change Key"
7. Now entry the short-cut you want to create. For example CapsLock-J
8. Then the cursor is in the text-input field. It supports autocomplete. Just type the key which should get executed if you entered the short-cut. For example "Backspace"
9. Click on "Apply"
10. Close the application, and check if it is working. The changes should survive a reboot of your machine.


I use CapsLock as modifier. Examples:

* CapsLock h: Pos1
* CapsLock ö: End
* CapsLock j: Backspace
* ...


Before I found this tool I wasted several hours, trying to remap keys via xmodmap or other tools. I am happy that I found input-remapper.

If you have two keyboards, for example your laptop and an external USB keyboard, then you can easily copy the config of one keyboard to the next:

```
cp $HOME/.config/input-remapper/presets/keyboard1/myconfig.json $HOME/.config/input-remapper/presets/keyboard2/
```

# I am happy with most other keys on the keyboard

I know that there are alternative keyboard layouts like neo2 or colemark. But I am happy with the default QWERTY or QWERTZ layout. 
I want to improve the default layout, I don't want to replace it. I want to be able to use the keyboard of my team mates and family members like I am used to.

# Related

More desktop related tips are in my [article "Desktop Tips"](https://github.com/guettli/desktop-tips-and-tricks).


# Maybe later: Command line

Pressing CapsLock twice should open a command line like tool. With this I want to:

  - move windows: for example put all terminals on the current screen beneath each other (no overlapping)
  - change window focus: For example bring webbroser, mail client or editor on the top of the screen.
  - insert fixed text: like "Regards, Firstname Lastname" or my ssh public key.
  - autocomplete search in the history of the copy and paste texts.


  
# Related Questions on Stackoverflow, Mailinglists, Groups

http://stackoverflow.com/questions/27813748/clone-input-device-with-python-uniput

http://stackoverflow.com/questions/27581500/hook-into-linux-key-event-handling

http://askubuntu.com/questions/401595/autocomplete-at-desktop-level

http://askubuntu.com/questions/627432/how-to-install-ibus-typing-booster

https://askubuntu.com/questions/1382762/hyper-key-u-like-arrow-up

# I love feedback

You found a typo, you have the same need as I, you know how to solve this?

Please send me advices via an github issue!



# More

* [Thomas WOL: Working out Loud](https://github.com/guettli/wol)
* [Desktop Tips](https://github.com/guettli/desktop-tips-and-tricks)


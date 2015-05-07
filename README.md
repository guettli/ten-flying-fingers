# ten-flying-fingers
My goal is to keep the pointing fingers on "F" and "J" as much as possible. Up to now I am gathering ideas and ways how to implement them. No coding was done yet, and no coding will be done during the next weeks.
My target plattform is a common linux desktop.

# Keys which are hard to access

These keys are hard to access if you want to keep the pointing fingers on "F" and "J"

- Pos1, End
- CursorUp, CursorDown, CursorLeft, CursorRight
- PageUp, PageDown
- Backspace
- Del
- ...

# History
I used Emacs for more than 15 years. Mostly for programming Python. In 2015 I switched from Emacs to PyCharm. In Emacs it is easy to go to the beginning/end of a line. During configuring PyCharm I asked myself: Why not configure this once for all applications? Why configure keyboard short cuts for every single application?

Other example: In thunderbird many users configure a shortcut to delete mails. The del-key works, but users configure a short cut. Why? Because "del" is too hard to access....

# I am happy with most other keys on the keyboard

I know that there are alternative keyboard layouts like neo2 or colemark. I am happy with the default QWERTY or QWERTZ layout. I want to improve the default layout, I don't want to replace it. I want to be able to use the keyboard of my team mates and family members like I am used to.

Related: http://forum.colemak.com/viewtopic.php?id=1914


# I love feedback

You found a typo, you have the same need as I, you know how to solve this?

Please send me advices via the github issue manager!

# Technologies for the implementation

https://help.ubuntu.com/community/ibus

http://en.wikipedia.org/wiki/Intelligent_Input_Bus

https://code.google.com/p/autokey/  https://github.com/autokey/autokey

# Why not xmodmap ...

The basics could be solved by using xmodmap. But later I would love to have a command line interface which does much more then just mapping one keyboard key to an other key.

# Command line

Pressing CapsLock twice should open a command line like tool. With this I want to:

  - move windows: for example put all terminals on the current screen beneath each other (no overlapping)
  - change window focus: For example bring webbroser, mail client or editor on the top of the screen.
  - insert fixed text: like "Regards, Firstname Lastname" or my ssh public key.
  - autocomplete search in the history of the copy and paste texts.

# Autocomplete for long words or common phrases

This is very common on mobile phones. But traditional PCs don't have this great feature. Some applications like LibreOffice or programming IDEs have it, but it is build into each single application. I dream of an autocomplete for long words which works for all applications on a linux desktop.

# Windowmanaging

I want to move, align, change focus .... of windows. I don't want to invent a new window manager. AFAIK it should be possible to control the layout of the windows with each window manager. My solution should work for unity desktops and others (mate, gnome, kde, ...). I guess, but don't know, that there is an API for this which works for all of them.

# Related Questions on Stackoverflow, Mailinglists, Groups

http://stackoverflow.com/questions/27813748/clone-input-device-with-python-uniput

http://stackoverflow.com/questions/27581500/hook-into-linux-key-event-handling

http://askubuntu.com/questions/585275/make-capslock-j-work-like-pos1

Question in colemark forum:
http://forum.colemak.com/viewtopic.php?pid=15700#p15700


#!/bin/bash
set -euxo pipefail
baseconfig=$1
if [ ! -e "$baseconfig" ]; then
    echo "$0 my-config.kmonad"
    echo "the string DEVICE will get replaced by the device path."
    exit 1
fi
for service in $(systemctl list-units --all --full --no-legend --plain | grep -Po '^kmonad-\S+'); do
    systemctl disable --now --force "$service" || true
done
rm -f /etc/systemd/system/kmonad-*

for device in \
    /dev/input/by-path/platform-i8042-serio-0-event-kbd \
    /dev/input/by-id/usb-Lenovo_ThinkPad_Compact_USB_Keyboard_with_TrackPoint-event-kbd \
    /dev/input/by-id/usb-04d9_USB-HID_Keyboard_000000000407-event-kbd; do
    exe="$(type -p kmonad)"
    name="$(basename $device)"
    config="/etc/systemd/system/kmonad-$name.conf"
    cp "$baseconfig" "$config"
    sed -i "s#DEVICE#$device#" "$config"
    cat <<EOF >"/etc/systemd/system/kmonad-$name.service"
[Unit]
Description=kmonad $name

[Service]
Restart=always
RestartSec=3
ExecStart=$exe -l debug "$config"
Nice=-20

[Install]
WantedBy=default.target
EOF

    systemctl enable --now "kmonad-$name.service"
done

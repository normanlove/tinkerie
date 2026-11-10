#!/bin/sh
# Lets you use USB scopes (Scopie) and logic analysers (Probie) on Linux and
# Raspberry Pi OS without root. Run once: ./install-usb.sh  (asks for your password)
# then unplug and replug the device. To remove: ./install-usb.sh --remove
set -e
rules=/etc/udev/rules.d/60-tinkerie.rules
if [ "$1" = "--remove" ]; then
  sudo rm -f "$rules" && sudo udevadm control --reload-rules
  echo "Removed Tinkerie's USB rule."
  exit 0
fi
tmp=$(mktemp)
{
  echo "# Tinkerie (nrmluvdev): USB scopes and logic analysers for the logged-in user."
  # Cypress FX2 based scopes and analysers (Hantek 6022, DDS120, Saleae-style clones),
  # sigrok/OpenHantek firmware, CWAV USBee, Instrustar, Hantek DSO-2xxx/5xxx,
  # USB-TMC bench scopes (Rigol, Siglent, Keysight/Agilent, Tektronix, OWON, R&S, LeCroy),
  # and the STM32 handheld scope.
  for v in 04b4 04b5 0925 1d50 08a9 d4a2 8102 1ab1 f4ec f4ed 0957 2a8d 0699 5345 049f 0aad 05ff; do
    echo "SUBSYSTEM==\"usb\", ATTR{idVendor}==\"$v\", MODE=\"0660\", GROUP=\"plugdev\", TAG+=\"uaccess\""
  done
  echo "SUBSYSTEM==\"usb\", ATTR{idVendor}==\"0483\", ATTR{idProduct}==\"2d42\", MODE=\"0660\", GROUP=\"plugdev\", TAG+=\"uaccess\""
} > "$tmp"
sudo install -m 0644 "$tmp" "$rules"
rm -f "$tmp"
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=usb || true
getent group plugdev >/dev/null && ! id -nG | grep -qw plugdev && sudo usermod -aG plugdev "$USER" && echo "Added you to the plugdev group: log out and back in once."
echo "Done. Unplug and replug your scope or logic analyser."

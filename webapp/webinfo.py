"""What each Tinkerie app does in its web version (Tinkerie web), compared with
the desktop app. One list, used by the web build (each app's "Web vs desktop"
box and the web home page) and by the website (each app's page).

Every app in the suite must be here: webapp/check.py fails the build when one
is missing, so a new app or feature is always either in the web version or
listed under `missing` with the reason. Made by nrmluvdev.
"""

# Chrome and Edge are the browsers the web versions are built and tested for.
BROWSERS = 'Chrome or Edge on a computer (Windows, Mac, Linux or Chromebook)'

# name: title, one-line description, devices the page may ask for
# ('serial' Web Serial, 'usb' WebUSB, 'hid' WebHID), and three lists:
#   works   - the same as the desktop app
#   differs - works, but differently in a browser
#   missing - only in the desktop app, and why
APPS = {
  'wirie': dict(title='Wirie', desc='Wiring diagrams, PCBs and Keebie keyboards', devices=[],
    works=['Wiring diagrams, schematics and PCB design, with every part and footprint',
           'Auto-route, the Check tab and Auto-fix',
           'Keebie keyboards: layouts, cases, plates and the keyboard PCB',
           'Exports: Gerbers, drill files, pick-and-place, parts lists, SVG and PNG',
           'The Part Creator, teaching mode and themes',
           'Autosave: your work comes back when you reopen the page'],
    differs=['Exports and saved projects are downloaded by the browser (or from the Files button) instead of going to your Documents folder',
             'To open a project from your computer, add it with Files → Add files first',
             '“Case in Makie” and “Firmware in Flashie” save the file for you to download; open Makie web or Flashie web and add it there'],
    missing=[]),
  'makie': dict(title='Makie', desc='3D part designer for 3D printing', devices=[],
    works=['Every shape, sketch, thread and piece of hardware, with smart drilling',
           'Cases for Wirie boards and Keebie keyboards',
           'STL export, made in the page',
           'Autosave and teaching mode'],
    differs=['Exports are downloaded by the browser instead of saved to a folder',
             'Case files from Wirie are added with Files → Add files'],
    missing=[]),
  'tunie': dict(title='Tunie', desc='3D printer calibration helper', devices=['serial'],
    works=['Every calibration test, for every printer preset and firmware (Marlin, Klipper, RepRapFirmware, Prusa, Bambu Lab)',
           'Saving tests as .gcode for the printer’s SD card or USB stick',
           'Printing straight to a USB-connected printer (Marlin, Prusa, RepRapFirmware) over Web Serial',
           'The bed mesh viewer, from a printer on USB or a pasted or added mesh file',
           'The belt tension helper, through your computer’s microphone',
           'Results history, profiles and slicer presets'],
    differs=['Pick the printer’s USB port once with the Connect button',
             '.gcode files and slicer presets are downloaded instead of saved to a drive you choose',
             'The browser asks once to use the microphone'],
    missing=['Moonraker (Klipper) and OctoPrint over the network: a secure (https) web page can’t talk to a printer on your home network; save the .gcode or use the desktop app',
             'Sharing the active profile with the other apps (the suite link)']),
  'flashie': dict(title='Flashie', desc='Firmware flasher and block coder for boards', devices=['serial'],
    works=['Flashing ESP32 and ESP8266 boards over USB serial (esptool’s protocol, built in)',
           'Flashing Arduino Uno, Nano, Mega and Pro Mini (.hex files) over serial',
           'STM32 boards through their serial bootloader',
           'MicroPython: installing it, the REPL, and uploading your program',
           'The Create tab: blocks to MicroPython, with teaching mode and templates',
           'The Cheap Yellow Display database and board tips'],
    differs=['Pick the board’s port once with the Connect button. A board that restarts as a different USB device (Leonardo and Micro in their bootloader, an ESP32-S2/S3 in download mode) has to be picked again the first time',
             'Firmware from your computer is added with Files → Add files',
             'Some sites in the firmware library don’t let web pages download their files: if one fails, download it yourself and add it with Files → Add files'],
    missing=['Writing SD card and USB images (Raspberry Pi OS, Windows and Linux installers): browsers can’t write to drives',
             'Copying .uf2 files to a Raspberry Pi Pico in BOOTSEL mode: browsers can’t write to drives; download the .uf2 and drag it onto the RPI-RP2 drive yourself',
             'Compiling Arduino code (blocks for Uno and Nano, and keyboard firmware): it needs arduino-cli on your computer',
             'Android phones and tablets over USB']),
  'scopie': dict(title='Scopie', desc='Oscilloscope app for many scopes', devices=['usb'],
    works=['The full scope screen: channels, trigger, cursors, 13 measurements, FFT, XY, persistence and averaging',
           'USB scopes over WebUSB: the Hantek 6022 family (firmware built in), OWON, Hantek handhelds and old Hantek DSOs, and Rigol, Keysight, Tektronix and Siglent bench scopes on USB',
           'Demo mode, CSV import and export, PNG screenshots, saved setups'],
    differs=['Pick the scope once with the Connect button. A Hantek 6022 restarts as a new device after Scopie loads its firmware, so pick it again the first time',
             'On Windows the browser can only use a USB scope with the WinUSB driver (Zadig installs it); libusb-win32, NI-VISA and makers’ own drivers don’t work in a browser, and the desktop app’s driver installer isn’t in the web version',
             'Old Hantek DSO firmware is imported from a file you add, not from the installed driver',
             'Screenshots and CSV files are downloaded instead of saved to a folder'],
    missing=['Network bench scopes (SCPI over LAN, VXI-11): web pages can’t open network connections to instruments',
             'PicoScope and other scopes that need their maker’s driver (DLL), and NI-VISA',
             '“Explain in Probie” opening Probie directly: save the CSV and add it in Probie web']),
  'meterie': dict(title='Meterie', desc='Read your multimeter on the computer', devices=['serial', 'hid'],
    works=['Meters with a USB-serial or RS-232 cable (with a USB adapter), over Web Serial',
           'UNI-T UT-D04, Victor, Brymen BU-86X and CP2110 USB-HID cables, over WebHID',
           'Reading a meter’s display with your webcam',
           'Live graphs, statistics, limits and alarms, the timeline and sessions',
           'Teaching mode and the meter wizards'],
    differs=['Pick the cable once with the Connect button',
             'Sessions and CSV exports are downloaded; to open one, add it with Files → Add files, then Open picks the newest',
             'The browser asks once to use the camera'],
    missing=['Bluetooth meters (not in the web version yet)',
             'Readings for Fixie and other apps (the suite API)']),
  'probie': dict(title='Probie', desc='Serial monitor, logic analyser and signal explainer', devices=['serial', 'usb'],
    works=['The serial monitor, with every baud rate, line ending, hex view, logging and plotting',
           'fx2lafw logic analysers (the cheap 8-channel “Saleae clones”), over WebUSB',
           'UART, I²C and SPI decoders',
           'The Explain tab and the Timeline tab',
           'Demo devices'],
    differs=['Pick the port or analyser once with the Connect button. An analyser restarts as a new device after Probie loads its firmware, so pick it again the first time',
             'On Windows the browser can only use an analyser with the WinUSB driver (Zadig installs it)',
             'The fx2lafw firmware isn’t found in a PulseView install: drop the firmware file on the Firmware needed box once',
             'Captures and logs are downloaded instead of saved to a folder; files from Scopie and Meterie are added with Files → Add files'],
    missing=['Opening captures from the other apps directly (Explain in Probie)']),
  'codie': dict(title='Codie', desc='Explains code, line by line, in plain words', devices=[],
    works=['Explaining code in every language Codie reads, with every tab: Explain, What it does, Outline, Concepts, Game logic, Warnings and Game school',
           'Pasting code or opening a file'],
    differs=['Files are added with Files → Add files, or pasted'],
    missing=[]),
  'fixie': dict(title='Fixie', desc='Step-by-step repair guide for almost any device', devices=[],
    works=['All three guides (Quick checks, Board level, My build) and Parts tests',
           'Pictures, reference cards and calculators',
           'Tools, signs, safety and the refusal list',
           'Your repair log and reports',
           'The Error logs tab: explaining Windows, macOS, Linux and Android logs you open, paste or drop'],
    differs=['Readings are typed in; reports are downloaded'],
    missing=['Scan this computer (Error logs): a web page can’t read the computer’s own logs; save one (the tab shows how) and open it',
             'Taking readings straight from Meterie, Scopie and Probie, and opening them from a step: the web versions can’t reach each other']),
}

ORDER = ['wirie', 'makie', 'tunie', 'flashie', 'scopie', 'meterie', 'probie', 'codie', 'fixie']

# Everything in the web versions, whatever the app.
ALL = ['Nothing is installed and nothing is uploaded: the app runs in the page, on your computer',
       'There are no accounts and no server: what you make is kept only in this browser on this computer, so nobody else can see it. Each tab and each app keeps to itself',
       'On a shared computer, use your own browser profile or a guest window, or use Clear my data (in Files) when you’ve finished. Download anything you want to keep',
       'Devices are used through Chrome and Edge’s Web Serial, WebUSB and WebHID. Firefox and Safari run the apps, but can’t connect to devices']


def html_list(name, esc):
    """The works / differs / missing lists for one app, as HTML."""
    a = APPS[name]
    out = []
    for key, head, cls in (('works', 'Works the same in the web version', 'yes'),
                           ('differs', 'Works, but differently', 'mid'),
                           ('missing', 'Only in the desktop app', 'no')):
        items = a[key] or (['Nothing: everything is in the web version'] if key == 'missing' else [])
        if not items:
            continue
        out.append('<div class="%s"><h4>%s</h4><ul>%s</ul></div>' % (cls, head, ''.join('<li>%s</li>' % esc(i) for i in items)))
    out.append('<div class="all"><h4>In every web app</h4><ul>%s</ul></div>' % ''.join('<li>%s</li>' % esc(i) for i in ALL))
    return '<div class="tk-web">%s</div>' % ''.join(out)

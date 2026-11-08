// Flashie teaching-mode explanations, shown by teach.js. The Create tab's
// blocks already carry their own teaching text. Created by nrmluvdev.
window.TEACH = {
  app: "Flashie",
  intro: "Point at any step, button or setting in Flashie to learn what it does.",
  tips: {
    // ---------------------------------------------------------------- tabs
    "Flash": ["Flash", "Puts new firmware (the program that runs on a microcontroller) onto your board in three steps: pick the board, pick the firmware, then flash."],
    "Write image": ["Write image", "Puts a whole operating system, like Raspberry Pi OS, onto an SD card or USB drive so a Raspberry Pi (or another computer) can start from it. It works like Raspberry Pi Imager."],
    "Create": ["Create", "Build your own program from blocks, like Scratch, without writing code. Flashie turns the blocks into MicroPython (or Arduino C++ for Arduino boards) and puts it on your board."],

    // ------------------------------------------------------- write image
    "Choose OS": ["Choose an operating system", "Pick what to put on the card. The list is Raspberry Pi's official one, the same one Raspberry Pi Imager shows. Flashie downloads it, checks the download hasn't been damaged, and keeps a copy so writing a second card is quicker. You can also use your own file: any Linux image or ISO, or a Windows installer ISO."],
    "Device filter": ["Which device?", "Pick the Raspberry Pi (or other board) the card is for, and the OS list only shows systems that run on it. Leave it on 'any device' to see everything."],
    "Use custom image": ["Your own image file", "Write an image you already have. Flashie understands .img and .iso files, and ones packed in .zip, .xz, .zst, .gz or .bz2, unpacking them as it writes. Linux ISOs (Ubuntu, Fedora, Mint…) are copied as they are. A Windows 10 or 11 ISO becomes a proper USB installer: Flashie formats the drive and copies the files across, like Rufus does."],
    "Image note": ["About this image", "Something worth knowing about the image you picked, such as how Flashie will write a Windows installer, or that an old CD-only ISO may not start from a USB drive."],
    "OS settings": ["OS settings", "Set the computer's name, your username and password, Wi-Fi, SSH and time zone before it ever starts, like Raspberry Pi Imager's OS customisation. Great for a Raspberry Pi with no screen. Flashie adds them to the card after writing, and the system sets itself up on its first start."],
    "Hostname setting": ["Hostname", "The name the computer uses on your network. Other computers can usually reach it as name.local, for example raspberrypi.local. Letters, numbers and dashes only."],
    "User setting": ["Username and password", "Makes your own login instead of a default one. The password is turned into a scrambled hash before it goes on the card, so it's never stored in plain text. Flashie doesn't save it, so you type it again next time."],
    "Wi-Fi setting": ["Wi-Fi", "Joins your Wi-Fi network on the first start, so you don't need a cable. The country matters: it sets which Wi-Fi channels are allowed, and Wi-Fi stays off until it's set."],
    "Hidden network": ["Hidden network", "Tick this if your Wi-Fi doesn't show its name in the list of networks, so the computer looks for it by name."],
    "Locale setting": ["Time zone and keyboard", "Sets the clock's time zone and the keyboard layout (gb for a UK keyboard, us for a US one), so the keys type what's printed on them."],
    "SSH setting": ["SSH", "Lets you log in from another computer over the network with a terminal: ssh username@hostname.local. Using a public key instead of the password is safer."],
    "SSH keys": ["Public keys", "Paste your public key here, the one-line text in a file like id_ed25519.pub. Never paste the private key (the file without .pub)."],
    "Save settings": ["Save", "Keeps these settings for every card you write, until you change them. Passwords are only kept until Flashie closes."],
    "Turn settings off": ["Don't use settings", "Writes the image exactly as it is, without adding any settings."],
    "Choose storage": ["Choose storage", "Pick the SD card or USB drive to write to. Only removable drives are listed, and never the disk your computer is running from, so you can't erase your computer by mistake."],
    "Write button": ["Write", "Erases the drive and copies the operating system onto it. Flashie asks you to confirm the drive first, then asks for your password because writing a whole drive needs administrator rights."],
    "Verify after writing": ["Check after writing", "Reads the whole drive back and compares it with the image. It takes extra time but catches faulty or fake cards, which are common."],
    "Eject when done": ["Eject when done", "Tells the computer you've finished with the drive, so it's safe to pull out straight away."],
    "Image progress": ["Progress", "Shows downloading, writing and checking. Flashie writes the start of the drive last, so if writing stops part way the card won't half-start a broken system."],
    "Cancel image write": ["Cancel", "Stops now. The drive will be only partly written, so write it again before you use it. A cancelled download carries on from where it stopped next time."],
    "Erase and write": ["Erase and write", "Last check: everything on this drive will be lost. Make sure it's the right one — the name and size help you tell drives apart."],
    "Clear image cache": ["Clear image cache", "Deletes the OS images Flashie has downloaded, to free up disk space. They'll download again when you next need them."],

    // ---------------------------------------------------------- 1: board
    "Pick board": ["Step 1: pick your board", "Plug your board in with a USB cable that carries data (some cables only charge). Flashie lists it here and guesses what it is from its USB chip."],
    "Rescan ports": ["Rescan", "Looks again for boards plugged in by USB. Flashie also keeps looking by itself."],
    "Connected ports": ["Connected ports", "Boards Flashie can see: serial (COM) ports, and Raspberry Pi Pico boards in BOOTSEL mode, which show up as a USB drive."],
    "No boards found yet": ["No board found yet", "Check the cable carries data, not just power. For a Pico, hold the BOOTSEL button while plugging it in. Some boards need a USB driver (CH340, CP2102)."],
    "Board type|#boardSel|Auto-detect": ["Board type", "Which board you have. Auto-detect works it out from the chip; pick it yourself if the guess is wrong, because the board decides how Flashie talks to it."],
    "Identify chip": ["Identify chip", "Asks the board which chip it has, how much flash memory, and its MAC address. Useful when a USB chip like the CH340 could be on many different boards."],

    // ------------------------------------------------------- 2: firmware
    "Pick firmware": ["Step 2: pick firmware", "Choose what to put on the board: a ready-made firmware from the library, or a file you already have."],
    "Firmware library|Browse firmware library": ["Firmware library", "Ready-made firmware for your board, downloaded straight from the official projects: MicroPython, WLED, ESPHome, Tasmota and more. Flashie checks every download."],
    "My own file": ["My own file", "Flash a file you built or downloaded yourself: .bin, .hex or .uf2. Flashie checks it matches your board before writing."],
    "Search firmware": ["Search firmware", "Type to find a firmware in the library by name or what it does."],
    "Choose a firmware file": ["Firmware file", "Click to pick a file, or drag it here. .bin files are raw memory images, .hex files carry their own addresses, and .uf2 files are for Pico-style boards."],
    "Flash offset": ["Flash offset", "Where in the board's memory the file goes. Flashie fills this in for you. Only change it if your firmware's instructions give a different address."],
    "Advanced": ["Advanced settings", "Speed and memory settings. The defaults work for almost every board."],
    "Speed": ["Speed", "How fast Flashie sends data to the board. Faster is quicker, but if flashing fails, try a slower speed (a long or cheap cable can't always keep up)."],
    "Flash size": ["Flash size", "How much flash memory the board's chip has. Identify chip tells you; 4 MB is the most common on ESP32 boards."],
    "Compress while sending": ["Compress while sending", "Squeezes the data before sending it, which makes flashing much faster. Turn it off only if flashing keeps failing."],

    // ---------------------------------------------------------- 3: flash
    "Write, verify, restart": ["Step 3: flash", "Writes the firmware, reads it back to check every byte arrived correctly, then restarts the board so it runs."],
    "Board": ["Board", "The board you picked in step 1."],
    "Port": ["Port", "The USB connection Flashie will use to reach the board."],
    "Firmware": ["Firmware", "The firmware you picked in step 2."],
    "Method": ["Method", "How Flashie talks to this board. Each chip family has its own bootloader (the tiny built-in program that accepts new firmware)."],
    "Writes to": ["Writes to", "Where in the board's memory the firmware will go."],
    "Technical details": ["Technical details", "The full log of what Flashie is doing, step by step. Handy if something goes wrong and you want to see why."],
    "Clear download cache": ["Clear download cache", "Deletes firmware Flashie downloaded earlier. It downloads them again when needed."],
    "Close": ["Close", "Closes this panel (Esc)."],
  },
};

"""Fixie's error log explainer: the messages it knows.

Builds web/logrules.json:  python3 apps/fixie/logs.py apps/fixie/web/logrules.json

Every rule is written by hand from the operating systems' own documentation
and long-standing repair knowledge. Fixie matches log lines against these
patterns in the window (web/logscan.js): no AI, and nothing leaves the
computer. Each rule has examples, and fixie's tests check every example
matches its rule (in Go and in JavaScript) and that sample logs from each
system give the findings they should.

A rule:
  id      unique
  os      windows, macos, linux, android or any
  sev     critical  failing hardware or data at risk: act now
          problem   something is broken
          warning   worth checking, often the cause of a symptom
          info      useful to know (why it shut down, what crashed)
          harmless  looks alarming but is normal: usually ignore it
  cat     disk, memory, power, heat, cpu, graphics, usb, network, boot,
          driver, software, security, filesystem, battery, audio
  t       what it is, in a few words
  means   what it means
  do      what to do, in order (easiest first)
  re      a JavaScript and Go (RE2) compatible regular expression, matched
          ignoring case; or ev=(source, [event ids]) for Windows events
  kw      a lower-case word every matching line contains (makes scanning fast)
  ex      example lines that must match
Created by nrmluvdev for Tinkerie.
"""
import json
import re
import sys

RULES = []
SEVS = ["critical", "problem", "warning", "info", "harmless"]
CATS = ["disk", "memory", "power", "heat", "cpu", "graphics", "usb", "network", "boot", "driver",
        "software", "security", "filesystem", "battery", "audio", "hardware"]


def esc(t):
    """Escape regex characters the same way for JavaScript and Go (RE2)."""
    return re.sub(r"([.^$*+?()\[\]{}|\\])", r"\\\1", t)


def R(id, os, sev, cat, t, means, do, re_=None, ev=None, kw=None, ex=(), stop=None):
    assert sev in SEVS and cat in CATS, id
    if ev:
        src, ids = ev
        re_ = r"^\[event\] [^|]*" + esc(src.lower()) + r"[^|]*\| id (?:" + "|".join(str(i) for i in ids) + r")\b"
        kw = kw or src.lower()
    assert re_ and kw is not None, id
    RULES.append(dict(id=id, os=os, sev=sev, cat=cat, t=t, means=means, do=list(do), re=re_, kw=kw, ex=list(ex)))


# ====================================================================== Windows
# Event log records are turned into one line each by logscan.js:
#   [event] <source> | id <event id> | <level> | <message>
W = "windows"
R("w.kp41", W, "warning", "power", "Kernel-Power 41: restarted without shutting down",
  "Windows started again without shutting down properly first. The power was cut, the computer froze and was reset, or it crashed so hard it couldn't write a crash report. This event is the result, not the cause: look at what happened just before it.",
  ["If you held the power button or the power went off, that explains it.",
   "If it happens by itself: look for a BugCheck (blue screen) event at the same time, and for WHEA, disk or display errors just before it.",
   "Random restarts with no blue screen often mean power: a failing power supply or battery, a loose power connector, or overheating. Fixie's Quick checks → Turns on then off walks through it."],
  ev=("Kernel-Power", [41]), ex=["[event] Microsoft-Windows-Kernel-Power | id 41 | Critical | The system has rebooted without cleanly shutting down first."])
R("w.6008", W, "info", "power", "The last shutdown was unexpected",
  "Windows noticed when it started that it hadn't shut down properly last time (EventLog 6008). It goes with Kernel-Power 41.",
  ["See Kernel-Power 41."], ev=("EventLog", [6008]),
  ex=["[event] EventLog | id 6008 | Error | The previous system shutdown at 10:02:11 on 14/11/2026 was unexpected."])
R("w.bugcheck", W, "problem", "boot", "Blue screen (BugCheck 1001)",
  "Windows crashed with a blue screen (stop error) and restarted. The message gives the stop code; Fixie explains it when it finds the code's name or number.",
  ["Find the stop code in the message (0x000000.. or a name like IRQL_NOT_LESS_OR_EQUAL).",
   "Note what you were doing and whether it repeats: one blue screen can be a fluke, the same one again is a fault.",
   "Crash dumps are in C:\\Windows\\Minidump; the driver named in them is usually the cause."],
  re_=r"^\[event\] [^|]*(?:bugcheck|wer-systemerrorreporting)[^|]*\| id 1001\b|rebooted from a bugcheck", kw="",
  ex=["[event] Microsoft-Windows-WER-SystemErrorReporting | id 1001 | Error | The computer has rebooted from a bugcheck.  The bugcheck was: 0x0000007e (0xffffffffc0000005, ...)."])
R("w.unexpectedboot", W, "harmless", "power", "Kernel-Boot / Kernel-General: start-up and time records",
  "Ordinary records of Windows starting, the clock being set, and so on.",
  ["Nothing to do."], re_=r"^\[event\] [^|]*(?:kernel-general|kernel-boot)[^|]*\| id (?:1|12|13|16|20|25|27|30|32|153|18)\b", kw="kernel-",
  ex=["[event] Microsoft-Windows-Kernel-General | id 12 | Information | The operating system started at system time 2026-11-14T09:00:00."])
R("w.disk7", W, "critical", "disk", "Disk 7: a bad block",
  "The disk reported a part it couldn't read or write (a bad block). On a hard drive or SSD this is a sign it's starting to fail; files in that part may be damaged.",
  ["Back up your files now, before anything else.",
   "Check the drive's health (SMART) with CrystalDiskInfo or the maker's tool: reallocated or pending sectors confirm it.",
   "Replace the drive. If it's a USB stick or SD card (\\Device\\Harddisk1 or higher), it's the stick or card."],
  ev=("Disk", [7]), ex=["[event] disk | id 7 | Error | The device, \\Device\\Harddisk0\\DR0, has a bad block."])
R("w.disk11", W, "problem", "disk", "Disk 11: controller error",
  "Windows got an error from the disk's controller. Usually a bad or loose cable, a failing drive, or (for USB drives) a poor USB connection or power.",
  ["Reseat or replace the SATA data and power cables (or try another USB port and cable).",
   "Check the drive's health (SMART).",
   "If it keeps happening on a good cable, the drive or the port is failing."],
  ev=("Disk", [11]), ex=["[event] Disk | id 11 | Error | The driver detected a controller error on \\Device\\Harddisk1\\DR1."])
R("w.disk51", W, "warning", "disk", "Disk 51: error during paging",
  "A read or write failed while Windows was moving memory to or from the disk. Once can happen when a USB drive is pulled out; many mean a failing drive or connection.",
  ["If you removed a USB drive at that time, ignore it.", "Otherwise check the drive's health and its cable."],
  ev=("Disk", [51]), ex=["[event] Disk | id 51 | Warning | An error was detected on device \\Device\\Harddisk0\\DR0 during a paging operation."])
R("w.disk153", W, "warning", "disk", "Disk 153: an operation was retried",
  "The disk took too long and Windows had to retry. A few now and then are normal on busy or sleepy drives; many point to a failing drive, cable or controller.",
  ["Check the drive's health (SMART).", "Reseat the cable; update the storage driver and the SSD's firmware."],
  ev=("Disk", [153]), ex=["[event] disk | id 153 | Warning | The IO operation at logical block address 0x1234 for Disk 0 (PDO name: \\Device\\00000034) was retried."])
R("w.disk157", W, "warning", "disk", "Disk 157: disk removed without warning",
  "A disk disappeared while in use. Normal when a USB drive is unplugged without ejecting; on an internal drive it means it lost power or its connection.",
  ["Eject USB drives before unplugging them.", "For an internal drive: check its cables, power and health."],
  ev=("Disk", [157]), ex=["[event] disk | id 157 | Warning | Disk 2 has been surprise removed."])
R("w.storport129", W, "warning", "disk", "Reset to the storage device (129)",
  "The storage driver (storahci, iaStorA, stornvme…) had to reset the drive because it stopped answering. Causes: power-saving settings, old drivers or SSD firmware, a bad cable, or a failing drive.",
  ["Update the storage driver and the SSD's firmware.", "Check the drive's health.", "Reseat or replace the cable."],
  re_=r"^\[event\] [^|]*(?:storahci|iastor\w*|stornvme|storport|nvme)[^|]*\| id (?:129|11)\b", kw="| id ",
  ex=["[event] storahci | id 129 | Warning | Reset to device, \\Device\\RaidPort0, was issued."])
R("w.ntfs55", W, "critical", "filesystem", "NTFS 55 / 98: the file system is damaged",
  "Windows found damage in the file system (NTFS) on a drive. Files may be lost or damaged. It often follows power cuts, but can mean the drive is failing.",
  ["Back up what you can.", "Run chkdsk (chkdsk C: /f, then restart) to repair it.", "Check the drive's health: if it's failing, repairs won't last."],
  ev=("Ntfs", [55, 98, 137]), ex=["[event] Ntfs | id 55 | Error | A corruption was discovered in the file system structure on volume C:."])
R("w.ntfs140", W, "warning", "filesystem", "NTFS 140: couldn't write to the drive",
  "Windows couldn't save a change to a drive, usually because it was removed or lost power while being written.",
  ["Eject drives before removing them.", "If it's an internal drive: check its health and cable."],
  ev=("Ntfs", [140]), ex=["[event] Microsoft-Windows-Ntfs | id 140 | Warning | The system failed to flush data to the transaction log. Corruption may occur in VolumeId: E:."])
R("w.volsnap", W, "warning", "disk", "Shadow copies were deleted (volsnap)",
  "Restore points or backup copies for a drive were removed, usually because the drive ran short of space or was too slow.",
  ["Free up space on the drive.", "Make a fresh restore point or backup."],
  ev=("volsnap", [25, 33, 36]), ex=["[event] volsnap | id 25 | Error | The shadow copies of volume C: were deleted because the shadow copy storage could not grow in time."])
R("w.whea", W, "critical", "hardware", "WHEA: a hardware error",
  "Windows Hardware Error Architecture reported an error from the processor, memory, or a PCI Express device. A fatal one (event 18 or 1) causes a blue screen; corrected ones (17, 19, 47) are warnings that hardware is struggling.",
  ["Look in the message for the component: Processor Core, Memory, PCI Express Root Port, or a device.",
   "Undo any overclocking or memory XMP/EXPO profile and set the BIOS back to defaults.",
   "Test the memory (Windows Memory Diagnostic or MemTest86) and watch temperatures.",
   "PCI Express errors: reseat the graphics card and NVMe SSD, update the BIOS and chipset drivers."],
  ev=("WHEA-Logger", [1, 17, 18, 19, 20, 46, 47]), kw="whea",
  ex=["[event] Microsoft-Windows-WHEA-Logger | id 18 | Error | A fatal hardware error has occurred. Reported by component: Processor Core"])
R("w.tdr", W, "problem", "graphics", "Display driver stopped responding (TDR)",
  "The graphics driver stopped responding and Windows restarted it: the screen goes black for a moment, or an app or game crashes. Causes: driver problems, an overheating or overclocked graphics card, or not enough power to it.",
  ["Update (or roll back) the graphics driver: use the maker's site, and DDU for a clean install.",
   "Undo graphics card overclocks; clean its fans and check its temperature.",
   "Check the card's power connectors are pushed fully home; a weak power supply causes this under load."],
  re_=r"^\[event\] [^|]*(?:display|nvlddmkm|amdkmdag|amdwddmg|igfx|dxgkrnl)[^|]*\| id (?:4101|4109|13|14|153)\b|display driver \S+ stopped responding and has successfully recovered", kw="display",
  ex=["[event] Display | id 4101 | Warning | Display driver nvlddmkm stopped responding and has successfully recovered."])
R("w.scm7000", W, "warning", "software", "A service failed to start",
  "A Windows service couldn't start. Often harmless (a leftover from uninstalled software), but if it's something you use (audio, network, a driver), it explains why that isn't working.",
  ["Note the service's name in the message.", "If it belongs to software you uninstalled, ignore it or remove the leftover.",
   "Otherwise reinstall or update the software or driver that owns it."],
  ev=("Service Control Manager", [7000, 7001, 7009, 7011, 7023, 7024, 7026, 7038]), kw="service control manager",
  ex=["[event] Service Control Manager | id 7000 | Error | The Foo Update Service service failed to start due to the following error: The system cannot find the file specified."])
R("w.scm7031", W, "warning", "software", "A service stopped unexpectedly",
  "A Windows service crashed (and may have been restarted). Once is not a worry; often means the software behind it is faulty.",
  ["Note which service it is.", "Update or reinstall the software that owns it."],
  ev=("Service Control Manager", [7031, 7034]),
  ex=["[event] Service Control Manager | id 7034 | Error | The Print Spooler service terminated unexpectedly.  It has done this 1 time(s)."])
R("w.apperr", W, "info", "software", "An app crashed (Application Error 1000)",
  "A program crashed. The message names the program, the module (file) where it crashed and an exception code, which Fixie explains when it knows it.",
  ["If it's the same program each time: update or reinstall it.",
   "If the faulting module is a driver or system file (ntdll.dll, nvwgf2umx.dll, atio6axx.dll…), update that driver or Windows.",
   "Many different programs crashing points to memory or overheating: test the memory."],
  ev=("Application Error", [1000]),
  ex=["[event] Application Error | id 1000 | Error | Faulting application name: game.exe, version: 1.0.0.0, Faulting module name: ntdll.dll, Exception code: 0xc0000005"])
R("w.apphang", W, "info", "software", "An app stopped responding (Application Hang 1002)",
  "A program froze and was closed.",
  ["Update or reinstall it.", "If it happens with files on a network or USB drive, the drive may be slow or failing."],
  ev=("Application Hang", [1002]), ex=["[event] Application Hang | id 1002 | Error | The program notepad.exe version 10.0 stopped interacting with Windows and was closed."])
R("w.dotnet", W, "info", "software", ".NET Runtime error (1026)",
  "A program written in .NET crashed with an unhandled exception. The message names the program and the exception.",
  ["Update or reinstall the program.", "Repair or update the .NET runtime it uses."],
  ev=(".NET Runtime", [1026, 1023]), ex=["[event] .NET Runtime | id 1026 | Error | Application: tool.exe Framework Version: v4.0.30319 Description: The process was terminated due to an unhandled exception."])
R("w.dcom10016", W, "harmless", "software", "DistributedCOM 10016: permission settings",
  "A program asked for a Windows component it isn't set up to use. Microsoft says these are expected and can be ignored: they don't cause crashes or slowdowns.",
  ["Nothing to do."], ev=("DistributedCOM", [10016, 10010, 10005]),
  ex=["[event] Microsoft-Windows-DistributedCOM | id 10016 | Warning | The application-specific permission settings do not grant Local Activation permission for the COM Server application with CLSID"])
R("w.tpm", W, "harmless", "security", "TPM / time service / Kerberos noise",
  "Records from the security chip (TPM), the clock and sign-in services that are common on healthy computers.",
  ["Nothing to do unless BitLocker or Windows Hello is giving you trouble."],
  re_=r"^\[event\] [^|]*(?:tpm-wmi|time-service|security-kerberos|user device registration)[^|]*\| id \d+", kw="| id ",
  ex=["[event] Microsoft-Windows-Time-Service | id 129 | Warning | NtpClient was unable to set a domain peer to use as a time source."])
R("w.cputhrottle", W, "warning", "heat", "The processor is being slowed down (Kernel-Processor-Power 37)",
  "The firmware is limiting the processor's speed. Usually heat (blocked vents, dried thermal paste, a dead fan), or power (a laptop on a weak charger).",
  ["Clean the vents and fans; check the fans spin.", "Replace the thermal paste if the computer is a few years old.",
   "On a laptop, use the maker's charger. Fixie's Quick checks → Overheating walks through it."],
  ev=("Kernel-Processor-Power", [37, 35]), kw="processor-power",
  ex=["[event] Microsoft-Windows-Kernel-Processor-Power | id 37 | Warning | The speed of processor 0 in group 0 is being limited by system firmware."])
R("w.pnp219", W, "warning", "driver", "A driver failed to load for a device (Kernel-PnP 219)",
  "Windows couldn't load the driver for a device at start-up. Often a device that's switched off or unplugged; otherwise a missing driver.",
  ["See which device it names; check Device Manager for a yellow warning sign.", "Install or update its driver."],
  ev=("Kernel-PnP", [219, 411, 441]), ex=["[event] Microsoft-Windows-Kernel-PnP | id 219 | Warning | The driver \\Driver\\WudfRd failed to load for the device SWD\\WPDBUSENUM\\..."])
R("w.usbreset", W, "warning", "usb", "A USB device kept resetting or drew too much power",
  "A USB device reset, failed to describe itself, or drew more power than the port allows. Bad cables, unpowered hubs, faulty devices, or a damaged port.",
  ["Try another cable and port (a port on the back of a desktop).", "Use a powered hub for drives and bigger devices.", "If one device does it everywhere, the device is faulty."],
  re_=r"^\[event\] [^|]*(?:usbhub|usbxhci|usb-usbhub3|kernel-pnp)[^|]*\| id (?:43|44|10|19|20|28|39)\b|port reset failed|device descriptor request failed|power surge on the hub port", kw="",
  ex=["[event] Microsoft-Windows-USB-USBHUB3 | id 43 | Error | Port reset failed.", "Unknown USB Device (Device Descriptor Request Failed)"])
R("w.wlan", W, "info", "network", "Wi-Fi disconnected or failed to connect",
  "The Wi-Fi adapter lost or couldn't make its connection. Weak signal, power saving, the router, or the adapter's driver.",
  ["Move closer or restart the router.", "Update the Wi-Fi driver; turn off its power saving in Device Manager.", "If it happens on every network, the adapter or its antenna cables may be faulty."],
  ev=("WLAN-AutoConfig", [4001, 8001, 8002, 8003, 10000, 11004, 11005, 11006]),
  ex=["[event] Microsoft-Windows-WLAN-AutoConfig | id 8003 | Information | WLAN AutoConfig service has successfully disconnected from a wireless network."])
R("w.netwtw", W, "warning", "network", "The Wi-Fi or network adapter driver reported an error",
  "The network adapter's driver (Intel Netwtw, Realtek rt640x, Intel e1i/e1d…) reported a problem or the link went down. Drivers, power saving, cables or the adapter.",
  ["Update the network driver from the laptop or adapter maker.", "Turn off the adapter's power saving.", "For wired: try another cable and switch port."],
  re_=r"^\[event\] [^|]*(?:netwtw\d*|rt640x64|rtwlane\w*|e1[a-z]express|e1[dir]\w*|athw\w*|bcmwl\w*|qcamain\w*)[^|]*\| id \d+", kw="| id ",
  ex=["[event] Netwtw10 | id 5002 | Information | 6062 - Intel(R) Wireless-AC 9560: disassociated"])
R("w.defender", W, "critical", "security", "Microsoft Defender found something harmful",
  "The virus scanner found (and usually removed or quarantined) malware.",
  ["Open Windows Security → Virus & threat protection → Protection history to see what it was and what was done.",
   "Run a full scan; change passwords if it was a password stealer."],
  ev=("Windows Defender", [1116, 1117, 1006, 1007]),
  ex=["[event] Microsoft-Windows-Windows Defender | id 1116 | Warning | Microsoft Defender Antivirus has detected malware or other potentially unwanted software."])
R("w.update", W, "info", "software", "A Windows update failed to install",
  "An update couldn't install. The message gives a code (0x8…) that Fixie explains when it knows it.",
  ["Run Settings → System → Troubleshoot → Windows Update.", "Free up disk space.", "Run sfc /scannow and DISM /Online /Cleanup-Image /RestoreHealth."],
  ev=("WindowsUpdateClient", [20, 31, 34]), ex=["[event] Microsoft-Windows-WindowsUpdateClient | id 20 | Error | Installation Failure: Windows failed to install the following update with error 0x800f081f"])
R("w.volmgr", W, "warning", "boot", "Crash dump couldn't be saved (volmgr 46/161)",
  "Windows couldn't set up or write the crash dump file, so blue screens leave no details behind. Usually the page file is turned off or too small.",
  ["Settings → System → About → Advanced system settings → Performance → Advanced → Virtual memory: let Windows manage it."],
  ev=("volmgr", [45, 46, 49, 161]), ex=["[event] volmgr | id 46 | Error | Crash dump initialization failed!"])
R("w.batt", W, "warning", "battery", "Battery or power warning",
  "Windows recorded a battery or power-source problem: a critical battery level, or the charger being refused.",
  ["Use the maker's charger; check its plug and cable.", "Check battery health (powercfg /batteryreport). Fixie's Quick checks → Won't charge walks through it."],
  re_=r"^\[event\] [^|]*(?:kernel-power|battery)[^|]*\| id (?:42|105|130|137|524)\b.*(?:battery|power source|ac)|critical battery", kw="",
  ex=["[event] Microsoft-Windows-Kernel-Power | id 524 | Critical | The system was shut down because the battery charge level was critically low."])
R("w.sleep", W, "harmless", "power", "Sleeping, waking and power plan changes",
  "Ordinary records of the computer going to sleep, waking, or changing power settings.",
  ["Nothing to do, unless it won't stay asleep: then powercfg /lastwake shows what woke it."],
  re_=r"^\[event\] [^|]*(?:kernel-power|power-troubleshooter)[^|]*\| id (?:1|42|105|107|109|172|187|506|507|566)\b", kw="power",
  ex=["[event] Microsoft-Windows-Power-Troubleshooter | id 1 | Information | The system has returned from a low power state."])

# Windows stop (blue screen) codes, by name and number
STOP = [
  ("0x0000000a", "IRQL_NOT_LESS_OR_EQUAL", "A driver used memory it shouldn't have. Almost always a faulty or out-of-date driver; sometimes bad memory.", "Update drivers (especially network, graphics and anything installed recently); test the memory."),
  ("0x000000d1", "DRIVER_IRQL_NOT_LESS_OR_EQUAL", "A driver used memory it shouldn't have. The crash dump usually names it (often a network or Wi-Fi driver).", "Update or roll back the driver it names; test the memory if drivers are up to date."),
  ("0x0000001a", "MEMORY_MANAGEMENT", "Windows found its memory bookkeeping damaged. Usually faulty RAM or unstable memory settings; sometimes a driver.", "Test the memory (MemTest86, several passes); turn off XMP/EXPO; try one stick at a time."),
  ("0x00000050", "PAGE_FAULT_IN_NONPAGED_AREA", "Something read memory that wasn't there. Faulty RAM, a bad driver, antivirus software, or a damaged disk.", "Test the memory; update drivers; check the disk (chkdsk, SMART)."),
  ("0x0000007e", "SYSTEM_THREAD_EXCEPTION_NOT_HANDLED", "A driver crashed inside Windows. The dump names the file.", "Update or remove the driver it names (graphics drivers are common)."),
  ("0x0000003b", "SYSTEM_SERVICE_EXCEPTION", "A crash while running Windows' own code for a program, usually caused by a driver or security software.", "Update graphics and security software; run sfc /scannow."),
  ("0x000000ef", "CRITICAL_PROCESS_DIED", "A process Windows can't live without stopped. Damaged system files, a failing disk, or malware.", "Run sfc /scannow and DISM; check the disk's health; scan for malware."),
  ("0x00000133", "DPC_WATCHDOG_VIOLATION", "A driver took too long. Common with old SSD/SATA (storahci, iaStorA) and network drivers, and SSD firmware.", "Update the storage driver and the SSD's firmware; update the chipset drivers."),
  ("0x00000124", "WHEA_UNCORRECTABLE_ERROR", "The processor reported a hardware error it couldn't fix. Overclocking, too little voltage, overheating, failing CPU, memory or power supply.", "Undo overclocks and set the BIOS to defaults; check temperatures; test memory; try another power supply."),
  ("0x00000101", "CLOCK_WATCHDOG_TIMEOUT", "One processor core stopped responding. Overclocking, overheating, BIOS or a failing CPU.", "BIOS to defaults and up to date; check temperatures."),
  ("0x00000116", "VIDEO_TDR_FAILURE", "The graphics card stopped responding and couldn't be reset.", "Clean-install the graphics driver; undo GPU overclocks; check its temperature and power cables."),
  ("0x00000119", "VIDEO_SCHEDULER_INTERNAL_ERROR", "The graphics system failed. Graphics driver, or the graphics card.", "Clean-install the graphics driver; test with another card if you can."),
  ("0x0000007a", "KERNEL_DATA_INPAGE_ERROR", "Windows couldn't read part of its memory back from the disk. Usually a failing disk, cable or controller; sometimes RAM.", "Back up; check the disk's health and run chkdsk; reseat the cable."),
  ("0x00000077", "KERNEL_STACK_INPAGE_ERROR", "Windows couldn't read memory back from the disk. A failing disk, cable or RAM.", "Back up; check the disk's health; test the memory."),
  ("0x00000154", "UNEXPECTED_STORE_EXCEPTION", "Usually the disk: a failing or overheating SSD, its firmware, or its driver.", "Back up; check the SSD's health and firmware; update the storage driver."),
  ("0x0000007b", "INACCESSIBLE_BOOT_DEVICE", "Windows lost access to the drive it starts from: a BIOS change (AHCI/RAID mode), a failing drive, or a damaged boot setup.", "Undo recent BIOS changes (SATA mode); check the drive; use Startup Repair."),
  ("0x000000f4", "CRITICAL_OBJECT_TERMINATION", "A vital process ended, usually because the disk couldn't be read.", "Check the disk and its cable; back up."),
  ("0x00000024", "NTFS_FILE_SYSTEM", "The file system driver crashed: damaged file system or failing disk.", "Run chkdsk /f; check the disk's health."),
  ("0x00000019", "BAD_POOL_HEADER", "Windows' memory pool was damaged, usually by a driver, sometimes by bad RAM.", "Update drivers; test the memory."),
  ("0x000000c2", "BAD_POOL_CALLER", "A driver misused memory.", "Update drivers, especially recent ones."),
  ("0x0000009f", "DRIVER_POWER_STATE_FAILURE", "A driver didn't handle sleep or waking properly.", "Update graphics, network and chipset drivers; check for BIOS updates."),
  ("0x0000001e", "KMODE_EXCEPTION_NOT_HANDLED", "A driver crashed.", "Update or remove recently installed drivers and software."),
  ("0x00000139", "KERNEL_SECURITY_CHECK_FAILURE", "Windows found damaged data, often from a driver or bad memory.", "Update drivers; test the memory; run sfc /scannow."),
  ("0x00000109", "CRITICAL_STRUCTURE_CORRUPTION", "Windows' core code or data was changed: a bad driver, faulty RAM, or software that patches Windows.", "Remove system tweaking or cheat software; test memory."),
  ("0x00000074", "BAD_SYSTEM_CONFIG_INFO", "The registry is damaged.", "Use Startup Repair or System Restore."),
  ("0xc000021a", "STATUS_SYSTEM_PROCESS_TERMINATED", "A core Windows process failed, often after a bad update or damaged system files.", "Use Startup Repair, System Restore or uninstall the latest update."),
  ("0x0000013a", "KERNEL_MODE_HEAP_CORRUPTION", "A driver damaged memory.", "Update graphics and other drivers; test the memory."),
  ("0x000000fc", "ATTEMPTED_EXECUTE_OF_NOEXECUTE_MEMORY", "A driver ran code from memory marked as data.", "Update drivers; test the memory."),
  ("0x0000003d", "INTERRUPT_EXCEPTION_NOT_HANDLED", "A hardware interrupt wasn't handled: driver or hardware fault.", "Update drivers; remove recently added hardware."),
  ("0x000000ea", "THREAD_STUCK_IN_DEVICE_DRIVER", "The graphics card got stuck in a loop.", "Update the graphics driver; check the card's cooling and power."),
  ("0x0000009c", "MACHINE_CHECK_EXCEPTION", "The processor reported a fatal hardware error (older Windows).", "Check temperatures, power supply and overclocks."),
  ("0x00000144", "BUGCODE_USB3_DRIVER", "The USB 3 driver crashed, usually because of a faulty USB device or hub.", "Unplug USB devices one by one; update chipset and USB drivers."),
]
for code, name, means, do in STOP:
    h = "%x" % int(code, 16)
    R("w.stop." + name.lower(), W, "problem", "boot", "Stop code %s (0x%s)" % (name, h),
      "A blue screen with this stop code. " + means, [do, "If it keeps happening, the crash dumps in C:\\Windows\\Minidump name the driver."],
      re_=r"\b" + name + r"\b|(?:bug ?check|stop code|stop error|bccode)\W[^|]{0,30}?\b(?:0x)?0*" + h + r"\b", kw="",
      ex=[name, "The bugcheck was: 0x%08x (0x0000000000000000, 0x0, 0x0, 0x0)." % int(code, 16), "BCCode: %s" % h])

# Exception and error codes found in crash and update records
CODES = [
  ("0xc0000005", "Access violation", "info", "software", "The program read or wrote memory it wasn't allowed to: a bug in the program or a plug-in, or (if many programs do it) faulty RAM.", "Update the program; if many programs crash like this, test the memory."),
  ("0xc0000409", "Stack buffer overrun", "info", "software", "The program detected that its memory was overwritten and stopped itself for safety. A bug, or a damaged install.", "Update or reinstall the program."),
  ("0xc0000374", "Heap corruption", "info", "software", "The program damaged its own memory: a bug, a bad plug-in or (if many programs do it) faulty RAM.", "Update or reinstall it; test the memory if it's widespread."),
  ("0xe0434352", ".NET exception", "info", "software", "An unhandled error in a .NET program; the .NET Runtime record next to it has details.", "Update or reinstall the program and its .NET runtime."),
  ("0xc000001d", "Illegal instruction", "info", "cpu", "The program used a processor instruction this CPU doesn't have (for example a game needing AVX on an old CPU), or memory is damaged.", "Check the program's minimum processor requirements."),
  ("0xc0000135", "DLL not found", "info", "software", "A file the program needs (a DLL, often a Visual C++ or .NET runtime) is missing.", "Reinstall the program, or install the Visual C++ Redistributable it needs."),
  ("0xc000007b", "Wrong DLL type", "info", "software", "The program found a 32-bit file where it needed a 64-bit one (or the other way round), usually a broken runtime install.", "Reinstall the Visual C++ Redistributables (x86 and x64) and the program."),
  ("0x80070005", "Access denied", "info", "software", "Windows wasn't allowed to change a file or setting: permissions, security software, or a file in use.", "Run as administrator; pause security software; restart and try again."),
  ("0x80070002", "File not found", "info", "software", "A file needed (often an update download) was missing.", "Run the Windows Update troubleshooter, or clear SoftwareDistribution\\Download."),
  ("0x80070070", "Not enough disk space", "warning", "disk", "The disk is full.", "Free up space (Disk Clean-up, Storage Sense)."),
  ("0x800f081f", "Component source missing", "info", "software", "Windows couldn't find files it needs to repair or update itself.", "Run DISM /Online /Cleanup-Image /RestoreHealth, then try again."),
  ("0x80073712", "Component store damaged", "info", "software", "Windows' store of system files is damaged.", "Run DISM /Online /Cleanup-Image /RestoreHealth and sfc /scannow."),
  ("0x8024402c", "Update server not reachable", "info", "network", "Windows Update couldn't reach Microsoft: no internet, a proxy or DNS problem.", "Check the internet connection, proxy and DNS settings."),
  ("0x80070643", "Install failed", "info", "software", "An installer failed, often the Windows Recovery partition being too small for an update.", "Look up the update's number; Microsoft documents a fix for the recovery partition case."),
  ("0x80004005", "Unspecified error", "info", "software", "A general failure that doesn't say why; the lines around it usually do.", "Look at the lines just before it."),
  ("0x8007000d", "Data is invalid", "info", "software", "Damaged update files or settings.", "Run the Windows Update troubleshooter and DISM."),
  ("0x80070020", "File in use", "info", "software", "Another program (often antivirus) had the file open.", "Restart and try again; pause security software."),
  ("0x800704c7", "Cancelled", "harmless", "software", "The operation was cancelled (by you, a restart or a timeout).", "Try again."),
  ("0x80240017", "Update not applicable", "harmless", "software", "The update doesn't apply to this computer.", "Nothing to do."),
]
for code, name, sev, cat, means, do in CODES:
    R("w.code." + code[2:], W, sev, cat, "Code %s: %s" % (code, name), means, [do], re_=r"\b" + code + r"\b", kw=code,
      ex=["Exception code: %s" % code])

# Windows text logs: CBS, setupapi, Windows Update
R("w.cbs", W, "warning", "software", "System file check found damaged files",
  "sfc or DISM found Windows system files that are damaged (and says whether it could repair them).",
  ["If it couldn't repair them, run DISM /Online /Cleanup-Image /RestoreHealth then sfc /scannow again.", "Damaged files that come back point to a failing disk or RAM."],
  re_=r"cannot repair member file|corrupt(?:ed)? (?:file|member|manifest)|found corrupt files", kw="",
  ex=["2026-11-14 10:00:00, Info  CSI  00000412 [SR] Cannot repair member file [l:24]\"amd64_microsoft-windows-foo\""])
R("w.setupapi", W, "info", "driver", "A driver install failed (setupapi)",
  "Installing a driver for a device didn't work. setupapi.dev.log has the device and the reason.",
  ["Download the driver from the device maker; for Arduino clones, the CH340 or CP210x driver.", "Try another USB port."],
  re_=r"!!!\s+(?:dvi|ndv|sto|flq|inf):|device install failed|failed to install (?:the )?driver", kw="",
  ex=["!!!  dvi: Device install failed for USB\\VID_1A86&PID_7523"])

# ====================================================================== Linux
# (dmesg, journalctl, /var/log/syslog, kern.log; Raspberry Pi OS and ChromeOS too)
L = "linux"
R("l.oom", L, "warning", "memory", "Out of memory: a program was killed",
  "The computer ran out of memory and Linux closed a program (named in the line) to keep going.",
  ["Close programs you don't need; check which one was using the most (it's usually the one killed).", "Add swap or more RAM; on a Raspberry Pi, a bigger swap file helps."],
  re_=r"out of memory: kill(?:ed)? process|oom-kill(?:er)?|invoked oom-killer|killed process \d+ \(", kw="",
  ex=["[123.456] Out of memory: Killed process 4242 (chrome) total-vm:4000000kB", "kernel: python3 invoked oom-killer: gfp_mask=0x100cca"])
R("l.ioerr", L, "critical", "disk", "A disk read or write failed (I/O error)",
  "Linux couldn't read or write part of a disk (named sda, nvme0n1, mmcblk0…). A failing drive, SD card or USB stick, or a bad cable or power.",
  ["Back up now.", "Check the drive's health: sudo smartctl -a /dev/sdX (for SD cards, there's no health report: replace it).",
   "Reseat or replace the cable; for USB drives, try another port and a powered hub."],
  re_=r"(?:blk_update_request|i/o error).{0,40}\b(?:dev|device)\s+\w+|buffer i/o error on dev\w*\s+\w+|critical medium error|\bunrecovered read error\b", kw="error",
  ex=["[ 812.2] blk_update_request: I/O error, dev sda, sector 123456 op 0x0:(READ)", "Buffer I/O error on dev mmcblk0p2, logical block 1234, lost async page write", "I/O error, dev nvme0n1, sector 9999 op 0x1:(WRITE)"])
R("l.ata", L, "problem", "disk", "SATA link errors (ataN)",
  "The SATA connection to a drive had errors or reset. Cables, the drive's power, or a failing drive. 'UNC' or 'media error' means the drive itself has bad sectors.",
  ["Reseat or replace the SATA data and power cables.", "Check the drive's health with smartctl.", "Errors with UNC: back up and replace the drive."],
  re_=r"\bata\d+(?:\.\d+)?: (?:failed command|exception emask|hard resetting link|sata link down|link is slow|status: \{ drdy err \}|error: \{ unc \}|limiting sata link speed|srst failed)", kw="ata",
  ex=["ata1.00: exception Emask 0x0 SAct 0x0 SErr 0x0 action 0x0", "ata1.00: failed command: READ FPDMA QUEUED", "ata3: hard resetting link"])
R("l.ataunc", L, "critical", "disk", "The drive has unreadable sectors (UNC)",
  "The drive itself reported sectors it can't read: it is failing.",
  ["Back up now.", "Replace the drive."], re_=r"error: \{ unc \}|\bunc\b.*media error|uncorrectable error", kw="unc",
  ex=["ata1.00: error: { UNC }"])
R("l.nvme", L, "problem", "disk", "The NVMe SSD stopped answering",
  "The NVMe SSD timed out or its controller reset. Power-saving settings (APST), its firmware, overheating, or a failing drive.",
  ["Update the SSD's firmware and the BIOS.", "Check its temperature and health: sudo smartctl -a /dev/nvme0.",
   "If it happens after waking from sleep, try the kernel option nvme_core.default_ps_max_latency_us=0."],
  re_=r"nvme\d*:? .*(?:i/o \d+ (?:qid \d+ )?timeout|controller is down|resetting controller|device not ready; aborting|removing after probe failure|frozen state error)", kw="nvme",
  ex=["nvme nvme0: I/O 25 QID 3 timeout, aborting", "nvme nvme0: controller is down; will reset: CSTS=0xffffffff"])
R("l.ext4", L, "critical", "filesystem", "File system errors (EXT4, XFS, Btrfs, FAT)",
  "Linux found damage in a drive's file system. Files may be lost. Often after power cuts or pulling out a drive; repeated errors mean the drive is failing.",
  ["Back up what you can.", "Repair it with fsck while it isn't mounted (for the system drive, from a live USB or at boot).", "Check the drive's health; on a Raspberry Pi, replace the SD card if it keeps happening."],
  re_=r"(?:ext[234]-fs|xfs|btrfs|f2fs)\s+(?:error|critical)\b|(?:ext[234]-fs|btrfs) warning.*(?:csum|checksum|corrupt|error)|xfs .*corruption detected|btrfs.*csum failed|fat-fs \(\w+\): .*(?:invalid|error)|filesystem has errors", kw="",
  ex=["EXT4-fs error (device mmcblk0p2): ext4_lookup:1700: inode #2: comm bash: deleted inode referenced", "BTRFS warning (device sda2): csum failed root 5 ino 257"])
R("l.ro", L, "critical", "filesystem", "The drive went read-only",
  "Linux switched a drive to read-only to protect it after errors. Nothing can be saved to it until it's repaired.",
  ["Back up what you can.", "Restart and let fsck repair it; check the drive's health."],
  re_=r"remount(?:ing)? (?:filesystem )?read-only|read-only file system", kw="read-only",
  ex=["EXT4-fs (sda1): Remounting filesystem read-only", "touch: cannot touch 'x': Read-only file system"])
R("l.mmc", L, "problem", "disk", "SD card or eMMC errors",
  "The SD card or eMMC timed out or failed a read or write. Worn-out or fake SD cards, a poor power supply, or a loose card.",
  ["Use a good-quality card from a known brand (fake cards fail like this).", "Use the proper power supply (under-voltage causes card errors on a Pi).",
   "Copy your files off and write the system to a new card."],
  re_=r"mmc\d+: (?:timeout waiting for hardware interrupt|card \w+ removed|error -\d+|cmd\d+ error|tuning|data error|request timed out)|mmcblk\d+: error", kw="mmc",
  ex=["mmc0: Timeout waiting for hardware interrupt.", "mmcblk0: error -110 sending stop command"])
R("l.mce", L, "critical", "hardware", "Machine check: a hardware error",
  "The processor reported a hardware error (Machine Check Exception) in itself, the memory or the bus. Corrected ones are warnings; uncorrected ones crash the computer.",
  ["Undo overclocks and set the BIOS to defaults; update the BIOS.", "Test the memory (MemTest86); check temperatures.", "rasdaemon or mcelog can decode the bank and address."],
  re_=r"\bmce: \[hardware error\]|machine check(?: exception)?:|\[hardware error\]: (?:cpu|machine check|corrected error)|mce: cpu\d+: .*(?:error|events logged)", kw="",
  ex=["mce: [Hardware Error]: Machine check events logged", "[Hardware Error]: CPU 2: Machine Check: 0 Bank 5: bea0000000000108"])
R("l.edac", L, "problem", "memory", "Memory errors (EDAC)",
  "The memory controller reported memory errors. 'CE' were corrected (ECC memory saved you); 'UE' weren't. Repeated errors on one module mean it's failing.",
  ["Note the DIMM or channel named and replace that module.", "Reseat the memory; run MemTest86."],
  re_=r"edac \w+: .*(?:\bce\b|\bue\b|corrected|uncorrected).*(?:memory|error)|edac mc\d+:", kw="edac",
  ex=["EDAC MC0: 1 CE memory read error on CPU_SrcID#0_Ha#0_Chan#1_DIMM#0"])
R("l.aer", L, "warning", "hardware", "PCI Express bus errors (AER)",
  "A PCI Express device (graphics card, SSD, Wi-Fi card) had link errors. 'Corrected' ones are usually harmless if rare, but floods of them mean a signal problem; 'Uncorrected' are serious.",
  ["Note the device (bus address) and see which it is with lspci.", "Reseat the card or SSD; update the BIOS.",
   "If corrected errors flood the log, the kernel option pcie_aspm=off often stops them."],
  re_=r"aer: (?:corrected|uncorrected|multiple corrected) error|pcie bus error: severity=(?:corrected|uncorrected|uncorrectable)", kw="",
  ex=["pcieport 0000:00:1c.0: AER: Corrected error received: 0000:02:00.0", "pcieport 0000:00:1c.5: PCIe Bus Error: severity=Corrected, type=Physical Layer"])
R("l.thermal", L, "warning", "heat", "Overheating: the processor was slowed down",
  "The processor got too hot and was slowed down to protect itself; if it reaches its critical temperature, the computer shuts down.",
  ["Clean the vents, fans and heatsinks; check the fans spin.", "Replace the thermal paste if it's a few years old; on a Pi, add a heatsink or fan.", "Fixie's Quick checks → Overheating walks through it."],
  re_=r"(?:core|package) temperature above threshold|cpu clock throttled|thermal_zone\d+: critical temperature|critical temperature reached|thermal throttling|temperature above threshold|cpu\d+: package temperature", kw="temperature",
  ex=["CPU0: Core temperature above threshold, cpu clock throttled (total events = 1)", "thermal thermal_zone0: critical temperature reached (105 C), shutting down"])
R("l.undervolt", L, "problem", "power", "Under-voltage detected (Raspberry Pi)",
  "The Raspberry Pi's supply dropped below about 4.63 V. It slows down, SD cards get damaged, and USB devices drop out.",
  ["Use the official supply (5.1 V; 3 A for a Pi 4, 5 A for a Pi 5) and a short, thick cable.",
   "Power hungry USB devices (drives) from a powered hub.", "vcgencmd get_throttled shows if it's still happening (0x0 means fine)."],
  re_=r"under-?voltage detected|undervoltage detected|hwmon hwmon\d+: undervoltage", kw="voltage",
  ex=["hwmon hwmon1: Undervoltage detected!", "Under-voltage detected! (0x00050005)"])
R("l.voltok", L, "info", "power", "Voltage back to normal (Raspberry Pi)",
  "The supply recovered after an under-voltage warning.", ["See Under-voltage detected."], re_=r"voltage normali[sz]ed", kw="normali",
  ex=["Voltage normalised (0x00000000)"])
R("l.usberr", L, "warning", "usb", "A USB device failed to start (error -71, -110, -32)",
  "A USB device didn't answer properly when plugged in. -71 is a communication error (cable, port, power, or a faulty device), -110 a timeout, -32 a refused request.",
  ["Try another cable (a data cable, not charge-only) and port.", "Use a powered hub; on a Pi, check the power supply.", "If one device does it on every computer, the device is faulty."],
  re_=r"usb \d+-[\d.]+: (?:device descriptor read/\w+, error -\d+|device not accepting address \d+, error -\d+|unable to enumerate usb device|can't set config #?\d+, error -\d+|string descriptor 0 read error: -\d+)", kw="usb",
  ex=["usb 1-1.2: device descriptor read/64, error -71", "usb 1-1: device not accepting address 5, error -110", "usb 3-2: unable to enumerate USB device"])
R("l.usbovercurrent", L, "problem", "usb", "USB over-current",
  "A USB port shut off because something drew too much current or shorted it: a faulty device or cable, or a damaged port.",
  ["Unplug everything from that port; plug devices back one at a time.", "Check cables and plugs for damage. Fixie's Quick checks → USB problems walks through it."],
  re_=r"over-?current (?:change|condition)|over-?current on port|usb_port_over_current|hub \S+: over-?current", kw="current",
  ex=["usb usb1-port2: over-current condition", "hub 1-0:1.0: over-current change on port 1"])
R("l.usbdisconnect", L, "harmless", "usb", "USB device connected or disconnected",
  "Ordinary records of plugging and unplugging USB devices. Many repeated disconnects of the same device point to a bad cable or power.",
  ["Nothing to do unless the same device keeps dropping out."], re_=r"usb \d+-[\d.]+: (?:usb disconnect|new (?:full|high|low|super)-speed|new usb device found|product:|manufacturer:|serialnumber:)", kw="usb",
  ex=["usb 1-1.3: USB disconnect, device number 7", "usb 1-1.3: new full-speed USB device number 8 using xhci_hcd"])
R("l.serialbusy", L, "info", "usb", "Serial adapter driver messages (ch341, cp210x, ftdi)",
  "The serial adapter's driver attached or detached. 'brltty' taking a CH340 adapter makes it vanish: a common Linux problem.",
  ["If the adapter vanishes straight away and brltty is mentioned, remove it: sudo apt remove brltty."],
  re_=r"(?:ch341|cp210x|ftdi|pl2303)\S* converter (?:now )?(?:attached|disconnected|detected)|interface \d+ claimed by (?:ch341|cp210x|ftdi_sio|pl2303)|brltty.*(?:ttyusb|ch341)|'brltty'", kw="",
  ex=["usb 1-1: ch341-uart converter now attached to ttyUSB0", "usbfs: interface 0 claimed by ch341 while 'brltty' sets config #1"])
R("l.fw", L, "warning", "driver", "Firmware file missing",
  "A driver couldn't find the firmware file its device needs, so the device (often Wi-Fi, Bluetooth or graphics) may not work.",
  ["Install the firmware package: linux-firmware (Ubuntu, Debian 'firmware-misc-nonfree', Fedora 'linux-firmware').", "If the device works, it's a fallback file it didn't need: ignore it."],
  re_=r"direct firmware load for \S+ failed with error -\d+|firmware: failed to load|failed to load firmware|firmware file .* not found", kw="firmware",
  ex=["iwlwifi 0000:00:14.3: Direct firmware load for iwlwifi-so-a0-gf-a0-86.ucode failed with error -2"])
R("l.gpunv", L, "problem", "graphics", "NVIDIA GPU error (Xid)",
  "The NVIDIA driver reported an error. Xid 13, 31, 43 are usually program bugs; 48, 63, 64, 74, 79 (fell off the bus) and 92-95 point to the card, its power or overheating.",
  ["Note the Xid number.", "Update the driver; check the card's temperature and power cables.", "Xid 79 or 74: reseat the card and check its power; it may be failing."],
  re_=r"nvrm: xid \(", kw="xid", ex=["NVRM: Xid (PCI:0000:01:00): 79, pid=1234, GPU has fallen off the bus."])
R("l.gpuhang", L, "problem", "graphics", "Graphics hang or reset (amdgpu, i915, nouveau)",
  "The graphics chip stopped responding and the driver reset it: the screen freezes, flickers or goes black. Drivers, overheating, or a failing graphics card.",
  ["Update the kernel and Mesa drivers.", "Check the graphics card's temperature and fans.", "If it's the same game or app every time, it may be a driver bug for that app."],
  re_=r"amdgpu: .*(?:ring \S+ timeout|gpu reset|gpu fault|page fault|failed to initialize)|i915 \S+: .*(?:gpu hang|resetting|\[drm\] gpu hang)|\[drm(?::\w+)?\] \*error\* .*(?:hang|timeout|flip_done timed out)|nouveau \S+: .*(?:fifo|gr): .*(?:fault|error)", kw="",
  ex=["amdgpu 0000:03:00.0: amdgpu: ring gfx_0.0.0 timeout, signaled seq=1234", "i915 0000:00:02.0: [drm] GPU HANG: ecode 9:1:85dffffb"])
R("l.panic", L, "problem", "boot", "Kernel panic",
  "Linux crashed completely and stopped. The lines just before it show the cause (a driver, a failing disk, or memory).",
  ["Read the lines just before the panic: the module or function named is usually the cause.", "Test the memory and check the disk; try the previous kernel from the boot menu."],
  re_=r"kernel panic - not syncing", kw="panic", ex=["Kernel panic - not syncing: Fatal exception in interrupt"])
R("l.oops", L, "problem", "driver", "Kernel bug or oops",
  "A part of the kernel (usually a driver) hit a bug. The computer may carry on but be unstable.",
  ["The 'Modules linked in' and RIP lines name the driver.", "Update the kernel; if it started after installing a driver (DKMS), update or remove that driver."],
  re_=r"\bbug: (?:unable to handle|kernel null pointer|scheduling while atomic|sleeping function)|\boops: \d+|general protection fault(?:, probably for)?|kernel bug at|------------\[ cut here \]------------", kw="",
  ex=["BUG: kernel NULL pointer dereference, address: 0000000000000008", "Oops: 0000 [#1] PREEMPT SMP NOPTI", "general protection fault, probably for non-canonical address 0xdead"])
R("l.lockup", L, "problem", "cpu", "The system froze (soft/hard lockup, hung task, RCU stall)",
  "Part of the system stopped responding for seconds or minutes. Hung tasks often wait on a slow or failing disk; lockups and RCU stalls point to drivers, overheating, or hardware.",
  ["Hung tasks: check the disks (especially USB and network drives).", "Lockups: update the kernel and BIOS; check temperatures; test memory."],
  re_=r"watchdog: bug: soft lockup|hard lockup|rcu_(?:sched|preempt)? ?(?:self-)?detected (?:stall|cpu stall)|rcu: .*stall|task \S+ blocked for more than \d+ seconds|hung_task", kw="",
  ex=["watchdog: BUG: soft lockup - CPU#2 stuck for 22s! [kworker/2:1:123]", "INFO: task jbd2/sda1-8:345 blocked for more than 120 seconds."])
R("l.segfault", L, "info", "software", "A program crashed (segfault)",
  "A program crashed by using memory it shouldn't. The line names it. One program: a bug; many different ones: suspect RAM.",
  ["Update or reinstall the program named.", "If lots of different programs segfault, test the memory."],
  re_=r"\bsegfault at [0-9a-f]+ ip|traps: \S+\[\d+\] (?:general protection|trap invalid opcode|trap divide error)|\bsegmentation fault\b", kw="",
  ex=["app[1234]: segfault at 0 ip 00007f sp 00007ffd error 4 in libfoo.so", "Segmentation fault (core dumped)"])
R("l.coredump", L, "info", "software", "A program crashed and left a core dump",
  "systemd-coredump saved a crashed program's memory for debugging.",
  ["coredumpctl list shows them; update or reinstall the program."], re_=r"systemd-coredump\[\d+\]: process \d+ \(\S+\) of user \d+ dumped core", kw="coredump",
  ex=["systemd-coredump[999]: Process 1234 (gnome-shell) of user 1000 dumped core."])
R("l.svcfail", L, "warning", "software", "A system service failed",
  "A systemd service (named in the line) stopped with an error or couldn't start.",
  ["See why: systemctl status NAME and journalctl -u NAME.", "If it's a leftover from removed software, disable it."],
  re_=r"(?:\.service|\.mount|\.socket|\.timer): (?:failed with result|main process exited, code=(?:exited|killed|dumped), status=[1-9])|failed to start [^.]+\.?$|\bfailed to start\b", kw="fail",
  ex=["systemd[1]: bluetooth.service: Failed with result 'exit-code'.", "systemd[1]: Failed to start Network Manager Wait Online."])
R("l.acpi", L, "harmless", "boot", "ACPI BIOS errors",
  "The BIOS describes something in a way Linux doesn't like. Very common, and almost always harmless: the computer works anyway.",
  ["Nothing to do unless something (sleep, fans, brightness) doesn't work; then a BIOS update may help."],
  re_=r"acpi (?:bios )?(?:error|warning)|acpi error: (?:ae_|aborting method|no handler)|acpi: \\\S+: .*(?:error|failed)", kw="acpi",
  ex=["ACPI BIOS Error (bug): Could not resolve symbol [\\_SB.PCI0.XHC.RHUB.HS11], AE_NOT_FOUND", "ACPI Error: Aborting method \\_SB.PCI0 due to previous error (AE_NOT_FOUND)"])
R("l.taint", L, "harmless", "driver", "Kernel tainted / out-of-tree module",
  "A driver not shipped with the kernel was loaded (NVIDIA, VirtualBox, ZFS…). Normal with those drivers; it only means kernel developers won't look at bug reports.",
  ["Nothing to do."], re_=r"module verification failed|loading out-of-tree module taints kernel|module license '[^']+' taints kernel|tainting kernel", kw="taint",
  ex=["nvidia: loading out-of-tree module taints kernel."])
R("l.wifi", L, "info", "network", "Wi-Fi disconnected or failed",
  "The Wi-Fi connection dropped or authentication failed. Weak signal, a wrong password, power saving, or the driver.",
  ["Check signal and password; restart the router.", "Turn off Wi-Fi power saving (iw dev wlan0 set power_save off).", "Repeated firmware errors (brcmfmac, iwlwifi): update linux-firmware."],
  re_=r"wlan\d+: (?:deauthenticating|disassociated|authentication with \S+ timed out|associating with \S+ timed out|deauthenticated from)|\b(?:brcmfmac|iwlwifi|ath\d+k|mt76\w*|rtw\w*)\b.*(?:error|failed|timeout|microcode sw error|firmware error)", kw="",
  ex=["wlan0: deauthenticating from 11:22:33:44:55:66 by local choice (Reason: 3=DEAUTH_LEAVING)", "iwlwifi 0000:00:14.3: Microcode SW error detected. Restarting 0x0."])
R("l.eth", L, "warning", "network", "Network link down or transmit timeout",
  "The wired network connection went down, or the adapter got stuck (transmit queue timed out). Cables, switch ports, power saving or the driver.",
  ["Try another cable and switch port.", "Update the kernel; Realtek r8169 adapters sometimes need the r8168 driver."],
  re_=r"netdev watchdog: \S+ \(\S+\): transmit queue \d+ timed out|\b(?:eth\d+|enp\w+|eno\d+|end\d+): link is down|nic link is down", kw="",
  ex=["NETDEV WATCHDOG: enp3s0 (r8169): transmit queue 0 timed out", "r8169 0000:03:00.0 enp3s0: Link is Down"])
R("l.raid", L, "critical", "disk", "A RAID disk failed",
  "Software RAID (md) marked a disk as failed and carried on without it. The data is no longer protected.",
  ["cat /proc/mdstat shows which disk.", "Replace the failed disk soon and add it back to the array."],
  re_=r"md/raid\d*:\S+: (?:disk failure|operation continuing on \d+ devices)|md: \S+ (?:has failed|kicking non-fresh)", kw="md",
  ex=["md/raid1:md0: Disk failure on sdb1, disabling device."])
R("l.smart", L, "critical", "disk", "SMART says the drive is failing",
  "The drive's own self-monitoring (SMART) reported problems: pending or reallocated sectors, or a failed health check.",
  ["Back up now and replace the drive."],
  re_=r"smartd\[\d+\]: .*(?:failed|prefail|currently unreadable|offline uncorrectable|pending sectors|reallocated)|smart (?:overall-health|health status).*(?:failed|failing)", kw="",
  ex=["smartd[812]: Device: /dev/sda [SAT], 8 Currently unreadable (pending) sectors"])
R("l.squash", L, "problem", "filesystem", "SquashFS / snap read errors",
  "A compressed image (snap package or a live USB) couldn't be read: a damaged download or a failing drive.",
  ["Reinstall the snap, or re-make the live USB.", "Check the drive."],
  re_=r"squashfs error: (?:unable to read|xz decompression failed|zlib decompression failed)|squashfs error", kw="squashfs",
  ex=["SQUASHFS error: Unable to read data cache entry [1234]"])
R("l.audio", L, "info", "audio", "Sound card errors",
  "The sound driver had a problem: no codecs found, a timeout or a firmware issue. Sound may not work.",
  ["Update the kernel and sof-firmware (Intel laptops need it).", "Check the right output is chosen."],
  re_=r"snd_hda_intel \S+: (?:no codecs found|azx_get_response timeout|codec_write: .*error)|sof-audio-pci\S*.*(?:error|failed)|alsa.*(?:error|failed)", kw="",
  ex=["snd_hda_intel 0000:00:1f.3: azx_get_response timeout, switching to polling mode"])
R("l.bt", L, "info", "network", "Bluetooth errors",
  "The Bluetooth adapter or its firmware reported a problem.",
  ["Install linux-firmware; turn Bluetooth off and on.", "On a Pi, update the system (rpi-update is not needed: sudo apt full-upgrade)."],
  re_=r"bluetooth: hci\d+: .*(?:failed|error|timeout|command 0x\w+ tx timeout)", kw="bluetooth",
  ex=["Bluetooth: hci0: command 0x0c03 tx timeout"])
R("l.pibrownout", L, "info", "power", "Raspberry Pi throttled",
  "vcgencmd get_throttled's number: bit 0 under-voltage now, 1 frequency capped, 2 throttled, 3 soft temperature limit; bits 16-19 the same, at some time since start-up. 0x50000 or 0x50005 means under-voltage happened.",
  ["0x0 is fine. Anything with 0x1 or 0x10000: the power supply. 0x8 or 0x80000: heat."],
  re_=r"throttled=0x[1-9a-f][0-9a-f]*", kw="throttled", ex=["throttled=0x50005"])
R("l.fsck", L, "warning", "filesystem", "The file system check found and fixed problems",
  "fsck repaired damage at start-up, often after the power was cut.",
  ["Shut down properly; check the drive's health if it keeps happening."],
  re_=r"(?:fsck|e2fsck)\S*: .*(?:clearing orphaned inode|was not cleanly unmounted|errors corrected|file system was modified|recovering journal)|\*\*\*\*\* file system was modified \*\*\*\*\*", kw="",
  ex=["fsck.ext4: /dev/sda1 was not cleanly unmounted, check forced."])
R("l.sdcardfake", L, "critical", "disk", "SD card or USB stick is fake or damaged",
  "The card or stick claims more space than it has, or reports impossible sizes: typical of fake cards, which lose data once they fill.",
  ["Test it with f3 or H2testw; use a card from a known brand and shop."],
  re_=r"attempt to access beyond end of device|access beyond end of device", kw="beyond end",
  ex=["attempt to access beyond end of device"])

# ====================================================================== macOS
M = "macos"
R("m.panic", M, "problem", "boot", "Kernel panic",
  "macOS crashed completely and restarted. The report names the cause: the line after 'panic(' and the kernel extensions in the backtrace.",
  ["Note the panic message and any third-party extension (not com.apple) named in the backtrace: update or remove it.",
   "Unplug accessories and see if it stops; run Apple Diagnostics (hold D, or the power button on Apple silicon).", "Repeated different panics point to hardware: memory, logic board or a swollen battery."],
  re_=r"\bpanic\((?:cpu \d+ caller|cpu)|panic(?:ked)? (?:report|string)|kernel panic", kw="panic",
  ex=["panic(cpu 2 caller 0xffffff8012345678): \"Spinlock timeout\" @/AppleIntelCPUPowerManagement", "Panic(CPU 0, time 123456): NMIPI for unresponsive processor"])
R("m.watchdog", M, "problem", "software", "Watchdog timeout",
  "Part of macOS stopped checking in and the Mac was restarted. Often a hung driver, a full or failing disk, or a third-party extension.",
  ["Update macOS and remove old system extensions.", "Check the disk with Disk Utility → First Aid; free up space."],
  re_=r"watchdog timeout: no checkins from watchdogd|userspace watchdog timeout|no successful checkins from", kw="checkin",
  ex=["userspace watchdog timeout: no successful checkins from WindowServer in 120 seconds"])
R("m.gpu", M, "problem", "graphics", "GPU panic or restart",
  "The graphics chip hung or was restarted. Older MacBook Pros with discrete graphics (2010-2013, some 2016-2019) are known to fail like this.",
  ["Update macOS.", "Try forcing the built-in graphics (gfxCardStatus or Energy settings).", "If it keeps happening, the graphics chip or its solder may be failing: a board-level repair."],
  re_=r"gpu panic|gpu restart|gpu reset|agpm|amdradeonx\d+.*(?:hang|timeout|error)|graphics hardware encountered an error", kw="gpu",
  ex=["GPU Panic: mux-regs 5 0 7f 0 0 0 severity 3 WS-ready 1"])
R("m.sleepwake", M, "warning", "power", "Sleep or wake failure",
  "The Mac failed to go to sleep or wake properly.",
  ["Reset the SMC (Intel Macs) and NVRAM.", "Unplug accessories; update macOS."],
  re_=r"sleep wake failure in efi|wake failure|sleep/wake hang|failure during sleep", kw="",
  ex=["Sleep Wake failure in EFI"])
R("m.disk", M, "critical", "disk", "Disk I/O or APFS errors",
  "macOS couldn't read or write the disk, or found damage in the APFS file system.",
  ["Back up now (Time Machine).", "Run Disk Utility → First Aid.", "On older Macs, a failing SSD or hard drive (or its cable, on 2012-era MacBook Pros) causes this."],
  re_=r"disk\d+s?\d*: i/o error|apfs.*(?:error|corrupt|failed)|\bi/o error\b.*disk\d|nx_corruption|hfs: .*(?:error|corrupt)", kw="",
  ex=["disk0s2: I/O error.", "apfs_vfsop_mount:2132: failed to mount container disk1"])
R("m.crash.segv", M, "info", "software", "A program crashed (EXC_BAD_ACCESS)",
  "The program used memory it shouldn't have (SIGSEGV or SIGBUS): a bug, a bad plug-in, or (if many programs do it) memory.",
  ["Update or reinstall the program; remove plug-ins.", "If many programs crash, run Apple Diagnostics."],
  re_=r"exc_bad_access|exception type:\s+exc_bad_access|\bsig(?:segv|bus)\b", kw="",
  ex=["Exception Type:        EXC_BAD_ACCESS (SIGSEGV)"])
R("m.crash.abrt", M, "info", "software", "A program stopped itself (EXC_CRASH / SIGABRT)",
  "The program found a problem and stopped itself on purpose. The 'Application Specific Information' section usually says why.",
  ["Update or reinstall it; reset its settings."], re_=r"exc_crash|\bsigabrt\b|abort\(\) called", kw="", ex=["Exception Type:  EXC_CRASH (SIGABRT)"])
R("m.crash.ill", M, "info", "software", "A program crashed (EXC_BAD_INSTRUCTION / SIGILL)",
  "The program hit an error check (common in Swift programs) or an instruction this Mac can't run.",
  ["Update the program; check it supports this Mac (Apple silicon or Intel)."], re_=r"exc_bad_instruction|exc_breakpoint|\bsigill\b|\bsigtrap\b", kw="",
  ex=["Exception Type:        EXC_BAD_INSTRUCTION (SIGILL)"])
R("m.codesign", M, "warning", "security", "Code signing problem",
  "macOS stopped a program because its signature is invalid or it was changed after signing.",
  ["Download the program again from its maker.", "For your own builds, re-sign them."], re_=r"namespace codesigning|code signature invalid|termination reason: .*codesigning", kw="",
  ex=["Termination Reason:    Namespace CODESIGNING, Code 2 Invalid Page"])
R("m.jetsam", M, "warning", "memory", "Out of memory (jetsam)",
  "The Mac (or iPhone) ran out of memory and closed programs.",
  ["Close programs you don't need; restart.", "Check Activity Monitor → Memory for the one using the most."],
  re_=r"\bjetsam\b|low memory|memory pressure (?:critical|warning)|vm-compressor-space-shortage", kw="",
  ex=["Jetsam Event Report: largestProcess: Safari"])
R("m.thermal", M, "warning", "heat", "Overheating or thermal pressure",
  "The Mac slowed down to cool off, or its sensors reported a high temperature.",
  ["Clean the vents (Intel Macs collect dust in the fans).", "Keep it on a hard surface; check Activity Monitor for a stuck process using the CPU."],
  re_=r"thermal (?:pressure|level|warning)|cpu_power.*throttl|thermalmonitord|prochot|smc (?:thermal|temperature)", kw="",
  ex=["thermalmonitord: thermal pressure level 3 (heavy)"])
R("m.usb", M, "warning", "usb", "USB over-current or power",
  "A USB device drew too much power and its port was turned off ('USB Accessories Disabled').",
  ["Unplug the device; use a powered hub.", "Check the cable and device for damage."],
  re_=r"over-?current|usb accessories disabled|requires more power than the port|iousbhost.*(?:enumeration failure|error)", kw="",
  ex=["IOUSBHostDevice: over-current on port 2", "USB Accessories Disabled"])
R("m.battery", M, "warning", "battery", "Battery needs service",
  "macOS reports the battery is worn or failing ('Service Recommended').",
  ["Check System Settings → Battery → Battery Health.", "A swollen battery lifts the trackpad or case: stop using it and have it replaced."],
  re_=r"battery (?:service|condition: (?:service|replace))|service (?:recommended|battery)|permanent failure.*battery", kw="",
  ex=["Battery condition: Service Recommended"])
SHUT = [
  ("5", "info", "power", "Normal shutdown", "The Mac was shut down normally (from the menu or software)."),
  ("3", "info", "power", "Power button held", "The power button was held down to force the Mac off."),
  ("0", "warning", "power", "Power lost", "The Mac lost power suddenly (no battery and the power was cut, or a power fault)."),
  ("-3", "warning", "heat", "Too hot", "The Mac shut down because it got too hot (several sensors over the limit)."),
  ("-60", "warning", "disk", "Bad disk", "Shut down because of a problem with the disk (bad master directory block)."),
  ("-62", "warning", "software", "Watchdog restart", "The watchdog restarted the Mac after it hung."),
  ("-64", "problem", "power", "Power problem", "Shut down for a power problem."),
  ("-71", "warning", "heat", "Memory too hot", "Shut down because the memory got too hot (SO-DIMM temperature)."),
  ("-74", "warning", "battery", "Battery too hot", "Shut down because the battery got too hot."),
  ("-75", "problem", "power", "Charger communication", "The Mac couldn't talk to the charger: a faulty charger, cable or charging port."),
  ("-78", "problem", "power", "Wrong current", "The current drawn was wrong: a power or logic board fault."),
  ("-86", "warning", "heat", "Proximity sensor too hot", "A temperature sensor near the CPU or GPU went over the limit."),
  ("-95", "warning", "heat", "CPU too hot", "The processor got too hot."),
  ("-100", "warning", "power", "Power supply too hot", "The power supply got too hot."),
  ("-101", "warning", "heat", "LCD too hot", "Shut down because the display got too hot."),
  ("-103", "warning", "battery", "Battery cell under voltage", "A battery cell's voltage dropped too low: the battery is worn or failing."),
  ("-104", "problem", "battery", "Battery problem", "An unknown battery problem: the battery may need replacing."),
  ("-112", "warning", "power", "Hardware watchdog", "A hardware watchdog or memory-related reset."),
  ("-128", "info", "power", "Unknown", "An unknown reason, often a power loss on Apple silicon."),
]
for code, sev, cat, name, means in SHUT:
    R("m.shut" + code, M, sev, cat, "Previous shutdown cause %s: %s" % (code, name),
      means + " (from 'Previous shutdown cause' in the log).",
      ["If it's heat: clean the vents and check the fans.", "If it's power or battery: check the charger and battery health.", "Note how often it happens."],
      re_=r"previous shutdown cause:? " + esc(code) + r"(?:$|[^0-9])", kw="shutdown cause",
      ex=["kernel: Previous shutdown cause: %s" % code])

# ====================================================================== Android
A = "android"
R("a.fatal", A, "info", "software", "An app crashed (FATAL EXCEPTION)",
  "An Android app crashed; the lines after it show the error and where in the app it happened.",
  ["Update the app, or clear its cache and data in Settings → Apps.", "Reinstall it."],
  re_=r"fatal exception:|androidruntime: fatal|androidruntime: process: ", kw="", ex=["E AndroidRuntime: FATAL EXCEPTION: main"])
R("a.anr", A, "info", "software", "An app stopped responding (ANR)",
  "An app froze for several seconds and Android offered to close it.",
  ["Update the app; restart the phone.", "If many apps freeze, the phone's storage may be full or worn."],
  re_=r"\banr in\b|application not responding|am_anr", kw="", ex=["E ActivityManager: ANR in com.example.app"])
R("a.native", A, "info", "software", "A native crash (Fatal signal)",
  "Native code in an app or system service crashed (a 'tombstone' is saved).",
  ["Update the app or the phone's software."], re_=r"fatal signal \d+ \(sig\w+\)|\*\*\* \*\*\* \*\*\*.*\*\*\*|tombstone written to", kw="",
  ex=["F libc    : Fatal signal 11 (SIGSEGV), code 1 (SEGV_MAPERR), fault addr 0x0 in tid 1234"])
R("a.lmk", A, "warning", "memory", "Low memory: apps were closed",
  "Android ran short of memory and closed background apps (lowmemorykiller / lmkd).",
  ["Close apps you don't need; restart.", "Free up storage: a full phone gets slow and short of memory."],
  re_=r"lowmemorykiller|\blmkd\b.*(?:kill|reclaim)|am_low_memory", kw="", ex=["I lowmemorykiller: Kill 'com.example' (1234), uid 10123, oom_score_adj 900"])
R("a.thermal", A, "warning", "heat", "Phone overheating",
  "The phone is too hot and is slowing down, or shut down.",
  ["Take it out of its case, stop charging and close games.", "If it's hot while doing nothing, the battery may be failing: check it for swelling."],
  re_=r"thermal(?:-engine|service|_hal)?.*(?:throttl|shutdown|overheat|critical|severe)|overheat", kw="", ex=["W ThermalEngine: Thermal shutdown triggered"])
R("a.battery", A, "warning", "battery", "Battery or charging problem",
  "Android reported a battery problem: overheating, over-voltage, a dead battery, or charging stopped.",
  ["Use the original charger and cable; clean the charging port.", "A swollen battery lifts the screen or back: stop using it."],
  re_=r"battery (?:health|status).*(?:overheat|dead|over_voltage|unspecified_failure|cold)|charging (?:disabled|stopped|suspended)", kw="",
  ex=["BatteryService: battery health=Overheat"])
R("a.storage", A, "warning", "disk", "Storage full or failing",
  "The phone's storage is full, or the system had trouble reading it.",
  ["Free up space.", "If storage errors continue, back up: the flash memory may be wearing out."],
  re_=r"(?:low on |insufficient )storage|no space left on device|enospc|f2fs-fs.*error|i/o error.*(?:mmcblk|sda|dm-)", kw="", ex=["E vold: No space left on device"])

# ====================================================================== any system
X = "any"
R("x.nospace", X, "warning", "disk", "The disk is full",
  "There's no space left on a drive, so programs can't save and updates fail.",
  ["Free up space: empty the bin, remove old downloads and big files you don't need."],
  re_=r"no space left on device|disk (?:is )?full|not enough (?:free )?(?:disk )?space|insufficient disk space", kw="space",
  ex=["write error: No space left on device"])
R("x.smartfail", X, "critical", "disk", "SMART: drive failure predicted",
  "The drive's own health check predicts it will fail.",
  ["Back up now and replace the drive."],
  re_=r"smart (?:failure predicted|status bad|status: (?:bad|fail))|failure is predicted|predictive failure|reallocated[_ ]sector[_ ]ct.*(?:fail|[1-9]\d* *$)", kw="",
  ex=["SMART Failure Predicted on Hard Disk 0"])
R("x.memtest", X, "critical", "memory", "Memory test failed",
  "A memory test (MemTest86, Windows Memory Diagnostic) found errors: the RAM (or its settings) is faulty.",
  ["Turn off XMP/EXPO and test again.", "Test one stick at a time to find the bad one, and replace it."],
  re_=r"memory diagnostic(?:s)? (?:tested the computer's memory and )?detected (?:hardware )?errors|memtest.*(?:fail|errors?: [1-9])|\b[1-9]\d* errors? found.*memory", kw="",
  ex=["The Windows Memory Diagnostic tested the computer's memory and detected hardware errors."])

if __name__ == "__main__":
    ids = [r["id"] for r in RULES]
    dup = {i for i in ids if ids.count(i) > 1}
    assert not dup, dup
    bad = []
    for r in RULES:
        rx = re.compile(r["re"], re.I)
        bad += [(r["id"], e) for e in r["ex"] if not rx.search(e)]
    assert not bad, "\n".join("%s: %s" % b for b in bad)
    out = sys.argv[1] if len(sys.argv) > 1 else "web/logrules.json"
    with open(out, "w") as f:
        json.dump({"version": 1, "rules": RULES}, f, ensure_ascii=False, separators=(",", ":"))
    print(len(RULES), "rules ->", out)

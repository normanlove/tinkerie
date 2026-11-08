// Meterie's icons for the Mac/Linux front end, as 24×24 line drawings. On
// Windows the same code points come from the Segoe Fluent Icons font.
// Paths are original drawings for Tinkerie by nrmluvdev. "f:" means filled.
const ICONS = {
  0xE80F: "M3 11l9-8 9 8M5 9.5V20h5v-6h4v6h5V9.5",                                  // home
  0xE713: "M12 8.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7zM12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9L7 7M17 17l2.1 2.1M4.9 19.1L7 17M17 7l2.1-2.1", // settings
  0xE768: "f:M7 4l13 8-13 8z",                                                       // play
  0xE769: "f:M6.5 4h3.5v16H6.5zM14 4h3.5v16H14z",                                    // pause
  0xE71A: "f:M6 6h12v12H6z",                                                         // stop
  0xE7C8: "f:M12 5a7 7 0 1 0 0 14 7 7 0 0 0 0-14z",                                  // record
  0xE74E: "M5 3h11l3 3v15H5zM8 3v5h7V3M8 21v-7h8v7",                                 // save
  0xE8E5: "M3 19V5h6l2 2h8v3M3 19l3-9h15l-3 9z",                                     // open
  0xE710: "M12 5v14M5 12h14",                                                        // add
  0xE711: "M6 6l12 12M18 6L6 18",                                                    // close
  0xE738: "M5 12h14",                                                                // remove
  0xE74D: "M4 6h16M9 6V4h6v2M6 6l1 14h10l1-14M10 10v7M14 10v7",                      // delete
  0xE72C: "M20 12a8 8 0 1 1-2.3-5.6M20 4v5h-5",                                      // refresh
  0xE722: "M4 8h3l2-3h6l2 3h3v11H4zM12 10a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7z",     // camera
  0xE702: "M7 7l10 10-5 4V3l5 4L7 17",                                               // bluetooth
  0xE88E: "M12 21V5M9 7l3-4 3 4M12 15l-5-3V9M12 12l5-3V7M5.5 7.5h3v2h-3zM17 5.5h.01", // usb
  0xE7BA: "M12 3l10 18H2zM12 10v5M12 18h.01",                                        // warning
  0xE946: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 11v6M12 7.5h.01",                 // info
  0xE945: "M13 2L4 14h7l-1 8 9-12h-7z",                                              // bolt
  0xE9D2: "M4 4v16h16M7 15l4-5 3 3 5-6",                                             // chart
  0xE8EF: "M6 3h12v18H6zM9 6h6v3H9zM9 13h.01M12 13h.01M15 13h.01M9 17h.01M12 17h.01M15 17h.01", // calculator
  0xEA80: "M9 18h6M10 21h4M12 3a6 6 0 0 0-4 10.5c.8.8 1 1.5 1 2.5h6c0-1 .2-1.7 1-2.5A6 6 0 0 0 12 3z", // bulb
  0xE7C1: "M5 21V4M5 4h11l-2 4 2 4H5",                                               // flag
  0xE70F: "M4 20h4L19 9l-4-4L4 16zM14 6l4 4",                                        // edit
  0xE8FD: "M9 6h11M9 12h11M9 18h11M4 6h.01M4 12h.01M4 18h.01",                       // list
  0xE8C8: "M9 9h11v11H9zM5 15H4V4h11v1",                                             // copy
  0xE8A3: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4M11 8v6M8 11h6",           // zoom in
  0xE71F: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4M8 11h6",                  // zoom out
  0xE740: "M4 9V4h5M20 9V4h-5M4 15v5h5M20 15v5h-5",                                  // fit
  0xE73E: "M4 12l5 5L20 6",                                                          // check
  0xE70D: "M6 9l6 6 6-6",                                                            // down
  0xE70E: "M6 15l6-6 6 6",                                                           // up
  0xE76B: "M15 6l-6 6 6 6",                                                          // left
  0xE76C: "M9 6l6 6-6 6",                                                            // right
  0xE767: "M4 9h4l5-4v14l-5-4H4zM16 9a4 4 0 0 1 0 6M18.5 6.5a8 8 0 0 1 0 11",        // volume
  0xE74F: "M4 9h4l5-4v14l-5-4H4zM16 9l5 6M21 9l-5 6",                                // mute
  0xE897: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .8-1 1.5V14M12 17.5h.01", // help
  0xE9D9: "M3 12h4l2-5 4 10 2-5h6",                                                  // diagnostics
  0xE8B7: "M3 6h6l2 2h10v11H3z",                                                     // folder
  0xE823: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 7v5l3 2",                         // clock
  0xE909: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18", // world
  0xEB9F: "M4 5h16v14H4zM4 16l5-5 4 4 3-3 4 4M15.5 8.5h.01",                         // photo
  0xE721: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4",                         // search
  0xE895: "M4 12a8 8 0 0 1 13.7-5.7L20 8M20 4v4h-4M20 12a8 8 0 0 1-13.7 5.7L4 16M4 20v-4h4", // sync
  0xE8EC: "M3 3h8l10 10-8 8L3 11zM7.5 7.5h.01",                                      // tag
  0xE890: "M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12zM12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z", // view
  0xE950: "M2 12h4l1.5-3 3 6 3-6 3 6 1.5-3h4",                                       // component
};

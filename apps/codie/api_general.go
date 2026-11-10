// Codie — api_general.go: well-known functions of everyday languages.
// Format: languages | names | kind | what | why | tags  (see api.go)
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

func addAPI(t string) bool { apiTables = append(apiTables, t); return true }

var _ = addAPI(`
# ---------------------------------------------------------------- Go
go | fmt.Println | print | Prints its values as text, with spaces between them and a new line at the end. | The simplest way to show output or check what a value is while testing. | output
go | fmt.Printf | print | Prints text built from a template: %d becomes a whole number, %s text, %v any value, %.2f a number with 2 decimals. | Printf gives exact control over how the output looks. | output
go | fmt.Print | print | Prints its values without a new line. | | output
go | fmt.Sprintf | call | Builds a piece of text from a template (like Printf) and gives it back instead of printing it. | Used to make messages, file names and labels. | text
go | fmt.Errorf | call | Makes an error with a formatted message; %w wraps another error inside it. | Adding context ('reading config: file not found') makes errors easy to trace. | errors
go | fmt.Fprintln,fmt.Fprintf | print | Prints to a chosen destination, such as a file or os.Stderr. | | output
go | errors.New | call | Makes a new error with a fixed message. | | errors
go | errors.Is,errors.As | call | Checks whether an error is (or wraps) a particular kind of error. | | errors
go | os.Open | call | Opens a file for reading. It gives back the file and an error. | Remember to close it (often with defer f.Close()). | files
go | os.Create | call | Creates (or empties) a file for writing. | | files
go | os.ReadFile | call | Reads a whole file into memory in one go. | Easiest way to load a small file. | files
go | os.WriteFile | call | Writes data into a file, replacing what was there. | The last number (like 0o644) sets who may read and write it on Linux and macOS. | files
go | os.MkdirAll | call | Makes a folder, and any missing folders above it. | | files
go | os.Remove,os.RemoveAll | call | Deletes a file (RemoveAll: a folder and everything in it). | | files
go | os.Exit | call | Ends the program immediately with an exit code (0 = success). | Deferred functions do NOT run when you call os.Exit. |
go | os.Getenv | call | Reads an environment variable, such as PATH or APPDATA. | Settings and folders are often found through environment variables. |
go | os.UserConfigDir | call | Gives the folder where programs keep their settings (on Windows, %AppData%). | |
go | os.Args | prop | The words typed after the program's name on the command line. | os.Args[0] is the program itself. |
go | os.Stat | call | Looks up a file's details (size, date, folder or not) and says if it exists. | | files
go | log.Println,log.Printf | print | Writes a log message with the date and time in front. | Logs help work out what a program did, especially when it has no window. | output
go | log.Fatal,log.Fatalf | error | Logs a message and then ends the program. | For errors the program can't continue after. | errors
go | strings.Split | call | Cuts text into a list of pieces wherever a separator appears. | | text
go | strings.Join | call | Joins a list of texts into one, with a separator between. | | text
go | strings.Contains | call | Answers whether one text appears inside another. | | text
go | strings.HasPrefix,strings.HasSuffix | call | Answers whether text starts (or ends) with something. | | text
go | strings.TrimSpace | call | Removes spaces and new lines from both ends of a text. | Useful on user input and lines read from files. | text
go | strings.ToLower,strings.ToUpper | call | Changes text to lower (or upper) case. | Handy for comparisons that ignore case. | text
go | strings.Replace,strings.ReplaceAll | call | Replaces parts of a text with something else. | | text
go | strings.Builder | call | A fast way to build up a long text bit by bit. | | text
go | strconv.Atoi | call | Turns text such as "42" into the number 42, with an error if it isn't a number. | | text
go | strconv.Itoa | call | Turns a whole number into text. | | text
go | strconv.ParseFloat | call | Turns text into a decimal number. | | text
go | time.Now | call | Gets the current date and time. | | timers
go | time.Sleep | call | Pauses this goroutine for a while (for example 500 * time.Millisecond). | Other goroutines keep running. | timers
go | time.Since | call | Works out how much time has passed since an earlier moment. | Used for timing and timeouts. | timers
go | time.NewTicker | call | Makes a ticker that sends a signal at a steady interval. | Good for doing something every second, like a simple game or status loop. | timers
go | time.After | call | Gives a channel that receives a value after a delay. | Used in select for timeouts. | timers
go | time.Second,time.Millisecond | prop | A unit of time. | Multiply it, for example 2 * time.Second. | timers
go | make | call | Makes a new slice, map or channel ready to use. | Maps and channels must be made before use. |
go | append | call | Adds items to the end of a slice and gives back the longer slice. | Always store the result: s = append(s, x). | list
go | len | call | Gives the length: how many items, bytes or characters. | | list
go | cap | call | Gives how many items a slice can hold before it needs to grow. | | list
go | copy | call | Copies items from one slice into another. | | list
go | delete | call | Removes a key from a map. | | list
go | close | call | Closes a channel, telling receivers nothing more will come. | | concurrency
go | recover | call | Catches a panic inside a deferred function so the program can carry on. | | errors
go | new | call | Makes a zeroed value and gives back a pointer to it. | | memory
go | filepath.Join | call | Joins parts of a file path with the right separator for this system (\ on Windows, / elsewhere). | Makes code work on every operating system. | files
go | filepath.Ext,filepath.Base,filepath.Dir | call | Takes a path apart: its extension, its last part or its folder. | | files
go | exec.Command | call | Prepares to run another program with arguments. | Call .Run(), .Start() or .Output() on the result to actually run it. |
go | http.HandleFunc | call | Tells the web server which function answers a web address. | | network
go | http.ListenAndServe | call | Starts a web server on an address and port and keeps serving forever. | | network
go | http.Get | call | Downloads a web page or API answer. | Remember to close resp.Body. | network
go | json.Marshal | call | Turns a Go value into JSON text. | | data
go | json.Unmarshal | call | Fills a Go value from JSON text. | | data
go | json.NewDecoder,json.NewEncoder | call | Reads (or writes) JSON straight from (or to) a stream such as a file or web request. | | data
go | bufio.NewScanner | call | Makes a scanner that reads input piece by piece, usually line by line. | Loop with scanner.Scan() and read scanner.Text(). | input
go | bufio.NewReader | call | Wraps a reader with a buffer for faster, easier reading. | | input
go | rand.Intn | call | Gives a random whole number from 0 up to (not including) n. | | random
go | rand.Float64 | call | Gives a random decimal between 0 and 1. | | random
go | math.Sqrt | call | Square root. | Used for distances: sqrt(dx*dx + dy*dy). | maths
go | math.Abs,math.Max,math.Min,math.Floor,math.Pow | call | A maths helper. | | maths
go | sync.WaitGroup | call | Counts running goroutines so you can wait for all of them to finish. | | concurrency
go | sync.Mutex | call | A lock: only one goroutine at a time may hold it. | Protects shared data from being changed by two goroutines at once. | concurrency
go | .Lock,.Unlock | call | Takes (or releases) a lock. | | concurrency
go | context.Background,context.WithTimeout,context.WithCancel | call | Makes a context, which carries deadlines and cancel signals through a program. | | concurrency
go | runtime.GOOS | prop | The operating system the program is running on: "windows", "linux", "darwin" (macOS). | Lets one program behave differently per system at run time. | windows-go
go | flag.Int,flag.String,flag.Bool | call | Declares a command-line option, such as -port 8080. | | input
go | flag.Parse | call | Reads the command-line options declared above. | | input
go | .Close | call | Closes the file, connection or handle. | Frees it for others; on Windows, open files can't be deleted until closed. | files
go | .Scan | call | Moves to the next piece of input (next line), answering false at the end. | | input
go | .Text | call | Gives the text the scanner just read. | | input
go | .Error | call | Gives an error's message as text. | | errors
go | .Run | call | Runs the prepared command and waits for it to finish. | |
go | .Start | call | Starts the command without waiting. | |
go | .Wait | call | Waits for something to finish (a command or a WaitGroup). | | concurrency
go | .Add | call | Adds to a count (a WaitGroup) or a collection. | |
go | .Done | call | Says one piece of work has finished. | | concurrency

# ---------------------------------------------------------------- Windows Go
go | syscall.NewLazyDLL | call | Prepares to use a Windows DLL (a library of Windows functions) such as kernel32.dll or user32.dll. The DLL is only loaded the first time one of its functions is used. | This is how standard-library Go reaches parts of Windows that Go has no package for, with no C compiler needed. | windows-go
go | windows.NewLazySystemDLL | call | Prepares to use a Windows DLL, loading it only from the Windows system folder. | Safer than NewLazyDLL: a fake DLL placed next to the program can't be loaded instead (DLL hijacking). | windows-go
go | .NewProc | call | Finds one function inside the DLL by its exact name. | The W at the end of names like MessageBoxW means the wide (UTF-16) text version, which supports every language. | windows-go
go | .Call | call | Calls the Windows function, passing each argument as a uintptr (a plain number). It gives back the result, a second result and the Windows error. | Windows only sees numbers, so text and structures must be passed as pointers. Check the first result: many functions return 0 when they fail. | windows-go
go | .Find | call | Checks the DLL function really exists, returning an error if not. | Useful when a function is only on newer versions of Windows. | windows-go
go | syscall.UTF16PtrFromString,windows.UTF16PtrFromString | call | Turns Go text into the UTF-16 text Windows expects and gives a pointer to it. | Windows 'W' functions use UTF-16; Go strings are UTF-8, so they must be converted. | windows-go
go | syscall.StringToUTF16Ptr | call | Turns Go text into a pointer to Windows UTF-16 text (older version; panics on text with a zero byte). | | windows-go
go | syscall.UTF16ToString,windows.UTF16ToString | call | Turns UTF-16 text received from Windows back into a Go string. | | windows-go
go | syscall.UTF16FromString | call | Turns Go text into a UTF-16 slice for Windows. | | windows-go
go | unsafe.Pointer | call | Turns a pointer into a raw memory address with no type. | Needed to pass Go data (buffers, structures) to Windows functions. Go's safety checks don't apply, so sizes must be exactly right. | windows-go memory
go | unsafe.Sizeof | call | Gives the size in bytes of a value. | Many Windows structures must have their own size written in their first field. | windows-go memory
go | uintptr | call | Converts a value to a plain number the size of a pointer. | Windows API calls take every argument as a uintptr. Convert pointers inside the Call(...) itself so the garbage collector can't move the memory first. | windows-go
go | syscall.Handle,windows.Handle | call | A Windows handle: the number Windows gives back to identify something it opened (a file, window, process or device). | Handles must be closed with CloseHandle when finished. | windows-go
go | syscall.CloseHandle,windows.CloseHandle | call | Closes a Windows handle. | Frees the file, device or process for others. | windows-go
go | syscall.CreateFile,windows.CreateFile | call | Opens a file, drive or device (such as a COM port or \\.\PhysicalDrive1) through Windows directly. | Lets a program set exact sharing and access options that os.Open doesn't offer. | windows-go files
go | syscall.ReadFile,windows.ReadFile,syscall.WriteFile,windows.WriteFile | call | Reads or writes through a Windows handle. | | windows-go files
go | windows.DeviceIoControl | call | Sends a control code to a device driver: lock a volume, get a disk's size, eject a drive… | Used by tools that write disk images or talk to hardware. | windows-go hardware
go | syscall.GetLastError,windows.GetLastError | call | Gets the error code of the last Windows call that failed. | | windows-go errors
go | syscall.SysProcAttr | call | Windows-specific options for starting another program, such as HideWindow or CreationFlags. | CreationFlags 0x08000000 (CREATE_NO_WINDOW) stops a console window flashing up. | windows-go
go | windows.ShellExecute | call | Asks Windows to open a file, folder or web address with its usual program (or 'runas' to ask for admin rights). | | windows-go
go | windows.GetCurrentProcess,windows.GetCurrentProcessId | call | Gets the running program's own process handle or ID. | | windows-go
go | windows.OpenProcessToken | call | Opens the security token of a process, to check things like administrator rights. | | windows-go
go | windows.KnownFolderPath | call | Gets a special Windows folder, such as Documents or Downloads. | | windows-go files
go | windows.LoadLibrary,syscall.LoadLibrary | call | Loads a DLL straight away. | | windows-go
go | windows.GetProcAddress,syscall.GetProcAddress | call | Finds a function's address inside a loaded DLL. | | windows-go
go | registry.OpenKey | call | Opens a key in the Windows registry. | The registry is Windows' database of settings. | windows-go
go | .GetStringValue,.GetIntegerValue | call | Reads a value from a registry key. | | windows-go
go | .SetStringValue,.SetDWordValue | call | Writes a value into a registry key. | | windows-go
go | registry.LOCAL_MACHINE,registry.CURRENT_USER | prop | A main registry branch (HKLM: whole computer; HKCU: this user). | | windows-go
go | svc.Run | call | Runs the program as a Windows service, handing control to the Service Control Manager. | Services start with Windows and run without anyone logged in. | windows-go
go | windows.SetConsoleTitle | call | Changes the console window's title. | | windows-go
`)

var _ = addAPI(`
# ---------------------------------------------------------------- C / C++
c,cpp | printf | print | Prints text built from a template: %d is a whole number, %f a decimal, %s text, \n a new line. | | output
c,cpp | scanf | input | Reads typed input into variables (note the & in front of each). | scanf is easy to misuse: it stops at spaces and doesn't check sizes. | input
c,cpp | puts | print | Prints text and a new line. | | output
c,cpp | putchar,getchar | call | Writes (or reads) a single character. | |
c,cpp | fgets | input | Reads a line of text safely, never more than the buffer can hold. | Safer than gets or scanf("%s"). | input
c,cpp | malloc | call | Asks for a block of memory of a given size. | Every malloc needs a matching free, or memory leaks. | memory
c,cpp | calloc | call | Asks for memory and fills it with zeros. | | memory
c,cpp | realloc | call | Makes a block of memory bigger or smaller. | | memory
c,cpp | free | call | Gives memory from malloc back. | Using memory after freeing it is a classic crash. | memory
c,cpp | sizeof | call | Gives the size of a type or variable in bytes. | | memory
c,cpp | strlen | call | Counts the characters in a text (up to its zero byte). | | text
c,cpp | strcpy,strncpy | call | Copies text into a buffer. | The buffer must be big enough; strncpy limits the length. | text
c,cpp | strcmp | call | Compares two texts: 0 means they're the same. | Don't compare C strings with ==. | text
c,cpp | strcat | call | Adds one text to the end of another. | | text
c,cpp | memcpy | call | Copies a block of bytes from one place to another. | | memory
c,cpp | memset | call | Fills a block of memory with one value, often zero. | | memory
c,cpp | fopen | call | Opens a file ("r" read, "w" write, "a" add to the end). | It gives NULL if the file can't be opened, so check it. | files
c,cpp | fclose | call | Closes a file. | | files
c,cpp | fprintf | print | Prints formatted text into a file. | | output files
c,cpp | fread,fwrite | call | Reads or writes blocks of bytes in a file. | | files
c,cpp | exit | call | Ends the program with an exit code. | |
c,cpp | rand | call | Gives a pseudo-random whole number. | rand() % n gives 0 to n-1. Seed it once with srand(time(NULL)). | random
c,cpp | srand | call | Seeds the random number generator. | Usually srand(time(NULL)), so each run differs. | random
c,cpp | time | call | Gets the current time in seconds. | | timers
c,cpp | atoi,atof | call | Turns text into a number. | | text
c,cpp | abs,sqrt,pow,sin,cos,atan2,floor | call | A maths function. | atan2(dy, dx) gives the angle towards something; sin and cos turn an angle into movement. | maths
cpp | std::cout,cout | print | The console output: values sent with << are printed. | | output
cpp | std::cin,cin | input | The console input: >> reads typed values. | | input
cpp | std::endl,endl | print | A new line (and flushes the output). | | output
cpp | std::vector,vector | call | A list that grows as you add to it. | | list
cpp | .push_back,.emplace_back | call | Adds an item to the end of the vector. | | list
cpp | .size | call | Gives how many items there are. | | list
cpp | std::make_unique,std::make_shared | call | Makes an object owned by a smart pointer, which deletes it automatically. | No need for delete. | memory
cpp | std::sort | call | Sorts a range of items. | | list
cpp | std::to_string | call | Turns a number into text. | | text
cpp | std::thread | call | Starts a new thread running a function. | | concurrency
cpp | std::this_thread::sleep_for | call | Pauses this thread for a while. | | timers
cpp | std::chrono::steady_clock::now | call | Reads a clock that never jumps, for measuring time. | Used to work out delta time between frames. | timers
c,cpp | MessageBoxA,MessageBoxW,MessageBox | call | Shows a Windows message box. | | windows-go
c,cpp | WinMain,wWinMain | call | The starting point of a Windows desktop program (instead of main). | | windows-go
c,cpp | CreateWindowEx,CreateWindowExW | call | Creates a Windows window. | | windows-go rendering
c,cpp | GetMessage,GetMessageW,PeekMessage,PeekMessageW | call | Gets the next message (click, key, close…) from Windows' queue. | GetMessage waits; PeekMessage doesn't, which games use so they can keep drawing. | windows-go input game-loop
c,cpp | TranslateMessage,DispatchMessage,DispatchMessageW | call | Passes the message on to the window's message handler. | | windows-go
c,cpp | GetAsyncKeyState | call | Says whether a key is held down right now. | | windows-go input
c,cpp | Sleep | call | Pauses for a number of milliseconds (Windows). | | timers
# Arduino
cpp | pinMode | call | Sets a pin as an INPUT or an OUTPUT. | Pins must be set up before use, usually in setup(). | hardware
cpp | digitalWrite | call | Turns a pin on (HIGH) or off (LOW). | Lights an LED, switches a relay. | hardware
cpp | digitalRead | call | Reads whether a pin is HIGH or LOW. | Reads buttons and switches. | hardware input
cpp | analogRead | call | Reads a voltage on an analogue pin as a number from 0 to 1023 (on most boards). | Reads potentiometers, light sensors and joysticks. | hardware input
cpp | analogWrite | call | Sets a PWM pin from 0 to 255, for dimming LEDs or motor speed. | | hardware
cpp | delay | call | Pauses for a number of milliseconds. | Nothing else happens during a delay, so programs that must react use millis() instead. | timers
cpp | millis | call | Gives the milliseconds since the board started. | Used for timers that don't block: if (millis() - last > 1000) … | timers
cpp | Serial.begin | call | Starts the serial (USB) connection at a speed such as 9600 or 115200 baud. | The Serial Monitor must use the same speed. | hardware
cpp | Serial.println,Serial.print | print | Sends text to the Serial Monitor. | The usual way to see what an Arduino is doing. | output
cpp | Serial.available | call | Says how many bytes have arrived over serial. | | input
cpp | Serial.read | input | Reads one byte that arrived over serial. | | input
cpp | tone | call | Plays a beep of a frequency on a pin (with a buzzer). | | sound

# ---------------------------------------------------------------- C#
csharp | Console.WriteLine | print | Prints a line of text in the console. | | output
csharp | Console.Write | print | Prints text without a new line. | | output
csharp | Console.ReadLine | input | Waits for the user to type a line and press Enter, and gives it back as text. | | input
csharp | Console.ReadKey | input | Waits for a single key press. | | input
csharp | Math.Max,Math.Min,Math.Abs,Math.Sqrt,Math.Round | call | A maths helper. | | maths
csharp | int.Parse,Convert.ToInt32 | call | Turns text into a whole number (an error if it isn't one). | int.TryParse is the safe version. | text
csharp | int.TryParse | call | Tries to turn text into a number, answering true or false instead of crashing. | | text
csharp | string.Format | call | Builds text from a template with {0}, {1} placeholders. | $"…{x}…" does the same more neatly. | text
csharp | File.ReadAllText,File.WriteAllText | call | Reads or writes a whole text file in one go. | | files
csharp | MessageBox.Show | call | Shows a Windows message box. | | windows-go
csharp | Task.Delay | call | Waits for a while without blocking the program (use with await). | | timers async
csharp | .Add | call | Adds an item to the list. | | list
csharp | .Remove,.RemoveAt | call | Removes an item from the list. | | list
csharp | .Contains | call | Answers whether the list or text contains something. | | list
csharp | .Count | prop | How many items are in the list. | | list
csharp | .Length | prop | How long the array or text is. | | list
csharp | .ToString | call | Turns the value into text. | | text

# ---------------------------------------------------------------- Java / Kotlin / Swift
java | System.out.println | print | Prints a line of text. | | output
java | System.out.print | print | Prints text without a new line. | | output
java | System.out.printf | print | Prints formatted text (%d, %s, %.2f). | | output
java | Integer.parseInt | call | Turns text into a whole number. | | text
java | Math.random | call | Gives a random decimal between 0 and 1. | (int)(Math.random() * n) gives 0 to n-1. | random
java | Thread.sleep | call | Pauses for a number of milliseconds. | | timers
java | System.currentTimeMillis,System.nanoTime | call | Reads the clock. | Used for timing and delta time in game loops. | timers
java | .nextInt,.nextLine | input | Reads the next number (or line) typed in. | | input
java | .add | call | Adds an item to the list. | | list
java | .size | call | Gives how many items there are. | | list
java | .get | call | Gets the item at a position (or for a key). | | list
java | .equals | call | Compares two objects' contents. | Compare Strings with .equals, not ==. | text
kotlin,swift | println,print | print | Prints text. | | output
kotlin | listOf,mutableListOf | call | Makes a list (mutableListOf can be changed later). | | list
kotlin | readLine | input | Reads a line typed by the user. | | input
swift | readLine | input | Reads a line typed by the user. | | input

# ---------------------------------------------------------------- JavaScript / TypeScript
javascript,typescript | console.log | print | Prints values to the browser's developer console (F12) or the terminal. | The everyday way to check what a value is. | output
javascript,typescript | console.error,console.warn | print | Prints an error (or warning) message to the console. | | output
javascript,typescript | alert | print | Shows a pop-up message the user must close. | | output
javascript,typescript | prompt | input | Shows a pop-up asking the user to type something. | | input
javascript,typescript | document.getElementById | call | Finds the element on the page with this id. | So the code can read or change it. | dom
javascript,typescript | document.querySelector,document.querySelectorAll | call | Finds the first (or every) element matching a CSS selector. | | dom
javascript,typescript | document.createElement | call | Makes a new page element (not shown until it's added). | | dom
javascript,typescript | .addEventListener | call | Tells the browser to run a function whenever an event happens (a click, a key press…). | Pages and browser games are driven by events. | dom input
javascript,typescript | .removeEventListener | call | Stops listening for that event. | | dom
javascript,typescript | .appendChild,.append | call | Adds an element (or item) to the end. | | dom
javascript,typescript | .textContent,.innerHTML | prop | The text (or HTML) inside an element. | | dom
javascript,typescript | setTimeout | call | Runs a function once after a delay in milliseconds. | | timers
javascript,typescript | setInterval | call | Runs a function again and again every few milliseconds. | Simple games use it, but requestAnimationFrame is smoother. | timers
javascript,typescript | clearInterval,clearTimeout | call | Cancels a timer made earlier. | | timers
javascript,typescript | requestAnimationFrame | call | Asks the browser to run a function just before the next screen refresh (usually 60 times a second). | The standard heartbeat of browser games: the function draws a frame and asks for the next one. | game-loop rendering
javascript,typescript | fetch | call | Downloads something from a web address; gives back a promise (await it). | | network async
javascript,typescript | JSON.parse | call | Turns JSON text into a JavaScript object. | | data
javascript,typescript | JSON.stringify | call | Turns an object into JSON text. | Used to save data or send it to a server. | data
javascript,typescript | Math.random | call | Gives a random decimal from 0 up to (not including) 1. | Math.floor(Math.random() * n) gives a whole number from 0 to n-1. | random
javascript,typescript | Math.floor,Math.round,Math.ceil | call | Rounds a number (down, nearest, up). | | maths
javascript,typescript | Math.min,Math.max | call | Gives the smallest (or biggest) of the values. | Often used to keep a value within limits. | maths
javascript,typescript | Math.sqrt,Math.hypot | call | Square root / length of a line. | Math.hypot(dx, dy) is the distance between two points. | maths
javascript,typescript | Math.sin,Math.cos,Math.atan2 | call | Trigonometry: turn angles into directions and back. | | maths
javascript,typescript | Math.abs | call | Removes a minus sign. | | maths
javascript,typescript | Date.now,performance.now | call | Reads the time in milliseconds. | The difference between two readings gives delta time. | timers
javascript,typescript | localStorage.setItem,localStorage.getItem | call | Saves (or loads) a small piece of text in the browser, kept between visits. | Browser games keep high scores and save games this way. | saving
javascript,typescript | parseInt,parseFloat,Number | call | Turns text into a number. | | text
javascript,typescript | .push | call | Adds an item to the end of the list (array). | | list
javascript,typescript | .pop,.shift | call | Takes the last (or first) item off the list. | | list
javascript,typescript | .splice | call | Removes (or inserts) items in the middle of a list. | Used to delete dead enemies or used bullets. | list
javascript,typescript | .map | call | Makes a new list by changing every item with a function. | | list
javascript,typescript | .filter | call | Makes a new list of only the items a test says yes to. | Often used to drop finished bullets or dead enemies. | list
javascript,typescript | .forEach | call | Runs a function for every item in the list. | | list loop
javascript,typescript | .reduce | call | Combines all the items into one value, such as a total. | | list
javascript,typescript | .includes | call | Answers whether the list or text contains something. | | list
javascript,typescript | .find,.findIndex | call | Finds the first item that passes a test. | | list
javascript,typescript | .length | prop | How many items (or characters) there are. | | list
javascript,typescript | .getContext | call | Gets the canvas's drawing tools ('2d' for flat drawing). | Everything drawn on a canvas goes through this context. | rendering
javascript,typescript | .fillRect | call | Draws a filled rectangle. | | rendering
javascript,typescript | .clearRect | call | Clears a rectangle (usually the whole canvas) before drawing the next frame. | Without clearing, old frames smear across the screen. | rendering
javascript,typescript | .drawImage | call | Draws a picture (or part of one, for sprite sheets) on the canvas. | | rendering
javascript,typescript | .fillText | call | Draws text on the canvas. | | rendering
javascript,typescript | .beginPath,.arc,.fill,.stroke,.moveTo,.lineTo | call | Builds and draws a shape on the canvas. | | rendering
javascript,typescript | .fillStyle,.strokeStyle | prop | The colour used for filling (or outlining). | | rendering
javascript,typescript | .play | call | Starts playing a sound or animation. | | sound
javascript,typescript | .preventDefault | call | Stops the browser's normal reaction (such as scrolling when arrow keys are pressed). | | input
javascript,typescript | Object.keys,Object.values,Object.entries | call | Lists an object's names, values, or both. | | list
javascript,typescript | Array.from | call | Makes a list from something list-like. | | list
javascript,typescript | Promise.all | call | Waits for several promises together. | | async

# ---------------------------------------------------------------- PHP
php | print_r,var_dump | print | Prints a value (and its structure) for debugging. | | output
php | isset | call | Answers whether a variable exists and isn't null. | | variable
php | empty | call | Answers whether a value is empty (null, "", 0 or an empty array). | |
php | count | call | Counts the items in an array. | | list
php | strlen | call | Counts the characters in a text. | | text
php | explode,implode | call | Splits text into an array (or joins an array into text). | | text
php | array_push | call | Adds items to the end of an array. | | list
php | json_encode,json_decode | call | Turns data into JSON text and back. | | data
php | htmlspecialchars | call | Makes text safe to show in a page (so < and > can't inject HTML). | Protects against cross-site scripting (XSS). | security
php | header | call | Sends an HTTP header, such as a redirect. | Must happen before any output. | network
php | session_start | call | Starts or resumes the visitor's session, so $_SESSION remembers them between pages. | | network
php | $_GET,$_POST | prop | The values sent to the page in the web address (GET) or a form (POST). | Never trust them without checking. | input

# ---------------------------------------------------------------- Python
python,gdscript | print | print | Shows its values as text, separated by spaces, ending with a new line. | The simplest way to see output or check values while testing. | output
python | input | input | Shows a prompt and waits for the user to type a line, giving it back as text. | Use int(...) or float(...) on it to get a number. | input
python | len | call | Gives how many items (or characters) there are. | | list
python | range | call | Gives a run of numbers, such as range(5) = 0, 1, 2, 3, 4. | | loop
python | int,float | call | Turns a value (often text) into a whole or decimal number. | | text
python | str | call | Turns a value into text. | | text
python | list,dict,set,tuple | call | Makes a list, dictionary, set or tuple. | | list
python | open | call | Opens a file ("r" read, "w" write, "a" append). | Best used with with, so the file closes itself. | files
python | enumerate | call | Gives each item together with its position. | | loop
python | zip | call | Pairs up items from two or more lists. | | list
python | sorted | call | Gives a sorted copy of a list. | | list
python | sum,min,max | call | Adds up (or finds the smallest/largest of) the values. | | maths
python | abs,round | call | Removes the minus sign / rounds a number. | | maths
python | isinstance,type | call | Checks (or gives) what type a value is. | |
python | .append | call | Adds an item to the end of the list. | | list
python | .remove,.pop | call | Takes an item out of the list. | | list
python | .split | call | Cuts text into a list of words (or pieces). | | text
python | .join | call | Joins a list of texts into one, with the text in front as the separator. | | text
python | .strip | call | Removes spaces and new lines from both ends of text. | | text
python | .format | call | Fills {} placeholders in a text. | f-strings (f"…{x}…") do the same more neatly. | text
python | .lower,.upper | call | Changes text to lower (or upper) case. | | text
python | .get | call | Gets a value from a dictionary, or a default if the key is missing. | | list
python | .items,.keys,.values | call | Gives a dictionary's pairs, keys or values to loop over. | | list
python | random.randint | call | Gives a random whole number between two numbers, including both. | | random
python | random.choice | call | Picks a random item from a list. | | random
python | random.random | call | Gives a random decimal between 0 and 1. | | random
python | random.shuffle | call | Mixes a list into a random order. | | random
python | time.sleep | call | Pauses the program for a number of seconds. | | timers
python | time.time,time.perf_counter | call | Reads the clock in seconds. | Subtract two readings to measure time. | timers
python | os.path.join | call | Joins parts of a file path correctly for this system. | | files
python | os.path.exists | call | Answers whether a file or folder exists. | | files
python | sys.exit | call | Ends the program. | |
python | math.sqrt,math.sin,math.cos,math.atan2,math.floor,math.pi | call | A maths function or constant. | | maths
python | json.load,json.dump,json.loads,json.dumps | call | Reads or writes JSON data. | Handy for saving settings and game progress. | data saving
python | requests.get,requests.post | call | Sends a web request and gives back the response. | | network
python | GPIO.setup,GPIO.output,GPIO.input | call | Sets up, switches or reads a Raspberry Pi pin. | | hardware
python | machine.Pin | call | Sets up a pin on a MicroPython board. | | hardware
python | turtle.forward,turtle.left,turtle.right,forward,left,right | call | Moves or turns the drawing turtle. | | rendering

# ---------------------------------------------------------------- Ruby
ruby | puts | print | Prints text with a new line. | | output
ruby | gets | input | Reads a line the user types (with .chomp to drop the new line). | | input
ruby | p | print | Prints a value in a debugging-friendly way. | | output
ruby | .each | call | Runs the block once for every item. | | loop
ruby | .map | call | Makes a new list by changing every item. | | list
ruby | .new | call | Makes a new object of this class (calling initialize). | | class
ruby | rand | call | Gives a random number. | | random
ruby | raise | error | Raises an error. | | errors
`)

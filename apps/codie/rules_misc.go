// Codie — rules_misc.go: shells (Bash, PowerShell, Windows batch), SQL,
// HTML and CSS.
// Part of the free nrmluvdev maker suite. Created by nrmluvdev.
package codie

import (
	"regexp"
	"strconv"
	"strings"
)

var psVerbs = map[string]string{
	"Get": "gets (reads)", "Set": "changes", "New": "creates", "Remove": "deletes", "Start": "starts", "Stop": "stops", "Write": "writes out",
	"Read": "reads", "Invoke": "runs", "Test": "checks", "Import": "loads", "Export": "saves", "Add": "adds", "Copy": "copies", "Move": "moves",
	"Out": "sends output to", "Select": "picks", "Where": "filters", "ForEach": "does something for each", "Sort": "sorts", "Measure": "counts or measures",
	"ConvertTo": "converts to", "ConvertFrom": "converts from", "Clear": "clears", "Enable": "turns on", "Disable": "turns off", "Install": "installs",
	"Uninstall": "uninstalls", "Restart": "restarts", "Wait": "waits for", "Join": "joins", "Split": "splits", "Resolve": "works out", "Format": "formats",
	"Update": "updates", "Rename": "renames", "Register": "registers", "Show": "shows", "Expand": "unpacks", "Compress": "zips",
}

var reTag = regexp.MustCompile(`^<(/?)([A-Za-z][\w-]*)([^>]*?)(/?)>(.*)$`)

var htmlTags = map[string]string{
	"html": "the whole page", "head": "the page's head: title, settings and links that aren't shown directly", "body": "everything visible on the page",
	"title": "the page title shown on the browser tab", "meta": "a setting for the browser or search engines", "link": "a link to another file, usually a stylesheet (CSS)",
	"script": "JavaScript code (or a link to a .js file)", "style": "CSS styling rules", "div": "a box that groups other things", "span": "a small piece of text to style",
	"p": "a paragraph", "h1": "the main heading", "h2": "a section heading", "h3": "a smaller heading", "a": "a link", "img": "a picture", "button": "a button",
	"input": "a box the user can type in or a tick box", "form": "a form that sends what the user filled in", "ul": "a bulleted list", "ol": "a numbered list",
	"li": "an item in a list", "table": "a table", "tr": "a table row", "td": "a table cell", "th": "a table heading cell", "canvas": "a drawing area, where browser games draw their graphics",
	"audio": "a sound player", "video": "a video player", "section": "a section of the page", "header": "the top part of the page or section", "footer": "the bottom part",
	"nav": "the menu (navigation) links", "main": "the main content", "label": "a label for a form box", "select": "a drop-down list", "option": "a choice in a drop-down",
	"textarea": "a big text box", "br": "a line break", "hr": "a dividing line", "iframe": "another page shown inside this one", "svg": "a vector drawing", "strong": "important (bold) text", "em": "emphasised (italic) text",
}

var cssProps = map[string]string{
	"color": "the text colour", "background": "the background", "background-color": "the background colour", "font-size": "the text size", "font-family": "the font",
	"font-weight": "how bold the text is", "margin": "the space outside the box", "padding": "the space inside the box, around its content", "border": "the outline of the box",
	"border-radius": "how rounded the corners are", "width": "the width", "height": "the height", "display": "how the box is laid out (block, inline, flex, grid, none = hidden)",
	"position": "how the box is positioned (relative, absolute, fixed)", "top": "the distance from the top", "left": "the distance from the left", "right": "the distance from the right",
	"bottom": "the distance from the bottom", "flex": "how much the item grows in a flex row", "justify-content": "how items spread along the row", "align-items": "how items line up across the row",
	"gap": "the space between items", "grid-template-columns": "the columns of a grid", "text-align": "text alignment (left, centre, right)", "opacity": "how see-through it is (0–1)",
	"transform": "moves, rotates or scales it", "transition": "animates changes smoothly", "animation": "plays a @keyframes animation", "cursor": "the mouse pointer shape",
	"z-index": "which boxes sit on top", "overflow": "what happens to content that doesn't fit", "box-shadow": "a shadow around the box", "line-height": "the space between lines of text",
	"max-width": "the largest width allowed", "min-height": "the smallest height allowed", "image-rendering": "how pictures are scaled (pixelated keeps pixel art sharp)",
}

func init() {
	// ------------------------------------------------------------ Bash
	R("bash", `^({I})=(.*)$`, "assign", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, "", m[1], "", m[2])
		if strings.HasPrefix(m[2], "$(") {
			p.What = "Runs " + code(unhide(m[2], x.strs)) + " and keeps what it prints in " + code(m[1]) + "."
		}
		return p.What, p.Why + " In shell scripts there must be no spaces around the =."
	}).setsVar(1)
	R("bash", `^export\s+({I})=(.*)$`, "assign", "Sets the environment variable $1 to $2 and shares it with programs this script starts.", "", "variable").setsVar(1)
	R("bash", `^(?:local|readonly|declare(?:\s+-\w+)?)\s+({I})(?:=(.*))?$`, "var", "Creates the variable $1 (set to $2).", "local keeps it inside the function; readonly stops it changing.", "variable").setsVar(1)
	R("bash", `^if\s+(.+?)(?:\s*;\s*then)?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		c := strings.TrimSpace(m[1])
		if w := shellCond(x, strings.Trim(c, "[] ")); w != "" {
			return "If-check: " + w + ".", "In shell scripts, [ … ] runs a test; the lines up to fi run when it passes."
		}
		return "If-check: " + checks(x, strings.Trim(c, "[] ")) + ".", "In shell scripts, [ … ] runs a test; the lines up to fi run when it passes."
	}).open("if", 0)
	R("bash", `^elif\s+(.+?)(?:\s*;\s*then)?$`, "if", "Otherwise, if $1…", "", "if")
	R("bash", `^then$`, "if", "Begins the lines that run when the check passes.", "", "if")
	R("bash", `^else$`, "if", "Otherwise, run the lines below (up to fi).", "", "if")
	R("bash", `^fi$`, "close", "", "").close()
	R("bash", `^for\s+({I})\s+in\s+(.+?)(?:\s*;\s*do)?$`, "loop", "Goes through each item in $2, calling it $1, and runs the lines up to done.", "Lists in shell scripts are words separated by spaces, or file patterns like *.txt.", "loop").open("loop", 1).setsVar(1)
	R("bash", `^for\s*\(\((.+)\)\)(?:\s*;\s*do)?$`, "loop", "A counting loop: $1.", "", "loop").open("loop", 0)
	R("bash", `^(while|until)\s+(.+?)(?:\s*;\s*do)?$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		if strings.HasPrefix(strings.TrimSpace(m[2]), "read") {
			return "Reads input one line at a time and runs the loop for each line.", "The usual way to process a file line by line."
		}
		return "Repeats the lines up to done " + m[1] + " " + code(unhide(m[2], x.strs)) + ".", ""
	}).open("loop", 0)
	R("bash", `^do$`, "loop", "Begins the body of the loop.", "", "loop")
	R("bash", `^done(.*)$`, "close", "", "").close()
	R("bash", `^case\s+(.+)\s+in$`, "switch", "Chooses what to do based on $1.", "Each pattern) below is tried in turn; ;; ends a branch.", "switch").open("switch", 0)
	R("bash", `^([^()]+)\)(.*)$`, "case", "If it matches $1, run the following.", "")
	R("bash", `^;;$`, "case", "Ends this branch of the case.", "")
	R("bash", `^esac$`, "close", "", "").close()
	R("bash", `^(?:function\s+)?({I})\s*\(\)\s*\{?$`, "def", "Defines the shell function $1.", "Call it like a command: $1 arg1 arg2. Inside, $1, $2… are its arguments.", "function").open("function", 1)
	R("bash", `^function\s+({I})\s*\{?$`, "def", "Defines the shell function $1.", "", "function").open("function", 1)
	R("bash", `^\}$`, "close", "", "").close()
	R("bash", `^echo\s*(.*)$`, "print", "Prints $1.", "echo shows text in the terminal; $NAME inside double quotes is replaced by the variable's value.", "output")
	R("bash", `^read\s+(?:-r\s+)?(?:-p\s+({S})\s+)?(.+)$`, "input", "Waits for the user to type a line and stores it in $2 (after showing $1).", "", "input").setsVar(2)
	R("bash", `^exit\s*(\d*)$`, "jump", "Ends the script (with exit code $1).", "0 means success; any other number tells whoever ran it that something failed.")
	R("bash", `^set\s+-(\w+)$`, "directive", "", "").with(func(m []string, x *ctx) (string, string) {
		var d []string
		for _, c := range m[1] {
			switch c {
			case 'e':
				d = append(d, "stop at the first failing command")
			case 'u':
				d = append(d, "treat unset variables as errors")
			case 'x':
				d = append(d, "print each command before running it")
			case 'o':
				d = append(d, "an option follows")
			}
		}
		return "Changes how the script runs: " + strings.Join(d, ", ") + ".", "set -e and -u make scripts safer by stopping on mistakes."
	})
	R("bash", `^(sudo\s+)?([\w./-]+)(\s+.*)?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		e := findKey(x, m[2])
		args := strings.TrimSpace(m[3])
		var w, why string
		if e != nil {
			w, why = e.What, e.Why
			x.tag(e.Tags...)
		} else if ln, ok := x.funcs[m[2]]; ok {
			w, why = "Runs the function "+code(m[2])+" from line "+itoa(ln)+".", ""
		} else {
			w, why = "Runs the command "+code(m[2])+".", "A program or script installed on the computer."
		}
		if args != "" {
			w = strings.TrimSuffix(w, ".") + " (with " + code(unhide(args, x.strs)) + ")."
		}
		if m[1] != "" {
			why += " sudo runs it as the administrator (root), so it may ask for your password."
		}
		if strings.Contains(args, "|") {
			why += " The | passes its output into the next command."
		}
		if strings.Contains(args, "&&") {
			why += " && runs the next command only if this one worked."
		}
		if strings.Contains(args, ">") {
			why += " > saves the output to a file (>> adds to the end)."
		}
		return w, strings.TrimSpace(why)
	})

	// ------------------------------------------------------------ PowerShell
	ps := "powershell"
	R(ps, `^\$({I}(?::{I})?(?:\.{I})*)\s*(\+|-|\*|/)?=\s*(.+)$`, "assign", "", "").with(func(m []string, x *ctx) (string, string) {
		p := assignPart(x, "", "$"+m[1], m[2], m[3])
		if strings.HasPrefix(strings.ToLower(m[1]), "env:") {
			p.Why = "$env: variables are environment variables, shared with programs this script starts."
		}
		return p.What, p.Why
	}).setsVar(1)
	R(ps, `^function\s+([\w-]+)\s*(?:\((.*)\))?$`, "def", "Defines the function $1.", "Call it by name, like a cmdlet. Verb-Noun names are PowerShell style.", "function").open("function", 1)
	R(ps, `^param\s*\((.*)\)?$`, "def", "Lists the inputs (parameters) this script or function accepts: $1.", "Callers pass them as -Name value.", "function")
	R(ps, `^if\s*(\(.*)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		c, _, ok := parenGroup(m[1])
		if !ok {
			return "", ""
		}
		x.tag("if")
		return "If-check: " + checks(x, c) + ".", "PowerShell compares with -eq, -ne, -gt and -lt instead of == and >."
	}).open("if", 0)
	R(ps, `^elseif\s*\((.*)\)$`, "if", "Otherwise, if $1…", "", "if").open("if", 0)
	R(ps, `^else$`, "if", "Otherwise, run the block below.", "", "if").open("if", 0)
	R(ps, `^foreach\s*\(\s*\$(\w+)\s+in\s+(.+)\)$`, "loop", "Goes through each item in $2, calling it $$1.", "", "loop").open("loop", 0).setsVar(1)
	R(ps, `^(for|while)\s*\((.*)\)$`, "loop", "A $1 loop: $2.", "", "loop").open("loop", 0)
	R(ps, `^switch\s*(?:-\w+\s*)*\((.*)\)$`, "switch", "Chooses what to do based on $1.", "", "switch").open("switch", 0)
	R(ps, `^try$`, "try", "Tries the block; errors jump to catch.", "Add -ErrorAction Stop to cmdlets so their errors are caught.", "errors").open("try", 0)
	R(ps, `^catch(.*)$`, "try", "Handles an error from the try block. $_ is the error.", "", "errors").open("try", 0)
	R(ps, `^finally$`, "try", "Always runs after try/catch.", "", "errors").open("try", 0)
	R(ps, `^return\s*(.*)$`, "return", "Leaves the function, giving back $1.", "")
	R(ps, `^([A-Za-z]+)-([A-Za-z]+)\b\s*(.*)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		name := m[1] + "-" + m[2]
		args := strings.TrimSpace(m[3])
		var w, why string
		if e := findKey(x, name); e != nil {
			w, why = e.What, e.Why
			x.tag(e.Tags...)
		} else if v, ok := psVerbs[strings.Title(strings.ToLower(m[1]))]; ok {
			w = "Runs the cmdlet " + code(name) + ", which " + v + " " + strings.ToLower(strings.Join(splitWords(m[2]), " ")) + "."
			why = "PowerShell commands are named Verb-Noun, so the name tells you what it does."
		} else {
			return "", ""
		}
		if args != "" {
			w = strings.TrimSuffix(w, ".") + " (with " + code(unhide(clip(args, 70), x.strs)) + ")."
		}
		if strings.Contains(args, "|") {
			why += " The | (pipeline) passes its results as objects to the next command."
		}
		return w, strings.TrimSpace(why)
	})
	R(ps, `^\|\s*(.+)$`, "call", "…passes the results on to $1.", "A pipeline split over lines.")

	// ------------------------------------------------------------ batch
	bt := "batch"
	R(bt, `^@?echo\s+off$`, "directive", "Stops the window from showing each command before running it.", "Almost every batch file starts with @echo off so only real messages appear. The @ hides this line too.")
	R(bt, `^@?echo\.?$`, "print", "Prints an empty line.", "", "output")
	R(bt, `^@?echo\s+(.+)$`, "print", "Prints $1.", "%NAME% inside the text is replaced by the variable's value.", "output")
	R(bt, `^set\s+/p\s+"?(\w+)=(.*?)"?$`, "input", "Shows $2 and waits for the user to type something into %$1%.", "", "input").setsVar(1)
	R(bt, `^set\s+/a\s+"?(.+?)"?$`, "assign", "Does the sum $1 (set /a means arithmetic).", "Without /a, set only stores text.", "variable")
	R(bt, `^set\s+"?(\w+)=(.*?)"?$`, "assign", "Sets the variable %$1% to $2.", "Use it later as %$1%. Spaces around = become part of the name or value, so leave them out.", "variable").setsVar(1)
	R(bt, `^setlocal(.*)$`, "directive", "Keeps variable changes inside this script.", "With EnableDelayedExpansion, !VAR! gives a variable's current value inside loops.")
	R(bt, `^endlocal$`, "directive", "Ends the setlocal section; variables go back to how they were.", "")
	R(bt, `^if\s+(.+)\s+\($`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		return "If-check: " + batchCond(m[1]) + ". If so, run the lines in the brackets below.", ""
	}).open("if", 0)
	R(bt, `^\)\s*else\s*\($`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		if b := x.pop(x.n); b != nil {
			x.push("if", "", x.n, 0)
		}
		return "Otherwise, run the lines in the next brackets.", ""
	})
	R(bt, `^if\s+(.+?)\s+((?:goto|echo|set|call|exit|del|copy|start)\b.*)$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("if")
		return "If-check: " + batchCond(m[1]) + ". If so, run " + code(unhide(m[2], x.strs)) + ".", ""
	})
	R(bt, `^for\s+(/\w\s+)?%%(\w)\s+in\s+\((.+?)\)\s+do\s+(.+)$`, "loop", "", "").with(func(m []string, x *ctx) (string, string) {
		x.tag("loop")
		w := "For each item in " + code(m[3]) + " (called %%" + m[2] + "), runs " + code(unhide(m[4], x.strs)) + "."
		switch strings.TrimSpace(strings.ToLower(m[1])) {
		case "/l":
			w = "Counts %%" + m[2] + " using (start, step, end) = " + code(m[3]) + ", running " + code(unhide(m[4], x.strs)) + " each time."
		case "/f":
			w = "Reads " + code(m[3]) + " line by line (or word by word) into %%" + m[2] + ", running " + code(unhide(m[4], x.strs)) + "."
		case "/d":
			w = "For each folder matching " + code(m[3]) + "…"
		}
		return w, "In a .bat file loop variables use two percent signs (%%i); typed at the prompt they use one."
	})
	R(bt, `^\)$`, "close", "", "").close()
	R(bt, `^:(\w+)$`, "label", "A label called :$1 that goto and call can jump to.", "Labels mark sections of a batch file.")
	R(bt, `^goto\s+:?(\w+)$`, "jump", "", "").with(func(m []string, x *ctx) (string, string) {
		if strings.EqualFold(m[1], "eof") {
			return "Jumps to the end of the file, which ends the script (or the current call).", ""
		}
		x.tag("goto")
		return "Jumps to the label :" + m[1] + ".", "Batch files use goto for loops and menus."
	})
	R(bt, `^call\s+:(\w+)(.*)$`, "call", "Runs the section :$1 like a subroutine and comes back here when it reaches exit /b.", "Inside it, %1 %2 are the extra words given ($2).", "subroutine")
	R(bt, `^call\s+(.+)$`, "call", "Runs the other batch file $1 and comes back when it finishes.", "Without call, the first batch file would stop.")
	R(bt, `^exit(\s+/b)?\s*(\d*)$`, "jump", "", "").with(func(m []string, x *ctx) (string, string) {
		if m[1] != "" {
			return "Leaves this script (or subroutine) with exit code " + code(m[2]) + ", without closing the window.", "0 means success."
		}
		return "Closes the command window.", ""
	})
	R(bt, `^(@?\w+(?:\.exe)?)\b\s*(.*)$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		e := findKey(x, strings.TrimPrefix(strings.ToLower(m[1]), "@"))
		if e == nil {
			return "Runs the command or program " + code(m[1]) + " " + code(unhide(m[2], x.strs)) + ".", ""
		}
		x.tag(e.Tags...)
		w := e.What
		if a := strings.TrimSpace(m[2]); a != "" {
			w = strings.TrimSuffix(w, ".") + " (" + code(unhide(a, x.strs)) + ")."
		}
		return w, e.Why
	})

	// ------------------------------------------------------------ SQL
	q := "sql"
	R(q, `^SELECT\s+(DISTINCT\s+)?(.+?)(?:\s+FROM\s+(.+?))?(?:\s+WHERE\s+(.+?))?;?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		cols := strings.TrimSpace(m[2])
		w := "Asks the database for " + code(cols)
		if cols == "*" {
			w = "Asks the database for every column"
		}
		if m[3] != "" {
			w += " from the table " + code(m[3])
		}
		if m[4] != "" {
			w += ", only where " + condWords(x, m[4])
		}
		why := "SELECT reads data without changing it; the answer comes back as rows and columns."
		if m[1] != "" {
			why += " DISTINCT removes duplicate rows."
		}
		return w + ".", why
	})
	R(q, `^FROM\s+(.+?);?$`, "call", "…from the table $1.", "")
	R(q, `^WHERE\s+(.+?);?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		return "Keeps only the rows where " + condWords(x, m[1]) + ".", "WHERE is the filter: rows that don't match are left out."
	})
	R(q, `^(AND|OR)\s+(.+?);?$`, "if", "", "").with(func(m []string, x *ctx) (string, string) {
		return "…" + strings.ToLower(m[1]) + " " + condWords(x, m[2]) + ".", ""
	})
	R(q, `^((?:INNER|LEFT|RIGHT|FULL)\s+)?(?:OUTER\s+)?JOIN\s+(.+?)(?:\s+ON\s+(.+?))?;?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Joins in the table " + code(m[2])
		if m[3] != "" {
			w += ", matching rows where " + condWords(x, m[3])
		}
		why := "A JOIN combines rows from two tables that belong together, such as orders and the customers who made them."
		if strings.HasPrefix(strings.ToUpper(m[1]), "LEFT") {
			why += " LEFT JOIN keeps every row of the first table even if there's no match."
		}
		return w + ".", why
	})
	R(q, `^ON\s+(.+?);?$`, "if", "…matching rows where $1.", "")
	R(q, `^GROUP\s+BY\s+(.+?);?$`, "call", "Groups rows that share the same $1, so COUNT, SUM or AVG work per group.", "")
	R(q, `^HAVING\s+(.+?);?$`, "if", "Keeps only the groups where $1.", "HAVING filters groups; WHERE filters single rows.")
	R(q, `^ORDER\s+BY\s+(.+?);?$`, "call", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "Sorts the results by " + code(m[1])
		if strings.Contains(strings.ToUpper(m[1]), "DESC") {
			w += " (largest first)"
		}
		return w + ".", "Without ORDER BY the order of rows isn't guaranteed."
	})
	R(q, `^LIMIT\s+(\d+)(.*)$`, "call", "Returns at most $1 rows.", "Handy for 'top 10' lists such as high scores.")
	R(q, `^INSERT\s+INTO\s+(\w+)\s*(\(.*?\))?(?:\s*VALUES\s*(.+?))?;?$`, "assign", "Adds a new row to the table $1 (columns $2, values $3).", "INSERT adds data; it never changes existing rows.")
	R(q, `^VALUES\s*(.+?);?$`, "assign", "…with the values $1.", "")
	R(q, `^UPDATE\s+(\w+)(?:\s+SET\s+(.+?))?;?$`, "assign", "Changes rows in the table $1 ($2).", "Always use a WHERE with UPDATE, or every row changes!")
	R(q, `^SET\s+(.+?);?$`, "assign", "…setting $1.", "")
	R(q, `^DELETE\s+FROM\s+(\w+)(.*)$`, "call", "Deletes rows from the table $1 $2.", "Without WHERE, DELETE removes every row in the table.")
	R(q, `^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([\w.]+)\s*\(?$`, "class", "Creates a new table called $1, with the columns listed below.", "A table is like a spreadsheet: each row is one record and each column one kind of value.").open("table", 1)
	R(q, `^(PRIMARY\s+KEY|FOREIGN\s+KEY|UNIQUE|CONSTRAINT)\b(.*)$`, "var", "Adds a rule: $1 $2.", "A primary key identifies each row uniquely; a foreign key links to another table.")
	R(q, `^(\w+)\s+(INT|INTEGER|BIGINT|SMALLINT|TEXT|VARCHAR|CHAR|REAL|FLOAT|DOUBLE|DECIMAL|NUMERIC|BOOLEAN|BOOL|DATE|DATETIME|TIMESTAMP|BLOB|SERIAL)\b(.*?),?$`, "var", "", "").with(func(m []string, x *ctx) (string, string) {
		w := "A column " + code(m[1]) + " holding " + strings.ToLower(m[2]) + " values"
		r := strings.ToUpper(m[3])
		var extra []string
		if strings.Contains(r, "PRIMARY KEY") {
			extra = append(extra, "it identifies each row")
		}
		if strings.Contains(r, "NOT NULL") {
			extra = append(extra, "it must always have a value")
		}
		if strings.Contains(r, "AUTOINCREMENT") || strings.Contains(r, "AUTO_INCREMENT") || strings.Contains(r, "SERIAL") {
			extra = append(extra, "it counts up by itself")
		}
		if strings.Contains(r, "DEFAULT") {
			extra = append(extra, "it has a default value")
		}
		if len(extra) > 0 {
			w += "; " + strings.Join(extra, ", ")
		}
		return w + ".", ""
	})
	R(q, `^\);?$`, "close", "", "").close()
	R(q, `^DROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?(\w+);?$`, "call", "Deletes the table $1 and all its data.", "There's no undo!")
	R(q, `^ALTER\s+TABLE\s+(\w+)\s+(.+?);?$`, "call", "Changes the table $1: $2.", "")
	R(q, `^CREATE\s+(UNIQUE\s+)?INDEX\s+(\w+)\s+ON\s+(.+?);?$`, "call", "Creates the index $2 on $3.", "An index makes searching that column much faster, like a book's index.")
	R(q, `^(BEGIN(?:\s+TRANSACTION)?|START\s+TRANSACTION)$;?`, "try", "Starts a transaction: the changes below happen all together or not at all.", "")
	R(q, `^COMMIT;?$`, "try", "Saves all the changes in the transaction.", "")
	R(q, `^ROLLBACK;?$`, "try", "Cancels all the changes in the transaction.", "")
	R(q, `^UNION(\s+ALL)?$`, "call", "Adds the results of the next query to this one.", "")
	R(q, `^([\w.*]+(?:\s+AS\s+\w+)?\s*,?\s*)+$`, "call", "More columns or names for the list above: $0.", "")

	// ------------------------------------------------------------ HTML
	R("html", `^<!DOCTYPE\s+html>$`, "directive", "Tells the browser this is a modern HTML5 page.", "Without it, browsers fall back to an old 'quirks' mode.")
	R("html", `^<.*$`, "other", "", "").with(func(m []string, x *ctx) (string, string) {
		t := reTag.FindStringSubmatch(x.cur)
		if t == nil {
			return "", ""
		}
		name := strings.ToLower(t[2])
		d := htmlTags[name]
		if d == "" {
			d = "the " + code(name) + " element"
		}
		if t[1] == "/" {
			return "Ends " + d + ".", ""
		}
		w := "Starts " + d + "."
		if strings.Contains(strings.ToLower(t[5]), "</"+name+">") {
			inner := t[5][:strings.Index(strings.ToLower(t[5]), "</"+name+">")]
			w = "Adds " + d
			if strings.TrimSpace(inner) != "" {
				w += " containing " + code(unhide(inner, x.strs))
			}
			w += "."
		}
		attrs := t[3]
		var notes []string
		for _, a := range []struct{ k, d string }{{"id=", "an id, so CSS and JavaScript can find it"}, {"class=", "a class, used by CSS for styling"}, {"src=", "the file to load"}, {"href=", "where it links to"}, {"onclick=", "JavaScript that runs when it's clicked"}, {"style=", "styling written inline"}, {"width=", "a width"}, {"type=", "its type"}} {
			if strings.Contains(attrs, a.k) {
				notes = append(notes, a.d)
			}
		}
		why := ""
		if len(notes) > 0 {
			why = "It has " + strings.Join(notes, ", ") + "."
		}
		if name == "canvas" {
			x.tag("rendering")
			why += " Browser games get the canvas with getContext('2d') and draw every frame."
		}
		return w, why
	})

	// ------------------------------------------------------------ CSS
	R("css", `^@media\s+(.+)$`, "if", "Styles inside only apply when $1 (for example on small screens).", "This is how pages adapt to phones and tablets.", "if").open("block", 0)
	R("css", `^@keyframes\s+([\w-]+)$`, "def", "Defines the animation $1.", "Steps such as from, to or 50% say how things look at each point.").open("block", 0)
	R("css", `^@import\s+(.+?);?$`, "import", "Loads another stylesheet: $1.", "", "import")
	R("css", `^:root$`, "class", "Styles for the whole page; often holds --variables (custom properties).", "").open("block", 0)
	R("css", `^([\w-]+)\s*:\s*(.+?);?$`, "assign", "", "").with(func(m []string, x *ctx) (string, string) {
		if x.opensBrace() {
			return "", ""
		}
		p := strings.ToLower(m[1])
		d := cssProps[p]
		if strings.HasPrefix(p, "--") {
			return "Sets the CSS variable " + code(m[1]) + " to " + code(m[2]) + ".", "Use it elsewhere with var(" + m[1] + ") to change a colour or size in one place."
		}
		if d == "" {
			return "Sets " + code(m[1]) + " to " + code(m[2]) + ".", ""
		}
		return "Sets " + d + " to " + code(m[2]) + ".", ""
	})
	R("css", `^(.+)$`, "class", "", "").with(func(m []string, x *ctx) (string, string) {
		if !x.opensBrace() {
			return "", ""
		}
		sel := strings.TrimSpace(m[1])
		var parts []string
		for _, s := range strings.Split(sel, ",") {
			s = strings.TrimSpace(s)
			switch {
			case strings.HasPrefix(s, "."):
				parts = append(parts, "things with the class "+code(s[1:]))
			case strings.HasPrefix(s, "#"):
				parts = append(parts, "the element with the id "+code(s[1:]))
			default:
				parts = append(parts, code(s))
			}
		}
		why := "Everything inside the { } applies to whatever the selector picks out."
		if strings.Contains(sel, ":hover") {
			why += " :hover means while the mouse is over it."
		}
		return "Styles for " + joinAnd(parts) + ".", why
	}).open("rule", 0)
}

var reShTest = regexp.MustCompile(`^(!\s*)?-([dfezn])\s+(.+)$`)
var reShCmp = regexp.MustCompile(`^(.+?)\s+(=|==|!=|-eq|-ne|-lt|-gt|-le|-ge)\s+(.+)$`)

// shellCond reads a [ … ] test in words.
func shellCond(x *ctx, c string) string {
	c = strings.TrimSpace(c)
	if m := reShTest.FindStringSubmatch(c); m != nil {
		v := code(unhide(m[3], x.strs))
		not := m[1] != ""
		d := map[string][2]string{"d": {"the folder " + v + " exists", "the folder " + v + " doesn't exist"}, "f": {"the file " + v + " exists", "the file " + v + " doesn't exist"}, "e": {v + " exists", v + " doesn't exist"}, "z": {v + " is empty", v + " isn't empty"}, "n": {v + " isn't empty", v + " is empty"}}[m[2]]
		if not {
			return d[1]
		}
		return d[0]
	}
	if m := reShCmp.FindStringSubmatch(c); m != nil {
		op := map[string]string{"=": "is the same text as", "==": "is the same text as", "!=": "is different from", "-eq": "equals", "-ne": "is not", "-lt": "is less than", "-gt": "is more than", "-le": "is at most", "-ge": "is at least"}[m[2]]
		return code(unhide(m[1], x.strs)) + " " + op + " " + code(unhide(m[3], x.strs))
	}
	return ""
}

func batchCond(c string) string {
	c = strings.TrimSpace(c)
	lo := strings.ToLower(c)
	not := ""
	if strings.HasPrefix(lo, "not ") {
		not, c, lo = "not ", strings.TrimSpace(c[4:]), strings.TrimSpace(lo[4:])
	}
	switch {
	case strings.HasPrefix(lo, "exist "):
		return "the file " + code(c[6:]) + " does " + not + "exist"
	case strings.HasPrefix(lo, "defined "):
		return "the variable " + code(c[8:]) + " is " + not + "set"
	case strings.HasPrefix(lo, "errorlevel "):
		return "the last command's exit code is " + not + "at least " + code(c[11:])
	}
	return not + code(c)
}

func itoa(n int) string { return strconv.Itoa(n) }

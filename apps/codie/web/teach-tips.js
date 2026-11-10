// Codie teaching-mode explanations, shown by teach.js. Created by nrmluvdev.
window.TEACH = {
  app: "Codie",
  intro: "Point at any button, menu or tab in Codie to learn what it does.",
  tips: {
    // top bar
    "Language": ["Language", "Which programming language the code is in. Leave it on Auto-detect and Codie works it out; pick one yourself if it guesses wrong (for example C versus C++, or Python versus GDScript)."],
    "Examples": ["Examples", "Ready-made code to explore: everyday programs, Windows Go, game engines (Unity, Godot, LÖVE, PICO-8, Roblox, pygame, Unreal, GameMaker…) and BASIC for older machines like the Commodore 64, ZX Spectrum and BBC Micro."],
    "Open a code file|Open file": ["Open a file", "Loads a code file from your computer into the editor. You can also drag a file onto the code box, or open files with Codie from Explorer or Finder."],
    "Explain (Ctrl+Enter)|Explain": ["Explain", "Explains the code now. With Live explain switched on, Codie also explains as you type. Shortcut: Ctrl+Enter."],
    "Clear the code|Clear": ["Clear", "Empties the code box so you can start again."],
    "Save the report|Save report": ["Save report", "Saves the whole explanation (line by line, outline, concepts, warnings and game logic) as a Markdown text file you can keep, print or share."],
    "Teaching tips (F2)": ["Teaching tips", "Turns these explanations on or off. The F2 key does the same."],
    // code pane
    "Code to explain|Your code": ["Your code", "Paste or type code here. Click a line to see its explanation on the right; the number on the left of each line also jumps there. Orange dots mark game logic, yellow dots warnings."],
    "Detected language": ["Detected language", "The language Codie thinks this is, the flavour (such as Windows Go, Unity or Commodore 64 BASIC), and how sure it is."],
    "Live explain": ["Live explain", "When ticked, the explanation updates as you type. Untick it for very long files and press Explain when you're ready."],
    // tabs
    "Explain": ["Explain", "Every line of your code, in order: what it does and why it's there. Click a card to jump to that line."],
    "Outline": ["Outline", "The shape of the program: its functions, classes, loops and checks, nested the way the code is. Click one to jump to it."],
    "Concepts": ["Concepts", "The programming ideas this code uses (variables, loops, functions, classes, error handling…), each with a short explanation and the lines that use it."],
    "Game logic": ["Game logic", "If this is game code: which engine or machine it's for, how that engine's game loop runs one frame, and where the code handles input, movement, collisions, spawning, timers and score."],
    "Warnings": ["Warnings", "Common mistakes and risky habits Codie spotted, with a tip for each. Codie doesn't run your code, so this is a helpful check, not a guarantee."],
    "Game school": ["Game school", "Short lessons on the ideas every game uses: the game loop, delta time, input, gravity and jumping, collisions, game states, spawning, timers, randomness, score and animation, plus games on 8-bit machines. Each has examples in many languages that Codie can explain line by line."],
    // explain options
    "Show why": ["Show why", "Shows or hides the second sentence on each card that says why the line is there."],
    "Show closing lines": ["Show closing lines", "Shows or hides cards for lines that only close a block, such as } or end."],
    "Show notes": ["Show notes", "Shows or hides the extra notes about well-known functions a line uses."],
    // game school
    "Lesson": ["Lesson", "Opens this Game school lesson: the idea in plain words, then working examples in several languages."],
    "Example language": ["Example", "Shows the same idea written for this engine or language, so you can compare how each one does it."],
    "Explain this example": ["Explain this example", "Puts this example into the editor and explains it line by line."],
    "Copy this example|Copy": ["Copy", "Copies the example's code so you can paste it into your own project."],
  },
};
